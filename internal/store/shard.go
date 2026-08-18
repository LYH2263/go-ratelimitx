package store

import (
	"sync"
	"time"
)

// Shard 是单分片内存表。调用方必须持有对应分片锁，或使用本类型自带的 mu。
type Shard struct {
	mu      sync.Mutex
	items   map[string]*Entry
	lookups uint64
	misses  uint64
	puts    uint64
	dels    uint64
}

// NewShard 创建空分片。
func NewShard() *Shard {
	return &Shard{items: make(map[string]*Entry)}
}

// Get 返回条目；不存在时 ok=false。
func (s *Shard) Get(key string) (*Entry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lookups++
	e, ok := s.items[key]
	if !ok {
		s.misses++
	}
	return e, ok
}

// GetUnlocked 假定调用方已持有 s.mu。
func (s *Shard) GetUnlocked(key string) (*Entry, bool) {
	s.lookups++
	e, ok := s.items[key]
	if !ok {
		s.misses++
	}
	return e, ok
}

// Put 插入或覆盖。返回被覆盖的旧条目（可能为 nil）。
func (s *Shard) Put(e *Entry) *Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.putUnlocked(e)
}

func (s *Shard) putUnlocked(e *Entry) *Entry {
	s.puts++
	old := s.items[e.Key]
	s.items[e.Key] = e
	return old
}

// PutUnlocked 假定调用方已持有 s.mu。
func (s *Shard) PutUnlocked(e *Entry) *Entry {
	return s.putUnlocked(e)
}

// Delete 删除键。存在则返回 true。
func (s *Shard) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteUnlocked(key)
}

func (s *Shard) deleteUnlocked(key string) bool {
	if _, ok := s.items[key]; !ok {
		return false
	}
	s.dels++
	delete(s.items, key)
	return true
}

// DeleteUnlocked 假定调用方已持有 s.mu。
func (s *Shard) DeleteUnlocked(key string) bool {
	return s.deleteUnlocked(key)
}

// Len 返回条目数。
func (s *Shard) Len() int {
	s.mu.Lock()
	n := len(s.items)
	s.mu.Unlock()
	return n
}

// ForEach 在锁内遍历。fn 返回 false 时提前结束。
func (s *Shard) ForEach(fn func(*Entry) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.items {
		if !fn(e) {
			return
		}
	}
}

// SweepIdle 删除空闲超过 ttl 的条目。ttl<=0 时不删除。返回删除数量。
func (s *Shard) SweepIdle(now time.Time, ttl time.Duration) int {
	if ttl <= 0 {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for k, e := range s.items {
		if e.Idle(now) >= ttl {
			delete(s.items, k)
			s.dels++
			n++
		}
	}
	return n
}

// Keys 返回当前键的快照。
func (s *Shard) Keys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.items))
	for k := range s.items {
		out = append(out, k)
	}
	return out
}

// Stats 是分片计数快照。
type Stats struct {
	Items   int
	Lookups uint64
	Misses  uint64
	Puts    uint64
	Deletes uint64
}

// SnapshotStats 返回计数快照。
func (s *Shard) SnapshotStats() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Stats{
		Items:   len(s.items),
		Lookups: s.lookups,
		Misses:  s.misses,
		Puts:    s.puts,
		Deletes: s.dels,
	}
}

// Lock 暴露底层互斥锁，便于与算法状态更新同锁。
func (s *Shard) Lock() { s.mu.Lock() }

// Unlock 释放底层互斥锁。
func (s *Shard) Unlock() { s.mu.Unlock() }
