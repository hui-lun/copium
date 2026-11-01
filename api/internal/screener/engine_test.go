package screener

import (
	"testing"
	"time"

	"copium/api/internal/market"
)

var day0 = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

// flatBars builds n flat bars at price p with volume v.
func flatBars(n int, p float64, v int64) []market.Bar {
	bars := make([]market.Bar, n)
	for i := range bars {
		bars[i] = market.Bar{Date: day0.AddDate(0, 0, i), Open: p, High: p, Low: p, Close: p, Volume: v}
	}
	return bars
}

const breakout = `{
  "name": "breakout",
  "universe": {"common_stock_only": true},
  "rule": {"op": "and", "rules": [
    {"label": "new high", "left": {"indicator": "close"}, "cmp": ">", "right": {"indicator": "highest_close", "period": 20}},
    {"label": "volume x2", "left": {"indicator": "volume_ratio", "period": 20}, "cmp": ">=", "right": {"value": 2}}
  ]}
}`

func TestScanBreakout(t *testing.T) {
	s, err := Parse([]byte(breakout))
	if err != nil {
		t.Fatal(err)
	}
	ds := NewDataset()
	bars := flatBars(30, 10, 1000)
	bars = append(bars, market.Bar{Date: day0.AddDate(0, 0, 30), Open: 10, High: 11, Low: 10, Close: 11, Volume: 5000})
	ds.Bars["1234"] = bars
	ds.Bars["0050"] = append(flatBars(30, 10, 1000), bars[30]) // ETF: excluded by universe
	ds.Stocks["1234"] = StockMeta{Code: "1234", Name: "Test"}

	if hits := Scan(s, ds, day0.AddDate(0, 0, 29)); len(hits) != 0 {
		t.Fatalf("expected no hits before breakout, got %v", hits)
	}
	hits := Scan(s, ds, day0.AddDate(0, 0, 30))
	if len(hits) != 1 || hits[0].Code != "1234" {
		t.Fatalf("expected 1234 to hit, got %v", hits)
	}
	if len(hits[0].Matched) != 2 {
		t.Fatalf("expected 2 matched labels, got %v", hits[0].Matched)
	}
}

func TestScanNotEnoughHistory(t *testing.T) {
	s, _ := Parse([]byte(breakout))
	ds := NewDataset()
	ds.Bars["1234"] = flatBars(5, 10, 1000)
	if hits := Scan(s, ds, day0.AddDate(0, 0, 4)); len(hits) != 0 {
		t.Fatalf("expected no hits with short history, got %v", hits)
	}
}

func TestRevenueIsPointInTime(t *testing.T) {
	s, err := Parse([]byte(`{"rule": {"left": {"indicator": "revenue_yoy"}, "cmp": ">", "right": {"value": 30}}}`))
	if err != nil {
		t.Fatal(err)
	}
	ds := NewDataset()
	ds.Bars["1234"] = flatBars(20, 10, 1000)
	announce := day0.AddDate(0, 0, 10)
	ds.Revenue["1234"] = []RevenuePoint{{AnnouncedOn: announce, YoY: 50}}

	if hits := Scan(s, ds, announce); len(hits) != 0 {
		t.Fatal("revenue must not be visible on its announcement day")
	}
	if hits := Scan(s, ds, announce.AddDate(0, 0, 1)); len(hits) != 1 {
		t.Fatal("revenue should be visible the day after announcement")
	}
}

func TestParseRejectsBadStrategies(t *testing.T) {
	bad := []string{
		`{"rule": {"left": {"indicator": "nope"}, "cmp": ">", "right": {"value": 1}}}`,
		`{"rule": {"left": {"indicator": "sma"}, "cmp": ">", "right": {"value": 1}}}`,
		`{"rule": {"op": "xor", "rules": []}}`,
		`{"rule": {"left": {"indicator": "close"}, "cmp": "!=", "right": {"value": 1}}}`,
		`{"unknown_field": 1}`,
	}
	for _, b := range bad {
		if _, err := Parse([]byte(b)); err == nil {
			t.Errorf("expected error for %s", b)
		}
	}
}
