package rpc

import (
	"context"
	"errors"
	"mime"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/kaeman-dev/kaeman-api/shared"
	"github.com/kaeman-dev/kaeman-api/shared/jwt"
	"github.com/labstack/echo/v4"
)

const PathPrefix = "/internal/rpc/v1"

type Service func(...connect.HandlerOption) (string, http.Handler)

func Register(e *echo.Echo, deps *shared.Services, services ...Service) {
	options := []connect.HandlerOption{
		connect.WithInterceptors(authentication(deps)),
		connect.WithRequireConnectProtocolHeader(),
		connect.WithReadMaxBytes(4 << 20),
		connect.WithSendMaxBytes(4 << 20),
	}

	for _, service := range services {
		if service == nil {
			panic("invalid ConnectRPC service registration")
		}

		path, handler := service(options...)
		if !strings.HasPrefix(path, "/") || !strings.HasSuffix(path, "/") || path == "/" || strings.ContainsAny(path, "*:?#") || handler == nil {
			panic("invalid ConnectRPC service registration")
		}

		if strings.Contains(path, "//") || strings.Contains(path, "/../") || strings.Contains(path, "/./") {
			panic("invalid ConnectRPC service registration")
		}

		handler = http.StripPrefix(PathPrefix, handler)
		e.POST(PathPrefix+path+"*", echo.WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || mediaType != "application/json" {
				w.WriteHeader(http.StatusUnsupportedMediaType)

				return
			}

			handler.ServeHTTP(w, r)
		})))
	}
}

func authentication(deps *shared.Services) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, request connect.AnyRequest) (connect.AnyResponse, error) {
			parts := strings.Fields(request.Header().Get("Authorization"))
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid API key"))
			}

			claims, err := deps.Tokens.ParseAPIKey(parts[1])
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid API key"))
			}

			if !claims.Data.HasRPC() {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.New("RPC access denied"))
			}

			ctx, cancel := context.WithTimeout(jwt.WithAPIKey(ctx, claims), 15*time.Second)
			defer cancel()

			started := time.Now()
			response, err := next(ctx, request)

			code := "ok"
			if err != nil {
				code = connect.CodeOf(err).String()
			}

			deps.Log.InfoContext(ctx, "rpc request", "sub", claims.Subject, "jti", claims.ID,
				"procedure", request.Spec().Procedure, "code", code, "duration", time.Since(started))

			return response, err
		}
	})
}
