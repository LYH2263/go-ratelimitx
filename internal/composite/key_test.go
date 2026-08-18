package composite_test

import (
	"reflect"
	"testing"

	"github.com/LYH2263/go-ratelimitx/internal/composite"
)

func TestEncodeParse(t *testing.T) {
	k := composite.Key{Tenant: "acme", Route: "/pay", IP: "1.1.1.1"}
	s := composite.Encode(k)
	if s != "acme|/pay|1.1.1.1" {
		t.Fatal(s)
	}
	got, err := composite.Parse(s)
	if err != nil || !composite.Equal(got, k) {
		t.Fatalf("%v %v", got, err)
	}
}

func TestFallbackOrder(t *testing.T) {
	k := composite.Key{Tenant: "acme", Route: "/pay", IP: "10.0.0.1"}
	got := composite.FallbackChain(k)
	want := []string{
		"acme|/pay|10.0.0.1",
		"acme|/pay|*",
		"acme|*|*",
		"*|*|*",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestFallbackMissingIP(t *testing.T) {
	k := composite.Key{Tenant: "acme", Route: "/pay"}
	got := composite.FallbackChain(k)
	if got[0] != "acme|/pay|*" {
		t.Fatalf("%v", got)
	}
	if len(got) != 3 { // acme|/pay|*, acme|*|*, *|*|*
		t.Fatalf("len=%d %v", len(got), got)
	}
}

func TestRejectSep(t *testing.T) {
	if composite.ValidDim("a|b") {
		t.Fatal()
	}
}
