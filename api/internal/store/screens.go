package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"

	"copium/api/internal/backtest"
	"copium/api/internal/screener"
)

// ---------- strategies ----------

type StrategyRow struct {
	ID         int64           `json:"id"`
	Name       string          `json:"name"`
	Definition json.RawMessage `json:"definition"`
	Enabled    bool            `json:"enabled"`
	CreatedAt  time.Time       `json:"created_at"`
}

func (s *Store) ListStrategies(ctx context.Context, enabledOnly bool) ([]StrategyRow, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, definition::text, enabled, created_at FROM strategies
		WHERE enabled OR NOT $1 ORDER BY id`, enabledOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []StrategyRow{}
	for rows.Next() {
		r, err := scanStrategy(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) GetStrategy(ctx context.Context, id int64) (StrategyRow, error) {
	row := s.pool.QueryRow(ctx, `SELECT id, name, definition::text, enabled, created_at FROM strategies WHERE id = $1`, id)
	r, err := scanStrategy(row)
	return r, notFound(err)
}

func (s *Store) CreateStrategy(ctx context.Context, name string, def json.RawMessage) (StrategyRow, error) {
	row := s.pool.QueryRow(ctx, `INSERT INTO strategies (name, definition) VALUES ($1, $2::jsonb)
		RETURNING id, name, definition::text, enabled, created_at`, name, string(def))
	return scanStrategy(row)
}

func scanStrategy(row pgx.Row) (StrategyRow, error) {
	var r StrategyRow
	var def string
	if err := row.Scan(&r.ID, &r.Name, &def, &r.Enabled, &r.CreatedAt); err != nil {
		return r, err
	}
	r.Definition = json.RawMessage(def)
	return r, nil
}

// ---------- screen results ----------

type ScreenHit struct {
	Date         time.Time `json:"date"`
	StrategyID   int64     `json:"strategy_id"`
	StrategyName string    `json:"strategy_name"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Close        float64   `json:"close"`
	Matched      []string  `json:"matched"`
}

// SaveScreenResults replaces one strategy's results for a date.
func (s *Store) SaveScreenResults(ctx context.Context, date time.Time, strategyID int64, hits []screener.Hit) error {
	b := &pgx.Batch{}
	b.Queue(`DELETE FROM screen_results WHERE screen_date = $1 AND strategy_id = $2`, date, strategyID)
	for _, h := range hits {
		matched, _ := json.Marshal(h.Matched)
		b.Queue(`INSERT INTO screen_results (screen_date, strategy_id, code, name, close, matched)
			VALUES ($1, $2, $3, $4, $5, $6::jsonb)`, date, strategyID, h.Code, h.Name, h.Close, string(matched))
	}
	return s.sendBatch(ctx, b)
}

// LatestScreenResults returns all hits from the most recent screening date.
func (s *Store) LatestScreenResults(ctx context.Context) ([]ScreenHit, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.screen_date, r.strategy_id, st.name, r.code, r.name, r.close, r.matched::text
		FROM screen_results r JOIN strategies st ON st.id = r.strategy_id
		WHERE r.screen_date = (SELECT max(screen_date) FROM screen_results)
		ORDER BY r.strategy_id, r.code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ScreenHit{}
	for rows.Next() {
		var h ScreenHit
		var matched string
		if err := rows.Scan(&h.Date, &h.StrategyID, &h.StrategyName, &h.Code, &h.Name, &h.Close, &matched); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(matched), &h.Matched)
		out = append(out, h)
	}
	return out, rows.Err()
}

// ---------- backtests ----------

type BacktestRow struct {
	ID         int64           `json:"id"`
	StrategyID *int64          `json:"strategy_id"`
	Status     string          `json:"status"`
	Config     json.RawMessage `json:"config"`
	Result     json.RawMessage `json:"result,omitempty"`
	Error      string          `json:"error,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	FinishedAt *time.Time      `json:"finished_at,omitempty"`
}

// CreateBacktest implements backtest.Store.
func (s *Store) CreateBacktest(ctx context.Context, strategyID int64, cfg backtest.Config) (int64, error) {
	b, err := json.Marshal(cfg)
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.pool.QueryRow(ctx, `INSERT INTO backtests (strategy_id, config, status) VALUES ($1, $2::jsonb, 'running')
		RETURNING id`, strategyID, string(b)).Scan(&id)
	return id, err
}

// FinishBacktest implements backtest.Store.
func (s *Store) FinishBacktest(ctx context.Context, id int64, res *backtest.Result, runErr error) error {
	if runErr != nil {
		_, err := s.pool.Exec(ctx, `UPDATE backtests SET status = 'failed', error = $2, finished_at = now() WHERE id = $1`,
			id, runErr.Error())
		return err
	}
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `UPDATE backtests SET status = 'done', result = $2::jsonb, finished_at = now() WHERE id = $1`,
		id, string(b))
	return err
}

// GetBacktest returns one backtest including its full result.
func (s *Store) GetBacktest(ctx context.Context, id int64) (BacktestRow, error) {
	row := s.pool.QueryRow(ctx, `SELECT id, strategy_id, status, config::text, COALESCE(result::text, ''),
		COALESCE(error, ''), created_at, finished_at FROM backtests WHERE id = $1`, id)
	var r BacktestRow
	var cfg, res string
	if err := row.Scan(&r.ID, &r.StrategyID, &r.Status, &cfg, &res, &r.Error, &r.CreatedAt, &r.FinishedAt); err != nil {
		return r, notFound(err)
	}
	r.Config = json.RawMessage(cfg)
	if res != "" {
		r.Result = json.RawMessage(res)
	}
	return r, nil
}

// ListBacktests returns recent backtests without their (large) results.
func (s *Store) ListBacktests(ctx context.Context, limit int) ([]BacktestRow, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, strategy_id, status, config::text, COALESCE(error, ''), created_at, finished_at
		FROM backtests ORDER BY id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BacktestRow{}
	for rows.Next() {
		var r BacktestRow
		var cfg string
		if err := rows.Scan(&r.ID, &r.StrategyID, &r.Status, &cfg, &r.Error, &r.CreatedAt, &r.FinishedAt); err != nil {
			return nil, err
		}
		r.Config = json.RawMessage(cfg)
		out = append(out, r)
	}
	return out, rows.Err()
}
