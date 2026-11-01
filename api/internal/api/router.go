// Package api exposes the REST API consumed by the Flutter app.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"copium/api/internal/ai"
	"copium/api/internal/backtest"
	"copium/api/internal/market"
	"copium/api/internal/store"
)

type Deps struct {
	Market    *market.Service
	Store     *store.Store
	Backtests *backtest.Manager
	AI        *ai.Client
}

type handlers struct{ Deps }

func NewRouter(d Deps) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), cors())
	h := &handlers{d}

	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	v1 := r.Group("/api/v1")
	{
		// Market data (public).
		v1.GET("/quotes", h.listQuotes)
		v1.GET("/quotes/:code", h.getQuote)
		v1.GET("/stocks/:code/kline", h.getKLine)

		// Screener.
		v1.GET("/indicators", h.listIndicators)
		v1.GET("/strategies", h.listStrategies)
		v1.GET("/strategies/:id", h.getStrategy)
		v1.POST("/strategies", h.createStrategy)
		v1.GET("/screens/today", h.todayScreens)

		// Backtests (background jobs).
		v1.POST("/backtests", h.createBacktest)
		v1.GET("/backtests", h.listBacktests)
		v1.GET("/backtests/:id", h.getBacktest)

		// AI chat (Gemini, proxied so the API key never reaches the app).
		v1.POST("/chat", h.chat)

		// Personal data (requires a user).
		auth := v1.Group("", requireUser())
		auth.GET("/watchlist", h.listWatchlist)
		auth.POST("/watchlist", h.addWatch)
		auth.DELETE("/watchlist/:code", h.removeWatch)
		auth.GET("/holdings", h.listHoldings)
		auth.PUT("/holdings/:code", h.saveHolding)
		auth.DELETE("/holdings/:code", h.deleteHolding)
	}
	return r
}
