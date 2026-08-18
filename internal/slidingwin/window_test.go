package slidingwin_test

import (
	"testing"
	"time"

	"github.com/LYH2263/go-ratelimitx/internal/clock"
	"github.com/LYH2263/go-ratelimitx/internal/slidingwin"
)

func TestLogLimitAndRollover(t *testing.T) {
	clk := clock.NewFake(time.Unix(0, 0).UTC())
	w := slidingwin.NewLog(time.Second, 5)
	for i := 0; i < 5; i++ {
		if !w.Allow(1, clk.Now()).OK {
			t.Fatalf("i=%d", i)
		}
	}
	if w.Allow(1, clk.Now()).OK {
		t.Fatal("6th")
	}
	clk.Advance(time.Second)
	if !w.Allow(1, clk.Now()).OK {
		t.Fatal("after window should allow")
	}
}

func TestLogBoundaryExclusive(t *testing.T) {
	clk := clock.NewFake(time.Unix(0, 0).UTC())
	w := slidingwin.NewLog(time.Second, 1)
	if !w.Allow(1, clk.Now()).OK {
		t.Fatal()
	}
	clk.Advance(time.Second - time.Nanosecond)
	if w.Allow(1, clk.Now()).OK {
		t.Fatal("still inside window")
	}
	clk.Advance(time.Nanosecond)
	if !w.Allow(1, clk.Now()).OK {
		t.Fatal("exactly size later old event must expire")
	}
}

func TestRingOrder(t *testing.T) {
	r := slidingwin.NewRing(3)
	r.Push(1)
	r.Push(2)
	r.Push(3)
	r.Push(4) // overwrite 1
	got := r.Times()
	if len(got) != 3 || got[0] != 2 || got[2] != 4 {
		t.Fatalf("%v", got)
	}
	r.Prune(2)
	got = r.Times()
	if len(got) != 2 || got[0] != 3 || got[1] != 4 {
		t.Fatalf("after prune %v", got)
	}
}

func TestCounterApproximate(t *testing.T) {
	clk := clock.NewFake(time.Unix(0, 0).UTC())
	c := slidingwin.NewCounter(time.Second, 10, 10)
	for i := 0; i < 10; i++ {
		if !c.Allow(1, clk.Now()).OK {
			t.Fatalf("i=%d est", i)
		}
	}
	if c.Allow(1, clk.Now()).OK {
		t.Fatal("over limit")
	}
	clk.Advance(time.Second)
	if !c.Allow(1, clk.Now()).OK {
		t.Fatal("after full window")
	}
}
