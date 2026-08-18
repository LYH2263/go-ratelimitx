package ratelimitx_test

import (
	"testing"
	"time"

	"github.com/LYH2263/go-ratelimitx"
)

func newEngine(t *testing.T, clk *ratelimitx.FakeClock) *ratelimitx.Engine {
	t.Helper()
	eng := ratelimitx.New(
		ratelimitx.WithClock(clk),
		ratelimitx.WithShards(8),
		ratelimitx.WithDefaultPolicy("default"),
	)
	err := eng.Register(ratelimitx.Spec{
		Name: "default",
		Rate: &ratelimitx.Rate{PerSecond: 10, Burst: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	return eng
}

func TestAllowBurstAndRefill(t *testing.T) {
	clk := ratelimitx.NewFakeClock(time.Unix(0, 0).UTC())
	eng := newEngine(t, clk)
	key := "acme|/api|10.0.0.1"
	for i := 0; i < 10; i++ {
		ok, _ := eng.Allow(key, 1)
		if !ok {
			t.Fatalf("burst i=%d", i)
		}
	}
	ok, wait := eng.Allow(key, 1)
	if ok {
		t.Fatal("expected deny")
	}
	if wait <= 0 {
		t.Fatalf("wait=%v", wait)
	}
	clk.Advance(100 * time.Millisecond)
	ok, _ = eng.Allow(key, 1)
	if !ok {
		t.Fatal("expected refill")
	}
}

func TestPeekResetReserve(t *testing.T) {
	clk := ratelimitx.NewFakeClock(time.Unix(0, 0).UTC())
	eng := newEngine(t, clk)
	key := "t|/r|1.1.1.1"
	p := eng.Peek(key)
	if p.Burst != 10 {
		t.Fatalf("burst=%d", p.Burst)
	}
	r := eng.Reserve(key, 10)
	if !r.OK() {
		t.Fatal("reserve")
	}
	ok, _ := eng.Allow(key, 1)
	if ok {
		t.Fatal("should be empty")
	}
	r.Cancel()
	ok, _ = eng.Allow(key, 10)
	if !ok {
		t.Fatal("after cancel")
	}
	eng.Reset(key)
	ok, _ = eng.Allow(key, 10)
	if !ok {
		t.Fatal("after reset")
	}
}

func TestSlidingWindowPolicy(t *testing.T) {
	clk := ratelimitx.NewFakeClock(time.Unix(0, 0).UTC())
	eng := ratelimitx.New(ratelimitx.WithClock(clk), ratelimitx.WithDefaultPolicy("win"))
	if err := eng.Register(ratelimitx.Spec{
		Name:   "win",
		Algo:   ratelimitx.AlgoSlidingLog,
		Window: &ratelimitx.Window{Size: time.Second, Limit: 3},
	}); err != nil {
		t.Fatal(err)
	}
	key := "k"
	for i := 0; i < 3; i++ {
		if ok, _ := eng.Allow(key, 1); !ok {
			t.Fatalf("i=%d", i)
		}
	}
	if ok, _ := eng.Allow(key, 1); ok {
		t.Fatal("over window")
	}
	clk.Advance(time.Second)
	if ok, _ := eng.Allow(key, 1); !ok {
		t.Fatal("rollover")
	}
}

func TestCompositeFallbackBind(t *testing.T) {
	clk := ratelimitx.NewFakeClock(time.Unix(0, 0).UTC())
	eng := ratelimitx.New(ratelimitx.WithClock(clk), ratelimitx.WithDefaultPolicy("slow"))
	if err := eng.Register(ratelimitx.Spec{
		Name: "slow",
		Rate: &ratelimitx.Rate{PerSecond: 1, Burst: 1},
	}); err != nil {
		t.Fatal(err)
	}
	if err := eng.Register(ratelimitx.Spec{
		Name: "fast",
		Rate: &ratelimitx.Rate{PerSecond: 100, Burst: 5},
	}); err != nil {
		t.Fatal(err)
	}
	if err := eng.Bind("acme|/pay|*", "fast"); err != nil {
		t.Fatal(err)
	}
	// exact ip still falls back to acme|/pay|*
	ok, _ := eng.Allow("acme|/pay|9.9.9.9", 5)
	if !ok {
		t.Fatal("fast bind")
	}
	ok, _ = eng.Allow("other|/pay|9.9.9.9", 2)
	if ok {
		t.Fatal("slow default burst=1")
	}
	chain := ratelimitx.FallbackPatterns(ratelimitx.NewKey("acme", "/pay", "9.9.9.9"))
	if chain[1] != "acme|/pay|*" {
		t.Fatalf("%v", chain)
	}
}

func TestQuotaChargeRefund(t *testing.T) {
	clk := ratelimitx.NewFakeClock(time.Unix(0, 0).UTC())
	eng := ratelimitx.New(ratelimitx.WithClock(clk), ratelimitx.WithDefaultPolicy("q"))
	if err := eng.Register(ratelimitx.Spec{
		Name:  "q",
		Quota: &ratelimitx.QuotaPlan{Period: time.Hour, Soft: 3, Hard: 5},
	}); err != nil {
		t.Fatal(err)
	}
	c := eng.Charge("acme", 4)
	if !c.OK || !c.SoftHit {
		t.Fatalf("%+v", c)
	}
	c2 := eng.Charge("acme", 2)
	if c2.OK {
		t.Fatal("hard")
	}
	rf := eng.Refund(c.Receipt)
	if !rf.OK || rf.Used != 0 {
		t.Fatalf("%+v", rf)
	}
	rf2 := eng.Refund(c.Receipt)
	if rf2.OK || rf2.Err != ratelimitx.ErrReceiptUsed {
		t.Fatalf("%+v", rf2)
	}
}

func TestQuotaDebt(t *testing.T) {
	clk := ratelimitx.NewFakeClock(time.Unix(0, 0).UTC())
	eng := ratelimitx.New(ratelimitx.WithClock(clk), ratelimitx.WithDefaultPolicy("d"))
	if err := eng.Register(ratelimitx.Spec{
		Name:  "d",
		Quota: &ratelimitx.QuotaPlan{Period: time.Hour, Hard: 2, Debt: true},
	}); err != nil {
		t.Fatal(err)
	}
	c := eng.Charge("t", 5)
	if !c.OK || c.Debt != 3 {
		t.Fatalf("%+v", c)
	}
}

func TestMetricsAndShards(t *testing.T) {
	clk := ratelimitx.NewFakeClock(time.Unix(0, 0).UTC())
	eng := newEngine(t, clk)
	eng.Allow("a", 1)
	eng.Allow("b", 1)
	snap := eng.Metrics()
	if snap.Allowed < 2 {
		t.Fatalf("%+v", snap)
	}
	if eng.Shards() != 8 {
		t.Fatalf("shards=%d", eng.Shards())
	}
	if eng.ShardOf("a") == eng.ShardOf("never-same-maybe") {
		// hash collision possible; only check range
	}
	if eng.ShardOf("a") < 0 || eng.ShardOf("a") >= 8 {
		t.Fatal("shard out of range")
	}
}

func TestGCRAEngine(t *testing.T) {
	clk := ratelimitx.NewFakeClock(time.Unix(0, 0).UTC())
	eng := ratelimitx.New(ratelimitx.WithClock(clk), ratelimitx.WithDefaultPolicy("g"))
	if err := eng.Register(ratelimitx.Spec{
		Name: "g",
		Algo: ratelimitx.AlgoGCRA,
		Rate: &ratelimitx.Rate{PerSecond: 10, Burst: 4},
	}); err != nil {
		t.Fatal(err)
	}
	okN := 0
	for i := 0; i < 6; i++ {
		if ok, _ := eng.Allow("x", 1); ok {
			okN++
		}
	}
	if okN != 4 {
		t.Fatalf("gcra allowed %d", okN)
	}
}

func TestLoadText(t *testing.T) {
	clk := ratelimitx.NewFakeClock(time.Unix(0, 0).UTC())
	eng := ratelimitx.New(ratelimitx.WithClock(clk), ratelimitx.WithDefaultPolicy("default"))
	src := `
policy default
  rate 50/s
  burst 2
`
	if err := eng.LoadText(src); err != nil {
		t.Fatal(err)
	}
	ok, _ := eng.Allow("z", 2)
	if !ok {
		t.Fatal()
	}
	ok, _ = eng.Allow("z", 1)
	if ok {
		t.Fatal()
	}
}

func TestImpossibleAllow(t *testing.T) {
	clk := ratelimitx.NewFakeClock(time.Unix(0, 0).UTC())
	eng := newEngine(t, clk)
	ok, wait := eng.Allow("k", 11)
	if ok || wait != ratelimitx.ImpossibleWait {
		t.Fatalf("ok=%v wait=%v", ok, wait)
	}
}

func TestInvalidN(t *testing.T) {
	clk := ratelimitx.NewFakeClock(time.Unix(0, 0).UTC())
	eng := newEngine(t, clk)
	d := eng.AllowN("k", 0)
	if d.OK || d.Err != ratelimitx.ErrInvalidN {
		t.Fatalf("%+v", d)
	}
}

func TestCombinedRateAndWindow(t *testing.T) {
	clk := ratelimitx.NewFakeClock(time.Unix(0, 0).UTC())
	eng := ratelimitx.New(ratelimitx.WithClock(clk), ratelimitx.WithDefaultPolicy("both"))
	if err := eng.Register(ratelimitx.Spec{
		Name:   "both",
		Rate:   &ratelimitx.Rate{PerSecond: 100, Burst: 10},
		Window: &ratelimitx.Window{Size: time.Second, Limit: 2},
	}); err != nil {
		t.Fatal(err)
	}
	key := "combo"
	if ok, _ := eng.Allow(key, 1); !ok {
		t.Fatal("first")
	}
	if ok, _ := eng.Allow(key, 1); !ok {
		t.Fatal("second")
	}
	if ok, _ := eng.Allow(key, 1); ok {
		t.Fatal("window should deny third")
	}
	p := eng.Peek(key)
	if p.Tokens < 8 {
		t.Fatalf("rate tokens should be restored after window deny, tokens=%v", p.Tokens)
	}
}

func TestSlidingCounterEngine(t *testing.T) {
	clk := ratelimitx.NewFakeClock(time.Unix(0, 0).UTC())
	eng := ratelimitx.New(ratelimitx.WithClock(clk), ratelimitx.WithDefaultPolicy("c"))
	if err := eng.Register(ratelimitx.Spec{
		Name:   "c",
		Algo:   ratelimitx.AlgoSlidingCounter,
		Window: &ratelimitx.Window{Size: time.Second, Limit: 4, Buckets: 10},
	}); err != nil {
		t.Fatal(err)
	}
	okN := 0
	for i := 0; i < 6; i++ {
		if ok, _ := eng.Allow("c", 1); ok {
			okN++
		}
	}
	if okN != 4 {
		t.Fatalf("counter allowed %d", okN)
	}
	clk.Advance(time.Second)
	if ok, _ := eng.Allow("c", 1); !ok {
		t.Fatal("after window")
	}
}
