package syncutil

import (
	"hash/fnv"
	"sync"
)

const defaultShards = 32

// Sharded 是按 key 哈希分片的 RWMutex 集合，降低热点锁竞争。
type Sharded struct {
	shards []sync.RWMutex
	n      uint32
}

// New 创建 n 个分片。n < 1 时使用 32。
func New(n int) *Sharded {
	if n < 1 {
		n = defaultShards
	}
	return &Sharded{
		shards: make([]sync.RWMutex, n),
		n:      uint32(n),
	}
}

// Len 返回分片数。
func (s *Sharded) Len() int { return int(s.n) }

// Index 返回 key 对应的分片下标。使用 FNV-1a 64 再取模。
func (s *Sharded) Index(key string) int {
	return int(Hash64(key) % uint64(s.n))
}

// Lock 锁住 key 所在分片。
func (s *Sharded) Lock(key string) { s.shards[s.Index(key)].Lock() }

// Unlock 解锁 key 所在分片。
func (s *Sharded) Unlock(key string) { s.shards[s.Index(key)].Unlock() }

// RLock 读锁 key 所在分片。
func (s *Sharded) RLock(key string) { s.shards[s.Index(key)].RLock() }

// RUnlock 解读锁。
func (s *Sharded) RUnlock(key string) { s.shards[s.Index(key)].RUnlock() }

// LockIndex 按分片下标加写锁。idx 越界时对 n 取模。
func (s *Sharded) LockIndex(idx int) {
	s.shards[s.norm(idx)].Lock()
}

// UnlockIndex 按分片下标解写锁。
func (s *Sharded) UnlockIndex(idx int) {
	s.shards[s.norm(idx)].Unlock()
}

// Mutex 返回第 idx 个互斥量指针，供需要与存储分片对齐的调用方使用。
func (s *Sharded) Mutex(idx int) *sync.RWMutex {
	return &s.shards[s.norm(idx)]
}

func (s *Sharded) norm(idx int) int {
	if s.n == 0 {
		return 0
	}
	if idx < 0 {
		idx = -idx
	}
	return idx % int(s.n)
}

// Do 在 key 的写锁内执行 fn。
func (s *Sharded) Do(key string, fn func()) {
	i := s.Index(key)
	s.shards[i].Lock()
	defer s.shards[i].Unlock()
	fn()
}

// DoR 在 key 的读锁内执行 fn。
func (s *Sharded) DoR(key string, fn func()) {
	i := s.Index(key)
	s.shards[i].RLock()
	defer s.shards[i].RUnlock()
	fn()
}

// Hash64 计算字符串的 FNV-1a 64 位哈希。
func Hash64(key string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	return h.Sum64()
}

// Hash64Bytes 计算字节切片的 FNV-1a 64 位哈希。
func Hash64Bytes(b []byte) uint64 {
	h := fnv.New64a()
	_, _ = h.Write(b)
	return h.Sum64()
}
