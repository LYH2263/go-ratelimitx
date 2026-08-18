package syncutil

import "sync"

// Guard 把值与互斥锁绑在一起，避免遗漏配对。
type Guard[T any] struct {
	mu    sync.Mutex
	value T
}

// NewGuard 包装初始值。
func NewGuard[T any](v T) *Guard[T] {
	return &Guard[T]{value: v}
}

// Do 在锁内把 value 交给 fn。fn 可就地修改。
func (g *Guard[T]) Do(fn func(*T)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	fn(&g.value)
}

// Get 返回值的拷贝。
func (g *Guard[T]) Get() T {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.value
}

// Set 覆盖值。
func (g *Guard[T]) Set(v T) {
	g.mu.Lock()
	g.value = v
	g.mu.Unlock()
}

// OnceValue 是带错误的一次性初始化。
type OnceValue[T any] struct {
	once sync.Once
	val  T
	err  error
}

// Do 只执行一次 fn，之后返回缓存结果。
func (o *OnceValue[T]) Do(fn func() (T, error)) (T, error) {
	o.once.Do(func() {
		o.val, o.err = fn()
	})
	return o.val, o.err
}
