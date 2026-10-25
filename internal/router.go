package internal

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

type Service struct {
	Path    string
	Handler http.Handler
}

func Register(e *echo.Echo, services ...Service) {
	for _, service := range services {
		if !strings.HasPrefix(service.Path, "/") || !strings.HasSuffix(service.Path, "/") || service.Path == "/" || strings.ContainsAny(service.Path, "*:?#") || service.Handler == nil {
			panic("invalid ConnectRPC service registration")
		}
		e.Any(service.Path+"*", echo.WrapHandler(service.Handler))
	}
}
