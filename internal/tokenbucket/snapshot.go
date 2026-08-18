package tokenbucket

import "time"

// Snapshot 是令牌桶/GCRA 的只读视图。
type Snapshot struct {
	Tokens    float64
	Burst     int
	PerSecond float64
	WaitOne   time.Duration
	Algo      string
}

// SnapshotBucket 在 refill 后生成快照，不扣减。
func SnapshotBucket(s *State, now time.Time) Snapshot {
	s.Refill(now)
	w := time.Duration(0)
	if s.Grains < Grain {
		w = s.waitFor(Grain)
	}
	per := 0.0
	if s.Rate > 0 {
		per = float64(s.Rate) / float64(Grain)
	}
	return Snapshot{
		Tokens:    s.Tokens(),
		Burst:     s.BurstTokens(),
		PerSecond: per,
		WaitOne:   w,
		Algo:      "tokenbucket",
	}
}

// SnapshotGCRA 生成 GCRA 快照。
func SnapshotGCRA(g *GCRAState, now time.Time) Snapshot {
	r := g.Peek(1, now)
	per := 0.0
	if g.Interval > 0 {
		per = float64(time.Second) / float64(g.Interval)
	}
	wait := time.Duration(0)
	if !r.OK {
		wait = r.Wait
	}
	return Snapshot{
		Tokens:    r.Remaining,
		Burst:     g.Burst,
		PerSecond: per,
		WaitOne:   wait,
		Algo:      "gcra",
	}
}

// ClonePtr 深拷贝令牌桶。State 仅含值类型字段，复制结构体即得独立副本。
func ClonePtr(s *State) *State {
	if s == nil {
		return nil
	}
	out := *s
	return &out
}

// CapN 把 n 钳到 (0, MaxTokens]。非法返回 0。
func CapN(n int) int {
	if n <= 0 {
		return 0
	}
	if n > MaxTokens {
		return MaxTokens
	}
	return n
}
