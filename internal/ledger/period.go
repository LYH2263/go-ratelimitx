package ledger

import (
	"time"

	iclk "github.com/LYH2263/go-ratelimitx/internal/clock"
)

// Align 返回 now 所在配额周期的起点。
func Align(now time.Time, period time.Duration) time.Time {
	return iclk.Truncate(now, period)
}

// Next 返回下一周期起点。
func Next(now time.Time, period time.Duration) time.Time {
	return iclk.NextBoundary(now, period)
}

// RemainingInPeriod 返回距本周期结束的时长。
func RemainingInPeriod(now time.Time, period time.Duration) time.Duration {
	end := Align(now, period).Add(period)
	d := end.Sub(now)
	if d < 0 {
		return 0
	}
	return d
}
