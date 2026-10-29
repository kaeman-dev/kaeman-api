package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
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
	_ "github.com/kaeman-dev/kaeman-api/docs"
	"github.com/kaeman-dev/kaeman-api/server"
	"github.com/kaeman-dev/kaeman-api/shared"
	"github.com/kaeman-dev/kaeman-api/shared/config"
	"github.com/kaeman-dev/kaeman-api/shared/gateway"
	"github.com/kaeman-dev/kaeman-api/shared/jwt"
	"github.com/kaeman-dev/kaeman-api/shared/minecraft"
	"github.com/kaeman-dev/kaeman-api/shared/storage"
	gocache_lib "github.com/patrickmn/go-cache"
	"github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	limiter_memstore "github.com/ulule/limiter/v3/drivers/store/memory"
	limiter_redisstore "github.com/ulule/limiter/v3/drivers/store/redis"
)

//go:generate swag init -g cli/main.go --parseDependency --parseDepth 2

// @title kaeman-api
// @version 1.0
// @description Kaeman public API。除 /v1/auth/minecraft/challenge 与 /v1/services/bsi/gateway 外，所有端点都经过每分钟 N 次（N 默认 60，public API key 可通过 ratelimit claim 自定义）的限流，每次响应携带 RateLimit-Limit / RateLimit-Remaining / RateLimit-Reset（IETF draft-ietf-httpapi-ratelimit-headers）与 X-RateLimit-Limit / X-RateLimit-Remaining / X-RateLimit-Reset 响应头，超限返回 429 并携带 Retry-After；意外错误统一返回 500 信封，data 中含 traceID。
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Bearer JWT。public API key 的 audience 为 public-api，携带 uid/permissions/ratelimit claims；Minecraft 会话 token 的 audience 为 minecraft-session，携带 minecraft identity claims。
func main() {
	os.Exit(run())
}

func run() int {
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

	slog.SetDefault(log.Logger)

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Error("load config", "error", err)

		return 1
	}

	db, err := storage.Open(cfg.Database.DSN, log.Logger)
	if err != nil {
		log.Error("initialize database", "error", err)

		return 1
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Error("get database connection", "error", err)

		return 1
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.Error("close database connection", "error", err)
		}
	}()

	deps := &shared.Services{
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
		defer func() {
			if err := client.Close(); err != nil {
				log.Error("close Redis", "error", err)
			}
		}()

		if err := client.Ping(context.Background()).Err(); err != nil {
			log.Error("connect to Redis", "error", err)

			return 1
		}

		deps.Cache = cache.New[string](redis_store.NewRedis(client))

		deps.RateLimit, err = limiter_redisstore.NewStoreWithOptions(client, limiter.StoreOptions{})
		if err != nil {
			log.Error("initialize ratelimiter", "error", err)

			return 1
		}
	} else {
		deps.Cache = cache.New[string](go_cache_store.NewGoCache(gocache_lib.New(5*time.Minute, 10*time.Minute)))
		deps.RateLimit = limiter_memstore.NewStore()
	}

	e := server.NewRouter(deps)
	e.Debug = cfg.Server.Debug
	httpServer := &http.Server{Addr: cfg.Server.ListenAddr, Handler: e, ErrorLog: slog.NewLogLogger(log.Handler(), slog.LevelError), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	httpServer.Protocols = new(http.Protocols)
	httpServer.Protocols.SetHTTP1(true)
	httpServer.Protocols.SetUnencryptedHTTP2(true)

	network, addr := "tcp", cfg.Server.ListenAddr
	if path, ok := strings.CutPrefix(addr, "unix://"); ok {
		network, addr = "unix", path
	}

	listener, err := net.Listen(network, addr)
	if err != nil {
		log.Error("listen HTTP", "address", cfg.Server.ListenAddr, "error", err)

		return 1
	}
	defer func() {
		if err := listener.Close(); err != nil {
			log.Error("close HTTP listener", "error", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("server listening", "address", cfg.Server.ListenAddr)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- httpServer.Serve(listener)
	}()

	select {
	case err := <-serveErr:
		log.Error("serve HTTP", "error", err)

		return 1
	case <-ctx.Done():
		deps.BSI.Close()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Error("shutdown HTTP", "error", err)

			if err := httpServer.Close(); err != nil {
				log.Error("close HTTP server", "error", err)
			}
		}
	}

	return 0
}
