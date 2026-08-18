package ratelimitx

import (
	"time"

	"github.com/LYH2263/go-ratelimitx/internal/clock"
	"github.com/LYH2263/go-ratelimitx/internal/metrics"
)

const (
	defaultShards     = 32
	defaultPolicyName = "default"
	maxShards         = 4096
	minShards         = 1
)

// Option 配置 Engine。
type Option func(*engineConfig)

type engineConfig struct {
	clock         clock.Clock
	shards        int
	defaultPolicy string
	idleTTL       time.Duration
	metrics       *metrics.Collector
}

func defaultConfig() engineConfig {
	return engineConfig{
		clock:         clock.Real{},
		shards:        defaultShards,
		defaultPolicy: defaultPolicyName,
	}
}

// Clock 是可注入时钟，测试使用 FakeClock。
type Clock interface {
	Now() time.Time
}

type clockAdapter struct {
	c Clock
}

func (a clockAdapter) Now() time.Time { return a.c.Now() }

// WithClock 注入时钟。nil 被忽略。
func WithClock(c Clock) Option {
	return func(cfg *engineConfig) {
		if c != nil {
			cfg.clock = clockAdapter{c: c}
		}
	}
}

// WithShards 设置存储与锁的分片数。越界值在 New 时钳到 [1, 4096]。
func WithShards(n int) Option {
	return func(cfg *engineConfig) {
		cfg.shards = n
	}
}

// WithDefaultPolicy 设置回退链末端使用的策略名。
func WithDefaultPolicy(name string) Option {
	return func(cfg *engineConfig) {
		if name != "" {
			cfg.defaultPolicy = name
		}
	}
}

// WithIdleTTL 设置状态空闲超过 ttl 后可被 Sweep 回收。<=0 表示关闭。
func WithIdleTTL(ttl time.Duration) Option {
	return func(cfg *engineConfig) {
		if ttl > 0 {
			cfg.idleTTL = ttl
		}
	}
}

// WithMetricsCollector 注入外部指标收集器。
func WithMetricsCollector(c *metrics.Collector) Option {
	return func(cfg *engineConfig) {
		if c != nil {
			cfg.metrics = c
		}
	}
}

func clampShards(n int) int {
	if n < minShards {
		return defaultShards
	}
	if n > maxShards {
		return maxShards
	}
	return n
}
