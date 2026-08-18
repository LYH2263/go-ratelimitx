package clock_test

import (
	"testing"
	"time"

	"github.com/LYH2263/go-ratelimitx/internal/clock"
)

func TestFakeAdvance(t *testing.T) {
	start := time.Unix(1000, 0).UTC()
	f := clock.NewFake(start)
	if !f.Now().Equal(start) {
		t.Fatalf("now = %v", f.Now())
	}
	f.Advance(250 * time.Millisecond)
	if got := f.Now().Sub(start); got != 250*time.Millisecond {
		t.Fatalf("delta = %v", got)
	}
	f.Advance(-time.Second) // no-op
	if got := f.Now().Sub(start); got != 250*time.Millisecond {
		t.Fatalf("negative advance changed clock: %v", got)
	}
}

func TestTruncatePeriod(t *testing.T) {
	tm := time.Unix(1000, 0).UTC()
	start := clock.Truncate(tm, time.Minute)
	if start.Unix()%60 != 0 {
		t.Fatalf("not aligned: %v", start)
	}
	s, e := clock.AlignPeriod(tm, time.Minute)
	if !clock.InPeriod(tm, s, time.Minute) {
		t.Fatal("expected in period")
	}
	if clock.InPeriod(e, s, time.Minute) {
		t.Fatal("end is exclusive")
	}
}

func TestSinceClockRewind(t *testing.T) {
	a := time.Unix(10, 0)
	b := time.Unix(5, 0)
	if d := clock.Since(b, a); d != 0 {
		t.Fatalf("rewind since = %v", d)
	}
}
