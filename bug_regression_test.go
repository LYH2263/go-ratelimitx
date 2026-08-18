package ratelimitx_test

import (
	"testing"
	"time"

	"github.com/LYH2263/go-ratelimitx"
)

func TestBug08_CancelRestoresWindow(t *testing.T) {
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
	r := eng.Reserve(key, 2)
	if !r.OK() {
		t.Fatal("reserve")
	}
	if ok, _ := eng.Allow(key, 1); ok {
		t.Fatal("window should be full after reserve")
	}
	r.Cancel()
	if ok, _ := eng.Allow(key, 2); !ok {
		t.Fatal("Cancel must restore window slots as well as rate tokens")
	}
}
