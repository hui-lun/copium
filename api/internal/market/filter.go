package market

// IsStockOrETF reports whether code is a tradable security we track, as
// opposed to warrants, which some feeds mix in by the thousands.
//
//   - 4 digits: common stock; 5 characters: preferred share (e.g. "2881A")
//   - 6 characters starting with 00 (ETF), 01 (REIT), 02 (ETN) or 91 (TDR)
//   - 6 characters with a letter in 5th place: preferred share (e.g. "2887Z1")
//
// Other 6-character codes (03xxxx–08xxxx, 7xxxxx, ...) are warrants.
func IsStockOrETF(code string) bool {
	switch len(code) {
	case 4, 5:
		return true
	case 6:
		if c := code[4]; c >= 'A' && c <= 'Z' {
			return true
		}
		switch code[:2] {
		case "00", "01", "02", "91":
			return true
		}
	}
	return false
}

// filterStocks drops warrants and other untracked instruments.
func filterStocks(quotes []Quote) []Quote {
	out := quotes[:0]
	for _, q := range quotes {
		if IsStockOrETF(q.Code) {
			out = append(out, q)
		}
	}
	return out
}
