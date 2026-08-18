package units_test

import (
	"testing"
	"time"

	"github.com/LYH2263/go-ratelimitx/internal/units"
)

func TestParseRate(t *testing.T) {
	p, err := units.ParseRate("100/s")
	if err != nil || p.Count != 100 || p.Period != time.Second {
		t.Fatalf("%+v %v", p, err)
	}
	if p.TokensPerSecond() != 100 {
		t.Fatal(p.TokensPerSecond())
	}
	p, err = units.ParseRate("60/m")
	if err != nil || p.TokensPerSecond() != 1 {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestParseDuration(t *testing.T) {
	d, err := units.ParseDuration("24h")
	if err != nil || d != 24*time.Hour {
		t.Fatal(d, err)
	}
}
