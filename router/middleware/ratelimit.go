package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kaeman-dev/kaeman-api/jwt"
	"github.com/kaeman-dev/kaeman-api/response"
	"github.com/ulule/limiter/v3"
)

func RateLimit(store limiter.Store) func(*gin.Context) error {
	return func(c *gin.Context) error {
		count := int64(60)
		key := "kaeman:ratelimit:" + c.FullPath() + ":ip:" + c.ClientIP()
		if value, ok := c.Get("publicAPIKey"); ok {
			if claims, ok := value.(*jwt.Claims[jwt.PublicAPIClaimsData]); ok {
				key = "kaeman:ratelimit:" + c.FullPath() + ":key:" + claims.ID + ":" + c.ClientIP()
				if claims.Data.Ratelimit != nil && *claims.Data.Ratelimit > 0 {
					count = int64(*claims.Data.Ratelimit)
				}
			}
		} else if value, ok := c.Get("minecraftKey"); ok {
			if claims, ok := value.(*jwt.Claims[jwt.MinecraftClaimsData]); ok {
				key = "kaeman:ratelimit:" + c.FullPath() + ":key:" + claims.ID + ":" + c.ClientIP()
			}
		}
		state, err := store.Get(c.Request.Context(), key, limiter.Rate{Period: time.Minute, Limit: count})
		if err != nil {
			return err
		}
		resetIn := max(0, state.Reset-time.Now().Unix())
		limit := strconv.FormatInt(state.Limit, 10)
		remaining := strconv.FormatInt(state.Remaining, 10)
		c.Header("RateLimit-Limit", limit)
		c.Header("RateLimit-Remaining", remaining)
		c.Header("RateLimit-Reset", strconv.FormatInt(resetIn, 10))
		c.Header("X-RateLimit-Limit", limit)
		c.Header("X-RateLimit-Remaining", remaining)
		c.Header("X-RateLimit-Reset", strconv.FormatInt(state.Reset, 10))
		if state.Reached {
			c.Header("Retry-After", strconv.FormatInt(resetIn, 10))
			return response.Write(c, http.StatusTooManyRequests, "", nil)
		}
		return nil
	}
}
