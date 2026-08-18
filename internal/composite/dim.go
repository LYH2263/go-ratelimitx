package composite

import (
	"strings"
	"unicode/utf8"
)

const (
	// Sep 是复合键维度分隔符。
	Sep = "|"
	// Wildcard 表示该维匹配任意值。
	Wildcard = "*"
	// Dims 是固定维度数：tenant, route, ip。
	Dims = 3
)

// Dim 标识复合键的一维。
type Dim int

const (
	DimTenant Dim = iota
	DimRoute
	DimIP
)

func (d Dim) String() string {
	switch d {
	case DimTenant:
		return "tenant"
	case DimRoute:
		return "route"
	case DimIP:
		return "ip"
	default:
		return "unknown"
	}
}

// Key 是三维度复合键。空字符串表示该维缺失（编码时仍占位）。
type Key struct {
	Tenant string
	Route  string
	IP     string
}

// Parts 返回固定顺序的三维切片。
func (k Key) Parts() [Dims]string {
	return [Dims]string{k.Tenant, k.Route, k.IP}
}

// Set 按维度写入。未知维度忽略。
func (k *Key) Set(d Dim, v string) {
	switch d {
	case DimTenant:
		k.Tenant = v
	case DimRoute:
		k.Route = v
	case DimIP:
		k.IP = v
	}
}

// Get 按维度读取。
func (k Key) Get(d Dim) string {
	switch d {
	case DimTenant:
		return k.Tenant
	case DimRoute:
		return k.Route
	case DimIP:
		return k.IP
	default:
		return ""
	}
}

// Has 报告该维是否非空且非通配。
func (k Key) Has(d Dim) bool {
	v := k.Get(d)
	return v != "" && v != Wildcard
}

// Specificity 返回已填充的具体维度数（非空且非 *）。
func (k Key) Specificity() int {
	n := 0
	for d := DimTenant; d <= DimIP; d++ {
		if k.Has(d) {
			n++
		}
	}
	return n
}

// Missing 列出缺失或通配的维度。
func (k Key) Missing() []Dim {
	var out []Dim
	for d := DimTenant; d <= DimIP; d++ {
		if !k.Has(d) {
			out = append(out, d)
		}
	}
	return out
}

// ContainsSep 报告 s 是否含分隔符。
func ContainsSep(s string) bool {
	return strings.Contains(s, Sep)
}

// ValidDim 报告维度值是否可编码：不含分隔符，且是合法 UTF-8。
func ValidDim(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	return !ContainsSep(s)
}

// Valid 报告三维均可编码。
func (k Key) Valid() bool {
	return ValidDim(k.Tenant) && ValidDim(k.Route) && ValidDim(k.IP)
}
