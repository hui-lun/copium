package market

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

const (
	keyQuotesAll = "quotes:all"
	quotesTTL    = 30 * time.Minute
)

var ErrNotFound = errors.New("not found")

// Cache is the small key/value interface the service needs (implemented by store.Cache).
type Cache interface {
	Get(ctx context.Context, key string, out any) (bool, error)
	Set(ctx context.Context, key string, v any, ttl time.Duration) error
}

// PriceStore reads historical bars from the database (implemented by store.Store).
type PriceStore interface {
	Bars(ctx context.Context, code string, from, to time.Time) ([]Bar, error)
}

// Service serves quotes and K-lines to the API, backed by a cache and the database.
type Service struct {
	twse   *TWSESource
	tpex   *TPExSource
	cache  Cache
	prices PriceStore
	loc    *time.Location
}

func NewService(twse *TWSESource, tpex *TPExSource, cache Cache, prices PriceStore, loc *time.Location) *Service {
	return &Service{twse: twse, tpex: tpex, cache: cache, prices: prices, loc: loc}
}

// RefreshQuotes fetches all TWSE + TPEx quotes and caches them.
// A partial failure still caches whatever was fetched.
func (s *Service) RefreshQuotes(ctx context.Context) error {
	today := Today(s.loc)
	var all []Quote
	var errs []error
	if q, err := s.twse.DailyQuotes(ctx, today); err != nil {
		errs = append(errs, fmt.Errorf("twse: %w", err))
	} else {
		all = append(all, q...)
	}
	if q, err := s.tpex.DailyQuotes(ctx, today); err != nil {
		errs = append(errs, fmt.Errorf("tpex: %w", err))
	} else {
		all = append(all, q...)
	}
	if len(all) == 0 {
		return errors.Join(errs...)
	}
	if err := s.cache.Set(ctx, keyQuotesAll, all, quotesTTL); err != nil {
		return err
	}
	if len(errs) > 0 {
		log.Printf("refresh quotes: partial failure: %v", errors.Join(errs...))
	}
	return nil
}

// AllQuotes returns every cached quote, fetching on a cache miss.
func (s *Service) AllQuotes(ctx context.Context) ([]Quote, error) {
	var quotes []Quote
	found, err := s.cache.Get(ctx, keyQuotesAll, &quotes)
	if err != nil {
		return nil, err
	}
	if !found {
		if err := s.RefreshQuotes(ctx); err != nil {
			return nil, err
		}
		if _, err := s.cache.Get(ctx, keyQuotesAll, &quotes); err != nil {
			return nil, err
		}
	}
	return quotes, nil
}

// Quotes returns quotes filtered by exchange ("" = all) and keyword (code prefix or name).
func (s *Service) Quotes(ctx context.Context, exchange, keyword string, limit int) ([]Quote, error) {
	all, err := s.AllQuotes(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]Quote, 0, min(len(all), max(limit, 0)))
	for _, q := range all {
		if exchange != "" && !strings.EqualFold(string(q.Exchange), exchange) {
			continue
		}
		if keyword != "" && !strings.HasPrefix(q.Code, keyword) && !strings.Contains(q.Name, keyword) {
			continue
		}
		res = append(res, q)
		if limit > 0 && len(res) >= limit {
			break
		}
	}
	return res, nil
}

// Quote returns a single security's quote.
func (s *Service) Quote(ctx context.Context, code string) (Quote, error) {
	all, err := s.AllQuotes(ctx)
	if err != nil {
		return Quote{}, err
	}
	for _, q := range all {
		if q.Code == code {
			return q, nil
		}
	}
	return Quote{}, ErrNotFound
}

// KLine returns the last `months` months of daily bars. It reads the database
// first and falls back to the exchange web endpoints when nothing is stored yet.
func (s *Service) KLine(ctx context.Context, code string, months int) ([]Bar, error) {
	to := Today(s.loc)
	from := to.AddDate(0, -months, 0)
	bars, err := s.prices.Bars(ctx, code, from, to)
	if err == nil && len(bars) > 0 {
		return bars, nil
	}

	q, err := s.Quote(ctx, code)
	if err != nil {
		return nil, err
	}
	var out []Bar
	for i := months; i >= 0; i-- {
		month := MonthStart(to).AddDate(0, -i, 0)
		var b []Bar
		if q.Exchange == TPEx {
			b, err = s.tpex.History(ctx, code, month)
		} else {
			b, err = s.twse.History(ctx, code, month)
		}
		if err != nil {
			return nil, err
		}
		for _, bar := range b {
			if !bar.Date.Before(from) {
				out = append(out, bar)
			}
		}
	}
	return out, nil
}
