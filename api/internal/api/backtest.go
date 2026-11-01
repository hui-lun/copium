package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"copium/api/internal/backtest"
	"copium/api/internal/screener"
	"copium/api/internal/store"
)

// POST /api/v1/backtests  {"strategy_id": 1, "config": {...}}
// Starts a background job and returns its id (202 Accepted).
func (h *handlers) createBacktest(c *gin.Context) {
	var body struct {
		StrategyID int64           `json:"strategy_id" binding:"required"`
		Config     backtest.Config `json:"config"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	row, err := h.Store.GetStrategy(c.Request.Context(), body.StrategyID)
	if errors.Is(err, store.ErrNotFound) {
		fail(c, http.StatusNotFound, "strategy not found")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	strat, err := screener.Parse(row.Definition)
	if err != nil {
		fail(c, http.StatusUnprocessableEntity, "stored strategy is invalid: "+err.Error())
		return
	}
	id, err := h.Backtests.Submit(c.Request.Context(), row.ID, strat, body.Config)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"data": gin.H{"id": id}})
}

// GET /api/v1/backtests
func (h *handlers) listBacktests(c *gin.Context) {
	rows, err := h.Store.ListBacktests(c.Request.Context(), 50)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, rows)
}

// GET /api/v1/backtests/:id — status, progress (0..1) and, when done, the result.
func (h *handlers) getBacktest(c *gin.Context) {
	id, valid := paramID(c)
	if !valid {
		return
	}
	row, err := h.Store.GetBacktest(c.Request.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		fail(c, http.StatusNotFound, "backtest not found")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	progress := 1.0
	if row.Status == "running" {
		progress, _ = h.Backtests.Progress(id)
	}
	ok(c, gin.H{"backtest": row, "progress": progress})
}
