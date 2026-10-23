package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Sn0wo2/caelum"
	"github.com/eko/gocache/lib/v4/cache"
	go_cache_store "github.com/eko/gocache/store/go_cache/v4"
	redis_store "github.com/eko/gocache/store/redis/v4"
	"github.com/kaeman-dev/kaeman-public-api/config"
	_ "github.com/kaeman-dev/kaeman-public-api/docs"
	"github.com/kaeman-dev/kaeman-public-api/gateway"
	"github.com/kaeman-dev/kaeman-public-api/jwt"
	"github.com/kaeman-dev/kaeman-public-api/minecraft"
	"github.com/kaeman-dev/kaeman-public-api/router"
	"github.com/kaeman-dev/kaeman-public-api/server"
	"github.com/kaeman-dev/kaeman-public-api/storage"
	"github.com/kaeman-dev/kaeman-public-api/utils"
	gocache_lib "github.com/patrickmn/go-cache"
	"github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	limiter_memstore "github.com/ulule/limiter/v3/drivers/store/memory"
	limiter_redisstore "github.com/ulule/limiter/v3/drivers/store/redis"
)

//go:generate swag init -g cli/main.go --parseDependency --parseDepth 2

// @title kaeman-public-api
// @version 1.0
// @description Kaeman public API。除 /v1/auth/minecraft/challenge 与 /v1/services/bsi/gateway 外，所有端点都经过每分钟 N 次（N 默认 60，public API key 可通过 ratelimit claim 自定义）的限流，每次响应携带 RateLimit-Limit / RateLimit-Remaining / RateLimit-Reset（IETF draft-ietf-httpapi-ratelimit-headers）与 X-RateLimit-Limit / X-RateLimit-Remaining / X-RateLimit-Reset 响应头，超限返回 429 并携带 Retry-After；意外错误统一返回 500 信封，data 中含 traceID。
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Bearer JWT。public API key 的 audience 为 public-api，携带 uid/permissions/ratelimit claims；Minecraft 会话 token 的 audience 为 minecraft-session，携带 minecraft identity claims。
func main() {
	configPath := flag.String("config", "./data/config.toml", "path to the TOML config file")
	flag.Parse()

	style := caelum.DefaultStyle()
	style.Icons = caelum.IconsNerd

	log := caelum.Init(caelum.Config{
		Targets: []caelum.Target{
			{
				Style: &style,
			},
		},
	})

	utils.RedirectGIN(log.Logger)
	slog.SetDefault(log.Logger)

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Error("load config", "error", err)
		os.Exit(1)
	}

	db, err := storage.Open(cfg.Database.DSN, log.Logger)
	if err != nil {
		log.Error("initialize database", "error", err)
		os.Exit(1)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	deps := &server.Services{
		DB:        db,
		Log:       log.Logger,
		Tokens:    &jwt.Token{Secret: []byte(cfg.Auth.JWTSecret)},
		Minecraft: &minecraft.Client{HTTP: http.DefaultClient, SessionBaseURL: cfg.Minecraft.MojangSessionBaseURL},
		BSI:       &gateway.Hub{},
	}

	if cfg.Redis.Addr != "" {
		network := "tcp"
		addr := cfg.Redis.Addr
		if path, ok := strings.CutPrefix(addr, "unix://"); ok {
			network, addr = "unix", path
		}
		client := redis.NewClient(&redis.Options{Network: network, Addr: addr, Password: cfg.Redis.Password})
		if err := client.Ping(context.Background()).Err(); err != nil {
			client.Close()
			log.Error("connect to Redis", "error", err)
			os.Exit(1)
		}
		deps.Cache = cache.New[string](redis_store.NewRedis(client))
		deps.RateLimit, err = limiter_redisstore.NewStoreWithOptions(client, limiter.StoreOptions{})
		if err != nil {
			log.Error("initialize ratelimiter", "error", err)
			os.Exit(1)
		}
	} else {
		deps.Cache = cache.New[string](go_cache_store.NewGoCache(gocache_lib.New(5*time.Minute, 10*time.Minute)))
		deps.RateLimit = limiter_memstore.NewStore()
	}

	httpServer := &http.Server{Addr: cfg.Server.ListenAddr, Handler: router.NewRouter(deps), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("server listening", "address", cfg.Server.ListenAddr)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		log.Error("serve HTTP", "error", err)
		os.Exit(1)
	case <-ctx.Done():
		deps.BSI.Close()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Error("shutdown HTTP", "error", err)
			httpServer.Close()
		}
	}
}
