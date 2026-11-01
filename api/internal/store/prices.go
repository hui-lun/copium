package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"copium/api/internal/market"
)

// UpsertQuotes stores stock metadata and daily prices from a set of quotes.
// Quotes without a trade (close == 0) only update the stock row.
func (s *Store) UpsertQuotes(ctx context.Context, quotes []market.Quote) (int, error) {
	b := &pgx.Batch{}
	n := 0
	for _, q := range quotes {
		if q.Code == "" {
			continue
		}
		b.Queue(`INSERT INTO stocks (code, name, exchange) VALUES ($1, $2, $3)
			ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, exchange = EXCLUDED.exchange, updated_at = now()`,
			q.Code, q.Name, string(q.Exchange))
		if q.Close <= 0 {
			continue
		}
		b.Queue(`INSERT INTO daily_prices (code, trade_date, open, high, low, close, volume, value)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (code, trade_date) DO UPDATE SET open = EXCLUDED.open, high = EXCLUDED.high,
			low = EXCLUDED.low, close = EXCLUDED.close, volume = EXCLUDED.volume, value = EXCLUDED.value`,
			q.Code, q.Date, q.Open, q.High, q.Low, q.Close, q.Volume, q.Value)
		n++
	}
	return n, s.sendBatch(ctx, b)
}

// UpsertBars stores bars for one stock.
func (s *Store) UpsertBars(ctx context.Context, code string, bars []market.Bar) error {
	b := &pgx.Batch{}
	for _, bar := range bars {
		if bar.Close <= 0 {
			continue
		}
		b.Queue(`INSERT INTO daily_prices (code, trade_date, open, high, low, close, volume, value)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (code, trade_date) DO UPDATE SET open = EXCLUDED.open, high = EXCLUDED.high,
			low = EXCLUDED.low, close = EXCLUDED.close, volume = EXCLUDED.volume, value = EXCLUDED.value`,
			code, bar.Date, bar.Open, bar.High, bar.Low, bar.Close, bar.Volume, bar.Value)
	}
	return s.sendBatch(ctx, b)
}

// Bars returns one stock's bars in [from, to], oldest first. Implements market.PriceStore.
func (s *Store) Bars(ctx context.Context, code string, from, to time.Time) ([]market.Bar, error) {
	rows, err := s.pool.Query(ctx, `SELECT trade_date, open, high, low, close, volume, value
		FROM daily_prices WHERE code = $1 AND trade_date BETWEEN $2 AND $3 ORDER BY trade_date`, code, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bars []market.Bar
	for rows.Next() {
		var b market.Bar
		if err := rows.Scan(&b.Date, &b.Open, &b.High, &b.Low, &b.Close, &b.Volume, &b.Value); err != nil {
			return nil, err
		}
		bars = append(bars, b)
	}
	return bars, rows.Err()
}

// LatestTradeDate returns the most recent date with any price data.
func (s *Store) LatestTradeDate(ctx context.Context) (time.Time, error) {
	var t *time.Time
	if err := s.pool.QueryRow(ctx, `SELECT max(trade_date) FROM daily_prices`).Scan(&t); err != nil {
		return time.Time{}, err
	}
	if t == nil {
		return time.Time{}, ErrNotFound
	}
	return *t, nil
}

// MarkInactiveStocks sets delisted_on for stocks with no trades since cutoff.
// This is a heuristic: long suspensions are also caught, and are cleared again
// as soon as the stock trades.
func (s *Store) MarkInactiveStocks(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx, `
		WITH last AS (SELECT code, max(trade_date) AS d FROM daily_prices GROUP BY code)
		UPDATE stocks s SET delisted_on = CASE WHEN last.d < $1 THEN last.d + 1 ELSE NULL END
		FROM last WHERE s.code = last.code
		  AND (s.delisted_on IS DISTINCT FROM CASE WHEN last.d < $1 THEN last.d + 1 ELSE NULL END)`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// BackfilledDays returns the dates already backfilled for a source.
func (s *Store) BackfilledDays(ctx context.Context, source string) (map[time.Time]bool, error) {
	rows, err := s.pool.Query(ctx, `SELECT trade_date FROM backfill_progress WHERE source = $1`, source)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	done := map[time.Time]bool{}
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		done[d] = true
	}
	return done, rows.Err()
}

func (s *Store) MarkBackfilled(ctx context.Context, source string, date time.Time, n int) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO backfill_progress (source, trade_date, rows) VALUES ($1, $2, $3)
		ON CONFLICT (source, trade_date) DO UPDATE SET rows = EXCLUDED.rows, done_at = now()`, source, date, n)
	return err
}
