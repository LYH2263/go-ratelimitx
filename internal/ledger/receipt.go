package ledger

import "time"

// Receipt 记录一次成功扣减，退款必须出示同一 ID。
type Receipt struct {
	ID     uint64
	Tenant string
	Units  int64
	At     time.Time
}

// receiptState 是内部收据状态。
type receiptState struct {
	Receipt
	Refunded bool
	RefundAt time.Time
}

// CanRefund 报告收据是否仍可退。
func (s *receiptState) CanRefund() bool {
	return s != nil && !s.Refunded && s.Units > 0
}
