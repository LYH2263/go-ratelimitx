package metrics

// Kind 是计数类别。
type Kind string

const (
	KindAllowed       Kind = "allowed"
	KindDenied        Kind = "denied"
	KindPeeked        Kind = "peeked"
	KindReset         Kind = "reset"
	KindReserved      Kind = "reserved"
	KindCanceled      Kind = "canceled"
	KindCharged       Kind = "charged"
	KindChargeDenied  Kind = "charge_denied"
	KindRefunded      Kind = "refunded"
	KindRefundDenied  Kind = "refund_denied"
	KindSoftHit       Kind = "soft_hit"
	KindSweep         Kind = "sweep"
)

// ReasonKey 把拒绝原因编进标签。
func ReasonKey(reason string) string {
	if reason == "" {
		return "none"
	}
	return reason
}
