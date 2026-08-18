package ratelimitx_test

import (
	"testing"
	"time"

	"github.com/LYH2263/go-ratelimitx"
)

func TestBug05_ChargeAfterCloseNoPanic(t *testing.T) {
	clk := ratelimitx.NewFakeClock(time.Unix(0, 0).UTC())
	eng := ratelimitx.New(ratelimitx.WithClock(clk), ratelimitx.WithDefaultPolicy("billed"))
	if err := eng.Register(ratelimitx.Spec{
		Name:  "billed",
		Quota: &ratelimitx.QuotaPlan{Period: time.Hour, Hard: 100},
	}); err != nil {
		t.Fatal(err)
	}
	res := eng.Charge("acme", 10)
	if !res.OK {
		t.Fatalf("charge: %v", res.Err)
	}
	if err := eng.Close(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("Charge/Refund after Close must not panic: %v", rec)
		}
	}()
	res2 := eng.Charge("acme", 1)
	if res2.Err != ratelimitx.ErrClosed {
		t.Fatalf("Charge after Close want ErrClosed, got ok=%v err=%v", res2.OK, res2.Err)
	}
	ref := eng.Refund(res.Receipt)
	if ref.Err != ratelimitx.ErrClosed {
		t.Fatalf("Refund after Close want ErrClosed, got ok=%v err=%v", ref.OK, ref.Err)
	}
}
