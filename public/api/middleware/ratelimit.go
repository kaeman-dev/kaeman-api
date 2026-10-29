package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/kaeman-dev/kaeman-api/public/api/response"
	"github.com/kaeman-dev/kaeman-api/shared/jwt"
	"github.com/labstack/echo/v4"
	"github.com/ulule/limiter/v3"
)

func RateLimit(store limiter.Store) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			count := int64(60)

			key := "kaeman:ratelimit:" + c.Path() + ":ip:" + c.RealIP()
			if value := c.Get("publicAPIKey"); value != nil {
				if claims, ok := value.(*jwt.APIKeyClaims); ok {
					if claims.Data.HasRPC() {
						return next(c)
					}

					key = "kaeman:ratelimit:" + c.Path() + ":key:" + claims.ID + ":" + c.RealIP()
					if claims.Data.Public != nil && claims.Data.Public.API != nil && claims.Data.Public.API.Ratelimit != nil && *claims.Data.Public.API.Ratelimit > 0 {
						count = int64(*claims.Data.Public.API.Ratelimit)
					}
				}
			} else if value := c.Get("minecraftKey"); value != nil {
				if claims, ok := value.(*jwt.Claims[jwt.MinecraftClaimsData]); ok {
					key = "kaeman:ratelimit:" + c.Path() + ":key:" + claims.ID + ":" + c.RealIP()
				}
			}

			state, err := store.Get(c.Request().Context(), key, limiter.Rate{Period: time.Minute, Limit: count})
			if err != nil {
				return err
			}

			resetIn := max(0, state.Reset-time.Now().Unix())
			limit := strconv.FormatInt(state.Limit, 10)
			remaining := strconv.FormatInt(state.Remaining, 10)

			c.Response().Header().Set("Ratelimit-Limit", limit)
			c.Response().Header().Set("Ratelimit-Remaining", remaining)
			c.Response().Header().Set("Ratelimit-Reset", strconv.FormatInt(resetIn, 10))
			c.Response().Header().Set("X-Ratelimit-Limit", limit)
			c.Response().Header().Set("X-Ratelimit-Remaining", remaining)
			c.Response().Header().Set("X-Ratelimit-Reset", strconv.FormatInt(state.Reset, 10))

			if state.Reached {
				c.Response().Header().Set("Retry-After", strconv.FormatInt(resetIn, 10))

				return response.Write(c, http.StatusTooManyRequests, "", nil)
			}

			return next(c)
		}
	}
}
