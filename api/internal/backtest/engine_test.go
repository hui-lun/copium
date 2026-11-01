package backtest

import (
	"context"
	"math"
	"testing"
	"time"

	"copium/api/internal/market"
	"copium/api/internal/screener"
)

var day0 = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

func d(i int) time.Time { return day0.AddDate(0, 0, i) }

const strategyJSON = `{"rule": {"op": "and", "rules": [
  {"left": {"indicator": "close"}, "cmp": ">", "right": {"indicator": "highest_close", "period": 20}},
  {"left": {"indicator": "volume_ratio", "period": 20}, "cmp": ">=", "right": {"value": 2}}
]}}`

// dataset: 30 flat days at 10, a breakout on day 30, then custom bars.
func dataset(after ...market.Bar) *screener.Dataset {
	ds := screener.NewDataset()
	var bars []market.Bar
	for i := 0; i < 30; i++ {
		bars = append(bars, market.Bar{Date: d(i), Open: 10, High: 10, Low: 10, Close: 10, Volume: 1000})
	}
	bars = append(bars, market.Bar{Date: d(30), Open: 10, High: 11, Low: 10, Close: 11, Volume: 5000})
	for i, b := range after {
		b.Date = d(31 + i)
		bars = append(bars, b)
	}
	ds.Bars["1234"] = bars
	ds.Stocks["1234"] = screener.StockMeta{Code: "1234", Name: "Test"}
	return ds
}

func cfg() Config {
	return Config{
		Start: "2025-01-01", End: "2025-03-31",
		InitialCapital: 1_000_000, PositionSize: 100_000, MaxPositions: 5,
		TakeProfitPct: 10,
	}
}

func TestEntersNextOpenAndTakesProfit(t *testing.T) {
	strat, err := screener.Parse([]byte(strategyJSON))
	if err != nil {
		t.Fatal(err)
	}
	ds := dataset(
		market.Bar{Open: 11.2, High: 11.5, Low: 11.0, Close: 11.3, Volume: 3000}, // day 31: entry at open 11.2
		market.Bar{Open: 11.4, High: 13.0, Low: 11.3, Close: 12.8, Volume: 3000}, // day 32: +10% target hit
	)
	res, err := Run(context.Background(), strat, ds, cfg(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Trades) == 0 {
		t.Fatal("expected a trade")
	}
	tr := res.Trades[0]
	if tr.EntryDate != "2025-02-01" || tr.EntryPrice != 11.2 {
		t.Fatalf("expected entry at day-31 open 11.2, got %s @ %v", tr.EntryDate, tr.EntryPrice)
	}
	if tr.ExitReason != ExitTakeProfit || math.Abs(tr.ExitPrice-11.2*1.1) > 1e-9 {
		t.Fatalf("expected take profit at 12.32, got %s @ %v", tr.ExitReason, tr.ExitPrice)
	}
	if tr.Shares != 8000 { // floor(100000 / (11.2*1000*1.001425)) = 8 lots
		t.Fatalf("expected 8000 shares, got %d", tr.Shares)
	}
	gross := 8000 * (11.2*1.1 - 11.2)
	if tr.PnL >= gross || tr.PnL <= 0 {
		t.Fatalf("PnL %v should be positive but below gross %v after costs", tr.PnL, gross)
	}
}

func TestNoEntryWhenLimitUpLocked(t *testing.T) {
	strat, _ := screener.Parse([]byte(strategyJSON))
	ds := dataset(
		market.Bar{Open: 12.1, High: 12.1, Low: 12.1, Close: 12.1, Volume: 200}, // day 31: locked limit-up
	)
	res, err := Run(context.Background(), strat, ds, cfg(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tr := range res.Trades {
		if tr.EntryDate == "2025-02-01" {
			t.Fatal("must not buy on a locked limit-up day")
		}
	}
}

func TestStopLossBeforeTakeProfit(t *testing.T) {
	strat, _ := screener.Parse([]byte(strategyJSON))
	c := cfg()
	c.StopLossPct = 5
	ds := dataset(
		market.Bar{Open: 11, High: 11.2, Low: 10.9, Close: 11, Volume: 3000},   // entry at 11
		market.Bar{Open: 11, High: 12.5, Low: 10.0, Close: 11.5, Volume: 3000}, // both stop and target touched
	)
	res, err := Run(context.Background(), strat, ds, c, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Trades) == 0 || res.Trades[0].ExitReason != ExitStopLoss {
		t.Fatalf("expected conservative stop-loss exit, got %+v", res.Trades)
	}
}
