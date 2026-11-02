package market

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	twseOpenAPI = "https://openapi.twse.com.tw/v1"
	twseWeb     = "https://www.twse.com.tw/rwd/zh"
)

// TWSESource reads data for listed (上市) securities.
//
// NOTE: field names below follow the public responses at the time of writing.
// The exchanges change formats occasionally; verify against a live response
// whenever a fetch starts returning zero rows.
type TWSESource struct{ c *Client }

func NewTWSE(c *Client) *TWSESource { return &TWSESource{c: c} }

// DailyQuotes returns the latest session's quotes for all listed securities
// (OpenAPI STOCK_DAY_ALL). fallbackDate is used when a row carries no date.
func (s *TWSESource) DailyQuotes(ctx context.Context, fallbackDate time.Time) ([]Quote, error) {
	var rows []struct {
		Date         string `json:"Date"`
		Code         string `json:"Code"`
		Name         string `json:"Name"`
		TradeVolume  string `json:"TradeVolume"`
		TradeValue   string `json:"TradeValue"`
		OpeningPrice string `json:"OpeningPrice"`
		HighestPrice string `json:"HighestPrice"`
		LowestPrice  string `json:"LowestPrice"`
		ClosingPrice string `json:"ClosingPrice"`
		Change       string `json:"Change"`
	}
	if err := s.c.getJSON(ctx, twseOpenAPI+"/exchangeReport/STOCK_DAY_ALL", &rows); err != nil {
		return nil, err
	}
	quotes := make([]Quote, 0, len(rows))
	for _, r := range rows {
		date := fallbackDate
		if d, err := parseDate(r.Date); err == nil {
			date = d
		}
		quotes = append(quotes, Quote{
			Code: r.Code, Name: r.Name, Exchange: TWSE, Date: date,
			Open: parseFloat(r.OpeningPrice), High: parseFloat(r.HighestPrice),
			Low: parseFloat(r.LowestPrice), Close: parseFloat(r.ClosingPrice),
			Change: parseFloat(r.Change), Volume: parseInt(r.TradeVolume), Value: parseInt(r.TradeValue),
		})
	}
	return filterStocks(quotes), nil
}

// DailyQuotesOn returns all listed securities' quotes for a past trading day
// (web endpoint MI_INDEX). One request covers the whole market, which makes
// historical backfill fast and also includes stocks that were later delisted.
// A non-trading day returns an empty slice.
func (s *TWSESource) DailyQuotesOn(ctx context.Context, date time.Time) ([]Quote, error) {
	var resp struct {
		Stat   string     `json:"stat"`
		Tables []rawTable `json:"tables"`
		// Older responses use numbered keys instead of "tables".
		Fields9 []string `json:"fields9"`
		Data9   [][]any  `json:"data9"`
	}
	url := fmt.Sprintf("%s/afterTrading/MI_INDEX?date=%s&type=ALLBUT0999&response=json", twseWeb, date.Format("20060102"))
	if err := s.c.getJSON(ctx, url, &resp); err != nil {
		return nil, err
	}
	tables := resp.Tables
	if len(resp.Fields9) > 0 {
		tables = append(tables, rawTable{Fields: resp.Fields9, Data: resp.Data9})
	}
	for _, t := range tables {
		iCode, iClose := t.col("證券代號"), t.col("收盤價")
		if iCode < 0 || iClose < 0 {
			continue
		}
		iName, iVol, iVal := t.col("證券名稱"), t.col("成交股數"), t.col("成交金額")
		iOpen, iHigh, iLow := t.col("開盤價"), t.col("最高價"), t.col("最低價")
		// The sign lives in its own column as HTML, e.g. <p style= color:red>+</p>.
		iSign, iDiff := t.col("漲跌(+/-)"), t.col("漲跌價差")
		quotes := make([]Quote, 0, len(t.Data))
		for _, row := range t.Data {
			change := parseFloat(cell(row, iDiff))
			if strings.Contains(cell(row, iSign), "-") {
				change = -change
			} else if !strings.Contains(cell(row, iSign), "+") {
				change = 0
			}
			quotes = append(quotes, Quote{
				Code: cell(row, iCode), Name: cell(row, iName), Exchange: TWSE, Date: date,
				Open: parseFloat(cell(row, iOpen)), High: parseFloat(cell(row, iHigh)),
				Low: parseFloat(cell(row, iLow)), Close: parseFloat(cell(row, iClose)), Change: change,
				Volume: parseInt(cell(row, iVol)), Value: parseInt(cell(row, iVal)),
			})
		}
		return filterStocks(quotes), nil
	}
	return nil, nil // holiday or no data
}

