package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GET /api/v1/watchlist
func (h *handlers) listWatchlist(c *gin.Context) {
	items, err := h.Store.Watchlist(c.Request.Context(), userID(c))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, items)
}

// POST /api/v1/watchlist  {"code": "2330"}
func (h *handlers) addWatch(c *gin.Context) {
	var body struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Store.AddWatch(c.Request.Context(), userID(c), body.Code); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

// DELETE /api/v1/watchlist/:code
func (h *handlers) removeWatch(c *gin.Context) {
	if err := h.Store.RemoveWatch(c.Request.Context(), userID(c), c.Param("code")); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

// GET /api/v1/holdings
func (h *handlers) listHoldings(c *gin.Context) {
	items, err := h.Store.Holdings(c.Request.Context(), userID(c))
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	ok(c, items)
}

// PUT /api/v1/holdings/:code  {"shares": 1000, "avg_cost": 580.5}
func (h *handlers) saveHolding(c *gin.Context) {
	var body struct {
		Shares  int64   `json:"shares" binding:"gte=0"`
		AvgCost float64 `json:"avg_cost" binding:"gte=0"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Store.SaveHolding(c.Request.Context(), userID(c), c.Param("code"), body.Shares, body.AvgCost); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

// DELETE /api/v1/holdings/:code
func (h *handlers) deleteHolding(c *gin.Context) {
	if err := h.Store.DeleteHolding(c.Request.Context(), userID(c), c.Param("code")); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}
