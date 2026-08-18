package policy

import (
	"fmt"
	"sync"

	"github.com/LYH2263/go-ratelimitx/internal/composite"
)

// Table 保存命名策略与模式绑定。查找按复合键回退链，先命中先返回。
type Table struct {
	mu       sync.RWMutex
	specs    map[string]Spec
	binds    []Binding
	byPat    map[string]string // pattern -> policy name
	defaultN string
}

// NewTable 创建空表。defaultName 为回退链末端仍未命中绑定时使用的策略名。
func NewTable(defaultName string) *Table {
	if defaultName == "" {
		defaultName = "default"
	}
	return &Table{
		specs:    make(map[string]Spec),
		byPat:    make(map[string]string),
		defaultN: defaultName,
	}
}

// Default 返回默认策略名。
func (t *Table) Default() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.defaultN
}

// SetDefault 设置默认策略名。
func (t *Table) SetDefault(name string) {
	if name == "" {
		return
	}
	t.mu.Lock()
	t.defaultN = name
	t.mu.Unlock()
}

// Register 注册策略。同名拒绝。
func (t *Table) Register(s Spec) error {
	if err := s.Validate(); err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.specs[s.Name]; ok {
		return fmt.Errorf("policy: duplicate %q", s.Name)
	}
	t.specs[s.Name] = s.Clone()
	return nil
}

// Replace 覆盖同名策略。
func (t *Table) Replace(s Spec) error {
	if err := s.Validate(); err != nil {
		return err
	}
	t.mu.Lock()
	t.specs[s.Name] = s.Clone()
	t.mu.Unlock()
	return nil
}

// Get 按名取策略。
func (t *Table) Get(name string) (Spec, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	s, ok := t.specs[name]
	if !ok {
		return Spec{}, false
	}
	return s.Clone(), true
}

// Bind 登记模式。pattern 建议为 tenant|route|ip 三段，允许 *。
func (t *Table) Bind(pattern, policyName string) error {
	if pattern == "" || policyName == "" {
		return fmt.Errorf("policy: empty bind")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.specs[policyName]; !ok {
		return fmt.Errorf("policy: bind unknown %q", policyName)
	}
	t.byPat[pattern] = policyName
	// 保持 binds 切片用于列举；覆盖同 pattern
	found := false
	for i := range t.binds {
		if t.binds[i].Pattern == pattern {
			t.binds[i].Policy = policyName
			found = true
			break
		}
	}
	if !found {
		t.binds = append(t.binds, Binding{Pattern: pattern, Policy: policyName})
	}
	return nil
}

// Resolve 按回退链解析。返回命中的策略与模式。
func (t *Table) Resolve(k composite.Key) (spec Spec, pattern string, ok bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, pat := range composite.FallbackChain(k) {
		if name, hit := t.byPat[pat]; hit {
			if s, exists := t.specs[name]; exists {
				return s.Clone(), pat, true
			}
		}
	}
	if s, exists := t.specs[t.defaultN]; exists {
		return s.Clone(), "*|*|*", true
	}
	return Spec{}, "", false
}

// Unregister 删除命名策略，保留指向它的绑定（随后 ResolveErr 报错）。
func (t *Table) Unregister(name string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.specs[name]; !ok {
		return false
	}
	delete(t.specs, name)
	return true
}

// ResolveErr 按回退链解析。绑定指向缺失策略时返回错误而非静默跳过。
func (t *Table) ResolveErr(k composite.Key) (spec Spec, pattern string, err error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, pat := range composite.FallbackChain(k) {
		if name, hit := t.byPat[pat]; hit {
			if s, exists := t.specs[name]; exists {
				return s.Clone(), pat, nil
			}
			return Spec{}, pat, fmt.Errorf("policy: dangling bind %q -> %q", pat, name)
		}
	}
	if s, exists := t.specs[t.defaultN]; exists {
		return s.Clone(), "*|*|*", nil
	}
	return Spec{}, "", fmt.Errorf("policy: no spec")
}

// ResolveName 只返回策略名。
func (t *Table) ResolveName(k composite.Key) (string, bool) {
	s, _, ok := t.Resolve(k)
	if !ok {
		return "", false
	}
	return s.Name, true
}

// Names 返回已注册策略名。
func (t *Table) Names() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]string, 0, len(t.specs))
	for n := range t.specs {
		out = append(out, n)
	}
	return out
}

// Bindings 返回绑定快照。
func (t *Table) Bindings() []Binding {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]Binding, len(t.binds))
	copy(out, t.binds)
	return out
}

// LoadText 解析并注册 DSL。
func (t *Table) LoadText(src string) error {
	specs, binds, err := ParseString(src)
	if err != nil {
		return err
	}
	for _, s := range specs {
		if err := t.Register(s); err != nil {
			return err
		}
	}
	for _, b := range binds {
		if err := t.Bind(b.Pattern, b.Policy); err != nil {
			return err
		}
	}
	return nil
}
