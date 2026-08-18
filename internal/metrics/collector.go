package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// Collector 聚合通过/拒绝等计数。并发安全。
type Collector struct {
	allowed      atomic.Int64
	denied       atomic.Int64
	peeked       atomic.Int64
	resets       atomic.Int64
	reserved     atomic.Int64
	canceled     atomic.Int64
	charged      atomic.Int64
	chargeDenied atomic.Int64
	refunded     atomic.Int64
	refundDenied atomic.Int64
	softHits     atomic.Int64
	sweeps       atomic.Int64
	waitNanos    atomic.Int64
	waitSamples  atomic.Int64

	mu      sync.Mutex
	byReason map[string]*atomic.Int64
	byPolicy map[string]*atomic.Int64
}

// New 创建空收集器。
func New() *Collector {
	return &Collector{
		byReason: make(map[string]*atomic.Int64),
		byPolicy: make(map[string]*atomic.Int64),
	}
}

func (c *Collector) bumpMap(m map[string]*atomic.Int64, key string, n int64) {
	if key == "" {
		key = "none"
	}
	c.mu.Lock()
	p, ok := m[key]
	if !ok {
		p = &atomic.Int64{}
		m[key] = p
	}
	c.mu.Unlock()
	p.Add(n)
}

// Allow 记录一次 Allow 结果。
func (c *Collector) Allow(ok bool, policy, reason string, wait time.Duration) {
	if ok {
		c.allowed.Add(1)
		c.bumpMap(c.byPolicy, policy, 1)
		return
	}
	c.denied.Add(1)
	c.bumpMap(c.byReason, reason, 1)
	if wait > 0 {
		c.waitNanos.Add(wait.Nanoseconds())
		c.waitSamples.Add(1)
	}
}

// Peek 记录 Peek。
func (c *Collector) Peek() { c.peeked.Add(1) }

// Reset 记录 Reset。
func (c *Collector) Reset() { c.resets.Add(1) }

// Reserve 记录预留。
func (c *Collector) Reserve(ok bool) {
	if ok {
		c.reserved.Add(1)
	}
}

// Cancel 记录取消预留。
func (c *Collector) Cancel() { c.canceled.Add(1) }

// Charge 记录配额扣减。
func (c *Collector) Charge(ok, soft bool) {
	if ok {
		c.charged.Add(1)
		if soft {
			c.softHits.Add(1)
		}
		return
	}
	c.chargeDenied.Add(1)
}

// Refund 记录退款。
func (c *Collector) Refund(ok bool) {
	if ok {
		c.refunded.Add(1)
		return
	}
	c.refundDenied.Add(1)
}

// Sweep 记录回收条目数。
func (c *Collector) Sweep(n int) {
	c.sweeps.Add(int64(n))
}

// Snapshot 导出当前计数。
func (c *Collector) Snapshot() Snapshot {
	c.mu.Lock()
	reasons := make(map[string]int64, len(c.byReason))
	for k, v := range c.byReason {
		reasons[k] = v.Load()
	}
	policies := make(map[string]int64, len(c.byPolicy))
	for k, v := range c.byPolicy {
		policies[k] = v.Load()
	}
	c.mu.Unlock()
	samples := c.waitSamples.Load()
	var avg time.Duration
	if samples > 0 {
		avg = time.Duration(c.waitNanos.Load() / samples)
	}
	return Snapshot{
		Allowed:      c.allowed.Load(),
		Denied:       c.denied.Load(),
		Peeked:       c.peeked.Load(),
		Resets:       c.resets.Load(),
		Reserved:     c.reserved.Load(),
		Canceled:     c.canceled.Load(),
		Charged:      c.charged.Load(),
		ChargeDenied: c.chargeDenied.Load(),
		Refunded:     c.refunded.Load(),
		RefundDenied: c.refundDenied.Load(),
		SoftHits:     c.softHits.Load(),
		Sweeps:       c.sweeps.Load(),
		WaitAvg:      avg,
		DeniedReason: reasons,
		AllowedPolicy: policies,
	}
}

// ResetAll 清零（测试用）。
func (c *Collector) ResetAll() {
	c.allowed.Store(0)
	c.denied.Store(0)
	c.peeked.Store(0)
	c.resets.Store(0)
	c.reserved.Store(0)
	c.canceled.Store(0)
	c.charged.Store(0)
	c.chargeDenied.Store(0)
	c.refunded.Store(0)
	c.refundDenied.Store(0)
	c.softHits.Store(0)
	c.sweeps.Store(0)
	c.waitNanos.Store(0)
	c.waitSamples.Store(0)
	c.mu.Lock()
	c.byReason = make(map[string]*atomic.Int64)
	c.byPolicy = make(map[string]*atomic.Int64)
	c.mu.Unlock()
}
