package slidingwin

import "time"

// Counter 是近似滑动窗口计数器：把窗口切成若干等宽桶，
// 估计值 = previous*(1-t) + current，其中 t 为当前桶已过去的比例。
// 这是 Redis 风格的滑动窗口计数，牺牲精确换固定内存。
type Counter struct {
	Size    time.Duration
	Limit   int
	Buckets int
	width   time.Duration
	counts  []int64
	start   int64 // counts[0] 对应的桶起点 Unix nano
	init    bool
}

// NewCounter 创建计数器。buckets<2 时按 10。
func NewCounter(size time.Duration, limit, buckets int) *Counter {
	if size <= 0 {
		size = time.Second
	}
	if limit < 1 {
		limit = 1
	}
	if buckets < 2 {
		buckets = 10
	}
	width := size / time.Duration(buckets)
	if width <= 0 {
		width = time.Nanosecond
		buckets = int(size / width)
		if buckets < 2 {
			buckets = 2
		}
	}
	return &Counter{
		Size:    size,
		Limit:   limit,
		Buckets: buckets,
		width:   width,
		counts:  make([]int64, buckets),
	}
}

func (c *Counter) bucketStart(ns int64) int64 {
	w := int64(c.width)
	if w <= 0 {
		return ns
	}
	return ns - ns%w
}

func (c *Counter) advance(now time.Time) {
	nowNs := now.UnixNano()
	cur := c.bucketStart(nowNs)
	if !c.init {
		c.start = cur - int64(c.width)*int64(c.Buckets-1)
		c.init = true
		return
	}
	end := c.start + int64(c.width)*int64(c.Buckets-1)
	if cur == end {
		return
	}
	if cur < c.start {
		// 时钟回拨：不移动桶，避免把历史计数作废后误放行
		return
	}
	steps := int((cur - end) / int64(c.width))
	if steps >= c.Buckets {
		for i := range c.counts {
			c.counts[i] = 0
		}
		c.start = cur - int64(c.width)*int64(c.Buckets-1)
		return
	}
	if steps > 0 {
		copy(c.counts, c.counts[steps:])
		for i := c.Buckets - steps; i < c.Buckets; i++ {
			c.counts[i] = 0
		}
		c.start += int64(c.width) * int64(steps)
	}
}

// Estimate 返回加权估计。t 为当前桶内已过去比例。
func (c *Counter) Estimate(now time.Time) float64 {
	c.advance(now)
	nowNs := now.UnixNano()
	curStart := c.start + int64(c.width)*int64(c.Buckets-1)
	elapsed := nowNs - curStart
	if elapsed < 0 {
		elapsed = 0
	}
	w := int64(c.width)
	var frac float64
	if w > 0 {
		frac = float64(elapsed) / float64(w)
	}
	if frac > 1 {
		frac = 1
	}
	var prev int64
	if c.Buckets >= 2 {
		prev = c.counts[c.Buckets-2]
	}
	cur := c.counts[c.Buckets-1]
	return float64(prev)*(1-frac) + float64(cur)
}

// Allow 若估计+n <= Limit 则把 n 记入当前桶。
func (c *Counter) Allow(n int, now time.Time) Result {
	if n <= 0 {
		return Result{}
	}
	if n > c.Limit {
		return Result{Wait: -1, Impossible: true}
	}
	est := c.Estimate(now)
	if est+float64(n) <= float64(c.Limit)+1e-9 {
		c.counts[c.Buckets-1] += int64(n)
		return Result{OK: true, InWindow: int(est) + n}
	}
	wait := c.wait(n, now, est)
	return Result{OK: false, Wait: wait, InWindow: int(est)}
}

// Peek 不写入。
func (c *Counter) Peek(n int, now time.Time) Result {
	if n <= 0 {
		return Result{}
	}
	if n > c.Limit {
		return Result{Wait: -1, Impossible: true}
	}
	est := c.Estimate(now)
	if est+float64(n) <= float64(c.Limit)+1e-9 {
		return Result{OK: true, InWindow: int(est)}
	}
	return Result{OK: false, Wait: c.wait(n, now, est), InWindow: int(est)}
}

// Restore 从当前桶减去 n，不低于 0。
func (c *Counter) Restore(n int) {
	if n <= 0 {
		return
	}
	idx := c.Buckets - 1
	c.counts[idx] -= int64(n)
	if c.counts[idx] < 0 {
		c.counts[idx] = 0
	}
}

func (c *Counter) wait(n int, now time.Time, est float64) time.Duration {
	// 粗略：等到当前桶滑出足够权重。用 width 作为下限。
	need := est + float64(n) - float64(c.Limit)
	if need <= 0 {
		return 0
	}
	w := float64(c.width)
	if w <= 0 {
		return c.Size
	}
	// 每滑过 1 个桶大约丢掉 previous 的一部分；用 need/max(est,1)*size 近似
	if est <= 0 {
		return 0
	}
	frac := need / est
	if frac > 1 {
		frac = 1
	}
	d := time.Duration(frac * float64(c.Size))
	if d < c.width {
		d = c.width
	}
	if d > c.Size {
		d = c.Size
	}
	return d
}

// Width 返回桶宽。
func (c *Counter) Width() time.Duration { return c.width }
