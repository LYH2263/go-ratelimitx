package ledger

import (
	"time"

	iclk "github.com/LYH2263/go-ratelimitx/internal/clock"
)

// Plan 是配额计划。
type Plan struct {
	Period time.Duration
	Soft   int64
	Hard   int64
	Debt   bool
}

// Valid 报告计划是否可运行。
func (p Plan) Valid() bool {
	return p.Period > 0 && p.Hard >= 0 && p.Soft >= 0
}

// Account 是单个租户在当前周期内的账本行。
type Account struct {
	Tenant      string
	PeriodStart time.Time
	Used        int64
	Soft        int64
	Hard        int64
	DebtMode    bool
}

// Remaining 返回 Hard-Used；负债模式下可为负。非负债且 Used>Hard 时返回 0。
func (a *Account) Remaining() int64 {
	left := a.Hard - a.Used
	if left < 0 && !a.DebtMode {
		return 0
	}
	return left
}

// Debt 返回 max(Used-Hard, 0)。仅负债模式有意义。
func (a *Account) Debt() int64 {
	if a.Used <= a.Hard {
		return 0
	}
	return a.Used - a.Hard
}

// SoftHit 报告 Used 是否已达到或超过 Soft（Soft==0 视为未配置）。
func (a *Account) SoftHit() bool {
	if a.Soft <= 0 {
		return false
	}
	return a.Used >= a.Soft
}

// WouldExceed 报告再扣 units 是否超过 Hard（非负债）。
func (a *Account) WouldExceed(units int64) bool {
	if a.DebtMode {
		return false
	}
	if units <= 0 {
		return true
	}
	return a.Used+units > a.Hard
}

// ApplyCharge 在已判定允许后记账。
func (a *Account) ApplyCharge(units int64) {
	a.Used += units
}

// ApplyRefund 归还 units。非负债模式 Used 不低于 0；负债模式允许 Used 降到当前值以下但通常仍 >=0。
func (a *Account) ApplyRefund(units int64, debtMode bool) {
	a.Used -= units
	if !debtMode && a.Used < 0 {
		a.Used = 0
	}
}

// Rollover 若 now 已离开当前周期则清零 Used 并对齐新周期起点。返回是否发生滚动。
func (a *Account) Rollover(now time.Time, period time.Duration) bool {
	if period <= 0 {
		return false
	}
	start, end := iclk.AlignPeriod(a.PeriodStart, period)
	if !a.PeriodStart.IsZero() && iclk.InPeriod(now, start, period) && now.Before(end) {
		return false
	}
	// 也处理 PeriodStart 为零或 now 越过 end
	if !a.PeriodStart.IsZero() && !now.Before(end) || a.PeriodStart.IsZero() || now.Before(start) {
		a.PeriodStart = iclk.Truncate(now, period)
		a.Used = 0
		return true
	}
	return false
}
