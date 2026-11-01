// Package screener evaluates configurable stock-picking strategies against
// point-in-time market data. The same engine powers the daily scan and the
// backtester, so a strategy behaves identically in both.
package screener

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Strategy is a JSON-described set of screening conditions.
//
// Example:
//
//	{
//	  "name": "Volume breakout",
//	  "universe": {"common_stock_only": true, "min_price": 10},
//	  "rule": {"op": "and", "rules": [
//	    {"left": {"indicator": "close"}, "cmp": ">", "right": {"indicator": "highest_close", "period": 60}},
//	    {"left": {"indicator": "volume_ratio", "period": 20}, "cmp": ">=", "right": {"value": 2}}
//	  ]}
//	}
type Strategy struct {
	Name     string   `json:"name"`
	Universe Universe `json:"universe"`
	Rule     Rule     `json:"rule"`
}

// Universe pre-filters which securities are considered at all.
type Universe struct {
	Exchanges       []string `json:"exchanges,omitempty"`         // empty = all
	CommonStockOnly bool     `json:"common_stock_only,omitempty"` // 4-digit codes not starting with 0 (skips ETFs etc.)
	MinPrice        float64  `json:"min_price,omitempty"`
	MaxPrice        float64  `json:"max_price,omitempty"`
	MinAvgVolume20  int64    `json:"min_avg_volume_20,omitempty"` // shares, average of the last 20 sessions
}

// Rule is either a group (Op + Rules) or a comparison (Left Cmp Right).
type Rule struct {
	Op    string `json:"op,omitempty"` // "and" | "or"
	Rules []Rule `json:"rules,omitempty"`

	Label string   `json:"label,omitempty"` // shown to users when the condition matches
	Left  *Operand `json:"left,omitempty"`
	Cmp   string   `json:"cmp,omitempty"` // ">", ">=", "<", "<=", "=="
	Right *Operand `json:"right,omitempty"`
}

// Operand is a constant (Value) or an indicator, optionally scaled by Mul.
type Operand struct {
	Indicator string   `json:"indicator,omitempty"`
	Period    int      `json:"period,omitempty"`
	Value     *float64 `json:"value,omitempty"`
	Mul       float64  `json:"mul,omitempty"`
}

const maxPeriod = 500

var validCmp = map[string]bool{">": true, ">=": true, "<": true, "<=": true, "==": true}

// Parse decodes and validates a strategy definition.
func Parse(data []byte) (*Strategy, error) {
	var s Strategy
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("invalid strategy JSON: %w", err)
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *Strategy) Validate() error {
	return s.Rule.validate("rule")
}

func (r *Rule) isGroup() bool { return r.Op != "" || len(r.Rules) > 0 }

func (r *Rule) validate(path string) error {
	if r.isGroup() {
		op := strings.ToLower(r.Op)
		if op != "and" && op != "or" {
			return fmt.Errorf("%s: op must be \"and\" or \"or\"", path)
		}
		if len(r.Rules) == 0 {
			return fmt.Errorf("%s: group has no rules", path)
		}
		if r.Left != nil || r.Right != nil || r.Cmp != "" {
			return fmt.Errorf("%s: a group cannot also be a comparison", path)
		}
		for i := range r.Rules {
			if err := r.Rules[i].validate(fmt.Sprintf("%s.rules[%d]", path, i)); err != nil {
				return err
			}
		}
		return nil
	}
	if r.Left == nil || r.Right == nil {
		return fmt.Errorf("%s: comparison needs left and right", path)
	}
	if !validCmp[r.Cmp] {
		return fmt.Errorf("%s: unknown cmp %q", path, r.Cmp)
	}
	if err := r.Left.validate(path + ".left"); err != nil {
		return err
	}
	return r.Right.validate(path + ".right")
}

func (o *Operand) validate(path string) error {
	if o.Value != nil {
		if o.Indicator != "" {
			return fmt.Errorf("%s: set either value or indicator, not both", path)
		}
		return nil
	}
	spec, ok := indicators[o.Indicator]
	if !ok {
		return fmt.Errorf("%s: unknown indicator %q", path, o.Indicator)
	}
	if spec.needsPeriod && (o.Period <= 0 || o.Period > maxPeriod) {
		return fmt.Errorf("%s: %s needs a period between 1 and %d", path, o.Indicator, maxPeriod)
	}
	return nil
}

// Lookback is the number of trading sessions of history the strategy needs.
func (s *Strategy) Lookback() int {
	n := 21 // universe avg volume uses 20 sessions + today
	var walk func(r *Rule)
	walk = func(r *Rule) {
		for i := range r.Rules {
			walk(&r.Rules[i])
		}
		for _, o := range []*Operand{r.Left, r.Right} {
			if o != nil && o.Period+1 > n {
				n = o.Period + 1
			}
		}
	}
	walk(&s.Rule)
	return n
}

// label returns the user-facing description of a comparison.
func (r *Rule) label() string {
	if r.Label != "" {
		return r.Label
	}
	return r.Left.String() + " " + r.Cmp + " " + r.Right.String()
}

func (o *Operand) String() string {
	if o.Value != nil {
		return strconv.FormatFloat(*o.Value, 'f', -1, 64)
	}
	s := o.Indicator
	if o.Period > 0 {
		s += "(" + strconv.Itoa(o.Period) + ")"
	}
	if o.Mul != 0 && o.Mul != 1 {
		s = strconv.FormatFloat(o.Mul, 'f', -1, 64) + "×" + s
	}
	return s
}
