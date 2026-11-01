package jobs

import (
	"fmt"

	"copium/api/internal/market"
)

// checkQuotes returns data-quality warnings for one ingest batch:
// an empty batch, duplicate rows, or an unusual share of zero prices.
func checkQuotes(quotes []market.Quote) []string {
	if len(quotes) == 0 {
		return []string{"no quotes received"}
	}
	var warnings []string
	seen := map[string]bool{}
	zero, dup := 0, 0
	for _, q := range quotes {
		key := q.Code + q.Date.Format("20060102")
		if seen[key] {
			dup++
		}
		seen[key] = true
		if q.Close <= 0 {
			zero++
		}
	}
	if dup > 0 {
		warnings = append(warnings, fmt.Sprintf("%d duplicate rows", dup))
	}
	// A few untraded securities per day is normal; a large share means bad data.
	if ratio := float64(zero) / float64(len(quotes)); ratio > 0.2 {
		warnings = append(warnings, fmt.Sprintf("%.0f%% of quotes have a zero close", ratio*100))
	}
	return warnings
}
