package composite

// FallbackChain 返回策略查找的固定回退序列（从最具体到默认）。
//
// 顺序（硬不变量）：
//  1. tenant|route|ip
//  2. tenant|route|*
//  3. tenant|*|*
//  4. *|*|*
//
// 某维本身已是空或 * 时，对应更具体的层级会与下一层重复，仍按序去重后返回。
func FallbackChain(k Key) []string {
	t := norm(k.Tenant)
	r := norm(k.Route)
	ip := norm(k.IP)
	raw := []string{
		t + Sep + r + Sep + ip,
		t + Sep + r + Sep + Wildcard,
		t + Sep + Wildcard + Sep + Wildcard,
		Wildcard + Sep + Wildcard + Sep + Wildcard,
	}
	return dedup(raw)
}

func norm(s string) string {
	if s == "" {
		return Wildcard
	}
	return s
}

func dedup(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

// FallbackKeys 返回回退链对应的 Key 值（通配写作 *）。
func FallbackKeys(k Key) []Key {
	chain := FallbackChain(k)
	out := make([]Key, 0, len(chain))
	for _, s := range chain {
		p, err := Parse(s)
		if err != nil {
			continue
		}
		out = append(out, p)
	}
	return out
}

// MatchPattern 判断具体键是否命中模式（模式中 * 匹配任意，含空）。
func MatchPattern(concrete, pattern Key) bool {
	return matchPart(concrete.Tenant, pattern.Tenant) &&
		matchPart(concrete.Route, pattern.Route) &&
		matchPart(concrete.IP, pattern.IP)
}

func matchPart(value, pat string) bool {
	if pat == "" || pat == Wildcard {
		return true
	}
	return value == pat
}

// MoreSpecific 报告 a 是否比 b 更具体（specificity 更大；同则按 tenant>route>ip 字典序更具体的已填维）。
func MoreSpecific(a, b Key) bool {
	sa, sb := a.Specificity(), b.Specificity()
	if sa != sb {
		return sa > sb
	}
	// 同特异性：优先有 tenant，再 route，再 ip
	for _, d := range []Dim{DimTenant, DimRoute, DimIP} {
		ha, hb := a.Has(d), b.Has(d)
		if ha != hb {
			return ha
		}
	}
	return false
}
