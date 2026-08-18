package store

import (
	"hash/fnv"
	"time"
)

// Kind 标识条目承载的算法状态。
type Kind uint8

const (
	KindNone Kind = iota
	KindTokenBucket
	KindGCRA
	KindSlidingLog
	KindSlidingCounter
	KindQuota
)

func (k Kind) String() string {
	switch k {
	case KindTokenBucket:
		return "tokenbucket"
	case KindGCRA:
		return "gcra"
	case KindSlidingLog:
		return "slidinglog"
	case KindSlidingCounter:
		return "slidingcounter"
	case KindQuota:
		return "quota"
	default:
		return "none"
	}
}

// Hash 计算键到分片下标。n==0 时返回 0。
func Hash(key string, n int) int {
	if n <= 1 {
		return 0
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	return int(h.Sum64() % uint64(n))
}

// Hash64 返回原始 FNV-1a 摘要。
func Hash64(key string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	return h.Sum64()
}

// SameShard 判断两个键是否落在同一分片。
func SameShard(a, b string, n int) bool {
	return Hash(a, n) == Hash(b, n)
}

// Entry 是分片内一条限流/配额状态。
type Entry struct {
	Key       string
	Kind      Kind
	Payload   any
	CreatedAt time.Time
	UpdatedAt time.Time
	Hits      uint64
	Denies    uint64
}

// Touch 更新命中时间与计数。ok 为 false 时记拒绝。
func (e *Entry) Touch(now time.Time, ok bool) {
	e.UpdatedAt = now
	if ok {
		e.Hits++
	} else {
		e.Denies++
	}
}

// Idle 报告 now 相对 UpdatedAt 的空闲时长。UpdatedAt 为零则用 CreatedAt。
func (e *Entry) Idle(now time.Time) time.Duration {
	base := e.UpdatedAt
	if base.IsZero() {
		base = e.CreatedAt
	}
	if now.Before(base) {
		return 0
	}
	return now.Sub(base)
}

// CloneMeta 复制元数据（不含 Payload 深拷贝）。
func (e *Entry) CloneMeta() Entry {
	return Entry{
		Key:       e.Key,
		Kind:      e.Kind,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
		Hits:      e.Hits,
		Denies:    e.Denies,
	}
}
