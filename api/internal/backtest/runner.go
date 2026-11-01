package backtest

import (
	"context"
	"log"
	"sync"
	"time"

	"copium/api/internal/screener"
)

// Store persists backtest jobs (implemented by store.Store).
type Store interface {
	CreateBacktest(ctx context.Context, strategyID int64, cfg Config) (int64, error)
	FinishBacktest(ctx context.Context, id int64, res *Result, runErr error) error
}

// DatasetLoader loads market data for [from, to].
type DatasetLoader func(ctx context.Context, from, to time.Time) (*screener.Dataset, error)

// Manager runs backtests as background jobs and tracks their progress.
type Manager struct {
	store Store
	load  DatasetLoader

	mu       sync.Mutex
	progress map[int64]float64
}

func NewManager(store Store, load DatasetLoader) *Manager {
	return &Manager{store: store, load: load, progress: map[int64]float64{}}
}

// Submit validates the config, records the job and starts it in the background.
func (m *Manager) Submit(ctx context.Context, strategyID int64, strat *screener.Strategy, cfg Config) (int64, error) {
	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return 0, err
	}
	id, err := m.store.CreateBacktest(ctx, strategyID, cfg)
	if err != nil {
		return 0, err
	}
	m.setProgress(id, 0)
	go m.run(id, strat, cfg)
	return id, nil
}

// Progress returns a running job's progress in [0, 1].
func (m *Manager) Progress(id int64) (float64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.progress[id]
	return p, ok
}

func (m *Manager) setProgress(id int64, p float64) {
	m.mu.Lock()
	m.progress[id] = p
	m.mu.Unlock()
}

func (m *Manager) run(id int64, strat *screener.Strategy, cfg Config) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	started := time.Now()

	from, to, _ := cfg.Range()
	// Load extra calendar days before the start so indicators have history.
	warmup := strat.Lookback()*2 + 30
	var res *Result
	ds, err := m.load(ctx, from.AddDate(0, 0, -warmup), to)
	if err == nil {
		res, err = Run(ctx, strat, ds, cfg, func(p float64) { m.setProgress(id, p) })
	}
	if ferr := m.store.FinishBacktest(context.Background(), id, res, err); ferr != nil {
		log.Printf("backtest %d: save result: %v", id, ferr)
	}
	log.Printf("backtest %d finished in %s (err=%v)", id, time.Since(started).Round(time.Millisecond), err)

	m.mu.Lock()
	delete(m.progress, id)
	m.mu.Unlock()
}
