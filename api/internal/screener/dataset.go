package screener

import (
	"sort"
	"sync"
	"time"

	"copium/api/internal/market"
)

// StockMeta is static information about a security.
type StockMeta struct {
	Code       string
	Name       string
	Exchange   string
	Industry   string
	Capital    int64      // paid-in capital, NT$
	DelistedOn *time.Time // nil while still listed
}

// RevenuePoint is a monthly revenue figure keyed by its announcement date.
type RevenuePoint struct {
	AnnouncedOn time.Time
	YoY         float64
	MoM         float64
}

// DailyValue is a single numeric value per trading day.
type DailyValue struct {
	Date  time.Time
	Value float64
}

// ValuationPoint holds daily valuation ratios.
type ValuationPoint struct {
	Date  time.Time
	PE    float64
	PB    float64
	Yield float64
}

// Dataset is an in-memory, read-only snapshot of everything the screener and
// backtester need. All per-code slices are sorted by date ascending.
type Dataset struct {
	Stocks    map[string]StockMeta
	Bars      map[string][]market.Bar
	Revenue   map[string][]RevenuePoint
	InstNet   map[string][]DailyValue // total institutional net shares
	Valuation map[string][]ValuationPoint

	codesOnce sync.Once
	codes     []string
}

func NewDataset() *Dataset {
	return &Dataset{
		Stocks:    map[string]StockMeta{},
		Bars:      map[string][]market.Bar{},
		Revenue:   map[string][]RevenuePoint{},
		InstNet:   map[string][]DailyValue{},
		Valuation: map[string][]ValuationPoint{},
	}
}

// Codes returns all codes that have bars, sorted. Do not mutate the dataset after calling it.
func (d *Dataset) Codes() []string {
	d.codesOnce.Do(func() {
		d.codes = make([]string, 0, len(d.Bars))
		for c := range d.Bars {
			d.codes = append(d.codes, c)
		}
		sort.Strings(d.codes)
	})
	return d.codes
}

// BarIndex returns the index of code's bar on date, or -1 if it did not trade that day.
func (d *Dataset) BarIndex(code string, date time.Time) int {
	bars := d.Bars[code]
	i := sort.Search(len(bars), func(i int) bool { return !bars[i].Date.Before(date) })
	if i < len(bars) && bars[i].Date.Equal(date) {
		return i
	}
	return -1
}

// TradingDays returns every date in [from, to] on which at least one security traded.
func (d *Dataset) TradingDays(from, to time.Time) []time.Time {
	seen := map[int64]time.Time{}
	for _, bars := range d.Bars {
		lo := sort.Search(len(bars), func(i int) bool { return !bars[i].Date.Before(from) })
		for i := lo; i < len(bars) && !bars[i].Date.After(to); i++ {
			seen[bars[i].Date.Unix()] = bars[i].Date
		}
	}
	days := make([]time.Time, 0, len(seen))
	for _, t := range seen {
		days = append(days, t)
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })
	return days
}

// latestRevenue returns the newest revenue announced strictly before asOf.
// Revenue is often published after the close, so same-day figures are excluded
// to avoid look-ahead bias.
func (d *Dataset) latestRevenue(code string, asOf time.Time) (RevenuePoint, bool) {
	pts := d.Revenue[code]
	i := sort.Search(len(pts), func(i int) bool { return !pts[i].AnnouncedOn.Before(asOf) })
	if i == 0 {
		return RevenuePoint{}, false
	}
	return pts[i-1], true
}

// instUpTo returns institutional data on or before asOf (published the same evening).
func (d *Dataset) instUpTo(code string, asOf time.Time) []DailyValue {
	pts := d.InstNet[code]
	i := sort.Search(len(pts), func(i int) bool { return pts[i].Date.After(asOf) })
	return pts[:i]
}

func (d *Dataset) latestValuation(code string, asOf time.Time) (ValuationPoint, bool) {
	pts := d.Valuation[code]
	i := sort.Search(len(pts), func(i int) bool { return pts[i].Date.After(asOf) })
	if i == 0 {
		return ValuationPoint{}, false
	}
	return pts[i-1], true
}
