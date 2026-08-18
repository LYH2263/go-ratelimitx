package policy

import (
	"fmt"
	"strings"
	"time"

	"github.com/LYH2263/go-ratelimitx/internal/units"
)

// Algorithm 与对外 API 对齐的算法名。
type Algorithm string

const (
	AlgoUnspecified    Algorithm = ""
	AlgoTokenBucket    Algorithm = "tokenbucket"
	AlgoGCRA           Algorithm = "gcra"
	AlgoSlidingLog     Algorithm = "slidinglog"
	AlgoSlidingCounter Algorithm = "slidingcounter"
)

// Rate 速率规格。
type Rate struct {
	PerSecond float64
	Burst     int
}

// Window 窗口规格。
type Window struct {
	Size    time.Duration
	Limit   int
	Buckets int
}

// QuotaPlan 配额规格。
type QuotaPlan struct {
	Period time.Duration
	Soft   int64
	Hard   int64
	Debt   bool
}

// Spec 命名策略。
type Spec struct {
	Name   string
	Algo   Algorithm
	Rate   *Rate
	Window *Window
	Quota  *QuotaPlan
}

// InferAlgo 在 Algo 为空时根据字段推断。
func (s Spec) InferAlgo() Algorithm {
	if s.Algo != "" {
		return s.Algo
	}
	if s.Rate != nil && s.Window == nil {
		return AlgoTokenBucket
	}
	if s.Window != nil && s.Rate == nil {
		return AlgoSlidingLog
	}
	if s.Rate != nil && s.Window != nil {
		return AlgoTokenBucket
	}
	return AlgoUnspecified
}

// Validate 检查规格。至少需要 Rate、Window 或 Quota 之一。
func (s Spec) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("policy: empty name")
	}
	if s.Rate == nil && s.Window == nil && s.Quota == nil {
		return fmt.Errorf("policy %q: need rate, window or quota", s.Name)
	}
	if s.Rate != nil {
		if s.Rate.PerSecond <= 0 {
			return fmt.Errorf("policy %q: rate must be positive", s.Name)
		}
		if s.Rate.Burst < 1 {
			return fmt.Errorf("policy %q: burst must be >= 1", s.Name)
		}
	}
	if s.Window != nil {
		if s.Window.Size <= 0 {
			return fmt.Errorf("policy %q: window size must be positive", s.Name)
		}
		if s.Window.Limit < 1 {
			return fmt.Errorf("policy %q: window limit must be >= 1", s.Name)
		}
		if s.Window.Buckets < 0 {
			return fmt.Errorf("policy %q: buckets must be >= 0", s.Name)
		}
	}
	if s.Quota != nil {
		if s.Quota.Period <= 0 {
			return fmt.Errorf("policy %q: quota period must be positive", s.Name)
		}
		if s.Quota.Hard < 0 || s.Quota.Soft < 0 {
			return fmt.Errorf("policy %q: quota limits must be >= 0", s.Name)
		}
	}
	algo := s.InferAlgo()
	switch algo {
	case AlgoUnspecified, AlgoTokenBucket, AlgoGCRA, AlgoSlidingLog, AlgoSlidingCounter:
	default:
		return fmt.Errorf("policy %q: unknown algorithm %q", s.Name, algo)
	}
	if algo == AlgoGCRA && s.Rate == nil {
		return fmt.Errorf("policy %q: gcra requires rate", s.Name)
	}
	if (algo == AlgoSlidingLog || algo == AlgoSlidingCounter) && s.Window == nil {
		return fmt.Errorf("policy %q: sliding window requires window", s.Name)
	}
	return nil
}

// HasLimiter 报告该策略是否参与 Allow。
func (s Spec) HasLimiter() bool {
	return s.Rate != nil || s.Window != nil
}

// ParseAlgo 解析算法名。
func ParseAlgo(s string) (Algorithm, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "tokenbucket", "tb", "token-bucket":
		return AlgoTokenBucket, nil
	case "gcra", "leaky", "leakybucket":
		return AlgoGCRA, nil
	case "slidinglog", "log", "sliding-log":
		return AlgoSlidingLog, nil
	case "slidingcounter", "counter", "sliding-counter":
		return AlgoSlidingCounter, nil
	default:
		return "", fmt.Errorf("policy: unknown algorithm %q", s)
	}
}

// ParseRateLine 解析 "100/s" 与可选 burst。
func ParseRateLine(rate string, burst int) (*Rate, error) {
	p, err := units.ParseRate(rate)
	if err != nil {
		return nil, err
	}
	if burst < 1 {
		return nil, fmt.Errorf("policy: burst must be >= 1")
	}
	return &Rate{PerSecond: p.TokensPerSecond(), Burst: burst}, nil
}
