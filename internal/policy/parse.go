package policy

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/LYH2263/go-ratelimitx/internal/units"
)

// ParseText 解析迷你 DSL：
//
//	policy default
//	  algo tokenbucket
//	  rate 100/s
//	  burst 20
//	policy api
//	  algo slidinglog
//	  window 1s
//	  limit 50
//	policy billed
//	  period 24h
//	  soft 10000
//	  hard 12000
//	  debt
//	bind acme|/checkout|* api
//	bind *|*|* default
func ParseText(r io.Reader) ([]Spec, []Binding, error) {
	sc := bufio.NewScanner(r)
	var specs []Spec
	var binds []Binding
	var cur *Spec
	lineNo := 0
	flush := func() {
		if cur != nil {
			specs = append(specs, *cur)
			cur = nil
		}
	}
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		kw := strings.ToLower(fields[0])
		switch kw {
		case "policy":
			flush()
			if len(fields) < 2 {
				return nil, nil, fmt.Errorf("policy: line %d: policy needs a name", lineNo)
			}
			cur = &Spec{Name: fields[1]}
		case "algo", "algorithm":
			if cur == nil {
				return nil, nil, fmt.Errorf("policy: line %d: algo outside policy", lineNo)
			}
			if len(fields) < 2 {
				return nil, nil, fmt.Errorf("policy: line %d: algo needs a value", lineNo)
			}
			a, err := ParseAlgo(fields[1])
			if err != nil {
				return nil, nil, fmt.Errorf("policy: line %d: %w", lineNo, err)
			}
			cur.Algo = a
		case "rate":
			if cur == nil {
				return nil, nil, fmt.Errorf("policy: line %d: rate outside policy", lineNo)
			}
			if len(fields) < 2 {
				return nil, nil, fmt.Errorf("policy: line %d: rate needs a value", lineNo)
			}
			p, err := units.ParseRate(fields[1])
			if err != nil {
				return nil, nil, fmt.Errorf("policy: line %d: %w", lineNo, err)
			}
			if cur.Rate == nil {
				cur.Rate = &Rate{}
			}
			cur.Rate.PerSecond = p.TokensPerSecond()
		case "burst":
			if cur == nil {
				return nil, nil, fmt.Errorf("policy: line %d: burst outside policy", lineNo)
			}
			if len(fields) < 2 {
				return nil, nil, fmt.Errorf("policy: line %d: burst needs a value", lineNo)
			}
			n, err := strconv.Atoi(fields[1])
			if err != nil {
				return nil, nil, fmt.Errorf("policy: line %d: bad burst", lineNo)
			}
			if cur.Rate == nil {
				cur.Rate = &Rate{}
			}
			cur.Rate.Burst = n
		case "window":
			if cur == nil {
				return nil, nil, fmt.Errorf("policy: line %d: window outside policy", lineNo)
			}
			if len(fields) < 2 {
				return nil, nil, fmt.Errorf("policy: line %d: window needs a duration", lineNo)
			}
			d, err := units.ParseDuration(fields[1])
			if err != nil {
				return nil, nil, fmt.Errorf("policy: line %d: %w", lineNo, err)
			}
			if cur.Window == nil {
				cur.Window = &Window{}
			}
			cur.Window.Size = d
		case "limit":
			if cur == nil {
				return nil, nil, fmt.Errorf("policy: line %d: limit outside policy", lineNo)
			}
			if len(fields) < 2 {
				return nil, nil, fmt.Errorf("policy: line %d: limit needs a value", lineNo)
			}
			n, err := strconv.Atoi(fields[1])
			if err != nil {
				return nil, nil, fmt.Errorf("policy: line %d: bad limit", lineNo)
			}
			if cur.Window == nil {
				cur.Window = &Window{}
			}
			cur.Window.Limit = n
		case "buckets":
			if cur == nil {
				return nil, nil, fmt.Errorf("policy: line %d: buckets outside policy", lineNo)
			}
			if len(fields) < 2 {
				return nil, nil, fmt.Errorf("policy: line %d: buckets needs a value", lineNo)
			}
			n, err := strconv.Atoi(fields[1])
			if err != nil {
				return nil, nil, fmt.Errorf("policy: line %d: bad buckets", lineNo)
			}
			if cur.Window == nil {
				cur.Window = &Window{}
			}
			cur.Window.Buckets = n
		case "period":
			if cur == nil {
				return nil, nil, fmt.Errorf("policy: line %d: period outside policy", lineNo)
			}
			if len(fields) < 2 {
				return nil, nil, fmt.Errorf("policy: line %d: period needs a duration", lineNo)
			}
			d, err := units.ParseDuration(fields[1])
			if err != nil {
				return nil, nil, fmt.Errorf("policy: line %d: %w", lineNo, err)
			}
			if cur.Quota == nil {
				cur.Quota = &QuotaPlan{}
			}
			cur.Quota.Period = d
		case "soft":
			if cur == nil {
				return nil, nil, fmt.Errorf("policy: line %d: soft outside policy", lineNo)
			}
			if len(fields) < 2 {
				return nil, nil, fmt.Errorf("policy: line %d: soft needs a value", lineNo)
			}
			n, err := strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				return nil, nil, fmt.Errorf("policy: line %d: bad soft", lineNo)
			}
			if cur.Quota == nil {
				cur.Quota = &QuotaPlan{}
			}
			cur.Quota.Soft = n
		case "hard":
			if cur == nil {
				return nil, nil, fmt.Errorf("policy: line %d: hard outside policy", lineNo)
			}
			if len(fields) < 2 {
				return nil, nil, fmt.Errorf("policy: line %d: hard needs a value", lineNo)
			}
			n, err := strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				return nil, nil, fmt.Errorf("policy: line %d: bad hard", lineNo)
			}
			if cur.Quota == nil {
				cur.Quota = &QuotaPlan{}
			}
			cur.Quota.Hard = n
		case "debt":
			if cur == nil {
				return nil, nil, fmt.Errorf("policy: line %d: debt outside policy", lineNo)
			}
			if cur.Quota == nil {
				cur.Quota = &QuotaPlan{}
			}
			cur.Quota.Debt = true
		case "bind":
			if len(fields) < 3 {
				return nil, nil, fmt.Errorf("policy: line %d: bind needs pattern and policy", lineNo)
			}
			binds = append(binds, Binding{Pattern: fields[1], Policy: fields[2]})
		default:
			return nil, nil, fmt.Errorf("policy: line %d: unknown keyword %q", lineNo, fields[0])
		}
	}
	flush()
	if err := sc.Err(); err != nil {
		return nil, nil, err
	}
	for i := range specs {
		if err := specs[i].Validate(); err != nil {
			return nil, nil, err
		}
	}
	return specs, binds, nil
}

// ParseString 解析 DSL 字符串。
func ParseString(s string) ([]Spec, []Binding, error) {
	return ParseText(strings.NewReader(s))
}
