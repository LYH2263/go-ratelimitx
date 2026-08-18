package units

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Per 表示「在 Period 内产生 Count 个令牌」。
type Per struct {
	Count  int64
	Period time.Duration
}

// TokensPerSecond 把 Per 换算成每秒令牌数。Period<=0 时返回 0。
func (p Per) TokensPerSecond() float64 {
	if p.Period <= 0 || p.Count <= 0 {
		return 0
	}
	return float64(p.Count) / p.Period.Seconds()
}

// GrainsPerSecond 把 Per 换算成 grains/秒。1 token = Grain 个 grains。
func (p Per) GrainsPerSecond(grain uint64) uint64 {
	if p.Period <= 0 || p.Count <= 0 || grain == 0 {
		return 0
	}
	// count * grain / period_seconds = count * grain * 1e9 / period_ns
	ns := uint64(p.Period.Nanoseconds())
	if ns == 0 {
		return 0
	}
	return mulDiv(uint64(p.Count)*grain, uint64(time.Second), ns)
}

func mulDiv(a, b, div uint64) uint64 {
	if div == 0 {
		return 0
	}
	// a is typically small; do it in two steps if needed
	hi := a / div
	lo := a % div
	return hi*b + lo*b/div
}

// ParseRate 解析 "100/s"、"10/ms"、"6000/m"、"1/h"、"50/sec" 等形式。
func ParseRate(s string) (Per, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Per{}, fmt.Errorf("units: empty rate")
	}
	i := strings.IndexByte(s, '/')
	if i <= 0 || i == len(s)-1 {
		return Per{}, fmt.Errorf("units: invalid rate %q", s)
	}
	countStr := strings.TrimSpace(s[:i])
	unitStr := strings.TrimSpace(s[i+1:])
	count, err := strconv.ParseInt(countStr, 10, 64)
	if err != nil || count <= 0 {
		return Per{}, fmt.Errorf("units: invalid count in %q", s)
	}
	period, err := ParseUnit(unitStr)
	if err != nil {
		return Per{}, err
	}
	return Per{Count: count, Period: period}, nil
}

// ParseUnit 把 s/sec/ms/m/h/d 等缩写解析为 Duration。
func ParseUnit(s string) (time.Duration, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "ns", "nsec", "nanosecond", "nanoseconds":
		return time.Nanosecond, nil
	case "us", "µs", "usec", "microsecond", "microseconds":
		return time.Microsecond, nil
	case "ms", "msec", "millisecond", "milliseconds":
		return time.Millisecond, nil
	case "s", "sec", "second", "seconds":
		return time.Second, nil
	case "m", "min", "minute", "minutes":
		return time.Minute, nil
	case "h", "hr", "hour", "hours":
		return time.Hour, nil
	case "d", "day", "days":
		return 24 * time.Hour, nil
	default:
		if d, err := time.ParseDuration(s); err == nil {
			return d, nil
		}
		return 0, fmt.Errorf("units: unknown period %q", s)
	}
}

// ParseDuration 接受 Go duration 或纯单位缩写（如 "24h"、"1s"、"500ms"）。
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("units: empty duration")
	}
	d, err := time.ParseDuration(s)
	if err == nil {
		if d <= 0 {
			return 0, fmt.Errorf("units: duration must be positive")
		}
		return d, nil
	}
	// 纯数字视为秒
	allDigit := true
	for _, r := range s {
		if !unicode.IsDigit(r) {
			allDigit = false
			break
		}
	}
	if allDigit {
		n, convErr := strconv.ParseInt(s, 10, 64)
		if convErr != nil || n <= 0 {
			return 0, fmt.Errorf("units: invalid duration %q", s)
		}
		return time.Duration(n) * time.Second, nil
	}
	return 0, fmt.Errorf("units: invalid duration %q", s)
}

// FormatRate 把 Per 格式化为 "count/unit"。
func FormatRate(p Per) string {
	switch p.Period {
	case time.Millisecond:
		return fmt.Sprintf("%d/ms", p.Count)
	case time.Second:
		return fmt.Sprintf("%d/s", p.Count)
	case time.Minute:
		return fmt.Sprintf("%d/m", p.Count)
	case time.Hour:
		return fmt.Sprintf("%d/h", p.Count)
	case 24 * time.Hour:
		return fmt.Sprintf("%d/d", p.Count)
	default:
		return fmt.Sprintf("%d/%s", p.Count, p.Period)
	}
}

// MustParseRate 解析失败时 panic，仅用于测试夹具。
func MustParseRate(s string) Per {
	p, err := ParseRate(s)
	if err != nil {
		panic(err)
	}
	return p
}
