package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kaeman-dev/kaeman-public-api/response"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Expose-Headers", "Trace-ID, Retry-After, WWW-Authenticate")
		if c.Request.Method != http.MethodOptions || c.GetHeader("Access-Control-Request-Method") == "" {
			return
		}
		if method := c.GetHeader("Access-Control-Request-Method"); method != http.MethodGet && method != http.MethodPost {
			_ = response.Write(c, http.StatusMethodNotAllowed, "", nil)
			c.Abort()
			return
		}
		for _, header := range strings.Split(c.GetHeader("Access-Control-Request-Headers"), ",") {
			switch strings.ToLower(strings.TrimSpace(header)) {
			case "", "authorization", "content-type", "public-api-key":
			default:
				_ = response.Write(c, http.StatusBadRequest, "", nil)
				c.Abort()
				return
			}
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Public-API-Key")
		c.Header("Access-Control-Max-Age", "600")
		c.AbortWithStatus(http.StatusNoContent)
	}
}
