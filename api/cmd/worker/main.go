// Command worker runs data jobs on demand.
//
//	worker backfill [-years 5 | -days N] [-exchange twse|tpex]   load historical prices (resumable)
//	worker ingest-close                                store today's closing prices
//	worker ingest-evening                              valuations, institutional, revenue + screens
//	worker screen [-date YYYY-MM-DD]                   run the daily screens for a date
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"copium/api/internal/config"
	"copium/api/internal/jobs"
	"copium/api/internal/market"
	"copium/api/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	cmd, args := os.Args[1], os.Args[2:]

	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer st.Close()

	client := market.NewClient(cfg.WebRequestInterval)
	in := jobs.NewIngestor(st, market.NewTWSE(client), market.NewTPEx(client), cfg.Location())

	switch cmd {
	case "backfill":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		years := fs.Int("years", 5, "years of history to load")
		days := fs.Int("days", 0, "days of history to load (overrides -years when > 0)")
		exchange := fs.String("exchange", "", "twse, tpex, or empty for both")
		_ = fs.Parse(args)
		start := market.Today(cfg.Location()).AddDate(-*years, 0, 0)
		if *days > 0 {
			start = market.Today(cfg.Location()).AddDate(0, 0, -*days)
		}
		err = in.Backfill(ctx, start, *exchange)
	case "ingest-close":
		err = in.IngestClose(ctx)
	case "ingest-evening":
		err = in.IngestEvening(ctx)
	case "screen":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		date := fs.String("date", "", "YYYY-MM-DD (default: latest trading day)")
		_ = fs.Parse(args)
		var d time.Time
		if *date == "" {
			d, err = st.LatestTradeDate(ctx)
		} else {
			d, err = time.Parse("2006-01-02", *date)
		}
		if err == nil {
			err = in.RunScreens(ctx, d)
		}
	default:
		usage()
	}
	if err != nil {
		log.Fatalf("%s: %v", cmd, err)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: worker <backfill|ingest-close|ingest-evening|screen> [flags]")
	os.Exit(2)
}
