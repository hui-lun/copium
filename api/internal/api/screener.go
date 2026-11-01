package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"copium/api/internal/screener"
	"copium/api/internal/store"
)

// GET /api/v1/indicators
func (h *handlers) listIndicators(c *gin.Context) {
	ok(c, screener.Indicators())
}

// GET /api/v1/strategies
func (h *handlers) listStrategies(c *gin.Context) {
	rows, err := h.Store.ListStrategies(c.Request.Context(), false)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, rows)
}

// GET /api/v1/strategies/:id
func (h *handlers) getStrategy(c *gin.Context) {
	id, valid := paramID(c)
	if !valid {
		return
	}
	row, err := h.Store.GetStrategy(c.Request.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		fail(c, http.StatusNotFound, "strategy not found")
		return
	}
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, row)
}

// POST /api/v1/strategies  {"name": "...", "definition": {...strategy JSON...}}
func (h *handlers) createStrategy(c *gin.Context) {
	var body struct {
		Name       string          `json:"name"`
		Definition json.RawMessage `json:"definition" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	strat, err := screener.Parse(body.Definition)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	name := body.Name
	if name == "" {
		name = strat.Name
	}
	if name == "" {
		fail(c, http.StatusBadRequest, "name is required")
		return
	}
	row, err := h.Store.CreateStrategy(c.Request.Context(), name, body.Definition)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": row})
}

// GET /api/v1/screens/today — hits from the most recent screening day.
func (h *handlers) todayScreens(c *gin.Context) {
	hits, err := h.Store.LatestScreenResults(c.Request.Context())
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, hits)
}
