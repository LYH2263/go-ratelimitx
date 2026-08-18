package clock

import (
	"sync"
	"time"
)

// Fake 是可注入的确定性时钟，供时间旅行测试使用。
// 所有方法并发安全。
type Fake struct {
	mu sync.Mutex
	t  time.Time
}

// NewFake 从给定时刻启动。若 t 为零值，使用 Unix epoch UTC。
func NewFake(t time.Time) *Fake {
	if t.IsZero() {
		t = time.Unix(0, 0).UTC()
	}
	return &Fake{t: t}
}

// Now 返回当前假时间的拷贝。
func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.t
}

// Set 将假时间设为 t。零值被忽略。
func (f *Fake) Set(t time.Time) {
	if t.IsZero() {
		return
	}
	f.mu.Lock()
	f.t = t
	f.mu.Unlock()
}

// Advance 将假时间向前推进 d。d <= 0 时无效果。
func (f *Fake) Advance(d time.Duration) {
	if d <= 0 {
		return
	}
	f.mu.Lock()
	f.t = f.t.Add(d)
	f.mu.Unlock()
}

// Rewind 将假时间向后拨 d。d <= 0 时无效果。限流算法必须能容忍回拨。
func (f *Fake) Rewind(d time.Duration) {
	if d <= 0 {
		return
	}
	f.mu.Lock()
	f.t = f.t.Add(-d)
	f.mu.Unlock()
}

// UnixNano 返回当前假时间的 Unix 纳秒。
func (f *Fake) UnixNano() int64 {
	return f.Now().UnixNano()
}
