package clock

import "time"

// AlignPeriod 返回 t 所在周期的 [start, end)。end = start + period。
func AlignPeriod(t time.Time, period time.Duration) (start, end time.Time) {
	start = Truncate(t, period)
	end = start.Add(period)
	return start, end
}

// InPeriod 判断 t 是否落在 [start, start+period)。
func InPeriod(t, start time.Time, period time.Duration) bool {
	if period <= 0 {
		return false
	}
	if t.Before(start) {
		return false
	}
	return t.Before(start.Add(period))
}

// NextBoundary 返回 t 之后的下一个周期边界（不含 t 自身若已在边界上则仍返回 t+period）。
func NextBoundary(t time.Time, period time.Duration) time.Time {
	start := Truncate(t, period)
	next := start.Add(period)
	if !t.Before(next) {
		return next.Add(period)
	}
	return next
}

// ElapsedInPeriod 返回 t 相对周期起点已过去的时长，钳到 [0, period]。
func ElapsedInPeriod(t time.Time, period time.Duration) time.Duration {
	start := Truncate(t, period)
	el := t.Sub(start)
	if el < 0 {
		return 0
	}
	if el > period {
		return period
	}
	return el
}
