package clock

import "time"

// Clock 提供当前时间。所有限流与账本状态变迁必须走该接口，禁止直接 time.Now。
type Clock interface {
	Now() time.Time
}

// Real 使用系统挂钟。
type Real struct{}

// Now 返回 time.Now。
func (Real) Now() time.Time { return time.Now() }

// UnixNano 把 t 转为 Unix 纳秒。零值时间返回 0。
func UnixNano(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixNano()
}

// Since 返回 now - then，若 now 早于 then 则返回 0（时钟回拨不产生负间隔）。
func Since(now, then time.Time) time.Duration {
	if now.Before(then) {
		return 0
	}
	return now.Sub(then)
}

// Add 在 t 上加 d。d < 0 时等价于回退。
func Add(t time.Time, d time.Duration) time.Time {
	return t.Add(d)
}

// Truncate 把 t 按 period 对齐到周期起点（Unix 纪元对齐）。
// period <= 0 时返回 t 本身。
func Truncate(t time.Time, period time.Duration) time.Time {
	if period <= 0 {
		return t
	}
	ns := t.UnixNano()
	p := int64(period)
	if p <= 0 {
		return t
	}
	aligned := ns - ns%p
	if ns < 0 && ns%p != 0 {
		aligned -= p
	}
	return time.Unix(0, aligned).In(t.Location())
}

// Deadline 返回 then + d，用于窗口过期时刻。
func Deadline(then time.Time, d time.Duration) time.Time {
	return then.Add(d)
}
