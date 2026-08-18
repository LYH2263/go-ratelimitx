package ratelimitx_test

import (
	"testing"
	"time"

	"github.com/LYH2263/go-ratelimitx"
)

func TestBug09_ResolveErrorDoesNotAllowAll(t *testing.T) {
	clk := ratelimitx.NewFakeClock(time.Unix(0, 0).UTC())
	eng := ratelimitx.New(ratelimitx.WithClock(clk), ratelimitx.WithDefaultPolicy("strict"))
	if err := eng.Register(ratelimitx.Spec{
		Name: "strict",
		Rate: &ratelimitx.Rate{PerSecond: 1, Burst: 1},
	}); err != nil {
		t.Fatal(err)
	}
	if err := eng.Bind("*|*|*", "strict"); err != nil {
		t.Fatal(err)
	}
	if !eng.Unregister("strict") {
		t.Fatal("unregister")
	}
	key := "acme|/api|10.0.0.1"
	ok, _ := eng.Allow(key, 100)
	if ok {
		t.Fatal("dangling bind / resolve error must not fall through to a zero policy that allows all")
	}
	d := eng.AllowN(key, 1)
	if d.Err != ratelimitx.ErrUnknownPolicy {
		t.Fatalf("want ErrUnknownPolicy, got ok=%v err=%v", d.OK, d.Err)
	}
}
