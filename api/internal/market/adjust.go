package market

import "sort"

// AdjustBars returns a back-adjusted copy of bars so that ex-dividend and
// ex-rights gaps do not look like price drops. Prices before each ex-date are
// multiplied by (prevClose - cash) / prevClose / (1 + stockRatio).
// bars must be sorted by date ascending; the input slice is not modified.
func AdjustBars(bars []Bar, actions []CorporateAction) []Bar {
	out := make([]Bar, len(bars))
	copy(out, bars)
	if len(actions) == 0 || len(bars) == 0 {
		return out
	}
	acts := make([]CorporateAction, len(actions))
	copy(acts, actions)
	sort.Slice(acts, func(i, j int) bool { return acts[i].ExDate.Before(acts[j].ExDate) })

	for _, a := range acts {
		idx := sort.Search(len(bars), func(i int) bool { return !bars[i].Date.Before(a.ExDate) })
		if idx <= 0 || idx >= len(bars) {
			continue
		}
		prev := bars[idx-1].Close // raw close: factors are scale-free, so they compound correctly
		if prev <= 0 {
			continue
		}
		factor := (prev - a.CashDividend) / prev / (1 + a.StockRatio)
		if factor <= 0 || factor >= 1 {
			continue // ignore nonsensical actions
		}
		for j := 0; j < idx; j++ {
			out[j].Open *= factor
			out[j].High *= factor
			out[j].Low *= factor
			out[j].Close *= factor
		}
	}
	return out
}
