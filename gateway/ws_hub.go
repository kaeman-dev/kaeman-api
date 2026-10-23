package gateway

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/kaeman-dev/kaeman-public-api/response"
)

type Hub struct {
	conns  sync.Map
	closed atomic.Bool
}

type connection struct {
	gin.ResponseWriter
	handshakeStatus int

	conn *websocket.Conn
	out  chan []byte
}

// Listen godoc
// @Summary BSI 实时消息 WebSocket 网关
// @Description 建立 WebSocket 连接后作为只读订阅者接收服务端推送的 BSIMessage 文本帧（JSON）：type 为 chat 时 data 为 BSIChatRequest，type 为 splash 时 data 为 BSISplashRequest。客户端发送会被 CloseRead 忽略，服务端每 30 秒 ping 一次，空闲写超时 5 秒。OpenAPI 不支持 WebSocket，仅作说明用。
// @Tags services
// @Success 101 {string} string "Switching Protocols (WebSocket upgrade)"
// @Failure 503 {object} response.Response[any] "Service Unavailable（hub 已关闭）"
// @Router /v1/services/bsi/gateway [get]
func (h *Hub) Listen(c *gin.Context) {
	if h.closed.Load() {
		_ = response.Write(c, http.StatusServiceUnavailable, "", nil)
		return
	}
	cn := &connection{
		ResponseWriter: c.Writer,
		out:            make(chan []byte, 32),
	}
	conn, err := websocket.Accept(cn, c.Request, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		_ = c.Error(err)
		if cn.handshakeStatus != 0 {
			_ = response.Write(c, cn.handshakeStatus, "", nil)
		}
		return
	}
	conn.SetReadLimit(1024)
	ctx := conn.CloseRead(c.Request.Context())
	cn.conn = conn
	h.conns.Store(cn, struct{}{})
	defer func() {
		h.conns.Delete(cn)
		_ = conn.CloseNow()
	}()
	if h.closed.Load() {
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-cn.out:
			writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Write(writeCtx, websocket.MessageText, data)
			cancel()
			if err != nil {
				return
			}
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

func (h *Hub) Broadcast(data []byte) {
	if h.closed.Load() {
		return
	}
	h.conns.Range(func(key, _ any) bool {
		c := key.(*connection)
		select {
		case c.out <- data:
		default:
			_ = c.conn.CloseNow()
		}
		return true
	})
}

func (c *connection) WriteHeader(status int) {
	if status >= 400 {
		c.handshakeStatus = status
		c.Header().Del("Content-Type")
		return
	}
	c.ResponseWriter.WriteHeader(status)
}

func (c *connection) Write(data []byte) (int, error) {
	if c.handshakeStatus != 0 {
		return len(data), nil
	}
	return c.ResponseWriter.Write(data)
}

func (c *connection) WriteString(data string) (int, error) {
	if c.handshakeStatus != 0 {
		return len(data), nil
	}
	return c.ResponseWriter.WriteString(data)
}

func (h *Hub) Close() {
	if !h.closed.CompareAndSwap(false, true) {
		return
	}
	h.conns.Range(func(key, _ any) bool {
		_ = key.(*connection).conn.CloseNow()
		return true
	})
}
