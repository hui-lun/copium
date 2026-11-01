// Package jobs contains the scheduled data ingestion, daily screening and
// historical backfill jobs.
package jobs

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"copium/api/internal/market"
	"copium/api/internal/screener"
	"copium/api/internal/store"
)

type Ingestor struct {
	store *store.Store
	twse  *market.TWSESource
	tpex  *market.TPExSource
	loc   *time.Location
}

func NewIngestor(st *store.Store, twse *market.TWSESource, tpex *market.TPExSource, loc *time.Location) *Ingestor {
	return &Ingestor{store: st, twse: twse, tpex: tpex, loc: loc}
}

// IngestClose stores today's closing prices for all TWSE and TPEx securities.
func (in *Ingestor) IngestClose(ctx context.Context) error {
	today := market.Today(in.loc)
	var all []market.Quote
	var errs []error
	if q, err := in.twse.DailyQuotes(ctx, today); err != nil {
		errs = append(errs, fmt.Errorf("twse quotes: %w", err))
	} else {
		all = append(all, q...)
	}
	if q, err := in.tpex.DailyQuotes(ctx, today); err != nil {
		errs = append(errs, fmt.Errorf("tpex quotes: %w", err))
	} else {
		all = append(all, q...)
	}
	for _, w := range checkQuotes(all) {
		log.Printf("ingest close: WARNING %s", w) // TODO: send an alert notification
	}
	n, err := in.store.UpsertQuotes(ctx, all)
	if err != nil {
		errs = append(errs, err)
	}
	log.Printf("ingest close: stored %d prices", n)
	return errors.Join(errs...)
}

// IngestEvening stores valuations, institutional trades and monthly revenue,
// then runs the daily screens. Institutional data is published around 16:30,
// so this runs in the evening.
func (in *Ingestor) IngestEvening(ctx context.Context) error {
	today := market.Today(in.loc)
	var errs []error
	collect := func(what string, err error) {
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", what, err))
		}
	}

	for _, fetch := range []func(context.Context, time.Time) ([]market.Valuation, error){in.twse.Valuations, in.tpex.Valuations} {
		v, err := fetch(ctx, today)
		collect("valuations", err)
		if err == nil {
			collect("store valuations", in.store.UpsertValuations(ctx, v))
		}
	}

	inst, err := in.twse.Institutional(ctx, today)
	collect("institutional", err)
	if err == nil {
		collect("store institutional", in.store.UpsertInstitutional(ctx, inst))
	}
	// TODO: TPEx institutional trades.

	for _, fetch := range []func(context.Context) ([]market.Revenue, error){in.twse.MonthlyRevenue, in.tpex.MonthlyRevenue} {
		r, err := fetch(ctx)
		collect("revenue", err)
		if err == nil {
			collect("store revenue", in.store.UpsertRevenue(ctx, r))
		}
	}

	date, err := in.store.LatestTradeDate(ctx)
	if err == nil {
		collect("screens", in.RunScreens(ctx, date))
	}
	return errors.Join(errs...)
}

// RunScreens runs every enabled strategy for date and stores the hits.
func (in *Ingestor) RunScreens(ctx context.Context, date time.Time) error {
	rows, err := in.store.ListStrategies(ctx, true)
	if err != nil {
		return err
	}
	type parsed struct {
		id    int64
		strat *screener.Strategy
	}
	var strategies []parsed
	lookback := 0
	for _, r := range rows {
		s, err := screener.Parse(r.Definition)
		if err != nil {
			log.Printf("screen: skip strategy %d: %v", r.ID, err)
			continue
		}
		strategies = append(strategies, parsed{r.ID, s})
		lookback = max(lookback, s.Lookback())
	}
	if len(strategies) == 0 {
		return nil
	}
	ds, err := in.store.LoadDataset(ctx, date.AddDate(0, 0, -(lookback*2+30)), date)
	if err != nil {
		return err
	}
	for _, p := range strategies {
		hits := screener.Scan(p.strat, ds, date)
		if err := in.store.SaveScreenResults(ctx, date, p.id, hits); err != nil {
			return err
		}
		log.Printf("screen %s: strategy %d matched %d stocks", date.Format("2006-01-02"), p.id, len(hits))
		// TODO: push hits to subscribed devices via Firebase Cloud Messaging.
	}
	return nil
}

// Backfill loads daily prices for the whole market from `start` until today,
// one trading day per request (TWSE MI_INDEX / TPEx daily quotes). Finished
// days are recorded, so an interrupted run resumes where it stopped. Because
// it reads whole-market daily files, delisted stocks are included automatically.
func (in *Ingestor) Backfill(ctx context.Context, start time.Time, exchange string) error {
	end := market.Today(in.loc)

	type source struct {
		name  string
		fetch func(context.Context, time.Time) ([]market.Quote, error)
	}
	var sources []source
	if exchange == "" || exchange == "twse" {
		sources = append(sources, source{"twse", in.twse.DailyQuotesOn})
	}
	if exchange == "" || exchange == "tpex" {
		sources = append(sources, source{"tpex", in.tpex.DailyQuotesOn})
	}

	// Each source has its own host and rate limit, so they run in parallel.
	var wg sync.WaitGroup
	errCh := make(chan error, len(sources))
	for _, src := range sources {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errCh <- in.backfillSource(ctx, src.name, src.fetch, start, end)
		}()
	}
	wg.Wait()
	close(errCh)
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}

	// Stocks with no trades in the last 30 days are treated as delisted.
	n, err := in.store.MarkInactiveStocks(ctx, end.AddDate(0, 0, -30))
	if err != nil {
		return err
	}
	log.Printf("backfill: updated delisted status of %d stocks", n)
	return nil
}

func (in *Ingestor) backfillSource(ctx context.Context, name string,
	fetch func(context.Context, time.Time) ([]market.Quote, error), start, end time.Time) error {
	done, err := in.store.BackfilledDays(ctx, name)
	if err != nil {
		return err
	}
	total := int(end.Sub(start).Hours() / 24)
	for i, day := 0, start; !day.After(end); i, day = i+1, day.AddDate(0, 0, 1) {
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday || done[day] {
			continue
		}
		quotes, err := fetch(ctx, day)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("backfill %s %s: %v (will retry next run)", name, day.Format("2006-01-02"), err)
			continue
		}
		n, err := in.store.UpsertQuotes(ctx, quotes)
		if err != nil {
			return err
		}
		// Today's data may still be incomplete, so do not mark it done.
		if day.Before(end) {
			if err := in.store.MarkBackfilled(ctx, name, day, n); err != nil {
				return err
			}
		}
		log.Printf("backfill %s %s: %d rows (%d/%d)", name, day.Format("2006-01-02"), n, i, total)
	}
	return nil
}
