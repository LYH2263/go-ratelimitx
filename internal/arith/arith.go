package arith

import "math/bits"

// MulDiv 计算 a*b/div，使用 128 位中间结果避免溢出。div==0 时返回 0。
func MulDiv(a, b, div uint64) uint64 {
	if div == 0 {
		return 0
	}
	hi, lo := bits.Mul64(a, b)
	q, _ := bits.Div64(hi, lo, div)
	return q
}

// MulDivRoundUp 计算 ceil(a*b/div)。
func MulDivRoundUp(a, b, div uint64) uint64 {
	if div == 0 {
		return 0
	}
	hi, lo := bits.Mul64(a, b)
	q, r := bits.Div64(hi, lo, div)
	if r > 0 {
		if q == ^uint64(0) {
			return q
		}
		return q + 1
	}
	return q
}

// MinU64 返回较小的无符号整数。
func MinU64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// MaxU64 返回较大的无符号整数。
func MaxU64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

// MinI64 返回较小的有符号整数。
func MinI64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// MaxI64 返回较大的有符号整数。
func MaxI64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// ClampI64 把 v 钳到 [lo, hi]。若 lo > hi 则返回 lo。
func ClampI64(v, lo, hi int64) int64 {
	if lo > hi {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ClampU64 把 v 钳到 [lo, hi]。
func ClampU64(v, lo, hi uint64) uint64 {
	if lo > hi {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// SaturatingAddI64 饱和加法，避免溢出。
func SaturatingAddI64(a, b int64) int64 {
	if b > 0 && a > (1<<63-1)-b {
		return 1<<63 - 1
	}
	if b < 0 && a < (-1<<63)-b {
		return -1 << 63
	}
	return a + b
}

// SaturatingSubI64 饱和减法。
func SaturatingSubI64(a, b int64) int64 {
	if b < 0 {
		return SaturatingAddI64(a, -b)
	}
	if a < (-1<<63)+b {
		return -1 << 63
	}
	return a - b
}

// SaturatingAddU64 饱和无符号加法。
func SaturatingAddU64(a, b uint64) uint64 {
	if a > ^uint64(0)-b {
		return ^uint64(0)
	}
	return a + b
}

// SubNonNeg 返回 max(a-b, 0)。
func SubNonNeg(a, b int64) int64 {
	if a <= b {
		return 0
	}
	return a - b
}

// CeilDivU64 返回 ceil(a/b)。b==0 时返回 0。
func CeilDivU64(a, b uint64) uint64 {
	if b == 0 {
		return 0
	}
	return (a + b - 1) / b
}
