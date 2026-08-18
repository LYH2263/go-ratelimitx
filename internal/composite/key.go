package composite

import (
	"fmt"
	"strings"
)

// Encode 把三维编码为 tenant|route|ip。始终输出三段，空维保留为空字段。
func Encode(k Key) string {
	return k.Tenant + Sep + k.Route + Sep + k.IP
}

// EncodePattern 把模式编码；空维写成 Wildcard，便于绑定表查找。
func EncodePattern(k Key) string {
	t, r, ip := k.Tenant, k.Route, k.IP
	if t == "" {
		t = Wildcard
	}
	if r == "" {
		r = Wildcard
	}
	if ip == "" {
		ip = Wildcard
	}
	return t + Sep + r + Sep + ip
}

// Parse 解析三段式复合键。字段数不是 3 时返回错误。
func Parse(s string) (Key, error) {
	if s == "" {
		return Key{}, fmt.Errorf("composite: empty key")
	}
	parts := strings.Split(s, Sep)
	if len(parts) != Dims {
		return Key{}, fmt.Errorf("composite: want %d fields, got %d in %q", Dims, len(parts), s)
	}
	k := Key{Tenant: parts[0], Route: parts[1], IP: parts[2]}
	if !k.Valid() {
		// Valid 只检查分隔符与 UTF-8；Split 后各段已不含 Sep。
		if !utf8OK(k.Tenant) || !utf8OK(k.Route) || !utf8OK(k.IP) {
			return Key{}, fmt.Errorf("composite: invalid utf-8")
		}
	}
	return k, nil
}

func utf8OK(s string) bool {
	return ValidDim(s) || !ContainsSep(s)
}

// ParseLoose 允许 1～3 段：一段当 tenant，两段当 tenant|route，三段完整。
func ParseLoose(s string) (Key, error) {
	if s == "" {
		return Key{}, fmt.Errorf("composite: empty key")
	}
	if strings.Count(s, Sep) == 0 {
		if !ValidDim(s) {
			return Key{}, fmt.Errorf("composite: invalid dimension")
		}
		return Key{Tenant: s}, nil
	}
	parts := strings.Split(s, Sep)
	switch len(parts) {
	case 1:
		return Key{Tenant: parts[0]}, nil
	case 2:
		return Key{Tenant: parts[0], Route: parts[1]}, nil
	case 3:
		return Key{Tenant: parts[0], Route: parts[1], IP: parts[2]}, nil
	default:
		return Key{}, fmt.Errorf("composite: too many fields in %q", s)
	}
}

// MustParse 解析失败时 panic，仅测试夹具使用。
func MustParse(s string) Key {
	k, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return k
}

// Opaque 把任意字符串当成单维 tenant 键（route/ip 空）。若含 Sep 则走 ParseLoose。
func Opaque(s string) Key {
	if s == "" {
		return Key{}
	}
	if strings.Contains(s, Sep) {
		k, err := ParseLoose(s)
		if err != nil {
			return Key{Tenant: strings.ReplaceAll(s, Sep, "_")}
		}
		return k
	}
	return Key{Tenant: s}
}

// Equal 三维全等。
func Equal(a, b Key) bool {
	return a.Tenant == b.Tenant && a.Route == b.Route && a.IP == b.IP
}