// History returns one month of daily bars for a listed stock (web endpoint STOCK_DAY).
func (s *TWSESource) History(ctx context.Context, code string, month time.Time) ([]Bar, error) {
	var resp struct {
		Stat string  `json:"stat"`
		Data [][]any `json:"data"`
	}
	url := fmt.Sprintf("%s/afterTrading/STOCK_DAY?date=%s01&stockNo=%s&response=json", twseWeb, month.Format("200601"), code)
	if err := s.c.getJSON(ctx, url, &resp); err != nil {
		return nil, err
	}
	if !strings.EqualFold(resp.Stat, "OK") {
		return nil, nil // no data for this month
	}
	// Columns: date (ROC), shares, value, open, high, low, close, change, transactions
	bars := make([]Bar, 0, len(resp.Data))
	for _, row := range resp.Data {
		d, err := parseDate(cell(row, 0))
		if err != nil {
			continue
		}
		bars = append(bars, Bar{
			Date: d, Volume: parseInt(cell(row, 1)), Value: parseInt(cell(row, 2)),
			Open: parseFloat(cell(row, 3)), High: parseFloat(cell(row, 4)),
			Low: parseFloat(cell(row, 5)), Close: parseFloat(cell(row, 6)),
		})
	}
	return bars, nil
}

// Valuations returns the latest PE / dividend yield / PB for all listed stocks.
func (s *TWSESource) Valuations(ctx context.Context, fallbackDate time.Time) ([]Valuation, error) {
	var rows []struct {
		Date          string `json:"Date"`
		Code          string `json:"Code"`
		PEratio       string `json:"PEratio"`
		DividendYield string `json:"DividendYield"`
		PBratio       string `json:"PBratio"`
	}
	if err := s.c.getJSON(ctx, twseOpenAPI+"/exchangeReport/BWIBBU_ALL", &rows); err != nil {
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
			PE: parseFloat(r.PEratio), DividendYield: parseFloat(r.DividendYield), PB: parseFloat(r.PBratio),
		})
	}
	return out, nil
}

// Institutional returns the three major institutional investors' net trades
// for one day (web endpoint T86). Columns are located by header name.
func (s *TWSESource) Institutional(ctx context.Context, date time.Time) ([]Institutional, error) {
	var t rawTable
	url := fmt.Sprintf("%s/fund/T86?date=%s&selectType=ALLBUT0999&response=json", twseWeb, date.Format("20060102"))
	if err := s.c.getJSON(ctx, url, &t); err != nil {
		return nil, err
	}
	if len(t.Data) == 0 {
		return nil, nil
	}
	iCode := t.col("證券代號")
	if iCode < 0 {
		return nil, fmt.Errorf("T86: code column not found in %v", t.Fields)
	}
	iForeign := t.col("外陸資買賣超股數(不含外資自營商)", "外資買賣超股數")
	iTrust := t.col("投信買賣超股數")
	iDealer := t.col("自營商買賣超股數")
	iTotal := t.col("三大法人買賣超股數")
	out := make([]Institutional, 0, len(t.Data))
	for _, row := range t.Data {
		out = append(out, Institutional{
			Code: cell(row, iCode), Date: date,
			ForeignNet: parseInt(cell(row, iForeign)), TrustNet: parseInt(cell(row, iTrust)),
			DealerNet: parseInt(cell(row, iDealer)), TotalNet: parseInt(cell(row, iTotal)),
		})
	}
	return out, nil
}

// MonthlyRevenue returns the latest monthly revenue report of all listed companies.
func (s *TWSESource) MonthlyRevenue(ctx context.Context) ([]Revenue, error) {
	var rows []map[string]string
	if err := s.c.getJSON(ctx, twseOpenAPI+"/opendata/t187ap05_L", &rows); err != nil {
		return nil, err
	}
	return parseRevenueRows(rows), nil
}

// parseRevenueRows converts the MOPS monthly revenue open-data format, shared
// by TWSE and TPEx. "出表日期" (report date) is used as the announcement date.
func parseRevenueRows(rows []map[string]string) []Revenue {
	out := make([]Revenue, 0, len(rows))
	for _, r := range rows {
		month, err := parseROCMonth(r["資料年月"])
		if err != nil {
			continue
		}
		announced, err := parseDate(r["出表日期"])
		if err != nil {
			continue
		}
		out = append(out, Revenue{
			Code:        strings.TrimSpace(r["公司代號"]),
			Month:       month,
			AnnouncedOn: announced,
			Revenue:     parseInt(r["營業收入-當月營收"]),
			MoM:         parseFloat(r["營業收入-上月比較增減(%)"]),
			YoY:         parseFloat(r["營業收入-去年同月增減(%)"]),
		})
	}
	return out
}
