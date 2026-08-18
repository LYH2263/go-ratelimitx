package ratelimitx

import "time"

// Algorithm 选择限流算法。空值在 Register 时按规格字段推断。
type Algorithm string

const (
	AlgoTokenBucket     Algorithm = "tokenbucket"
	AlgoGCRA            Algorithm = "gcra"
	AlgoSlidingLog      Algorithm = "slidinglog"
	AlgoSlidingCounter  Algorithm = "slidingcounter"
)

// Rate 描述令牌桶/GCRA 的速率与突发。
type Rate struct {
	// PerSecond 为每秒产生的令牌数，必须 > 0。
	PerSecond float64
	// Burst 为桶容量（令牌个数），必须 >= 1。
	Burst int
}

// Window 描述滑动窗口的时长与上限。
type Window struct {
	// Size 为窗口长度，必须 > 0。
	Size time.Duration
	// Limit 为窗口内允许的事件数，必须 >= 1。
	Limit int
	// Buckets 仅滑动计数器使用；0 表示默认 10。
	Buckets int
}

// QuotaPlan 描述租户配额周期与软/硬限额。
type QuotaPlan struct {
	// Period 为账本滚动周期，必须 > 0。
	Period time.Duration
	// Soft 为软限额（超过仍可通过，仅记指标）。0 表示无软限额。
	Soft int64
	// Hard 为硬限额。负债模式关闭时 Used+units > Hard 则拒绝。
	Hard int64
	// Debt 为显式负债模式：允许 Used 超过 Hard，Debt = Used-Hard。
	Debt bool
}

// Spec 是一条命名策略。Rate 与 Window 可同时存在：Allow 先过速率再过窗口。
type Spec struct {
	Name   string
	Algo   Algorithm
	Rate   *Rate
	Window *Window
	Quota  *QuotaPlan
}

// Key 是复合键的三个维度。空字符串表示该维缺失。
type Key struct {
	Tenant string
	Route  string
	IP     string
}

// Decision 是 AllowN 的详细结果。
type Decision struct {
	OK         bool
	Wait       time.Duration
	Remaining  float64
	Limit      int
	RetryAfter time.Duration
	Policy     string
	Algorithm  Algorithm
	Reason     DenyReason
	Err        error
}

// DenyReason 解释拒绝原因，便于指标按原因聚合。
type DenyReason string

const (
	DenyNone        DenyReason = ""
	DenyEmptyKey    DenyReason = "empty_key"
	DenyInvalidN    DenyReason = "invalid_n"
	DenyRate        DenyReason = "rate"
	DenyWindow      DenyReason = "window"
	DenyImpossible  DenyReason = "impossible"
	DenyNoPolicy    DenyReason = "no_policy"
)

// PeekInfo 是 Peek 的只读快照，不消耗令牌/窗口名额。
type PeekInfo struct {
	Policy    string
	Algorithm Algorithm
	Tokens    float64
	Burst     int
	WindowIn  int
	WindowLim int
	Wait      time.Duration
	Exists    bool
}

// ChargeResult 是配额扣减结果。
type ChargeResult struct {
	OK        bool
	Remaining int64
	Used      int64
	SoftHit   bool
	Debt      int64
	Receipt   Receipt
	Err       error
}

// RefundResult 是配额归还结果。
type RefundResult struct {
	OK        bool
	Remaining int64
	Used      int64
	Debt      int64
	Err       error
}

// Receipt 标识一次成功 Charge，Refund 必须持有同一张收据。
type Receipt struct {
	ID     uint64
	Tenant string
	Units  int64
	At     time.Time
}

// Reservation 是可取消的预留。OK 为真时已扣减，Cancel 归还。
type Reservation struct {
	ok     bool
	wait   time.Duration
	n      int
	key    string
	policy string
	cancel func()
}

// OK 报告预留是否成功。
func (r Reservation) OK() bool { return r.ok }

// Delay 返回建议等待时长；失败且不可能时为 ImpossibleWait。
func (r Reservation) Delay() time.Duration { return r.wait }

// N 返回本次预留的单位数。
func (r Reservation) N() int { return r.n }

// Key 返回预留对应的限流键。
func (r Reservation) Key() string { return r.key }

// Policy 返回命中的策略名。
func (r Reservation) Policy() string { return r.policy }

// Cancel 归还已扣减的令牌/窗口名额。未成功或已取消时为 no-op。
func (r Reservation) Cancel() {
	if r.cancel != nil {
		r.cancel()
	}
}
