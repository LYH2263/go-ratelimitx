package ratelimitx

import "github.com/LYH2263/go-ratelimitx/internal/composite"

// NewKey 构造复合键。
func NewKey(tenant, route, ip string) Key {
	return Key{Tenant: tenant, Route: route, IP: ip}
}

// Encode 编码为 tenant|route|ip。
func (k Key) Encode() string {
	return composite.Encode(toComposite(k))
}

// ParseKey 解析三段式键。
func ParseKey(s string) (Key, error) {
	ck, err := composite.Parse(s)
	if err != nil {
		return Key{}, err
	}
	return fromComposite(ck), nil
}

// ParseKeyLoose 允许 1～3 段。
func ParseKeyLoose(s string) (Key, error) {
	ck, err := composite.ParseLoose(s)
	if err != nil {
		return Key{}, err
	}
	return fromComposite(ck), nil
}

func toComposite(k Key) composite.Key {
	return composite.Key{Tenant: k.Tenant, Route: k.Route, IP: k.IP}
}

func fromComposite(k composite.Key) Key {
	return Key{Tenant: k.Tenant, Route: k.Route, IP: k.IP}
}

func parseAllowKey(s string) (Key, error) {
	if s == "" {
		return Key{}, ErrEmptyKey
	}
	ck := composite.Opaque(s)
	if !ck.Valid() {
		return Key{}, ErrBadDimension
	}
	return fromComposite(ck), nil
}
