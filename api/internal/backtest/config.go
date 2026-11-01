// Package backtest simulates trading a screener strategy day by day over
// historical data, with realistic entry timing, costs and liquidity limits.
package backtest

import (
	"errors"
	"time"
)

const dateLayout = "2006-01-02"

// Config controls a backtest run. Percentages are expressed as e.g. 10 for 10%.
type Config struct {
	Start          string  `json:"start"` // YYYY-MM-DD
	End            string  `json:"end"`
	InitialCapital float64 `json:"initial_capital"`
	PositionSize   float64 `json:"position_size"` // NT$ per new position; 0 = equity / max_positions
	MaxPositions   int     `json:"max_positions"`

	// Exit rules; 0 disables a rule.
	TakeProfitPct    float64 `json:"take_profit_pct"`
	StopLossPct      float64 `json:"stop_loss_pct"`
	TrailingStopPct  float64 `json:"trailing_stop_pct"`
	MaxHoldDays      int     `json:"max_hold_days"`
	ExitOnSignalLost bool    `json:"exit_on_signal_lost"`

	// Costs (Taiwan defaults).
	FeeRate     float64 `json:"fee_rate"`     // brokerage fee, 0.001425
	FeeDiscount float64 `json:"fee_discount"` // e.g. 0.6 for a 40% discount
	MinFee      float64 `json:"min_fee"`      // NT$ per order
	TaxRate     float64 `json:"tax_rate"`     // securities transaction tax on sells, 0.003

	// Liquidity.
	LotSize   int64 `json:"lot_size"`   // shares per board lot
	MinVolume int64 `json:"min_volume"` // skip signals whose signal-day volume (shares) is below this

	Benchmark string `json:"benchmark"` // buy-and-hold comparison, default 0050
}

func (c *Config) applyDefaults() {
	if c.InitialCapital == 0 {
		c.InitialCapital = 1_000_000
	}
	if c.MaxPositions == 0 {
		c.MaxPositions = 10
	}
	if c.FeeRate == 0 {
		c.FeeRate = 0.001425
	}
	if c.FeeDiscount == 0 {
		c.FeeDiscount = 1
	}
	if c.MinFee == 0 {
		c.MinFee = 20
	}
	if c.TaxRate == 0 {
		c.TaxRate = 0.003
	}
	if c.LotSize == 0 {
		c.LotSize = 1000
	}
	if c.Benchmark == "" {
		c.Benchmark = "0050"
	}
}

// Range parses Start and End.
func (c Config) Range() (time.Time, time.Time, error) {
	from, err := time.Parse(dateLayout, c.Start)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("start must be YYYY-MM-DD")
	}
	to, err := time.Parse(dateLayout, c.End)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("end must be YYYY-MM-DD")
	}
	if !from.Before(to) {
		return time.Time{}, time.Time{}, errors.New("start must be before end")
	}
	return from, to, nil
}

// Validate checks the config after defaults are applied.
func (c Config) Validate() error {
	if _, _, err := c.Range(); err != nil {
		return err
	}
	switch {
	case c.InitialCapital <= 0:
		return errors.New("initial_capital must be positive")
	case c.MaxPositions < 1 || c.MaxPositions > 100:
		return errors.New("max_positions must be between 1 and 100")
	case c.PositionSize < 0, c.TakeProfitPct < 0, c.TrailingStopPct < 0, c.MaxHoldDays < 0:
		return errors.New("sizes and exit parameters cannot be negative")
	case c.StopLossPct < 0 || c.StopLossPct >= 100:
		return errors.New("stop_loss_pct must be between 0 and 100")
	case c.FeeDiscount <= 0 || c.FeeDiscount > 1:
		return errors.New("fee_discount must be in (0, 1]")
	}
	return nil
}
