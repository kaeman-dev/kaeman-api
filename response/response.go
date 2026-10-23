package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response[T any] struct {
	// Msg 响应消息，通用消息为 HTTP 状态文本，业务消息为具体语义
	Msg string `json:"msg"`
	// Data 响应数据，失败或无数据时省略
	Data T `json:"data,omitempty"`
}

func Write(c *gin.Context, status int, msg string, data any) error {
	if msg == "" {
		msg = http.StatusText(status)
	}
	switch {
	case status == http.StatusUnauthorized:
		c.Header("WWW-Authenticate", "Bearer")
	case status >= 400:
		c.Header("Cache-Control", "no-store")
	}

	c.JSON(status, Response[any]{Msg: msg, Data: data})
	return nil
}
