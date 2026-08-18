package policy_test

import (
	"testing"

	"github.com/LYH2263/go-ratelimitx/internal/composite"
	"github.com/LYH2263/go-ratelimitx/internal/policy"
)

func TestParseAndResolve(t *testing.T) {
	src := `
policy default
  algo tokenbucket
  rate 100/s
  burst 20
policy api
  algo slidinglog
  window 1s
  limit 5
bind acme|/pay|* api
bind *|*|* default
`
	tab := policy.NewTable("default")
	if err := tab.LoadText(src); err != nil {
		t.Fatal(err)
	}
	s, pat, ok := tab.Resolve(composite.MustParse("acme|/pay|1.2.3.4"))
	if !ok || s.Name != "api" || pat != "acme|/pay|*" {
		t.Fatalf("name=%s pat=%s ok=%v", s.Name, pat, ok)
	}
	s, _, ok = tab.Resolve(composite.MustParse("other|/x|1.1.1.1"))
	if !ok || s.Name != "default" {
		t.Fatalf("fallback %s", s.Name)
	}
	if s.Rate == nil || s.Rate.Burst != 20 {
		t.Fatalf("%+v", s.Rate)
	}
}

func TestValidate(t *testing.T) {
	err := (policy.Spec{Name: "x"}).Validate()
	if err == nil {
		t.Fatal("expected error")
	}
}
