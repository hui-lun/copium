package screener

import (
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Hit is one security that matched a strategy on a given day.
type Hit struct {
	Code    string   `json:"code"`
	Name    string   `json:"name"`
	Close   float64  `json:"close"`
	Volume  int64    `json:"volume"`
	Matched []string `json:"matched"` // labels of the conditions that were true
}

// Scan applies the strategy to every security that traded on date, using only
// data available up to that day. Results are sorted by code.
func Scan(s *Strategy, ds *Dataset, date time.Time) []Hit {
	codes := ds.Codes()
	results := make([]*Hit, len(codes))

	var next atomic.Int64
	var wg sync.WaitGroup
	for w := 0; w < runtime.NumCPU(); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := int(next.Add(1) - 1)
				if i >= len(codes) {
					return
				}
				results[i] = s.evalCode(ds, codes[i], date)
			}
		}()
	}
	wg.Wait()

	hits := make([]Hit, 0)
	for _, h := range results {
		if h != nil {
			hits = append(hits, *h)
		}
	}
	return hits
}

func (s *Strategy) evalCode(ds *Dataset, code string, date time.Time) *Hit {
	idx := ds.BarIndex(code, date)
	if idx < 0 {
		return nil
	}
	meta, ok := ds.Stocks[code]
	if !ok {
		meta = StockMeta{Code: code, Name: code}
	}
	if meta.DelistedOn != nil && !date.Before(*meta.DelistedOn) {
		return nil
	}
	// Point-in-time view: bars after `date` are cut off here.
	c := &evalCtx{ds: ds, code: code, bars: ds.Bars[code][:idx+1], asOf: date}
	if c.last().Close <= 0 || !s.Universe.accept(meta, c) {
		return nil
	}
	var matched []string
	if !s.Rule.eval(c, &matched) {
		return nil
	}
	return &Hit{Code: code, Name: meta.Name, Close: c.last().Close, Volume: c.last().Volume, Matched: matched}
}

func (u *Universe) accept(m StockMeta, c *evalCtx) bool {
	if len(u.Exchanges) > 0 {
		found := false
		for _, e := range u.Exchanges {
			if strings.EqualFold(e, m.Exchange) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if u.CommonStockOnly && !isCommonStock(c.code) {
		return false
	}
	price := c.last().Close
	if u.MinPrice > 0 && price < u.MinPrice {
		return false
	}
	if u.MaxPrice > 0 && price > u.MaxPrice {
		return false
	}
	if u.MinAvgVolume20 > 0 {
		n := min(20, len(c.bars))
		if avgVolume(c.bars[len(c.bars)-n:]) < float64(u.MinAvgVolume20) {
			return false
		}
	}
	return true
}

// isCommonStock reports whether code looks like an ordinary share (4 digits, not 0xxx ETFs).
func isCommonStock(code string) bool {
	if len(code) != 4 || code[0] == '0' {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// eval evaluates the rule tree. Every branch is evaluated (no short-circuit)
// so users can see all the conditions a stock satisfied.
func (r *Rule) eval(c *evalCtx, matched *[]string) bool {
	if r.isGroup() {
		isAnd := strings.EqualFold(r.Op, "and")
		result := isAnd
		for i := range r.Rules {
			ok := r.Rules[i].eval(c, matched)
			if isAnd {
				result = result && ok
			} else {
				result = result || ok
			}
		}
		return result
	}
	l, ok1 := r.Left.resolve(c)
	rv, ok2 := r.Right.resolve(c)
	if !ok1 || !ok2 {
		return false // missing data never matches
	}
	var ok bool
	switch r.Cmp {
	case ">":
		ok = l > rv
	case ">=":
		ok = l >= rv
	case "<":
		ok = l < rv
	case "<=":
		ok = l <= rv
	case "==":
		ok = l == rv
	}
	if ok {
		*matched = append(*matched, r.label())
	}
	return ok
}

func (o *Operand) resolve(c *evalCtx) (float64, bool) {
	if o.Value != nil {
		return *o.Value, true
	}
	v, ok := indicators[o.Indicator].fn(c, o.Period)
	if o.Mul != 0 {
		v *= o.Mul
	}
	return v, ok
}
