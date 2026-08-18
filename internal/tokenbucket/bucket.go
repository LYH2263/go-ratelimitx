package tokenbucket

import (
	"time"

	"github.com/LYH2263/go-ratelimitx/internal/arith"
)

// Grain 是一个令牌被拆成的内部整数单位，避免浮点漂移。
const Grain uint64 = 1_000_000

// MaxTokens 是 Burst/单次 n 的上限，防止 grains 溢出。
const MaxTokens = 1_000_000_000

// State 是经典令牌桶的可变状态。
type State struct {
	Grains uint64 // 当前库存，单位 grain
	Last   int64  // 上次填充的 Unix 纳秒
	Rate   uint64 // grains / second
	Burst  uint64 // grains 上限
}

// Config 由速率（令牌/秒）与突发（令牌个数）构造。
type Config struct {
	PerSecond float64
	Burst     int
}

// GrainsPerSecond 把令牌/秒换成 grains/秒。四舍五入到最近整数，至少为 1（若 PerSecond>0）。
func GrainsPerSecond(perSecond float64) uint64 {
	if perSecond <= 0 {
		return 0
	}
	g := perSecond * float64(Grain)
	if g >= float64(^uint64(0)>>1) {
		return ^uint64(0) >> 1
	}
	u := uint64(g + 0.5)
	if u == 0 {
		return 1
	}
	return u
}

// BurstGrains 把突发令牌个数换成 grains。
func BurstGrains(burst int) uint64 {
	if burst <= 0 {
		return 0
	}
	if burst > MaxTokens {
		burst = MaxTokens
	}
	return uint64(burst) * Grain
}

// NewState 创建满桶状态。now 为初始 Last。速率或突发非法时返回零值 State。
func NewState(cfg Config, now time.Time) State {
	rate := GrainsPerSecond(cfg.PerSecond)
	burst := BurstGrains(cfg.Burst)
	return State{
		Grains: burst,
		Last:   now.UnixNano(),
		Rate:   rate,
		Burst:  burst,
	}
}

// Valid 报告配置是否可运行。
func (c Config) Valid() bool {
	return c.PerSecond > 0 && c.Burst >= 1 && c.Burst <= MaxTokens
}

// Refill 按 elapsed 把令牌补到 Burst。时钟回拨不补不扣。
func (s *State) Refill(now time.Time) {
	if s.Rate == 0 || s.Burst == 0 {
		return
	}
	nowNs := now.UnixNano()
	if nowNs <= s.Last {
		s.Last = nowNs
		return
	}
	elapsed := uint64(nowNs - s.Last)
	add := arith.MulDiv(s.Rate, elapsed, uint64(time.Second))
	s.Grains = arith.MinU64(s.Burst, arith.SaturatingAddU64(s.Grains, add))
	s.Last = nowNs
}

// Tokens 返回当前令牌数的浮点近似（不 refill）。
func (s *State) Tokens() float64 {
	return float64(s.Grains) / float64(Grain)
}

// BurstTokens 返回突发上限的令牌数。
func (s *State) BurstTokens() int {
	if Grain == 0 {
		return 0
	}
	return int(s.Burst / Grain)
}

// Need 把 n 个令牌换成 grains。n<=0 返回 0。
func Need(n int) uint64 {
	if n <= 0 {
		return 0
	}
	if n > MaxTokens {
		n = MaxTokens
	}
	return uint64(n) * Grain
}

// Result 是一次 Allow 的结果。
type Result struct {
	OK        bool
	Wait      time.Duration
	Remaining float64
	Impossible bool
}

// Allow 尝试消耗 n 个令牌。n<=0 视为拒绝且 Wait=0。
func (s *State) Allow(n int, now time.Time) Result {
	if n <= 0 {
		return Result{OK: false, Remaining: s.Tokens()}
	}
	need := Need(n)
	if need > s.Burst {
		return Result{OK: false, Wait: -1, Remaining: s.Tokens(), Impossible: true}
	}
	s.Refill(now)
	if s.Grains >= need {
		s.Grains -= need
		return Result{OK: true, Remaining: s.Tokens()}
	}
	wait := s.waitFor(need)
	return Result{OK: false, Wait: wait, Remaining: s.Tokens()}
}

// Peek 只 refill 并计算是否可过，不扣减。
func (s *State) Peek(n int, now time.Time) Result {
	if n <= 0 {
		return Result{OK: false, Remaining: s.Tokens()}
	}
	need := Need(n)
	if need > s.Burst {
		return Result{OK: false, Wait: -1, Remaining: s.Tokens(), Impossible: true}
	}
	s.Refill(now)
	if s.Grains >= need {
		return Result{OK: true, Remaining: s.Tokens()}
	}
	return Result{OK: false, Wait: s.waitFor(need), Remaining: s.Tokens()}
}

// Restore 把 n 个令牌还回桶内，不超过 Burst。用于 Reservation.Cancel。
func (s *State) Restore(n int, now time.Time) {
	if n <= 0 {
		return
	}
	s.Refill(now)
	s.Grains = arith.MinU64(s.Burst, arith.SaturatingAddU64(s.Grains, Need(n)))
}

func (s *State) waitFor(need uint64) time.Duration {
	if s.Rate == 0 {
		return -1
	}
	if need <= s.Grains {
		return 0
	}
	deficit := need - s.Grains
	ns := arith.MulDivRoundUp(deficit, uint64(time.Second), s.Rate)
	if ns > uint64(time.Hour*24*365) {
		return time.Hour * 24 * 365
	}
	return time.Duration(ns)
}
