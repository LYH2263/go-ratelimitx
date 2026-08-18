package tokenbucket_test

import (
	"testing"
	"time"

	"github.com/LYH2263/go-ratelimitx/internal/clock"
	"github.com/LYH2263/go-ratelimitx/internal/tokenbucket"
)

func TestBurstThenRefill(t *testing.T) {
	clk := clock.NewFake(time.Unix(0, 0).UTC())
	st := tokenbucket.NewState(tokenbucket.Config{PerSecond: 10, Burst: 10}, clk.Now())
	for i := 0; i < 10; i++ {
		r := st.Allow(1, clk.Now())
		if !r.OK {
			t.Fatalf("burst i=%d denied", i)
		}
	}
	r := st.Allow(1, clk.Now())
	if r.OK {
		t.Fatal("11th should deny")
	}
	if r.Wait <= 0 {
		t.Fatalf("wait=%v", r.Wait)
	}
	clk.Advance(100 * time.Millisecond)
	r = st.Allow(1, clk.Now())
	if !r.OK {
		t.Fatalf("after 100ms expected 1 token, wait=%v rem=%v", r.Wait, r.Remaining)
	}
}

func TestPeekDoesNotConsume(t *testing.T) {
	now := time.Unix(0, 0).UTC()
	st := tokenbucket.NewState(tokenbucket.Config{PerSecond: 1, Burst: 1}, now)
	p := st.Peek(1, now)
	if !p.OK {
		t.Fatal("peek should ok")
	}
	r := st.Allow(1, now)
	if !r.OK {
		t.Fatal("allow after peek should still ok")
	}
	r = st.Allow(1, now)
	if r.OK {
		t.Fatal("consumed")
	}
}

func TestImpossibleN(t *testing.T) {
	now := time.Unix(0, 0).UTC()
	st := tokenbucket.NewState(tokenbucket.Config{PerSecond: 5, Burst: 3}, now)
	r := st.Allow(4, now)
	if r.OK || !r.Impossible {
		t.Fatalf("got %+v", r)
	}
}

func TestRestore(t *testing.T) {
	now := time.Unix(0, 0).UTC()
	st := tokenbucket.NewState(tokenbucket.Config{PerSecond: 1, Burst: 2}, now)
	if !st.Allow(2, now).OK {
		t.Fatal()
	}
	st.Restore(2, now)
	if !st.Allow(2, now).OK {
		t.Fatal("restore failed")
	}
}

func TestGCRABurst(t *testing.T) {
	clk := clock.NewFake(time.Unix(0, 0).UTC())
	g := tokenbucket.NewGCRA(tokenbucket.Config{PerSecond: 10, Burst: 5}, clk.Now())
	ok := 0
	for i := 0; i < 8; i++ {
		if g.Allow(1, clk.Now()).OK {
			ok++
		}
	}
	if ok != 5 {
		t.Fatalf("gcra burst allowed %d", ok)
	}
	clk.Advance(200 * time.Millisecond)
	if !g.Allow(1, clk.Now()).OK {
		t.Fatal("gcra should refill ~2 tokens in 200ms")
	}
}
