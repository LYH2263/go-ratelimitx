package ratelimitx_test

import (
	"testing"
	"time"

	"github.com/LYH2263/go-ratelimitx"
)

func TestBug03_AllowAfterCloseNoPanic(t *testing.T) {
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
		t.Fatal("warmup allow")
	}
	if err := eng.Close(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("Allow after Close must not panic: %v", rec)
		}
	}()
	d := eng.AllowN(key, 1)
	if d.Err != ratelimitx.ErrClosed {
		t.Fatalf("want ErrClosed, got ok=%v err=%v", d.OK, d.Err)
	}
}
