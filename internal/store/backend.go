package store

import "sync"

// Backend 是可选持久化层。Set 失败时调用方必须回滚内存状态。
type Backend interface {
	Set(key string, blob []byte) error
	Get(key string) ([]byte, bool)
}

// MemoryBackend 是可注入失败的内存后端，供测试与嵌入式部署。
type MemoryBackend struct {
	mu     sync.Mutex
	data   map[string][]byte
	setErr error
}

// NewMemoryBackend 创建空后端。
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{data: make(map[string][]byte)}
}

// SetError 使后续 Set 返回 err；err 为 nil 时恢复正常写入。
func (m *MemoryBackend) SetError(err error) {
	m.mu.Lock()
	m.setErr = err
	m.mu.Unlock()
}

// Set 写入 blob。setErr 非空时不写入并返回该错误。
func (m *MemoryBackend) Set(key string, blob []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.setErr != nil {
		return m.setErr
	}
	m.data[key] = append([]byte(nil), blob...)
	return nil
}

// Get 读取 blob。
func (m *MemoryBackend) Get(key string) ([]byte, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.data[key]
	if !ok {
		return nil, false
	}
	return append([]byte(nil), b...), true
}
