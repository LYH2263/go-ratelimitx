package ratelimitx

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/LYH2263/go-ratelimitx/internal/policy"
	"github.com/LYH2263/go-ratelimitx/internal/store"
)

// Allow 尝试消耗 n 个单位。ok 为假时 wait 是建议等待；永远不可能则为 ImpossibleWait。
func (e *Engine) Allow(key string, n int) (ok bool, wait time.Duration) {
	d := e.AllowN(key, n)
	return d.OK, d.Wait
}

// AllowKey 使用结构化复合键。
func (e *Engine) AllowKey(k Key, n int) (ok bool, wait time.Duration) {
	d := e.AllowN(k.Encode(), n)
	return d.OK, d.Wait
}

// AllowN 返回详细 Decision。
func (e *Engine) AllowN(key string, n int) Decision {
	d, _ := e.decide(key, n, true)
	return d
}

// Peek 只读查询，不消耗。
func (e *Engine) Peek(key string) PeekInfo {
	k, err := parseAllowKey(key)
	if err != nil {
		return PeekInfo{}
	}
	spec, _, ok := e.resolve(k)
	if !ok || !spec.HasLimiter() {
		return PeekInfo{Exists: false}
	}
	now := e.now()
	sk := e.storeKey(spec.Name, k.Encode())
	e.locks.Lock(sk)
	defer e.locks.Unlock(sk)
	ent, exists := e.store.Get(sk)
	var st *limiterState
	if exists {
		st, _ = ent.Payload.(*limiterState)
	}
	if st == nil {
		st = newLimiterState(spec, now)
	}
	r := st.peek(1, now)
	e.metrics.Peek()
	wait := r.wait
	if r.impossible {
		wait = ImpossibleWait
	}
	return PeekInfo{
		Policy:    spec.Name,
		Algorithm: Algorithm(spec.InferAlgo()),
		Tokens:    st.remainingTokens(),
		Burst:     st.burst(),
		WindowIn:  st.windowIn(),
		WindowLim: st.windowLim(),
		Wait:      wait,
		Exists:    exists,
	}
}

// Reset 删除该键的限流状态（不影响配额账本）。
func (e *Engine) Reset(key string) {
	k, err := parseAllowKey(key)
	if err != nil {
		return
	}
	spec, _, ok := e.resolve(k)
	if !ok {
		return
	}
	sk := e.storeKey(spec.Name, k.Encode())
	e.locks.Lock(sk)
	e.store.Delete(sk)
	e.locks.Unlock(sk)
	e.metrics.Reset()
}

// Reserve 预留 n 个单位，失败不扣减。成功后 Cancel 归还。
func (e *Engine) Reserve(key string, n int) Reservation {
	d, cancel := e.decide(key, n, true)
	if !d.OK {
		return Reservation{ok: false, wait: d.Wait, n: n, key: key, policy: d.Policy}
	}
	var once bool
	return Reservation{
		ok:     true,
		wait:   0,
		n:      n,
		key:    key,
		policy: d.Policy,
		cancel: func() {
			if once || cancel == nil {
				return
			}
			once = true
			cancel()
			e.metrics.Cancel()
		},
	}
}

func (e *Engine) decide(key string, n int, consume bool) (Decision, func()) {
	if n <= 0 {
		d := Decision{OK: false, Wait: 0, Reason: DenyInvalidN, Err: ErrInvalidN}
		e.metrics.Allow(false, "", string(DenyInvalidN), 0)
		return d, nil
	}
	k, err := parseAllowKey(key)
	if err != nil {
		reason := DenyEmptyKey
		if err == ErrBadDimension {
			reason = DenyEmptyKey
		}
		d := Decision{OK: false, Reason: reason, Err: err}
		e.metrics.Allow(false, "", string(reason), 0)
		return d, nil
	}
	spec, _, ok := e.resolve(k)
	if !ok || !spec.HasLimiter() {
		d := Decision{OK: false, Reason: DenyNoPolicy, Err: ErrUnknownPolicy}
		e.metrics.Allow(false, "", string(DenyNoPolicy), 0)
		return d, nil
	}
	now := e.now()
	sk := e.storeKey(spec.Name, k.Encode())
	e.locks.Lock(sk)
	defer e.locks.Unlock(sk)

	ent, _ := e.store.GetOrCreate(sk, kindOf(spec), func() any {
		return newLimiterState(spec, now)
	})
	st, _ := ent.Payload.(*limiterState)
	if st == nil {
		st = newLimiterState(spec, now)
		ent.Payload = st
		ent.Kind = kindOf(spec)
	}

	var r stepResult
	if consume {
		r = st.allow(n, now)
	} else {
		r = st.peek(n, now)
	}
	// 持久化失败时必须回滚内存扣减，否则 Peek（内存态）与持久态失步。
	var persistErr error
	if consume && r.ok {
		persistErr = e.store.Persist(sk, persistBlob(st))
		if persistErr != nil {
			st.restore(n, now)
			r.ok = false
			r.remaining = st.remainingTokens()
			r.reason = DenyPersist
		}
	}
	wait := r.wait
	if r.impossible {
		wait = ImpossibleWait
	}
	d := Decision{
		OK:         r.ok,
		Wait:       wait,
		Remaining:  r.remaining,
		Limit:      st.burst(),
		RetryAfter: wait,
		Policy:     spec.Name,
		Algorithm:  Algorithm(spec.InferAlgo()),
		Reason:     r.reason,
	}
	if r.impossible {
		d.Err = ErrImpossible
		d.Reason = DenyImpossible
	}
	if persistErr != nil {
		d.Err = fmt.Errorf("%w: %w", ErrPersist, persistErr)
	}
	ent.Touch(now, r.ok)
	e.metrics.Allow(r.ok, spec.Name, string(d.Reason), wait)
	if consume && r.ok {
		e.metrics.Reserve(true)
		encoded := k.Encode()
		pname := spec.Name
		nn := n
		cancel := func() {
			e.restore(pname, encoded, nn)
		}
		return d, cancel
	}
	return d, nil
}

func (e *Engine) restore(policyName, encoded string, n int) {
	sk := e.storeKey(policyName, encoded)
	e.locks.Lock(sk)
	defer e.locks.Unlock(sk)
	ent, ok := e.store.Get(sk)
	if !ok {
		return
	}
	st, _ := ent.Payload.(*limiterState)
	if st == nil {
		return
	}
	st.restore(n, e.now())
}

func kindOf(spec policy.Spec) store.Kind {
	switch spec.InferAlgo() {
	case policy.AlgoGCRA:
		return store.KindGCRA
	case policy.AlgoSlidingLog:
		return store.KindSlidingLog
	case policy.AlgoSlidingCounter:
		return store.KindSlidingCounter
	default:
		return store.KindTokenBucket
	}
}

// UseFailingBackend 注入一个 Set 总失败的内存后端。
func (e *Engine) UseFailingBackend(err error) {
	b := store.NewMemoryBackend()
	b.SetError(err)
	e.store.Attach(b)
}

func persistBlob(st *limiterState) []byte {
	if st == nil || st.tb == nil {
		return []byte{0}
	}
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, st.tb.Grains)
	return b
}
