package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const ctxUserID = "userID"

// requireUser identifies the caller from the X-User-ID header.
// Development only; TODO: replace with real authentication (JWT) in Phase 5.
func requireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.GetHeader("X-User-ID"), 10, 64)
		if err != nil || id <= 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Set(ctxUserID, id)
		c.Next()
	}
}

func userID(c *gin.Context) int64 { return c.GetInt64(ctxUserID) }

// cors allows browser clients (e.g. Flutter web during development).
func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Content-Type, X-User-ID, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func ok(c *gin.Context, data any) { c.JSON(http.StatusOK, gin.H{"data": data}) }

func fail(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, gin.H{"error": msg})
}

func paramID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		fail(c, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}
