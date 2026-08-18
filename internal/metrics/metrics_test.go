package metrics_test

import (
	"testing"
	"time"

	"github.com/LYH2263/go-ratelimitx/internal/metrics"
)

func TestCollector(t *testing.T) {
	c := metrics.New()
	c.Allow(true, "default", "", 0)
	c.Allow(false, "default", "rate", 50*time.Millisecond)
	c.Charge(true, true)
	s := c.Snapshot()
	if s.Allowed != 1 || s.Denied != 1 || s.SoftHits != 1 {
		t.Fatalf("%+v", s)
	}
	if s.Reason("rate") != 1 {
		t.Fatal(s.DeniedReason)
	}
	if s.DenyRatio() <= 0 {
		t.Fatal()
	}
}
