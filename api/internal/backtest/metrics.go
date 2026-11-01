package backtest

import (
	"math"
	"time"
)

// Metrics summarizes a backtest.
type Metrics struct {
	FinalEquity        float64 `json:"final_equity"`
	TotalReturnPct     float64 `json:"total_return_pct"`
	CAGRPct            float64 `json:"cagr_pct"`
	MaxDrawdownPct     float64 `json:"max_drawdown_pct"` // negative number
	BenchmarkReturnPct float64 `json:"benchmark_return_pct"`
	Trades             int     `json:"trades"`
	Stocks             int     `json:"stocks"` // distinct securities traded
	WinRatePct         float64 `json:"win_rate_pct"`
	AvgWinPct          float64 `json:"avg_win_pct"`
	AvgLossPct         float64 `json:"avg_loss_pct"`
	PayoffRatio        float64 `json:"payoff_ratio"` // avg win / |avg loss|
	AvgHoldDays        float64 `json:"avg_hold_days"`
}

// Bucket counts trades within a return range. Hidden-gem strategies depend on
// a few big winners, so the distribution says more than the average.
type Bucket struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

func computeMetrics(cfg Config, trades []Trade, equity []EquityPoint) Metrics {
	var m Metrics
	if len(equity) == 0 {
		return m
	}
	first, last := equity[0], equity[len(equity)-1]
	m.FinalEquity = last.Equity
	m.TotalReturnPct = (last.Equity/cfg.InitialCapital - 1) * 100
	if last.Benchmark > 0 {
		m.BenchmarkReturnPct = (last.Benchmark/cfg.InitialCapital - 1) * 100
	}

	t0, _ := time.Parse(dateLayout, first.Date)
	t1, _ := time.Parse(dateLayout, last.Date)
	if years := t1.Sub(t0).Hours() / 24 / 365.25; years > 0 && last.Equity > 0 {
		m.CAGRPct = (math.Pow(last.Equity/cfg.InitialCapital, 1/years) - 1) * 100
	}

	peak := equity[0].Equity
	for _, e := range equity {
		peak = math.Max(peak, e.Equity)
		if peak > 0 {
			m.MaxDrawdownPct = math.Min(m.MaxDrawdownPct, (e.Equity/peak-1)*100)
		}
	}

	m.Trades = len(trades)
	if m.Trades == 0 {
		return m
	}
	stocks := map[string]bool{}
	var wins, losses int
	var winSum, lossSum float64
	var holdSum int
	for _, t := range trades {
		stocks[t.Code] = true
		holdSum += t.HoldDays
		if t.PnL > 0 {
			wins++
			winSum += t.ReturnPct
		} else {
			losses++
			lossSum += t.ReturnPct
		}
	}
	m.Stocks = len(stocks)
	m.WinRatePct = float64(wins) / float64(m.Trades) * 100
	m.AvgHoldDays = float64(holdSum) / float64(m.Trades)
	if wins > 0 {
		m.AvgWinPct = winSum / float64(wins)
	}
	if losses > 0 {
		m.AvgLossPct = lossSum / float64(losses)
	}
	if m.AvgLossPct < 0 {
		m.PayoffRatio = m.AvgWinPct / -m.AvgLossPct
	}
	return m
}

func distribution(trades []Trade) []Bucket {
	buckets := []Bucket{
		{Label: "≥ +100%"}, {Label: "+50% ~ +100%"}, {Label: "0 ~ +50%"},
		{Label: "-30% ~ 0"}, {Label: "≤ -30%"},
	}
	for _, t := range trades {
		r := t.ReturnPct
		switch {
		case r >= 100:
			buckets[0].Count++
		case r >= 50:
			buckets[1].Count++
		case r > 0:
			buckets[2].Count++
		case r > -30:
			buckets[3].Count++
		default:
			buckets[4].Count++
		}
	}
	return buckets
}
