// Command server runs the REST API and the scheduled jobs.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"copium/api/internal/api"
	"copium/api/internal/backtest"
	"copium/api/internal/config"
	"copium/api/internal/jobs"
	"copium/api/internal/market"
	"copium/api/internal/store"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer st.Close()

	cache, err := store.NewCache(ctx, cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer cache.Close()

	client := market.NewClient(cfg.WebRequestInterval)
	twse, tpex := market.NewTWSE(client), market.NewTPEx(client)
	mkt := market.NewService(twse, tpex, cache, st, cfg.Location())
	ingestor := jobs.NewIngestor(st, twse, tpex, cfg.Location())
	backtests := backtest.NewManager(st, st.LoadDataset)

	go jobs.NewScheduler(cfg, mkt, ingestor).Run(ctx)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           api.NewRouter(api.Deps{Market: mkt, Store: st, Backtests: backtests}),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		log.Printf("listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Println("server stopped")
}
