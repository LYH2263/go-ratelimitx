package metrics

import "time"

// Snapshot 是指标只读拷贝。
type Snapshot struct {
	Allowed       int64
	Denied        int64
	Peeked        int64
	Resets        int64
	Reserved      int64
	Canceled      int64
	Charged       int64
	ChargeDenied  int64
	Refunded      int64
	RefundDenied  int64
	SoftHits      int64
	Sweeps        int64
	WaitAvg       time.Duration
	DeniedReason  map[string]int64
	AllowedPolicy map[string]int64
}

// TotalDecisions 返回 Allowed+Denied。
func (s Snapshot) TotalDecisions() int64 {
	return s.Allowed + s.Denied
}

// DenyRatio 返回拒绝占比。无决策时为 0。
func (s Snapshot) DenyRatio() float64 {
	tot := s.TotalDecisions()
	if tot == 0 {
		return 0
	}
	return float64(s.Denied) / float64(tot)
}

// Reason 读取某一拒绝原因计数。
func (s Snapshot) Reason(r string) int64 {
	if s.DeniedReason == nil {
		return 0
	}
	return s.DeniedReason[r]
}
