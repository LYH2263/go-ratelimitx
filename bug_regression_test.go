package ratelimitx_test

import (
	"testing"
	"time"

	"github.com/LYH2263/go-ratelimitx"
)

func TestBug02_PeekDoesNotConsume(t *testing.T) {
	clk := ratelimitx.NewFakeClock(time.Unix(0, 0).UTC())
	eng := ratelimitx.New(ratelimitx.WithClock(clk), ratelimitx.WithDefaultPolicy("default"))
	if err := eng.Register(ratelimitx.Spec{
		Name: "default",
		Rate: &ratelimitx.Rate{PerSecond: 10, Burst: 10},
	}); err != nil {
		t.Fatal(err)
	}
	key := "acme|/api|10.0.0.1"
	p := eng.Peek(key)
	if p.Tokens < 9.5 {
		t.Fatalf("Peek must not consume, tokens=%v", p.Tokens)
	}
	ok, _ := eng.Allow(key, 10)
	if !ok {
		t.Fatal("full burst must still be available after Peek")
	}
}
