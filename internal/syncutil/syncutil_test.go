package syncutil_test

import (
	"testing"

	"github.com/LYH2263/go-ratelimitx/internal/syncutil"
)

func TestShardedIndexRange(t *testing.T) {
	s := syncutil.New(8)
	for _, k := range []string{"a", "b", "tenant|route|ip"} {
		i := s.Index(k)
		if i < 0 || i >= 8 {
			t.Fatal(i)
		}
	}
	var hit int
	s.Do("x", func() { hit = 1 })
	if hit != 1 {
		t.Fatal()
	}
}

func TestGuard(t *testing.T) {
	g := syncutil.NewGuard(3)
	g.Do(func(v *int) { *v++ })
	if g.Get() != 4 {
		t.Fatal(g.Get())
	}
}
