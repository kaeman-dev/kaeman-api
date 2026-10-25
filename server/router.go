package server

import (
	"fmt"
	"github.com/kaeman-dev/kaeman-api/internal/rpc"
	"github.com/kaeman-dev/kaeman-api/public/api"
	"github.com/kaeman-dev/kaeman-api/public/api/middleware"
	"github.com/kaeman-dev/kaeman-api/public/api/response"
	"github.com/kaeman-dev/kaeman-api/shared"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

func NewRouter(deps *shared.Services, services ...rpc.Service) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.IPExtractor = echo.ExtractIPDirect()
	e.JSONSerializer = response.JSONSerializer{}
	e.HTTPErrorHandler = middleware.Errors(deps.Log)
	e.Use(echomiddleware.RequestIDWithConfig(echomiddleware.RequestIDConfig{TargetHeader: "Trace-ID"}),
		echomiddleware.RequestLoggerWithConfig(echomiddleware.RequestLoggerConfig{
			HandleError: true,
			LogMethod:   true,
			LogURIPath:  true,
			LogStatus:   true,
			LogLatency:  true,
			LogValuesFunc: func(c echo.Context, v echomiddleware.RequestLoggerValues) error {
				deps.Log.InfoContext(c.Request().Context(), "http request", "trace_id", c.Response().Header().Get("Trace-ID"),
					"method", v.Method, "path", v.URIPath, "status", v.Status, "duration", v.Latency)
				return nil
			},
		}),
		echomiddleware.RecoverWithConfig(echomiddleware.RecoverConfig{
			DisableStackAll:     true,
			DisableErrorHandler: true,
			LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
				return fmt.Errorf("panic: %v\n%s", err, stack)
			},
		}))

	api.Register(e, deps)
	rpc.Register(e, deps, services...)

	return e
}
