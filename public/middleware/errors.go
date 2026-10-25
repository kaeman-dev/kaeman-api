package middleware

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/kaeman-dev/kaeman-api/public/response"
	"github.com/labstack/echo/v4"
)

func Errors(log *slog.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}
		status := http.StatusInternalServerError
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			status = httpErr.Code
		}
		traceID := c.Response().Header().Get("Trace-ID")
		if status >= http.StatusInternalServerError {
			log.ErrorContext(c.Request().Context(), "unexpected error", "trace_id", traceID,
				"method", c.Request().Method, "path", c.Request().URL.Path, "status", status, "error", err)
		}
		msg := ""
		var data any
		if status >= http.StatusInternalServerError {
			msg = "oops, something went wrong"
			data = echo.Map{"traceID": traceID}
		}
		if err := response.Write(c, status, msg, data); err != nil {
			log.ErrorContext(c.Request().Context(), "write error response", "trace_id", traceID, "error", err)
		}
	}
}
