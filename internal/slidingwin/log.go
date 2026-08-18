package slidingwin

// Ring 是固定容量的时间戳环形缓冲区，按插入顺序保存 Unix 纳秒。
// 容量通常等于窗口 Limit。一律用 (start+i)%cap 寻址，避免满/未满两套语义。
type Ring struct {
	times []int64
	start int // 最旧事件下标
	len   int
	cap   int
}

// NewRing 分配容量为 cap 的环。cap<1 时按 1。
func NewRing(cap int) *Ring {
	if cap < 1 {
		cap = 1
	}
	return &Ring{times: make([]int64, cap), cap: cap}
}

// Cap 返回容量。
func (r *Ring) Cap() int { return r.cap }

// Len 返回当前事件数。
func (r *Ring) Len() int { return r.len }

// Reset 清空。
func (r *Ring) Reset() {
	r.start = 0
	r.len = 0
}

// At 返回从最旧开始第 i 个事件的时间戳。越界返回 0。
func (r *Ring) At(i int) int64 {
	if i < 0 || i >= r.len || r.cap == 0 {
		return 0
	}
	return r.times[(r.start+i)%r.cap]
}

// Oldest 返回最旧时间戳。空环返回 0。
func (r *Ring) Oldest() int64 {
	if r.len == 0 {
		return 0
	}
	return r.At(0)
}

// Newest 返回最新时间戳。空环返回 0。
func (r *Ring) Newest() int64 {
	if r.len == 0 {
		return 0
	}
	return r.At(r.len - 1)
}

// Push 追加 nowNs。满时覆盖最旧。
func (r *Ring) Push(nowNs int64) {
	if r.len < r.cap {
		r.times[(r.start+r.len)%r.cap] = nowNs
		r.len++
		return
	}
	r.times[r.start] = nowNs
	r.start = (r.start + 1) % r.cap
}

// PushN 追加 n 次相同时间戳。
func (r *Ring) PushN(nowNs int64, n int) {
	for i := 0; i < n; i++ {
		r.Push(nowNs)
	}
}

// Prune 丢弃 ts <= cutoff 的前缀。事件有效当 ts > cutoff。
func (r *Ring) Prune(cutoff int64) int {
	dropped := 0
	for r.len > 0 && r.At(0) <= cutoff {
		r.dropOldest()
		dropped++
	}
	return dropped
}

func (r *Ring) dropOldest() {
	if r.len == 0 {
		return
	}
	r.start = (r.start + 1) % r.cap
	r.len--
}

// PopNewest 弹出最新一条，用于 Restore。空环返回 false。
func (r *Ring) PopNewest() (int64, bool) {
	if r.len == 0 {
		return 0, false
	}
	ts := r.Newest()
	r.len--
	return ts, true
}

// PopNewestN 弹出最多 n 条最新事件，返回实际弹出数。
func (r *Ring) PopNewestN(n int) int {
	got := 0
	for i := 0; i < n; i++ {
		if _, ok := r.PopNewest(); !ok {
			break
		}
		got++
	}
	return got
}

// Clone 深拷贝。
func (r *Ring) Clone() *Ring {
	return &Ring{
		times: append([]int64(nil), r.times...),
		start: r.start,
		len:   r.len,
		cap:   r.cap,
	}
}

// CountAfter 统计 ts > cutoff 的事件数（不修改）。
func (r *Ring) CountAfter(cutoff int64) int {
	n := 0
	for i := 0; i < r.len; i++ {
		if r.At(i) > cutoff {
			n++
		}
	}
	return n
}

// FirstAfter 返回第一个 ts > cutoff 的时间戳。
func (r *Ring) FirstAfter(cutoff int64) (int64, bool) {
	for i := 0; i < r.len; i++ {
		ts := r.At(i)
		if ts > cutoff {
			return ts, true
		}
	}
	return 0, false
}

// Times 按从旧到新复制出切片，便于测试。
func (r *Ring) Times() []int64 {
	out := make([]int64, r.len)
	for i := 0; i < r.len; i++ {
		out[i] = r.At(i)
	}
	return out
}
