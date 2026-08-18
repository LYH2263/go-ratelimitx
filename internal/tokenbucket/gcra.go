package tokenbucket

import (
	"time"

	"github.com/LYH2263/go-ratelimitx/internal/arith"
)

// GCRAState 实现 Generic Cell Rate Algorithm。
// TAT（theoretical arrival time）是下一允许到达的理论时刻。
// limit = burst * emissionInterval，允许在 limit 窗口内突发。
type GCRAState struct {
	TAT      int64  // Unix nano
	Interval uint64 // nano per token (emission interval)
	Limit    uint64 // nano, = burst * interval
	Burst    int
}

// NewGCRA 由速率与突发构造。PerSecond<=0 或 Burst<1 时返回零值。
func NewGCRA(cfg Config, now time.Time) GCRAState {
	if !cfg.Valid() {
		return GCRAState{}
	}
	// interval = 1e9 / perSecond nanoseconds per token
	per := cfg.PerSecond
	interval := uint64(float64(time.Second) / per)
	if interval == 0 {
		interval = 1
	}
	limit := interval * uint64(cfg.Burst)
	return GCRAState{
		TAT:      now.UnixNano(),
		Interval: interval,
		Limit:    limit,
		Burst:    cfg.Burst,
	}
}

// Allow 尝试接纳 n 个事件。GCRA 对 n>1 按 n*interval 推进 TAT。
func (g *GCRAState) Allow(n int, now time.Time) Result {
	if n <= 0 {
		return Result{}
	}
	if g.Interval == 0 || g.Limit == 0 {
		return Result{Wait: -1, Impossible: true}
	}
	if n > g.Burst {
		return Result{Wait: -1, Impossible: true}
	}
	nowNs := now.UnixNano()
	tat := g.TAT
	if nowNs > tat {
		tat = nowNs
	}
	inc := g.Interval * uint64(n)
	newTAT := uint64(tat) + inc
	// 允许条件：newTAT - now <= limit
	if newTAT < uint64(nowNs) {
		// overflow / clock jump: 视为 now 领先，直接接纳
		g.TAT = int64(uint64(nowNs) + inc)
		return Result{OK: true, Remaining: g.remaining(nowNs, g.TAT)}
	}
	ahead := newTAT - uint64(nowNs)
	if ahead <= g.Limit {
		g.TAT = int64(newTAT)
		return Result{OK: true, Remaining: g.remaining(nowNs, g.TAT)}
	}
	waitNs := ahead - g.Limit
	return Result{OK: false, Wait: time.Duration(waitNs), Remaining: g.remaining(nowNs, g.TAT)}
}

// Peek 不修改 TAT。
func (g *GCRAState) Peek(n int, now time.Time) Result {
	if n <= 0 {
		return Result{}
	}
	if g.Interval == 0 || g.Limit == 0 {
		return Result{Wait: -1, Impossible: true}
	}
	if n > g.Burst {
		return Result{Wait: -1, Impossible: true}
	}
	nowNs := now.UnixNano()
	tat := g.TAT
	if nowNs > tat {
		tat = nowNs
	}
	inc := g.Interval * uint64(n)
	newTAT := uint64(tat) + inc
	if newTAT < uint64(nowNs) {
		return Result{OK: true, Remaining: g.remaining(nowNs, int64(uint64(nowNs)+inc))}
	}
	ahead := newTAT - uint64(nowNs)
	if ahead <= g.Limit {
		return Result{OK: true, Remaining: g.remaining(nowNs, int64(newTAT))}
	}
	return Result{OK: false, Wait: time.Duration(ahead - g.Limit), Remaining: g.remaining(nowNs, g.TAT)}
}

// Restore 回退 TAT，相当于取消一次 Allow。不会把 TAT 推到 now 之前超过 Limit。
func (g *GCRAState) Restore(n int, now time.Time) {
	if n <= 0 || g.Interval == 0 {
		return
	}
	dec := int64(g.Interval * uint64(n))
	g.TAT -= dec
	floor := now.UnixNano() - int64(g.Limit)
	if g.TAT < floor {
		g.TAT = floor
	}
}

func (g *GCRAState) remaining(nowNs, tat int64) float64 {
	if g.Interval == 0 {
		return 0
	}
	// remaining tokens ≈ (limit - max(tat-now, 0)) / interval
	var ahead uint64
	if tat > nowNs {
		ahead = uint64(tat - nowNs)
	}
	if ahead >= g.Limit {
		return 0
	}
	left := g.Limit - ahead
	return float64(left) / float64(g.Interval)
}

// WaitNanos 计算还差 deficit grains 时令牌桶需要等待的纳秒。
func WaitNanos(deficitGrains, rateGrainsPerSec uint64) uint64 {
	if rateGrainsPerSec == 0 {
		return ^uint64(0)
	}
	return arith.MulDivRoundUp(deficitGrains, uint64(time.Second), rateGrainsPerSec)
}
