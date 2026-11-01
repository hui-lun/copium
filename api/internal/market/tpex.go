package market

import (
	"context"
	"fmt"
	"time"
)

const (
	tpexOpenAPI = "https://www.tpex.org.tw/openapi/v1"
	tpexWeb     = "https://www.tpex.org.tw/www/zh-tw"
)

// TPExSource reads data for OTC (上櫃) securities. Most hidden-gem candidates
// live here, so its historical data matters as much as TWSE's.
//
// NOTE: verify field names against a live response if a fetch returns zero rows.
type TPExSource struct{ c *Client }

func NewTPEx(c *Client) *TPExSource { return &TPExSource{c: c} }

// DailyQuotes returns the latest session's closing quotes for all OTC securities.
func (s *TPExSource) DailyQuotes(ctx context.Context, fallbackDate time.Time) ([]Quote, error) {
	var rows []struct {
		Date              string `json:"Date"`
		Code              string `json:"SecuritiesCompanyCode"`
		Name              string `json:"CompanyName"`
		Close             string `json:"Close"`
		Change            string `json:"Change"`
		Open              string `json:"Open"`
		High              string `json:"High"`
		Low               string `json:"Low"`
		TradingShares     string `json:"TradingShares"`
		TransactionAmount string `json:"TransactionAmount"`
	}
	if err := s.c.getJSON(ctx, tpexOpenAPI+"/tpex_mainboard_daily_close_quotes", &rows); err != nil {
		return nil, err
	}
	quotes := make([]Quote, 0, len(rows))
	for _, r := range rows {
		date := fallbackDate
		if d, err := parseDate(r.Date); err == nil {
			date = d
		}
		quotes = append(quotes, Quote{
			Code: r.Code, Name: r.Name, Exchange: TPEx, Date: date,
			Open: parseFloat(r.Open), High: parseFloat(r.High),
			Low: parseFloat(r.Low), Close: parseFloat(r.Close),
			Change: parseFloat(r.Change), Volume: parseInt(r.TradingShares), Value: parseInt(r.TransactionAmount),
		})
	}
	return quotes, nil
}

// DailyQuotesOn returns all OTC securities' quotes for a past trading day.
// A non-trading day returns an empty slice.
func (s *TPExSource) DailyQuotesOn(ctx context.Context, date time.Time) ([]Quote, error) {
	var resp struct {
		Stat   string     `json:"stat"`
		Tables []rawTable `json:"tables"`
	}
	url := fmt.Sprintf("%s/afterTrading/otc?date=%s&type=EW&response=json", tpexWeb, date.Format("2006/01/02"))
	if err := s.c.getJSON(ctx, url, &resp); err != nil {
		return nil, err
	}
	for _, t := range resp.Tables {
		iCode, iClose := t.col("代號"), t.col("收盤")
		if iCode < 0 || iClose < 0 {
			continue
		}
		iName, iVol, iVal := t.col("名稱"), t.col("成交股數"), t.col("成交金額")
		iOpen, iHigh, iLow := t.col("開盤"), t.col("最高"), t.col("最低")
		quotes := make([]Quote, 0, len(t.Data))
		for _, row := range t.Data {
			quotes = append(quotes, Quote{
				Code: cell(row, iCode), Name: cell(row, iName), Exchange: TPEx, Date: date,
				Open: parseFloat(cell(row, iOpen)), High: parseFloat(cell(row, iHigh)),
				Low: parseFloat(cell(row, iLow)), Close: parseFloat(cell(row, iClose)),
				Volume: parseInt(cell(row, iVol)), Value: parseInt(cell(row, iVal)),
			})
		}
		return quotes, nil
	}
	return nil, nil
}

// History returns one month of daily bars for an OTC stock.
func (s *TPExSource) History(ctx context.Context, code string, month time.Time) ([]Bar, error) {
	var resp struct {
		Stat   string     `json:"stat"`
		Tables []rawTable `json:"tables"`
	}
	url := fmt.Sprintf("%s/afterTrading/tradingStock?code=%s&date=%s&response=json", tpexWeb, code, month.Format("2006/01/02"))
	if err := s.c.getJSON(ctx, url, &resp); err != nil {
		return nil, err
	}
	if len(resp.Tables) == 0 {
		return nil, nil
	}
	// Columns: date (ROC), volume (thousand shares), value (NT$ thousand),
	// open, high, low, close, change, transactions
	var bars []Bar
	for _, row := range resp.Tables[0].Data {
		d, err := parseDate(cell(row, 0))
		if err != nil {
			continue
		}
		bars = append(bars, Bar{
			Date: d, Volume: parseInt(cell(row, 1)) * 1000, Value: parseInt(cell(row, 2)) * 1000,
			Open: parseFloat(cell(row, 3)), High: parseFloat(cell(row, 4)),
			Low: parseFloat(cell(row, 5)), Close: parseFloat(cell(row, 6)),
		})
	}
	return bars, nil
}

// Valuations returns the latest PE / dividend yield / PB for all OTC stocks.
func (s *TPExSource) Valuations(ctx context.Context, fallbackDate time.Time) ([]Valuation, error) {
	var rows []struct {
		Date              string `json:"Date"`
		Code              string `json:"SecuritiesCompanyCode"`
		PriceEarningRatio string `json:"PriceEarningRatio"`
		YieldRatio        string `json:"YieldRatio"`
		PriceBookRatio    string `json:"PriceBookRatio"`
	}
	if err := s.c.getJSON(ctx, tpexOpenAPI+"/tpex_mainboard_peratio_analysis", &rows); err != nil {
		return nil, err
	}
	out := make([]Valuation, 0, len(rows))
	for _, r := range rows {
		date := fallbackDate
		if d, err := parseDate(r.Date); err == nil {
			date = d
		}
		out = append(out, Valuation{
			Code: r.Code, Date: date,
			PE: parseFloat(r.PriceEarningRatio), DividendYield: parseFloat(r.YieldRatio), PB: parseFloat(r.PriceBookRatio),
		})
	}
	return out, nil
}

// MonthlyRevenue returns the latest monthly revenue report of all OTC companies.
func (s *TPExSource) MonthlyRevenue(ctx context.Context) ([]Revenue, error) {
	var rows []map[string]string
	if err := s.c.getJSON(ctx, tpexOpenAPI+"/mopsfin_t187ap05_O", &rows); err != nil {
		return nil, err
	}
	return parseRevenueRows(rows), nil
}

// TODO: OTC institutional investor trades (TPEx 3insti endpoint).
