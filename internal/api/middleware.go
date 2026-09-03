package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIKeyAuth requires header "X-API-Key" to match key on every request
// in the group it's attached to. Passing an empty key disables the
// check entirely (handy for local development) — set app.api_key /
// APP_API_KEY before exposing the service beyond localhost.
func APIKeyAuth(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if key == "" {
			c.Next()
			return
		}
		if c.GetHeader("X-API-Key") != key {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing api key"})
			return
		}
		c.Next()
	}
}
