package screener

import (
	"sort"
	"time"

	"copium/api/internal/market"
)

// evalCtx is the view of one security on one day. bars ends at asOf, so an
// indicator cannot see the future even by mistake.
type evalCtx struct {
	ds   *Dataset
	code string
	bars []market.Bar
	asOf time.Time
}

func (c *evalCtx) last() market.Bar { return c.bars[len(c.bars)-1] }

// prev returns the n bars before today (excluding today), or false if there is not enough history.
func (c *evalCtx) prev(n int) ([]market.Bar, bool) {
	if len(c.bars) < n+1 {
		return nil, false
	}
	return c.bars[len(c.bars)-1-n : len(c.bars)-1], true
}

type indicatorFunc func(c *evalCtx, period int) (float64, bool)

type indicatorSpec struct {
	fn          indicatorFunc
	needsPeriod bool
	desc        string
}

// IndicatorInfo describes an indicator for clients building strategy forms.
type IndicatorInfo struct {
	Name        string `json:"name"`
	NeedsPeriod bool   `json:"needs_period"`
	Description string `json:"description"`
}

// Indicators lists every supported indicator, sorted by name.
func Indicators() []IndicatorInfo {
	out := make([]IndicatorInfo, 0, len(indicators))
	for name, s := range indicators {
		out = append(out, IndicatorInfo{Name: name, NeedsPeriod: s.needsPeriod, Description: s.desc})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

var indicators = map[string]indicatorSpec{
	"close":  {desc: "Today's close", fn: func(c *evalCtx, _ int) (float64, bool) { return c.last().Close, true }},
	"open":   {desc: "Today's open", fn: func(c *evalCtx, _ int) (float64, bool) { return c.last().Open, true }},
	"high":   {desc: "Today's high", fn: func(c *evalCtx, _ int) (float64, bool) { return c.last().High, true }},
	"low":    {desc: "Today's low", fn: func(c *evalCtx, _ int) (float64, bool) { return c.last().Low, true }},
	"volume": {desc: "Today's volume (shares)", fn: func(c *evalCtx, _ int) (float64, bool) { return float64(c.last().Volume), true }},
	"value":  {desc: "Today's turnover (NT$)", fn: func(c *evalCtx, _ int) (float64, bool) { return float64(c.last().Value), true }},

	"change_pct": {desc: "1-day % change", fn: func(c *evalCtx, _ int) (float64, bool) {
		p, ok := c.prev(1)
		if !ok || p[0].Close <= 0 {
			return 0, false
		}
		return (c.last().Close/p[0].Close - 1) * 100, true
	}},
	"return_pct": {needsPeriod: true, desc: "N-day % change", fn: func(c *evalCtx, n int) (float64, bool) {
		p, ok := c.prev(n)
		if !ok || p[0].Close <= 0 {
			return 0, false
		}
		return (c.last().Close/p[0].Close - 1) * 100, true
	}},
	"sma": {needsPeriod: true, desc: "N-day simple moving average of close (incl. today)", fn: func(c *evalCtx, n int) (float64, bool) {
		if len(c.bars) < n {
			return 0, false
		}
		sum := 0.0
		for _, b := range c.bars[len(c.bars)-n:] {
			sum += b.Close
		}
		return sum / float64(n), true
	}},
	"avg_volume": {needsPeriod: true, desc: "Average volume of the previous N days (excl. today)", fn: func(c *evalCtx, n int) (float64, bool) {
		p, ok := c.prev(n)
		if !ok {
			return 0, false
		}
		return avgVolume(p), true
	}},
	"volume_ratio": {needsPeriod: true, desc: "Today's volume / average volume of the previous N days", fn: func(c *evalCtx, n int) (float64, bool) {
		p, ok := c.prev(n)
		if !ok {
			return 0, false
		}
		avg := avgVolume(p)
		if avg <= 0 {
			return 0, false
		}
		return float64(c.last().Volume) / avg, true
	}},
	"highest_close": {needsPeriod: true, desc: "Highest close of the previous N days (close > this = N-day high)", fn: func(c *evalCtx, n int) (float64, bool) {
		p, ok := c.prev(n)
		if !ok {
			return 0, false
		}
		m := p[0].Close
		for _, b := range p {
			m = max(m, b.Close)
		}
		return m, true
	}},
	"lowest_close": {needsPeriod: true, desc: "Lowest close of the previous N days", fn: func(c *evalCtx, n int) (float64, bool) {
		p, ok := c.prev(n)
		if !ok {
			return 0, false
		}
		m := p[0].Close
		for _, b := range p {
			m = min(m, b.Close)
		}
		return m, true
	}},
	"highest_high": {needsPeriod: true, desc: "Highest high of the previous N days", fn: func(c *evalCtx, n int) (float64, bool) {
		p, ok := c.prev(n)
		if !ok {
			return 0, false
		}
		m := p[0].High
		for _, b := range p {
			m = max(m, b.High)
		}
		return m, true
	}},

	"revenue_yoy": {desc: "Latest announced monthly revenue YoY %", fn: func(c *evalCtx, _ int) (float64, bool) {
		r, ok := c.ds.latestRevenue(c.code, c.asOf)
		return r.YoY, ok
	}},
	"revenue_mom": {desc: "Latest announced monthly revenue MoM %", fn: func(c *evalCtx, _ int) (float64, bool) {
		r, ok := c.ds.latestRevenue(c.code, c.asOf)
		return r.MoM, ok
	}},
	"inst_net": {desc: "Latest institutional total net buy (shares)", fn: func(c *evalCtx, _ int) (float64, bool) {
		pts := c.ds.instUpTo(c.code, c.asOf)
		if len(pts) == 0 {
			return 0, false
		}
		return pts[len(pts)-1].Value, true
	}},
	"inst_buy_streak": {desc: "Consecutive days of institutional net buying", fn: func(c *evalCtx, _ int) (float64, bool) {
		pts := c.ds.instUpTo(c.code, c.asOf)
		if len(pts) == 0 {
			return 0, false
		}
		n := 0
		for i := len(pts) - 1; i >= 0 && pts[i].Value > 0; i-- {
			n++
		}
		return float64(n), true
	}},
	"pe": {desc: "Price / earnings ratio", fn: func(c *evalCtx, _ int) (float64, bool) {
		v, ok := c.ds.latestValuation(c.code, c.asOf)
		return v.PE, ok && v.PE > 0
	}},
	"pb": {desc: "Price / book ratio", fn: func(c *evalCtx, _ int) (float64, bool) {
		v, ok := c.ds.latestValuation(c.code, c.asOf)
		return v.PB, ok && v.PB > 0
	}},
	"dividend_yield": {desc: "Dividend yield %", fn: func(c *evalCtx, _ int) (float64, bool) {
		v, ok := c.ds.latestValuation(c.code, c.asOf)
		return v.Yield, ok
	}},
	"capital": {desc: "Paid-in capital (NT$)", fn: func(c *evalCtx, _ int) (float64, bool) {
		m := c.ds.Stocks[c.code]
		return float64(m.Capital), m.Capital > 0
	}},
}

func avgVolume(bars []market.Bar) float64 {
	if len(bars) == 0 {
		return 0
	}
	var sum int64
	for _, b := range bars {
		sum += b.Volume
	}
	return float64(sum) / float64(len(bars))
}
