package ratelimitx_test

import (
	"testing"
	"time"

	"github.com/LYH2263/go-ratelimitx"
)

func TestBug06_CloneDoesNotShareBucket(t *testing.T) {
	clk := ratelimitx.NewFakeClock(time.Unix(0, 0).UTC())
	eng := ratelimitx.New(ratelimitx.WithClock(clk), ratelimitx.WithDefaultPolicy("default"))
	if err := eng.Register(ratelimitx.Spec{
		Name: "default",
		Rate: &ratelimitx.Rate{PerSecond: 10, Burst: 10},
	}); err != nil {
		t.Fatal(err)
	}
	key := "acme|/api|10.0.0.1"
	if ok, _ := eng.Allow(key, 1); !ok {
		t.Fatal("warmup")
	}
	cl := eng.Clone()
	if ok, _ := cl.Allow(key, 9); !ok {
		t.Fatal("clone should still have 9 tokens")
	}
	ok, _ := eng.Allow(key, 9)
	if !ok {
		t.Fatal("mutating clone must not drain the live engine token bucket")
	}
}
