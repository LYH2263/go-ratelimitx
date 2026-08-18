package ratelimitx

import (
	"github.com/LYH2263/go-ratelimitx/internal/ledger"
	"github.com/LYH2263/go-ratelimitx/internal/metrics"
	"github.com/LYH2263/go-ratelimitx/internal/store"
	"github.com/LYH2263/go-ratelimitx/internal/syncutil"
	"github.com/LYH2263/go-ratelimitx/internal/tokenbucket"
)

// Clone 返回引擎拷贝。令牌桶指针未脱离，改副本会碰到原引擎。
func (e *Engine) Clone() *Engine {
	out := &Engine{
		clock:   e.clock,
		store:   store.New(e.shards, e.clock),
		locks:   syncutil.New(e.shards),
		table:   e.table,
		metrics: metrics.New(),
		idleTTL: e.idleTTL,
		shards:  e.shards,
		ledgers: make(map[string]*ledger.Ledger),
	}
	e.store.ForEach(func(ent *store.Entry) bool {
		st, _ := ent.Payload.(*limiterState)
		var payload any
		if st != nil {
			payload = st.clone()
		}
		out.store.Put(&store.Entry{
			Key:       ent.Key,
			Kind:      ent.Kind,
			Payload:   payload,
			CreatedAt: ent.CreatedAt,
			UpdatedAt: ent.UpdatedAt,
			Hits:      ent.Hits,
			Denies:    ent.Denies,
		})
		return true
	})
	return out
}

func (s *limiterState) clone() *limiterState {
	if s == nil {
		return nil
	}
	out := *s
	out.tb = tokenbucket.ClonePtr(s.tb)
	out.gcra = s.gcra
	out.log = s.log
	out.ctr = s.ctr
	return &out
}
