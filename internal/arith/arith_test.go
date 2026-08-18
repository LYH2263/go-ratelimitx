package arith_test

import (
	"testing"

	"github.com/LYH2263/go-ratelimitx/internal/arith"
)

func TestMulDiv(t *testing.T) {
	if arith.MulDiv(10, 2, 4) != 5 {
		t.Fatal()
	}
	if arith.MulDivRoundUp(10, 1, 3) != 4 {
		t.Fatal(arith.MulDivRoundUp(10, 1, 3))
	}
	if arith.MulDiv(1, 1, 0) != 0 {
		t.Fatal()
	}
}

func TestSaturating(t *testing.T) {
	if arith.SaturatingAddU64(^uint64(0), 1) != ^uint64(0) {
		t.Fatal()
	}
	if arith.SubNonNeg(3, 5) != 0 {
		t.Fatal()
	}
}
