package ledger_test

import (
	"testing"
	"time"

	"github.com/LYH2263/go-ratelimitx/internal/clock"
	"github.com/LYH2263/go-ratelimitx/internal/ledger"
)

func TestChargeRefundPair(t *testing.T) {
	clk := clock.NewFake(time.Unix(0, 0).UTC())
	l := ledger.New(clk, ledger.Plan{Period: time.Hour, Soft: 5, Hard: 10})
	c := l.Charge("acme", 8)
	if !c.OK || c.Used != 8 || !c.SoftHit {
		t.Fatalf("%+v", c)
	}
	if l.Charge("acme", 3).OK {
		t.Fatal("should exceed hard")
	}
	rf := l.Refund(c.Receipt)
	if !rf.OK || rf.Used != 0 {
		t.Fatalf("%+v", rf)
	}
	rf2 := l.Refund(c.Receipt)
	if rf2.OK {
		t.Fatal("double refund")
	}
}

func TestNoNegative(t *testing.T) {
	clk := clock.NewFake(time.Unix(0, 0).UTC())
	l := ledger.New(clk, ledger.Plan{Period: time.Hour, Hard: 10})
	c := l.Charge("t", 4)
	if !c.OK {
		t.Fatal()
	}
	_ = l.Refund(c.Receipt)
	a := l.Peek("t")
	if a.Used < 0 {
		t.Fatalf("negative used %d", a.Used)
	}
}

func TestDebtMode(t *testing.T) {
	clk := clock.NewFake(time.Unix(0, 0).UTC())
	l := ledger.New(clk, ledger.Plan{Period: time.Hour, Hard: 10, Debt: true})
	c := l.Charge("t", 15)
	if !c.OK || c.Debt != 5 {
		t.Fatalf("%+v", c)
	}
}

func TestPeriodRollover(t *testing.T) {
	clk := clock.NewFake(time.Unix(0, 0).UTC())
	l := ledger.New(clk, ledger.Plan{Period: time.Second, Hard: 3})
	if !l.Charge("t", 3).OK {
		t.Fatal()
	}
	if l.Charge("t", 1).OK {
		t.Fatal()
	}
	clk.Advance(time.Second)
	if !l.Charge("t", 3).OK {
		t.Fatal("new period")
	}
}
