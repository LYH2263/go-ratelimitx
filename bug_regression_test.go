package ratelimitx_test

import (
	"errors"
	"testing"
	"time"

	"github.com/LYH2263/go-ratelimitx"
)

func TestBug01_AllowSurfacesPersistError(t *testing.T) {
	clk := ratelimitx.NewFakeClock(time.Unix(0, 0).UTC())
	eng := ratelimitx.New(ratelimitx.WithClock(clk), ratelimitx.WithDefaultPolicy("default"))
	if err := eng.Register(ratelimitx.Spec{
		Name: "default",
		Rate: &ratelimitx.Rate{PerSecond: 10, Burst: 10},
	}); err != nil {
		t.Fatal(err)
	}
	eng.UseFailingBackend(errors.New("disk full"))
	key := "acme|/api|10.0.0.1"
	d := eng.AllowN(key, 1)
	if d.Err == nil {
		t.Fatal("Allow must return the persist/store.Set error; in-memory consume with a nil error desyncs Peek from durable state")
	}
	p := eng.Peek(key)
	if p.Tokens < 9.5 {
		t.Fatalf("persist failure must roll back limiter memory, tokens=%v", p.Tokens)
	}
}
