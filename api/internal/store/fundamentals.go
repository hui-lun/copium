package store

import (
	"context"

	"github.com/jackc/pgx/v5"

	"copium/api/internal/market"
)

func (s *Store) UpsertValuations(ctx context.Context, vals []market.Valuation) error {
	b := &pgx.Batch{}
	for _, v := range vals {
		b.Queue(`INSERT INTO valuations (code, trade_date, pe, dividend_yield, pb) VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (code, trade_date) DO UPDATE SET pe = EXCLUDED.pe,
			dividend_yield = EXCLUDED.dividend_yield, pb = EXCLUDED.pb`,
			v.Code, v.Date, v.PE, v.DividendYield, v.PB)
	}
	return s.sendBatch(ctx, b)
}

// UpsertRevenue stores monthly revenue. The original announcement date is kept
// on conflict, because re-fetching later must not move it forward or backward.
func (s *Store) UpsertRevenue(ctx context.Context, revs []market.Revenue) error {
	b := &pgx.Batch{}
	for _, r := range revs {
		b.Queue(`INSERT INTO monthly_revenue (code, year_month, announced_on, revenue, yoy, mom)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (code, year_month) DO UPDATE SET revenue = EXCLUDED.revenue,
			yoy = EXCLUDED.yoy, mom = EXCLUDED.mom`,
			r.Code, r.Month, r.AnnouncedOn, r.Revenue, r.YoY, r.MoM)
	}
	return s.sendBatch(ctx, b)
}

func (s *Store) UpsertInstitutional(ctx context.Context, rows []market.Institutional) error {
	b := &pgx.Batch{}
	for _, r := range rows {
		b.Queue(`INSERT INTO institutional (code, trade_date, foreign_net, trust_net, dealer_net, total_net)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (code, trade_date) DO UPDATE SET foreign_net = EXCLUDED.foreign_net,
			trust_net = EXCLUDED.trust_net, dealer_net = EXCLUDED.dealer_net, total_net = EXCLUDED.total_net`,
			r.Code, r.Date, r.ForeignNet, r.TrustNet, r.DealerNet, r.TotalNet)
	}
	return s.sendBatch(ctx, b)
}
