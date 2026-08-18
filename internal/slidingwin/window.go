package slidingwin

import "time"

// Log 是精确滑动窗口：窗口内事件时间戳个数不超过 Limit。
// 有效事件满足 now.Sub(ts) < Size，边界上恰好 Size 时旧事件离开（避免 off-by-one 把边界算进窗口）。
type Log struct {
	Size  time.Duration
	Limit int
	ring  *Ring
}

// NewLog 创建空窗口。limit<1 时按 1；size<=0 时按 1s。
func NewLog(size time.Duration, limit int) *Log {
	if size <= 0 {
		size = time.Second
	}
	if limit < 1 {
		limit = 1
	}
	return &Log{Size: size, Limit: limit, ring: NewRing(limit)}
}

// Result 是窗口判定结果。
type Result struct {
	OK         bool
	Wait       time.Duration
	InWindow   int
	Impossible bool
}

// cutoff 返回 now-size 的纳秒。ts > cutoff 才算在窗口内。
func (l *Log) cutoff(now time.Time) int64 {
	return now.Add(-l.Size).UnixNano()
}

// Allow 尝试记录 n 个发生在 now 的事件。
func (l *Log) Allow(n int, now time.Time) Result {
	if n <= 0 {
		return Result{}
	}
	if n > l.Limit {
		return Result{Wait: -1, Impossible: true, InWindow: l.ring.Len()}
	}
	cut := l.cutoff(now)
	l.ring.Prune(cut)
	in := l.ring.Len()
	if in+n <= l.Limit {
		l.ring.PushN(now.UnixNano(), n)
		return Result{OK: true, InWindow: l.ring.Len()}
	}
	return Result{OK: false, Wait: l.wait(n, now, cut), InWindow: in}
}

// Peek 不写入。
func (l *Log) Peek(n int, now time.Time) Result {
	if n <= 0 {
		return Result{}
	}
	if n > l.Limit {
		return Result{Wait: -1, Impossible: true, InWindow: l.ring.Len()}
	}
	cut := l.cutoff(now)
	in := l.ring.CountAfter(cut)
	if in+n <= l.Limit {
		return Result{OK: true, InWindow: in}
	}
	wait := time.Duration(0)
	// 需要挤出 extra = in+n-limit 个最旧仍有效事件
	extra := in + n - l.Limit
	seen := 0
	for i := 0; i < l.ring.Len(); i++ {
		ts := l.ring.At(i)
		if ts <= cut {
			continue
		}
		seen++
		if seen == extra {
			exp := time.Unix(0, ts).Add(l.Size)
			wait = exp.Sub(now)
			if wait < 0 {
				wait = 0
			}
			break
		}
	}
	return Result{OK: false, Wait: wait, InWindow: in}
}

// Restore 弹出最多 n 条最新事件。
func (l *Log) Restore(n int) int {
	return l.ring.PopNewestN(n)
}

// InWindow 返回当前环长度（调用方应先 Prune 或接受可能含过期项）。
func (l *Log) InWindow() int { return l.ring.Len() }

func (l *Log) wait(n int, now time.Time, cut int64) time.Duration {
	in := l.ring.Len()
	extra := in + n - l.Limit
	if extra <= 0 {
		return 0
	}
	seen := 0
	for i := 0; i < l.ring.Len(); i++ {
		ts := l.ring.At(i)
		if ts <= cut {
			continue
		}
		seen++
		if seen == extra {
			exp := time.Unix(0, ts).Add(l.Size)
			d := exp.Sub(now)
			if d < 0 {
				return 0
			}
			return d
		}
	}
	return 0
}

// Reset 清空窗口占用。
func (l *Log) Reset() {
	if l.ring != nil {
		l.ring.Reset()
	}
}
