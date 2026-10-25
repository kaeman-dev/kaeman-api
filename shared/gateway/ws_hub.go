package gateway

import (
	"context"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

type Hub struct {
	conns  sync.Map
	closed atomic.Bool
}

type connection struct {
	http.ResponseWriter
	handshakeStatus int

	conn *websocket.Conn
	out  chan []byte
}

func (h *Hub) Listen(w http.ResponseWriter, r *http.Request) (int, error) {
	if h.closed.Load() {
		return http.StatusServiceUnavailable, nil
	}
	cn := &connection{
		ResponseWriter: w,
		out:            make(chan []byte, 32),
	}
	conn, err := websocket.Accept(cn, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		if cn.handshakeStatus != 0 {
			return cn.handshakeStatus, err
		}
		return 0, err
	}
	conn.SetReadLimit(1024)
	ctx := conn.CloseRead(r.Context())
	cn.conn = conn
	h.conns.Store(cn, struct{}{})
	defer func() {
		h.conns.Delete(cn)
		_ = conn.CloseNow()
	}()
	if h.closed.Load() {
		return 0, nil
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return 0, nil
		case data := <-cn.out:
			writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Write(writeCtx, websocket.MessageText, data)
			cancel()
			if err != nil {
				return 0, nil
			}
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				return 0, nil
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
	return io.WriteString(c.ResponseWriter, data)
}

func (c *connection) Unwrap() http.ResponseWriter {
	return c.ResponseWriter
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
