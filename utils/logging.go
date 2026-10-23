package utils

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"
)

func RedirectGIN(l *slog.Logger) {
	gin.DebugPrintRouteFunc = func(method, path, handler string, handlers int) {
		l.Info("route registered", "method", method, "path", path, "handler", handler, "handlers", handlers)
	}
	gin.DebugPrintFunc = func(format string, values ...any) {
		message := strings.Join(strings.Fields(fmt.Sprintf(format, values...)), " ")
		level := slog.LevelInfo
		// gin 的 debugPrint 只会在前面加一个 [WARNING]
		// debug.go:83/87/93/101、gin.go:544、response_writer.go:70
		// [ERROR] 由 debugPrintError 直接写 DefaultErrorWriter（debug.go:112），不经过这里。
		if tag, rest, ok := strings.Cut(message, "]"); ok && strings.HasPrefix(tag, "[") {
			if strings.TrimPrefix(strings.ToUpper(tag), "[") == "WARNING" {
				level = slog.LevelWarn
			}
			message = strings.TrimSpace(rest)
		}
		l.Log(context.Background(), level, message)
	}
	gin.DefaultWriter = slog.NewLogLogger(l.Handler(), slog.LevelInfo).Writer()
	gin.DefaultErrorWriter = slog.NewLogLogger(l.Handler(), slog.LevelError).Writer()
}
