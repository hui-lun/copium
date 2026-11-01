// Package market fetches raw market data from public Taiwan exchange sources
// (TWSE / TPEx) and defines the shared market data types.
package market

import "time"

// Exchange identifies where a security is listed.
type Exchange string

const (
	TWSE Exchange = "TWSE" // Taiwan Stock Exchange (listed, 上市)
	TPEx Exchange = "TPEx" // Taipei Exchange (OTC, 上櫃)
)

// Quote is one security's daily trading summary.
type Quote struct {
	Code     string    `json:"code"`
	Name     string    `json:"name"`
	Exchange Exchange  `json:"exchange"`
	Date     time.Time `json:"date"`
	Open     float64   `json:"open"`
	High     float64   `json:"high"`
	Low      float64   `json:"low"`
	Close    float64   `json:"close"`
	Change   float64   `json:"change"`
	Volume   int64     `json:"volume"` // shares
	Value    int64     `json:"value"`  // turnover in NT$
}

// Bar is one daily OHLCV candle. Dates are midnight UTC of the trading day.
type Bar struct {
	Date   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64 // shares
	Value  int64 // NT$
}

// Valuation holds daily valuation ratios.
type Valuation struct {
	Code          string
	Date          time.Time
	PE            float64
	DividendYield float64
	PB            float64
}

// Revenue is one month of reported revenue. AnnouncedOn is the date the figure
// became public; backtests must only use it after that date.
type Revenue struct {
	Code        string
	Month       time.Time // first day of the reported month
	AnnouncedOn time.Time
	Revenue     int64 // NT$ thousand
	YoY         float64
	MoM         float64
}

// Institutional is the daily net buy/sell (in shares) of the three major
// institutional investor groups.
type Institutional struct {
	Code       string
	Date       time.Time
	ForeignNet int64
	TrustNet   int64
	DealerNet  int64
	TotalNet   int64
}

// CorporateAction is an ex-rights / ex-dividend event used for price adjustment.
type CorporateAction struct {
	Code         string
	ExDate       time.Time
	CashDividend float64 // NT$ per share
	StockRatio   float64 // new shares per share, e.g. 0.1 for a 10% stock dividend
}
