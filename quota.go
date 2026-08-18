package ratelimitx

import "github.com/LYH2263/go-ratelimitx/internal/ledger"

// Charge 按租户扣减配额。策略经 Key{Tenant: tenant} 回退链解析。
func (e *Engine) Charge(tenant string, units int64) ChargeResult {
	if tenant == "" {
		return ChargeResult{Err: ErrEmptyKey}
	}
	if units <= 0 {
		return ChargeResult{Err: ErrInvalidUnits}
	}
	spec, _, ok := e.resolve(Key{Tenant: tenant})
	if !ok || spec.Quota == nil {
		return ChargeResult{Err: ErrUnknownPolicy}
	}
	l := e.ledgerFor(spec)
	if l == nil { // 引擎已关闭
		return ChargeResult{Err: ErrClosed}
	}
	out := l.Charge(tenant, units)
	e.metrics.Charge(out.OK, out.SoftHit)
	res := ChargeResult{
		OK:        out.OK,
		Remaining: out.Remaining,
		Used:      out.Used,
		SoftHit:   out.SoftHit,
		Debt:      out.Debt,
		Receipt: Receipt{
			ID:     out.Receipt.ID,
			Tenant: out.Receipt.Tenant,
			Units:  out.Receipt.Units,
			At:     out.Receipt.At,
		},
	}
	if !out.OK {
		res.Err = ErrQuotaExceeded
	}
	return res
}

// Refund 凭收据归还。同一收据只能成功一次。
func (e *Engine) Refund(rec Receipt) RefundResult {
	if rec.ID == 0 {
		e.metrics.Refund(false)
		return RefundResult{Err: ErrUnknownReceipt}
	}
	if rec.Tenant == "" {
		e.metrics.Refund(false)
		return RefundResult{Err: ErrReceiptMismatch}
	}
	spec, _, ok := e.resolve(Key{Tenant: rec.Tenant})
	if !ok || spec.Quota == nil {
		e.metrics.Refund(false)
		return RefundResult{Err: ErrUnknownPolicy}
	}
	l := e.ledgerFor(spec)
	if l == nil { // 引擎已关闭
		e.metrics.Refund(false)
		return RefundResult{Err: ErrClosed}
	}
	out := l.Refund(ledger.Receipt{
		ID:     rec.ID,
		Tenant: rec.Tenant,
		Units:  rec.Units,
		At:     rec.At,
	})
	e.metrics.Refund(out.OK)
	res := RefundResult{
		OK:        out.OK,
		Remaining: out.Remaining,
		Used:      out.Used,
		Debt:      out.Debt,
	}
	if !out.OK {
		switch out.Reason {
		case "already_refunded":
			res.Err = ErrReceiptUsed
		case "tenant_mismatch":
			res.Err = ErrReceiptMismatch
		default:
			res.Err = ErrUnknownReceipt
		}
	}
	return res
}

// PeekQuota 只读租户配额。
func (e *Engine) PeekQuota(tenant string) (used, remaining, debt int64, ok bool) {
	spec, _, hit := e.resolve(Key{Tenant: tenant})
	if !hit || spec.Quota == nil {
		return 0, 0, 0, false
	}
	l := e.ledgerFor(spec)
	if l == nil { // 引擎已关闭
		return 0, 0, 0, false
	}
	a := l.Peek(tenant)
	return a.Used, a.Remaining(), a.Debt(), true
}

// ResetQuota 清除租户当前周期账户（收据仍按 ID 防双花）。
func (e *Engine) ResetQuota(tenant string) {
	spec, _, hit := e.resolve(Key{Tenant: tenant})
	if !hit || spec.Quota == nil {
		return
	}
	l := e.ledgerFor(spec)
	if l == nil { // 引擎已关闭
		return
	}
	l.Reset(tenant)
}
