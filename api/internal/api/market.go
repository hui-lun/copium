package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"copium/api/internal/market"
)

type candleDTO struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int64   `json:"volume"`
}

// GET /api/v1/quotes?exchange=TWSE&q=台積&limit=50
func (h *handlers) listQuotes(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))
	quotes, err := h.Market.Quotes(c.Request.Context(), c.Query("exchange"), c.Query("q"), limit)
	if err != nil {
		fail(c, http.StatusBadGateway, err.Error())
		return
	}
	ok(c, quotes)
}

// GET /api/v1/quotes/:code
func (h *handlers) getQuote(c *gin.Context) {
	q, err := h.Market.Quote(c.Request.Context(), c.Param("code"))
	if errors.Is(err, market.ErrNotFound) {
		fail(c, http.StatusNotFound, "stock not found")
		return
	}
	if err != nil {
		fail(c, http.StatusBadGateway, err.Error())
		return
	}
	ok(c, q)
}

// GET /api/v1/stocks/:code/kline?months=6
func (h *handlers) getKLine(c *gin.Context) {
	months, _ := strconv.Atoi(c.DefaultQuery("months", "6"))
	if months < 1 || months > 60 {
		months = 6
	}
	bars, err := h.Market.KLine(c.Request.Context(), c.Param("code"), months)
	if errors.Is(err, market.ErrNotFound) {
		fail(c, http.StatusNotFound, "stock not found")
		return
	}
	if err != nil {
		fail(c, http.StatusBadGateway, err.Error())
		return
	}
	out := make([]candleDTO, len(bars))
	for i, b := range bars {
		out[i] = candleDTO{Date: b.Date.Format("2006-01-02"), Open: b.Open, High: b.High, Low: b.Low, Close: b.Close, Volume: b.Volume}
	}
	ok(c, out)
}
