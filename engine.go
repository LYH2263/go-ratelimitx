package ratelimitx

import (
	"sync"
	"time"

	"github.com/LYH2263/go-ratelimitx/internal/clock"
	"github.com/LYH2263/go-ratelimitx/internal/composite"
	"github.com/LYH2263/go-ratelimitx/internal/ledger"
	"github.com/LYH2263/go-ratelimitx/internal/metrics"
	"github.com/LYH2263/go-ratelimitx/internal/policy"
	"github.com/LYH2263/go-ratelimitx/internal/store"
	"github.com/LYH2263/go-ratelimitx/internal/syncutil"
)

// Engine 是限流与配额引擎。
type Engine struct {
	clock   clock.Clock
	store   *store.Store
	locks   *syncutil.Sharded
	table   *policy.Table
	metrics *metrics.Collector
	idleTTL time.Duration
	shards  int

	ledgerMu sync.Mutex
	ledgers  map[string]*ledger.Ledger // policy name -> ledger
	closed   bool
}

// New 构造引擎。未注册任何策略时 Allow 会因无策略而拒绝。
func New(opts ...Option) *Engine {
	cfg := defaultConfig()
	for _, o := range opts {
		if o != nil {
			o(&cfg)
		}
	}
	n := clampShards(cfg.shards)
	clk := cfg.clock
	if clk == nil {
		clk = clock.Real{}
	}
	mc := cfg.metrics
	if mc == nil {
		mc = metrics.New()
	}
	return &Engine{
		clock:   clk,
		store:   store.New(n, clk),
		locks:   syncutil.New(n),
		table:   policy.NewTable(cfg.defaultPolicy),
		metrics: mc,
		idleTTL: cfg.idleTTL,
		shards:  n,
		ledgers: make(map[string]*ledger.Ledger),
	}
}

// Shards 返回分片数。
func (e *Engine) Shards() int { return e.shards }

// Register 注册命名策略。
func (e *Engine) Register(s Spec) error {
	ps, err := toPolicySpec(s)
	if err != nil {
		return err
	}
	if err := e.table.Register(ps); err != nil {
		return ErrDuplicatePolicy
	}
	return nil
}

// Bind 把模式（tenant|route|ip，可用 *）绑到策略。
func (e *Engine) Bind(pattern, policyName string) error {
	if err := e.table.Bind(pattern, policyName); err != nil {
		return err
	}
	return nil
}

// LoadText 加载策略 DSL。
func (e *Engine) LoadText(src string) error {
	return e.table.LoadText(src)
}

// Sweep 回收空闲状态。未配置 IdleTTL 时返回 0。
func (e *Engine) Sweep() int {
	n := e.store.Sweep(e.idleTTL)
	e.metrics.Sweep(n)
	return n
}

// StoreLen 返回当前状态条目数。
func (e *Engine) StoreLen() int { return e.store.Len() }

// StoreStats 返回分片存储汇总。
func (e *Engine) StoreStats() store.Stats { return e.store.AggregateStats() }

// Metrics 返回指标快照。
func (e *Engine) Metrics() metrics.Snapshot { return e.metrics.Snapshot() }

// Policies 列出已注册策略名。
func (e *Engine) Policies() []string { return e.table.Names() }

func (e *Engine) resolve(k Key) (policy.Spec, string, bool) {
	return e.table.Resolve(toComposite(k))
}

func (e *Engine) now() time.Time { return e.clock.Now() }

func (e *Engine) storeKey(policyName string, encoded string) string {
	return policyName + "\x00" + encoded
}

func (e *Engine) shardIndex(key string) int {
	return store.Hash(key, e.shards)
}

// Close 关闭引擎与底层存储。之后 Allow 必须返回 ErrClosed，不得 panic。
func (e *Engine) Close() error {
	e.closed = true
	return e.store.Close()
}

func (e *Engine) ledgerFor(spec policy.Spec) *ledger.Ledger {
	if spec.Quota == nil {
		return nil
	}
	e.ledgerMu.Lock()
	defer e.ledgerMu.Unlock()
	if l, ok := e.ledgers[spec.Name]; ok {
		return l
	}
	l := ledger.New(e.clock, ledger.Plan{
		Period: spec.Quota.Period,
		Soft:   spec.Quota.Soft,
		Hard:   spec.Quota.Hard,
		Debt:   spec.Quota.Debt,
	})
	e.ledgers[spec.Name] = l
	return l
}

func toPolicySpec(s Spec) (policy.Spec, error) {
	ps := policy.Spec{
		Name: s.Name,
		Algo: policy.Algorithm(s.Algo),
	}
	if s.Rate != nil {
		ps.Rate = &policy.Rate{PerSecond: s.Rate.PerSecond, Burst: s.Rate.Burst}
	}
	if s.Window != nil {
		ps.Window = &policy.Window{Size: s.Window.Size, Limit: s.Window.Limit, Buckets: s.Window.Buckets}
	}
	if s.Quota != nil {
		ps.Quota = &policy.QuotaPlan{Period: s.Quota.Period, Soft: s.Quota.Soft, Hard: s.Quota.Hard, Debt: s.Quota.Debt}
	}
	if err := ps.Validate(); err != nil {
		return policy.Spec{}, ErrInvalidSpec
	}
	return ps, nil
}

func fromPolicySpec(s policy.Spec) Spec {
	out := Spec{Name: s.Name, Algo: Algorithm(s.InferAlgo())}
	if s.Rate != nil {
		out.Rate = &Rate{PerSecond: s.Rate.PerSecond, Burst: s.Rate.Burst}
	}
	if s.Window != nil {
		out.Window = &Window{Size: s.Window.Size, Limit: s.Window.Limit, Buckets: s.Window.Buckets}
	}
	if s.Quota != nil {
		out.Quota = &QuotaPlan{Period: s.Quota.Period, Soft: s.Quota.Soft, Hard: s.Quota.Hard, Debt: s.Quota.Debt}
	}
	return out
}

// FallbackPatterns 导出固定回退链，便于测试与文档对齐。
func FallbackPatterns(k Key) []string {
	return composite.FallbackChain(toComposite(k))
}

// ShardOf 返回键对应的分片下标。
func (e *Engine) ShardOf(key string) int {
	return e.shardIndex(key)
}
