package backtest

import (
	"context"
	"errors"
	"math"
	"sort"
	"time"

	"copium/api/internal/market"
	"copium/api/internal/screener"
)

// Trade is one completed round trip.
type Trade struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	EntryDate  string  `json:"entry_date"`
	ExitDate   string  `json:"exit_date"`
	EntryPrice float64 `json:"entry_price"`
	ExitPrice  float64 `json:"exit_price"`
	Shares     int64   `json:"shares"`
	PnL        float64 `json:"pnl"`        // after all costs, NT$
	ReturnPct  float64 `json:"return_pct"` // PnL / total entry cost
	HoldDays   int     `json:"hold_days"`  // trading days
	ExitReason string  `json:"exit_reason"`
}

// EquityPoint is the portfolio value at a day's close.
type EquityPoint struct {
	Date      string  `json:"date"`
	Equity    float64 `json:"equity"`
	Benchmark float64 `json:"benchmark"` // buy-and-hold benchmark scaled to the initial capital
}

// Result is the full output of a backtest.
type Result struct {
	Metrics      Metrics       `json:"metrics"`
	Distribution []Bucket      `json:"distribution"`
	Trades       []Trade       `json:"trades"`
	Equity       []EquityPoint `json:"equity"`
}

// Exit reasons.
const (
	ExitStopLoss     = "stop_loss"
	ExitTrailing     = "trailing_stop"
	ExitTakeProfit   = "take_profit"
	ExitMaxHold      = "max_hold"
	ExitSignalLost   = "signal_lost"
	ExitDelisted     = "delisted"
	ExitOpenAtEnd    = "open_at_end"
	limitUpThreshold = 1.095 // Taiwan daily limit is ±10%; allow for tick rounding
	limitDnThreshold = 0.905
)

type position struct {
	code, name string
	shares     int64
	entryPrice float64
	cost       float64 // total cash paid including fees
	entryDate  time.Time
	entryDay   int
	peak       float64 // highest close since entry, for the trailing stop
	lastClose  float64
	exitReason string // set when an exit is scheduled for the next open
}

type simulator struct {
	cfg       Config
	strat     *screener.Strategy
	ds        *screener.Dataset
	cash      float64
	positions map[string]*position
	pending   []screener.Hit // yesterday's signals, entered at today's open
	trades    []Trade
	equity    []EquityPoint
	benchBase float64
	benchLast float64
}

// Run backtests strat over ds between cfg.Start and cfg.End.
//
// Timeline for each trading day:
//  1. Enter yesterday's signals at today's OPEN (never at the signal day's
//     close, which would be look-ahead).
//  2. Check exits for open positions using today's bar.
//  3. Run the screener at today's close to produce tomorrow's signals.
//  4. Mark the portfolio to market.
func Run(ctx context.Context, strat *screener.Strategy, ds *screener.Dataset, cfg Config, progress func(float64)) (*Result, error) {
	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	from, to, _ := cfg.Range()
	days := ds.TradingDays(from, to)
	if len(days) == 0 {
		return nil, errors.New("no trading data in the selected period")
	}

	s := &simulator{cfg: cfg, strat: strat, ds: ds, cash: cfg.InitialCapital, positions: map[string]*position{}}
	for di, day := range days {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		s.enter(di, day)
		s.exit(di, day)
		s.signal(di, day, di == len(days)-1)
		s.mark(day)
		if progress != nil {
			progress(float64(di+1) / float64(len(days)))
		}
	}
	last := len(days) - 1
	for _, code := range s.sortedCodes() {
		p := s.positions[code]
		s.closePosition(p, days[last], last, p.lastClose, ExitOpenAtEnd)
	}
	return &Result{
		Metrics:      computeMetrics(cfg, s.trades, s.equity),
		Distribution: distribution(s.trades),
		Trades:       s.trades,
		Equity:       s.equity,
	}, nil
}

func (s *simulator) enter(di int, day time.Time) {
	signals := s.pending
	s.pending = nil
	for _, h := range signals {
		if len(s.positions) >= s.cfg.MaxPositions {
			return
		}
		if _, held := s.positions[h.Code]; held {
			continue
		}
		bars := s.ds.Bars[h.Code]
		idx := s.ds.BarIndex(h.Code, day)
		if idx <= 0 {
			continue // not trading today: the signal expires
		}
		bar, prevClose := bars[idx], bars[idx-1].Close
		if isLimitUpLocked(bar, prevClose) {
			continue // locked limit-up: no sellers, cannot buy
		}
		price := bar.Open
		if price <= 0 {
			continue
		}
		budget := s.cfg.PositionSize
		if budget <= 0 {
			budget = s.equityValue() / float64(s.cfg.MaxPositions)
		}
		budget = math.Min(budget, s.cash)
		lotCost := price * float64(s.cfg.LotSize) * (1 + s.cfg.FeeRate*s.cfg.FeeDiscount)
		lots := int64(budget / lotCost)
		if lots < 1 {
			continue
		}
		shares := lots * s.cfg.LotSize
		gross := float64(shares) * price
		total := gross + s.fee(gross)
		if total > s.cash {
			continue
		}
		s.cash -= total
		s.positions[h.Code] = &position{
			code: h.Code, name: h.Name, shares: shares, entryPrice: price, cost: total,
			entryDate: day, entryDay: di, peak: price, lastClose: bar.Close,
		}
	}
}

