package store

import (
	"context"
	"time"
)

// WatchItem is one stock on a user's watchlist.
type WatchItem struct {
	Code      string    `json:"code"`
	CreatedAt time.Time `json:"created_at"`
}

// Holding is one position in a user's portfolio.
type Holding struct {
	Code      string    `json:"code"`
	Shares    int64     `json:"shares"`
	AvgCost   float64   `json:"avg_cost"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *Store) Watchlist(ctx context.Context, userID int64) ([]WatchItem, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT code, created_at FROM watchlist WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []WatchItem{}
	for rows.Next() {
		var it WatchItem
		if err := rows.Scan(&it.Code, &it.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (s *Store) AddWatch(ctx context.Context, userID int64, code string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO watchlist (user_id, code) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, code)
	return err
}

func (s *Store) RemoveWatch(ctx context.Context, userID int64, code string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM watchlist WHERE user_id = $1 AND code = $2`, userID, code)
	return err
}

func (s *Store) Holdings(ctx context.Context, userID int64) ([]Holding, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT code, shares, avg_cost, updated_at FROM holdings WHERE user_id = $1 ORDER BY code`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Holding{}
	for rows.Next() {
		var h Holding
		if err := rows.Scan(&h.Code, &h.Shares, &h.AvgCost, &h.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, h)
	}
	return items, rows.Err()
}

// SaveHolding inserts or overwrites the position for one stock.
func (s *Store) SaveHolding(ctx context.Context, userID int64, code string, shares int64, avgCost float64) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO holdings (user_id, code, shares, avg_cost) VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, code) DO UPDATE
		SET shares = EXCLUDED.shares, avg_cost = EXCLUDED.avg_cost, updated_at = now()`,
		userID, code, shares, avgCost)
	return err
}

func (s *Store) DeleteHolding(ctx context.Context, userID int64, code string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM holdings WHERE user_id = $1 AND code = $2`, userID, code)
	return err
}
