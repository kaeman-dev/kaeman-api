package public

import (
	"net/http"

	auth "github.com/kaeman-dev/kaeman-api/public/handler/v1/auth/api"
	minecraft "github.com/kaeman-dev/kaeman-api/public/handler/v1/auth/minecraft/api"
	bsi "github.com/kaeman-dev/kaeman-api/public/handler/v1/services/bsi/api"
	"github.com/kaeman-dev/kaeman-api/public/middleware"
	"github.com/kaeman-dev/kaeman-api/shared"
	"github.com/kaeman-dev/kaeman-api/shared/permission"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
)

func Register(e *echo.Echo, deps *shared.Services) {
	routes := e.Group("", echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins:  []string{"*"},
		AllowMethods:  []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowHeaders:  []string{"Authorization", "Content-Type", "Public-API-Key"},
		ExposeHeaders: []string{"Trace-ID", "Retry-After", "WWW-Authenticate"},
		MaxAge:        600,
	}))
	routes.GET("/swagger/*", func(c echo.Context) error {
		if !e.Debug {
			return echo.ErrNotFound
		}
		return echoSwagger.WrapHandler(c)
	})
	authRoutes := routes.Group("/v1/auth")
	authRoutes.GET("/check", auth.Handler{Services: deps}.Check,
		middleware.PublicAPIKey(deps, 0), middleware.RateLimit(deps.RateLimit))
	minecraftRoutes := authRoutes.Group("/minecraft")
	minecraftHandler := minecraft.Handler{Services: deps}
	minecraftRoutes.GET("/challenge", minecraftHandler.Challenge, middleware.RateLimit(deps.RateLimit))
	minecraftRoutes.POST("/verify", minecraftHandler.Verify, middleware.RateLimit(deps.RateLimit))
	bsiRoutes := routes.Group("/v1/services/bsi")
	bsiRoutes.POST("/queue", bsi.Handler{Services: deps}.Send,
		middleware.PublicAPIKey(deps, permission.SplasherQueue), middleware.MinecraftToken(deps), middleware.RateLimit(deps.RateLimit))
	bsiRoutes.GET("/gateway", bsi.Handler{Services: deps}.Listen)
}
