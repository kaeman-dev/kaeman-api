package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/kaeman-dev/kaeman-api/response"
)

func Chain(h func(*gin.Context) error, mws ...func(*gin.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, mw := range mws {
			err := mw(c)
			if err == nil && !c.Writer.Written() {
				continue
			}
			if err != nil {
				_ = c.Error(err)
			}
			c.Abort()
			return
		}
		if err := h(c); err != nil {
			_ = c.Error(err)
		}
	}
}

func Errors(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		defer func() {
			if len(c.Errors) > 0 {
				causes := make([]error, 0, len(c.Errors))
				for _, entry := range c.Errors {
					causes = append(causes, entry.Err)
				}
				log.ErrorContext(c.Request.Context(), "unexpected error", "trace_id", requestid.Get(c), "method", c.Request.Method, "path", c.Request.URL.Path, "status", c.Writer.Status(), "stack", string(debug.Stack()), "error", errors.Join(causes...))
			}
			log.InfoContext(c.Request.Context(), "http request", "trace_id", requestid.Get(c), "method", c.Request.Method, "path", c.Request.URL.Path, "status", c.Writer.Status(), "duration", time.Since(start))
		}()
		c.Next()

		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}

		_ = response.Write(c, http.StatusInternalServerError, "oops, something went wrong", gin.H{"traceID": requestid.Get(c)})
	}
}
