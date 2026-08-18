package slidingwin

import "time"

// Snapshot 是窗口只读视图。
type Snapshot struct {
	InWindow int
	Limit    int
	Size     time.Duration
	WaitOne  time.Duration
	Algo     string
}

// SnapshotLog 生成精确窗口快照。
func SnapshotLog(l *Log, now time.Time) Snapshot {
	r := l.Peek(1, now)
	w := time.Duration(0)
	if !r.OK {
		w = r.Wait
	}
	return Snapshot{
		InWindow: r.InWindow,
		Limit:    l.Limit,
		Size:     l.Size,
		WaitOne:  w,
		Algo:     "slidinglog",
	}
}

// SnapshotCounter 生成近似窗口快照。
func SnapshotCounter(c *Counter, now time.Time) Snapshot {
	r := c.Peek(1, now)
	w := time.Duration(0)
	if !r.OK {
		w = r.Wait
	}
	return Snapshot{
		InWindow: r.InWindow,
		Limit:    c.Limit,
		Size:     c.Size,
		WaitOne:  w,
		Algo:     "slidingcounter",
	}
}
