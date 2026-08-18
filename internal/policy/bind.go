package policy

// Binding 把复合键模式绑到策略名。
type Binding struct {
	Pattern string
	Policy  string
}

// Clone 深拷贝规格中的指针字段。
func (s Spec) Clone() Spec {
	out := s
	if s.Rate != nil {
		r := *s.Rate
		out.Rate = &r
	}
	if s.Window != nil {
		w := *s.Window
		out.Window = &w
	}
	if s.Quota != nil {
		q := *s.Quota
		out.Quota = &q
	}
	return out
}
