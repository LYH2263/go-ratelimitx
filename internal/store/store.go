package store

import (
	"time"

	iclk "github.com/LYH2263/go-ratelimitx/internal/clock"
)

// Store 是分片内存状态表。分片下标 = hash(key) % N。
type Store struct {
	shards   []*Shard
	n        int
	clock    iclk.Clock
	backend  Backend
}

// New 创建 n 个分片的存储。n<1 时按 32。clock 为 nil 时用系统时钟（仅用于时间戳）。
func New(n int, clk iclk.Clock) *Store {
	if n < 1 {
		n = 32
	}
	if clk == nil {
		clk = iclk.Real{}
	}
	ss := make([]*Shard, n)
	for i := range ss {
		ss[i] = NewShard()
	}
	return &Store{shards: ss, n: n, clock: clk}
}

// Shards 返回分片数。
func (s *Store) Shards() int { return s.n }

// Index 返回 key 的分片下标。
func (s *Store) Index(key string) int { return Hash(key, s.n) }

// Shard 返回下标对应的分片。越界按模回绕。
func (s *Store) Shard(idx int) *Shard {
	if s.n == 0 {
		return nil
	}
	if idx < 0 {
		idx = -idx
	}
	return s.shards[idx%s.n]
}

// ShardOf 返回 key 所在分片。
func (s *Store) ShardOf(key string) *Shard {
	return s.shards[s.Index(key)]
}

// Get 读取条目。
func (s *Store) Get(key string) (*Entry, bool) {
	return s.ShardOf(key).Get(key)
}

// Put 写入条目。若 CreatedAt 为零则填当前时钟。
func (s *Store) Put(e *Entry) *Entry {
	if e.CreatedAt.IsZero() {
		e.CreatedAt = s.clock.Now()
	}
	if e.UpdatedAt.IsZero() {
		e.UpdatedAt = e.CreatedAt
	}
	return s.ShardOf(e.Key).Put(e)
}

// Delete 删除键。
func (s *Store) Delete(key string) bool {
	return s.ShardOf(key).Delete(key)
}

// GetOrCreate 在分片锁内取或建条目。created 表示新建。
func (s *Store) GetOrCreate(key string, kind Kind, build func() any) (e *Entry, created bool) {
	sh := s.ShardOf(key)
	sh.Lock()
	defer sh.Unlock()
	if old, ok := sh.GetUnlocked(key); ok {
		return old, false
	}
	now := s.clock.Now()
	e = &Entry{
		Key:       key,
		Kind:      kind,
		Payload:   build(),
		CreatedAt: now,
		UpdatedAt: now,
	}
	sh.PutUnlocked(e)
	return e, true
}

// LoadAndTouch 在分片锁内加载并调用 fn。fn 返回 ok 用于更新 Hits/Denies。
// 若不存在则用 build 创建。fn 可替换 Payload。
func (s *Store) LoadAndTouch(key string, kind Kind, build func() any, fn func(*Entry) bool) bool {
	sh := s.ShardOf(key)
	sh.Lock()
	defer sh.Unlock()
	e, ok := sh.GetUnlocked(key)
	if !ok {
		now := s.clock.Now()
		e = &Entry{
			Key:       key,
			Kind:      kind,
			Payload:   build(),
			CreatedAt: now,
			UpdatedAt: now,
		}
		sh.PutUnlocked(e)
	}
	ok2 := fn(e)
	e.Touch(s.clock.Now(), ok2)
	return ok2
}

// Reset 删除键对应状态。
func (s *Store) Reset(key string) bool {
	return s.Delete(key)
}

// Len 返回所有分片条目总数。
func (s *Store) Len() int {
	n := 0
	for _, sh := range s.shards {
		n += sh.Len()
	}
	return n
}

// Sweep 按空闲 TTL 回收。ttl<=0 时返回 0。
func (s *Store) Sweep(ttl time.Duration) int {
	if ttl <= 0 {
		return 0
	}
	now := s.clock.Now()
	n := 0
	for _, sh := range s.shards {
		n += sh.SweepIdle(now, ttl)
	}
	return n
}

// Keys 收集全部分片键（无序）。
func (s *Store) Keys() []string {
	out := make([]string, 0, s.Len())
	for _, sh := range s.shards {
		out = append(out, sh.Keys()...)
	}
	return out
}

// AggregateStats 汇总各分片计数。
func (s *Store) AggregateStats() Stats {
	var tot Stats
	for _, sh := range s.shards {
		st := sh.SnapshotStats()
		tot.Items += st.Items
		tot.Lookups += st.Lookups
		tot.Misses += st.Misses
		tot.Puts += st.Puts
		tot.Deletes += st.Deletes
	}
	return tot
}

// Attach 设置可选持久化后端。
func (s *Store) Attach(b Backend) { s.backend = b }

// Persist 把状态写入后端。无后端时为 no-op。
// 返回 backend.Set 的错误，调用方据此回滚内存状态。
func (s *Store) Persist(key string, blob []byte) error {
	if s.backend == nil {
		return nil
	}
	return s.backend.Set(key, blob)
}

// ForEach 遍历所有条目。fn 在对应分片锁内被调用。
func (s *Store) ForEach(fn func(*Entry) bool) {
	for _, sh := range s.shards {
		stop := false
		sh.ForEach(func(e *Entry) bool {
			if !fn(e) {
				stop = true
				return false
			}
			return true
		})
		if stop {
			return
		}
	}
}
