package router

import (
	"fmt"
	"io"
	"net/http"
	"runtime/debug"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/kaeman-dev/kaeman-public-api/permission"
	"github.com/kaeman-dev/kaeman-public-api/response"
	auth "github.com/kaeman-dev/kaeman-public-api/router/handler/v1/auth/api"
	minecraft "github.com/kaeman-dev/kaeman-public-api/router/handler/v1/auth/minecraft/api"
	bsi "github.com/kaeman-dev/kaeman-public-api/router/handler/v1/services/bsi/api"
	"github.com/kaeman-dev/kaeman-public-api/router/middleware"
	"github.com/kaeman-dev/kaeman-public-api/server"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func NewRouter(deps *server.Services) *gin.Engine {
	e := gin.New()
	e.Use(requestid.New(requestid.WithCustomHeaderStrKey("Trace-ID")), middleware.Errors(deps.Log),
		gin.RecoveryWithWriter(io.Discard, func(c *gin.Context, err any) {
			_ = c.Error(fmt.Errorf("panic: %v\n%s", err, debug.Stack()))
		}), middleware.CORS())

	_ = e.SetTrustedProxies(nil)

	e.RedirectTrailingSlash = false
	e.NoRoute(func(c *gin.Context) {
		_ = response.Write(c, http.StatusNotFound, "", nil)
	})

	e.HandleMethodNotAllowed = true
	e.NoMethod(func(c *gin.Context) {
		_ = response.Write(c, http.StatusMethodNotAllowed, "", nil)
	})

	if gin.Mode() == gin.DebugMode {
		e.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	authRoutes := e.Group("/v1/auth")
	authRoutes.GET("/check", middleware.Chain(auth.Handler{Services: deps}.Check,
		middleware.PublicAPIKey(deps, 0), middleware.RateLimit(deps.RateLimit)))
	minecraftRoutes := authRoutes.Group("/minecraft")
	minecraftHandler := minecraft.Handler{Services: deps}
	minecraftRoutes.GET("/challenge", middleware.Chain(minecraftHandler.Challenge, middleware.RateLimit(deps.RateLimit)))
	minecraftRoutes.POST("/verify", middleware.Chain(minecraftHandler.Verify, middleware.RateLimit(deps.RateLimit)))
	bsiRoutes := e.Group("/v1/services/bsi")
	bsiRoutes.POST("/queue", middleware.Chain(bsi.Handler{Services: deps}.Send,
		middleware.PublicAPIKey(deps, permission.SplasherQueue), middleware.MinecraftToken(deps), middleware.RateLimit(deps.RateLimit)))
	bsiRoutes.GET("/gateway", deps.BSI.Listen)

	return e
}
