package ratelimitx_test

import (
	"context"
	"testing"
	"time"

	"github.com/LYH2263/go-ratelimitx"
)

func TestBug04_WaitHonorsContextCancel(t *testing.T) {
	clk := ratelimitx.NewFakeClock(time.Unix(0, 0).UTC())
	eng := ratelimitx.New(ratelimitx.WithClock(clk), ratelimitx.WithDefaultPolicy("default"))
	if err := eng.Register(ratelimitx.Spec{
		Name: "default",
		Rate: &ratelimitx.Rate{PerSecond: 10, Burst: 10},
	}); err != nil {
		t.Fatal(err)
	}
	key := "acme|/api|10.0.0.1"
	if ok, _ := eng.Allow(key, 10); !ok {
		t.Fatal("drain burst")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	err := eng.Wait(ctx, key, 1)
	if err != context.Canceled {
		t.Fatalf("Wait must return context.Canceled, got %v", err)
	}
	if time.Since(start) > 50*time.Millisecond {
		t.Fatal("already-canceled ctx must not sleep the refill interval")
	}
}
