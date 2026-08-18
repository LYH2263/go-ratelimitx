package ratelimitx

import (
	"time"

	"github.com/LYH2263/go-ratelimitx/internal/clock"
)

// FakeClock 是可注入的确定性时钟，用于时间旅行测试。
type FakeClock struct {
	inner *clock.Fake
}

// NewFakeClock 从 now 启动。零值视为 Unix epoch UTC。
func NewFakeClock(now time.Time) *FakeClock {
	return &FakeClock{inner: clock.NewFake(now)}
}

// Now 实现 Clock。
func (f *FakeClock) Now() time.Time { return f.inner.Now() }

// Advance 向前推进 d。d<=0 无效果。
func (f *FakeClock) Advance(d time.Duration) { f.inner.Advance(d) }

// Set 设置绝对时间。
func (f *FakeClock) Set(t time.Time) { f.inner.Set(t) }

// Rewind 向后拨钟，算法必须容忍。
func (f *FakeClock) Rewind(d time.Duration) { f.inner.Rewind(d) }