func (s *simulator) exit(di int, day time.Time) {
	for _, code := range s.sortedCodes() {
		p := s.positions[code]
		bars := s.ds.Bars[code]
		idx := s.ds.BarIndex(code, day)
		if idx < 0 {
			// Not trading today. A delisted stock is closed at its last price so
			// that blow-ups count against the strategy (survivorship bias).
			if m, ok := s.ds.Stocks[code]; ok && m.DelistedOn != nil && !day.Before(*m.DelistedOn) {
				s.closePosition(p, day, di, p.lastClose, ExitDelisted)
			}
			continue
		}
		bar := bars[idx]
		if p.entryDay == di {
			p.peak = math.Max(p.peak, bar.Close) // no same-day exits
			p.lastClose = bar.Close
			continue
		}
		prevClose := 0.0
		if idx > 0 {
			prevClose = bars[idx-1].Close
		}
		if isLimitDownLocked(bar, prevClose) {
			p.lastClose = bar.Close // locked limit-down: no buyers, cannot sell
			continue
		}

		price, reason := bar.Open, p.exitReason
		if reason == "" {
			price, reason = s.checkExit(p, bar, di)
		}
		if reason != "" {
			s.closePosition(p, day, di, price, reason)
			continue
		}
		p.lastClose = bar.Close
		p.peak = math.Max(p.peak, bar.Close)
	}
}

// checkExit applies exit rules to today's bar. When several could trigger on
// the same bar the conservative order is used: stops before take-profit.
func (s *simulator) checkExit(p *position, bar market.Bar, di int) (float64, string) {
	c := s.cfg
	if c.StopLossPct > 0 {
		stop := p.entryPrice * (1 - c.StopLossPct/100)
		if bar.Low <= stop {
			return math.Min(bar.Open, stop), ExitStopLoss // a gap down fills at the open
		}
	}
	if c.TrailingStopPct > 0 {
		stop := p.peak * (1 - c.TrailingStopPct/100)
		if bar.Low <= stop {
			return math.Min(bar.Open, stop), ExitTrailing
		}
	}
	if c.TakeProfitPct > 0 {
		target := p.entryPrice * (1 + c.TakeProfitPct/100)
		if bar.High >= target {
			return math.Max(bar.Open, target), ExitTakeProfit // a gap up fills at the open
		}
	}
	if c.MaxHoldDays > 0 && di-p.entryDay >= c.MaxHoldDays {
		return bar.Close, ExitMaxHold
	}
	return 0, ""
}

func (s *simulator) signal(di int, day time.Time, lastDay bool) {
	hits := screener.Scan(s.strat, s.ds, day)

	if s.cfg.ExitOnSignalLost {
		still := make(map[string]bool, len(hits))
		for _, h := range hits {
			still[h.Code] = true
		}
		for code, p := range s.positions {
			if !still[code] && p.exitReason == "" {
				p.exitReason = ExitSignalLost
			}
		}
	}
	if lastDay {
		return
	}
	for _, h := range hits {
		if _, held := s.positions[h.Code]; held {
			continue
		}
		if s.cfg.MinVolume > 0 && h.Volume < s.cfg.MinVolume {
			continue
		}
		s.pending = append(s.pending, h)
	}
}

func (s *simulator) mark(day time.Time) {
	if idx := s.ds.BarIndex(s.cfg.Benchmark, day); idx >= 0 {
		c := s.ds.Bars[s.cfg.Benchmark][idx].Close
		if s.benchBase == 0 && c > 0 {
			s.benchBase = c
		}
		if s.benchBase > 0 {
			s.benchLast = s.cfg.InitialCapital * c / s.benchBase
		}
	}
	s.equity = append(s.equity, EquityPoint{Date: day.Format(dateLayout), Equity: s.equityValue(), Benchmark: s.benchLast})
}

func (s *simulator) closePosition(p *position, day time.Time, di int, price float64, reason string) {
	gross := float64(p.shares) * price
	net := gross - s.fee(gross) - gross*s.cfg.TaxRate
	s.cash += net
	pnl := net - p.cost
	s.trades = append(s.trades, Trade{
		Code: p.code, Name: p.name,
		EntryDate: p.entryDate.Format(dateLayout), ExitDate: day.Format(dateLayout),
		EntryPrice: p.entryPrice, ExitPrice: price, Shares: p.shares,
		PnL: pnl, ReturnPct: pnl / p.cost * 100, HoldDays: di - p.entryDay, ExitReason: reason,
	})
	delete(s.positions, p.code)
}

func (s *simulator) fee(gross float64) float64 {
	return math.Max(s.cfg.MinFee, gross*s.cfg.FeeRate*s.cfg.FeeDiscount)
}

func (s *simulator) equityValue() float64 {
	v := s.cash
	for _, p := range s.positions {
		v += float64(p.shares) * p.lastClose
	}
	return v
}

func (s *simulator) sortedCodes() []string {
	codes := make([]string, 0, len(s.positions))
	for c := range s.positions {
		codes = append(codes, c)
	}
	sort.Strings(codes)
	return codes
}

// A bar is "locked" when it traded at a single price at the daily limit.
func isLimitUpLocked(b market.Bar, prevClose float64) bool {
	return prevClose > 0 && b.Open == b.High && b.High == b.Low && b.Close >= prevClose*limitUpThreshold
}

func isLimitDownLocked(b market.Bar, prevClose float64) bool {
	return prevClose > 0 && b.Open == b.High && b.High == b.Low && b.Close <= prevClose*limitDnThreshold
}
