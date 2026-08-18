package ratelimitx_test

import (
	"testing"
	"time"

	"github.com/LYH2263/go-ratelimitx"
)

func TestBug07_ResetClearsWindowOccupancy(t *testing.T) {
	clk := ratelimitx.NewFakeClock(time.Unix(0, 0).UTC())
	eng := ratelimitx.New(ratelimitx.WithClock(clk), ratelimitx.WithDefaultPolicy("both"))
	if err := eng.Register(ratelimitx.Spec{
		Name:   "both",
		Rate:   &ratelimitx.Rate{PerSecond: 100, Burst: 10},
		Window: &ratelimitx.Window{Size: time.Second, Limit: 2},
	}); err != nil {
		t.Fatal(err)
	}
	key := "acme|/api|10.0.0.1"
	if ok, _ := eng.Allow(key, 2); !ok {
		t.Fatal("fill window")
	}
	if ok, _ := eng.Allow(key, 1); ok {
		t.Fatal("window should be full")
	}
	eng.Reset(key)
	if ok, _ := eng.Allow(key, 2); !ok {
		t.Fatal("Reset must clear sliding-window occupancy as well as the rate bucket")
	}
}
