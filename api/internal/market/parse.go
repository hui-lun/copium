package market

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// parseFloat parses exchange number strings such as "1,234.5", "+1.5",
// "X0.00" or "--". Unparseable values return 0.
func parseFloat(s string) float64 {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	s = strings.TrimPrefix(s, "X")
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

func parseInt(s string) int64 {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}
	return int64(parseFloat(s))
}

// parseDate accepts ROC dates ("1151031", "115/10/31", "115-10-31") and
// Gregorian dates ("20261031", "2026/10/31", "2026-10-31").
// The result is midnight UTC of that calendar day.
func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	var parts []string
	switch {
	case strings.ContainsAny(s, "/-"):
		parts = strings.FieldsFunc(s, func(r rune) bool { return r == '/' || r == '-' })
	case len(s) == 7: // ROC yyyMMdd
		parts = []string{s[:3], s[3:5], s[5:]}
	case len(s) == 8: // yyyyMMdd
		parts = []string{s[:4], s[4:6], s[6:]}
	}
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("bad date %q", s)
	}
	y, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	d, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return time.Time{}, fmt.Errorf("bad date %q", s)
	}
	if y < 1911 {
		y += 1911 // ROC era
	}
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC), nil
}

// parseROCMonth parses "11510" (ROC year + month) into the first day of that month.
func parseROCMonth(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if len(s) < 4 {
		return time.Time{}, fmt.Errorf("bad month %q", s)
	}
	y, err1 := strconv.Atoi(s[:len(s)-2])
	m, err2 := strconv.Atoi(s[len(s)-2:])
	if err1 != nil || err2 != nil {
		return time.Time{}, fmt.Errorf("bad month %q", s)
	}
	return time.Date(y+1911, time.Month(m), 1, 0, 0, 0, 0, time.UTC), nil
}

// Today returns today's calendar date in loc, as midnight UTC.
func Today(loc *time.Location) time.Time {
	return DateOf(time.Now().In(loc))
}

// DateOf truncates t to its calendar date, as midnight UTC.
func DateOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// MonthStart returns the first day of t's month.
func MonthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// rawTable is the generic "fields + data" table shape used by the exchange web endpoints.
type rawTable struct {
	Title  string   `json:"title"`
	Fields []string `json:"fields"`
	Data   [][]any  `json:"data"`
}

// col returns the index of the first field whose name starts with one of the
// given prefixes, or -1.
func (t rawTable) col(prefixes ...string) int {
	for _, p := range prefixes {
		for i, f := range t.Fields {
			if strings.HasPrefix(strings.TrimSpace(f), p) {
				return i
			}
		}
	}
	return -1
}

func cell(row []any, i int) string {
	if i < 0 || i >= len(row) || row[i] == nil {
		return ""
	}
	if s, ok := row[i].(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprint(row[i]))
}
