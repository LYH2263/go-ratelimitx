package ratelimitx

import (
	"time"

	"github.com/LYH2263/go-ratelimitx/internal/policy"
	"github.com/LYH2263/go-ratelimitx/internal/slidingwin"
	"github.com/LYH2263/go-ratelimitx/internal/tokenbucket"
)

// limiterState 保存在分片存储里的算法状态。Rate 与 Window 可同时存在。
type limiterState struct {
	algo policy.Algorithm
	tb   *tokenbucket.State
	gcra *tokenbucket.GCRAState
	log  *slidingwin.Log
	ctr  *slidingwin.Counter
}

func newLimiterState(spec policy.Spec, now time.Time) *limiterState {
	algo := spec.InferAlgo()
	st := &limiterState{algo: algo}
	if spec.Rate != nil {
		cfg := tokenbucket.Config{PerSecond: spec.Rate.PerSecond, Burst: spec.Rate.Burst}
		switch algo {
		case policy.AlgoGCRA:
			g := tokenbucket.NewGCRA(cfg, now)
			st.gcra = &g
		default:
			tb := tokenbucket.NewState(cfg, now)
			st.tb = &tb
		}
	}
	if spec.Window != nil {
		switch algo {
		case policy.AlgoSlidingCounter:
			st.ctr = slidingwin.NewCounter(spec.Window.Size, spec.Window.Limit, spec.Window.Buckets)
		default:
			st.log = slidingwin.NewLog(spec.Window.Size, spec.Window.Limit)
		}
		// 若仅有窗口，algo 已是 sliding*
		// 若同时有 rate，窗口作为第二道闸，tb/gcra 已建
	}
	return st
}

type stepResult struct {
	ok         bool
	wait       time.Duration
	remaining  float64
	inWindow   int
	impossible bool
	reason     DenyReason
}

func (s *limiterState) allow(n int, now time.Time) stepResult {
	rateRes, rateApplied := s.allowRate(n, now)
	if rateApplied && !rateRes.ok {
		return rateRes
	}
	winRes, winApplied := s.allowWindow(n, now)
	if winApplied && !winRes.ok {
		if rateApplied && rateRes.ok {
			s.restoreRate(n, now)
		}
		return winRes
	}
	if rateApplied {
		return rateRes
	}
	if winApplied {
		return winRes
	}
	return stepResult{ok: true}
}

func (s *limiterState) peek(n int, now time.Time) stepResult {
	if s.tb != nil {
		r := s.tb.Peek(n, now)
		out := tbToStep(r, DenyRate)
		if !out.ok {
			return out
		}
	}
	if s.gcra != nil {
		r := s.gcra.Peek(n, now)
		out := tbToStep(r, DenyRate)
		if !out.ok {
			return out
		}
	}
	if s.log != nil {
		r := s.log.Peek(n, now)
		out := winToStep(r)
		if !out.ok {
			return out
		}
		return out
	}
	if s.ctr != nil {
		r := s.ctr.Peek(n, now)
		out := winToStep(r)
		if !out.ok {
			return out
		}
		return out
	}
	if s.tb != nil {
		r := s.tb.Peek(n, now)
		return tbToStep(r, DenyRate)
	}
	if s.gcra != nil {
		r := s.gcra.Peek(n, now)
		return tbToStep(r, DenyRate)
	}
	return stepResult{ok: true}
}

func (s *limiterState) allowRate(n int, now time.Time) (stepResult, bool) {
	if s.tb != nil {
		return tbToStep(s.tb.Allow(n, now), DenyRate), true
	}
	if s.gcra != nil {
		return tbToStep(s.gcra.Allow(n, now), DenyRate), true
	}
	return stepResult{}, false
}

func (s *limiterState) allowWindow(n int, now time.Time) (stepResult, bool) {
	if s.log != nil {
		return winToStep(s.log.Allow(n, now)), true
	}
	if s.ctr != nil {
		return winToStep(s.ctr.Allow(n, now)), true
	}
	return stepResult{}, false
}

func (s *limiterState) restoreRate(n int, now time.Time) {
	if s.tb != nil {
		s.tb.Restore(n, now)
	}
	if s.gcra != nil {
		s.gcra.Restore(n, now)
	}
}

func (s *limiterState) restore(n int, now time.Time) {
	if s.log != nil {
		s.log.Restore(n)
	}
	if s.ctr != nil {
		s.ctr.Restore(n)
	}
	s.restoreRate(n, now)
}

func (s *limiterState) remainingTokens() float64 {
	if s.tb != nil {
		return s.tb.Tokens()
	}
	if s.gcra != nil {
		return 0
	}
	return 0
}

func (s *limiterState) burst() int {
	if s.tb != nil {
		return s.tb.BurstTokens()
	}
	if s.gcra != nil {
		return s.gcra.Burst
	}
	return 0
}

func (s *limiterState) windowIn() int {
	if s.log != nil {
		return s.log.InWindow()
	}
	return 0
}

func (s *limiterState) windowLim() int {
	if s.log != nil {
		return s.log.Limit
	}
	if s.ctr != nil {
		return s.ctr.Limit
	}
	return 0
}

func tbToStep(r tokenbucket.Result, reason DenyReason) stepResult {
	out := stepResult{
		ok:         r.OK,
		wait:       r.Wait,
		remaining:  r.Remaining,
		impossible: r.Impossible,
	}
	if !r.OK {
		if r.Impossible {
			out.reason = DenyImpossible
		} else {
			out.reason = reason
		}
	}
	return out
}

func winToStep(r slidingwin.Result) stepResult {
	out := stepResult{
		ok:         r.OK,
		wait:       r.Wait,
		inWindow:   r.InWindow,
		impossible: r.Impossible,
	}
	if !r.OK {
		if r.Impossible {
			out.reason = DenyImpossible
		} else {
			out.reason = DenyWindow
		}
	}
	return out
}

func (s *limiterState) reset(now time.Time) {
	s.resetRateFull(now)
	s.resetWindow()
}

func (s *limiterState) resetRateFull(now time.Time) {
	if s.tb != nil {
		s.tb.Grains = s.tb.Burst
		s.tb.Last = now.UnixNano()
	}
	if s.gcra != nil {
		s.gcra.TAT = now.UnixNano()
	}
}

func (s *limiterState) resetWindow() {
	if s.log != nil {
		s.log.Reset()
	}
	if s.ctr != nil {
		s.ctr.Reset()
	}
}
