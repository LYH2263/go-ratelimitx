package ledger

import (
	"sync"
	"time"

	iclk "github.com/LYH2263/go-ratelimitx/internal/clock"
)

// Result 是 Charge 的结果。
type ChargeOut struct {
	OK        bool
	Remaining int64
	Used      int64
	SoftHit   bool
	Debt      int64
	Receipt   Receipt
	Reason    string
}

// RefundOut 是 Refund 的结果。
type RefundOut struct {
	OK        bool
	Remaining int64
	Used      int64
	Debt      int64
	Reason    string
}

// Ledger 是租户配额账本。默认禁止 Used 变负；Debt 模式允许超过 Hard。
type Ledger struct {
	mu       sync.Mutex
	clock    iclk.Clock
	plan     Plan
	accounts map[string]*Account
	receipts map[uint64]*receiptState
	nextID   uint64
}

// New 创建账本。clk 为 nil 时用系统时钟。
func New(clk iclk.Clock, plan Plan) *Ledger {
	if clk == nil {
		clk = iclk.Real{}
	}
	if !plan.Valid() {
		plan = Plan{Period: 24 * time.Hour, Hard: 0}
	}
	return &Ledger{
		clock:    clk,
		plan:     plan,
		accounts: make(map[string]*Account),
		receipts: make(map[uint64]*receiptState),
	}
}

// Plan 返回当前计划副本。
func (l *Ledger) Plan() Plan {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.plan
}

// SetPlan 替换计划（不影响已有 Used，但新周期按新 Period 对齐）。
func (l *Ledger) SetPlan(p Plan) {
	l.mu.Lock()
	l.plan = p
	l.mu.Unlock()
}

func (l *Ledger) accountLocked(tenant string, now time.Time) *Account {
	a, ok := l.accounts[tenant]
	if !ok {
		a = &Account{
			Tenant:      tenant,
			PeriodStart: Align(now, l.plan.Period),
			Soft:        l.plan.Soft,
			Hard:        l.plan.Hard,
			DebtMode:    l.plan.Debt,
		}
		l.accounts[tenant] = a
		return a
	}
	a.Soft = l.plan.Soft
	a.Hard = l.plan.Hard
	a.DebtMode = l.plan.Debt
	newStart := Align(now, l.plan.Period)
	if !a.PeriodStart.Equal(newStart) {
		a.PeriodStart = newStart
		a.Used = 0
	}
	return a
}

// Charge 扣减 units。units<=0 拒绝。超额在非负债模式下拒绝且 Used 不变。
func (l *Ledger) Charge(tenant string, units int64) ChargeOut {
	if tenant == "" {
		return ChargeOut{Reason: "empty_tenant"}
	}
	if units <= 0 {
		return ChargeOut{Reason: "invalid_units"}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.clock.Now()
	a := l.accountLocked(tenant, now)
	if a.WouldExceed(units) {
		return ChargeOut{
			OK:        false,
			Remaining: a.Remaining(),
			Used:      a.Used,
			SoftHit:   a.SoftHit(),
			Debt:      a.Debt(),
			Reason:    "hard_limit",
		}
	}
	a.ApplyCharge(units)
	l.nextID++
	id := l.nextID
	rec := Receipt{ID: id, Tenant: tenant, Units: units, At: now}
	l.receipts[id] = &receiptState{Receipt: rec}
	return ChargeOut{
		OK:        true,
		Remaining: a.Remaining(),
		Used:      a.Used,
		SoftHit:   a.SoftHit(),
		Debt:      a.Debt(),
		Receipt:   rec,
	}
}

// Refund 按收据归还。已退或未知则失败。租户必须一致。
func (l *Ledger) Refund(rec Receipt) RefundOut {
	if rec.ID == 0 {
		return RefundOut{Reason: "unknown_receipt"}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	st, ok := l.receipts[rec.ID]
	if !ok {
		return RefundOut{Reason: "unknown_receipt"}
	}
	if rec.Tenant != "" && rec.Tenant != st.Tenant {
		return RefundOut{Reason: "tenant_mismatch"}
	}
	if st.Refunded {
		now := l.clock.Now()
		a := l.accountLocked(st.Tenant, now)
		return RefundOut{
			Remaining: a.Remaining(),
			Used:      a.Used,
			Debt:      a.Debt(),
			Reason:    "already_refunded",
		}
	}
	now := l.clock.Now()
	a := l.accountLocked(st.Tenant, now)
	a.ApplyRefund(st.Units, l.plan.Debt)
	st.Refunded = true
	st.RefundAt = now
	return RefundOut{
		OK:        true,
		Remaining: a.Remaining(),
		Used:      a.Used,
		Debt:      a.Debt(),
	}
}

// Peek 只读账户。不存在时 Used=0。
func (l *Ledger) Peek(tenant string) Account {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.clock.Now()
	a := l.accountLocked(tenant, now)
	cp := *a
	return cp
}

// Reset 删除租户账户（不使旧收据失效，退款时若账户被删会重建后归还）。
func (l *Ledger) Reset(tenant string) {
	l.mu.Lock()
	delete(l.accounts, tenant)
	l.mu.Unlock()
}

// Tenants 返回当前账户名快照。
func (l *Ledger) Tenants() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, 0, len(l.accounts))
	for t := range l.accounts {
		out = append(out, t)
	}
	return out
}
