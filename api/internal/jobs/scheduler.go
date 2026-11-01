package jobs

import (
	"context"
	"log"
	"time"

	"copium/api/internal/config"
	"copium/api/internal/market"
)

// Scheduler triggers the recurring jobs:
//   - quote cache refresh every QuoteRefreshInterval (weekdays)
//   - closing price ingest at CloseIngestAt
//   - evening ingest + daily screens at EveningIngestAt
type Scheduler struct {
	cfg config.Config
	mkt *market.Service
	in  *Ingestor
	loc *time.Location
}

func NewScheduler(cfg config.Config, mkt *market.Service, in *Ingestor) *Scheduler {
	return &Scheduler{cfg: cfg, mkt: mkt, in: in, loc: cfg.Location()}
}

func (s *Scheduler) Run(ctx context.Context) {
	go s.refreshLoop(ctx)

	lastRun := map[string]string{} // job name -> date it last ran
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		now := time.Now().In(s.loc)
		if isWeekend(now) {
			continue
		}
		today, hm := now.Format("2006-01-02"), now.Format("15:04")
		s.runOnce(ctx, lastRun, "close", today, hm == s.cfg.CloseIngestAt, s.in.IngestClose)
		s.runOnce(ctx, lastRun, "evening", today, hm == s.cfg.EveningIngestAt, s.in.IngestEvening)
	}
}

func (s *Scheduler) runOnce(ctx context.Context, lastRun map[string]string, name, today string, due bool,
	job func(context.Context) error) {
	if !due || lastRun[name] == today {
		return
	}
	lastRun[name] = today
	jobCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	start := time.Now()
	if err := job(jobCtx); err != nil {
		log.Printf("job %s failed: %v", name, err) // TODO: alert on failure
		return
	}
	log.Printf("job %s done in %s", name, time.Since(start).Round(time.Second))
}

func (s *Scheduler) refreshLoop(ctx context.Context) {
	refresh := func() {
		if isWeekend(time.Now().In(s.loc)) {
			return
		}
		rctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		if err := s.mkt.RefreshQuotes(rctx); err != nil {
			log.Printf("refresh quotes: %v", err)
		}
	}
	refresh()
	t := time.NewTicker(s.cfg.QuoteRefreshInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			refresh()
		}
	}
}

func isWeekend(t time.Time) bool {
	return t.Weekday() == time.Saturday || t.Weekday() == time.Sunday
}
