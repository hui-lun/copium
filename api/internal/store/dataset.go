package store

import (
	"context"
	"time"

	"copium/api/internal/market"
	"copium/api/internal/screener"
)

// LoadDataset loads everything the screener/backtester needs for [from, to]
// into memory. Revenue is loaded with an extra year of history so the latest
// announcement before `from` is available.
func (s *Store) LoadDataset(ctx context.Context, from, to time.Time) (*screener.Dataset, error) {
	ds := screener.NewDataset()

	rows, err := s.pool.Query(ctx, `SELECT code, name, exchange, COALESCE(industry, ''), COALESCE(capital, 0), delisted_on FROM stocks`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var m screener.StockMeta
		if err := rows.Scan(&m.Code, &m.Name, &m.Exchange, &m.Industry, &m.Capital, &m.DelistedOn); err != nil {
			rows.Close()
			return nil, err
		}
		ds.Stocks[m.Code] = m
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = s.pool.Query(ctx, `SELECT code, trade_date, open, high, low, close, volume, value
		FROM daily_prices WHERE trade_date BETWEEN $1 AND $2 ORDER BY code, trade_date`, from, to)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var code string
		var b market.Bar
		if err := rows.Scan(&code, &b.Date, &b.Open, &b.High, &b.Low, &b.Close, &b.Volume, &b.Value); err != nil {
			rows.Close()
			return nil, err
		}
		ds.Bars[code] = append(ds.Bars[code], b)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = s.pool.Query(ctx, `SELECT code, announced_on, yoy, mom FROM monthly_revenue
		WHERE announced_on BETWEEN ($1::date - 400) AND $2 ORDER BY code, announced_on`, from, to)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var code string
		var p screener.RevenuePoint
		if err := rows.Scan(&code, &p.AnnouncedOn, &p.YoY, &p.MoM); err != nil {
			rows.Close()
			return nil, err
		}
		ds.Revenue[code] = append(ds.Revenue[code], p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = s.pool.Query(ctx, `SELECT code, trade_date, total_net FROM institutional
		WHERE trade_date BETWEEN $1 AND $2 ORDER BY code, trade_date`, from, to)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var code string
		var p screener.DailyValue
		var net int64
		if err := rows.Scan(&code, &p.Date, &net); err != nil {
			rows.Close()
			return nil, err
		}
		p.Value = float64(net)
		ds.InstNet[code] = append(ds.InstNet[code], p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = s.pool.Query(ctx, `SELECT code, trade_date, pe, pb, dividend_yield FROM valuations
		WHERE trade_date BETWEEN $1 AND $2 ORDER BY code, trade_date`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var code string
		var p screener.ValuationPoint
		if err := rows.Scan(&code, &p.Date, &p.PE, &p.PB, &p.Yield); err != nil {
			return nil, err
		}
		ds.Valuation[code] = append(ds.Valuation[code], p)
	}
	return ds, rows.Err()
}
