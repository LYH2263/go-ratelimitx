package store_test

import (
	"testing"
	"time"

	"github.com/LYH2263/go-ratelimitx/internal/clock"
	"github.com/LYH2263/go-ratelimitx/internal/store"
)

func TestShardHashAndGetOrCreate(t *testing.T) {
	clk := clock.NewFake(time.Unix(0, 0).UTC())
	s := store.New(4, clk)
	e, created := s.GetOrCreate("alpha", store.KindTokenBucket, func() any { return 1 })
	if !created || e.Key != "alpha" {
		t.Fatalf("%+v created=%v", e, created)
	}
	_, created = s.GetOrCreate("alpha", store.KindTokenBucket, func() any { return 2 })
	if created {
		t.Fatal("second create")
	}
	if s.Index("alpha") != store.Hash("alpha", 4) {
		t.Fatal("index mismatch")
	}
	if s.Shards() != 4 {
		t.Fatal()
	}
}

func TestSweepIdle(t *testing.T) {
	clk := clock.NewFake(time.Unix(0, 0).UTC())
	s := store.New(2, clk)
	s.Put(&store.Entry{Key: "old", Kind: store.KindNone, CreatedAt: clk.Now(), UpdatedAt: clk.Now()})
	clk.Advance(time.Second)
	n := s.Sweep(500 * time.Millisecond)
	if n != 1 || s.Len() != 0 {
		t.Fatalf("n=%d len=%d", n, s.Len())
	}
}

func TestSyncutilAligned(t *testing.T) {
	if store.Hash("", 1) != 0 {
		t.Fatal()
	}
}
