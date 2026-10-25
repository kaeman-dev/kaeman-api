package shared

import (
	"log/slog"

	"github.com/eko/gocache/lib/v4/cache"
	"github.com/kaeman-dev/kaeman-api/shared/gateway"
	"github.com/kaeman-dev/kaeman-api/shared/jwt"
	"github.com/kaeman-dev/kaeman-api/shared/minecraft"
	"github.com/ulule/limiter/v3"
	"gorm.io/gorm"
)

type Services struct {
	DB        *gorm.DB
	Log       *slog.Logger
	Tokens    *jwt.Token
	Minecraft *minecraft.Client
	Cache     *cache.Cache[string]
	RateLimit limiter.Store
	BSI       *gateway.Hub
}
