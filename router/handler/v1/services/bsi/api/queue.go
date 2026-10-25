package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kaeman-dev/kaeman-api/response"
	"github.com/kaeman-dev/kaeman-api/server"
	"github.com/kaeman-dev/kaeman-api/utils"
)

type Handler struct{ Services *server.Services }

type BSIType string

const (
	BSITypeChat   BSIType = "chat"
	BSITypeSplash BSIType = "splash"
)

type BSIChatRequest struct {
	// Content 消息内容，不能为空白
	Content string `json:"content" binding:"required" example:"Hello kaeman!"`
}

type BSISplashRequest struct {
	// Splasher 发起 splash 的玩家名
	Splasher string `json:"splasher" example:"Notch"`
	// Time splash 时间，必填
	Time time.Time `json:"time" binding:"required" example:"2026-09-29T12:00:00Z"`
	// Lobby 大厅名称，必填
	Lobby string `json:"lobby" binding:"required" example:"lobby-1"`
	// LobbyID 大厅编号，从 1 开始，必填
	LobbyID int64 `json:"lobbyID" binding:"required" example:"1"`
	// Location splash 坐标位置
	Location string `json:"location,omitempty" example:"256 64 -128"`
	// ServerID splash 所在服务器 ID
	ServerID string `json:"serverID,omitempty" example:"lobby"`
	// Note 附加备注
	Note string `json:"note,omitempty" example:"every 10 mins"`
}

type BSIMessage struct {
	// ID 消息 ID，服务端签发的小写无横杠 UUID
	ID string `json:"id,omitempty" example:"0f8f8c8e5d3c4b2a9e7d6c5b4a3f2e1d"`
	// Type 消息类型
	Type BSIType `json:"type"`
	// Data 消息体
	Data any `json:"data"`
}

type BSIEnvelope struct {
	// Type 消息类型，chat 或 splash
	Type BSIType `json:"type" binding:"required"`
	// Data 消息体，type 为 chat 时为 BSIChatRequest，为 splash 时为 BSISplashRequest
	Data json.RawMessage `json:"data" example:"{\"content\":\"Hello kaeman!\"}"`
}

// Send godoc
// @Summary 向 BSI 广播一条消息
// @Description 需要 splasher:queue 权限的 public api key，且 Authorization 中的 JWT 需同时携带 public API 与 Minecraft 会话 claims（audience 含 public-api 和 minecraft-session，data 同时包含 uid/permissions/ratelimit 与 minecraft）。消息经校验后分配无横杠 UUID 作为 id，实时广播给所有 /gateway WebSocket 订阅者。成功返回 202，data 为 null。
// @Tags services
// @Accept json
// @Security BearerAuth
// @Param body body BSIEnvelope true "type 为 chat 时 data 为 BSIChatRequest，type 为 splash 时 data 为 BSISplashRequest"
// @Success 202 {object} response.Response[any]
// @Failure 400 {object} response.Response[any] "Invalid request body / Message must not be blank / Valid time, lobby and lobbyID are required / Unknown message kind"
// @Failure 401 {object} response.Response[any] "Key required / Invalid public api key / Required permission missing / Invalid minecraft key / Minecraft identity not bound to this public api key"
// @Failure 429 {object} response.Response[any] "Too Many Requests"
// @Failure 500 {object} response.Response[any]
// @Router /v1/services/bsi/queue [post]
func (h Handler) Send(c *gin.Context) error {
	var env BSIEnvelope
	if err := utils.DecodeJSON(http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10), &env); err != nil {
		return response.Write(c, http.StatusBadRequest, "Invalid request body", nil)
	}

	msg := BSIMessage{Type: env.Type}
	switch env.Type {
	case BSITypeChat:
		var req BSIChatRequest
		if err := utils.DecodeJSON(bytes.NewReader(env.Data), &req); err != nil {
			return response.Write(c, http.StatusBadRequest, "Invalid request body", nil)
		}
		if strings.TrimSpace(req.Content) == "" {
			return response.Write(c, http.StatusBadRequest, "Message must not be blank", nil)
		}
		msg.Data = req
	case BSITypeSplash:
		var req BSISplashRequest
		if err := utils.DecodeJSON(bytes.NewReader(env.Data), &req); err != nil {
			return response.Write(c, http.StatusBadRequest, "Invalid request body", nil)
		}
		if req.Time.IsZero() || strings.TrimSpace(req.Lobby) == "" || req.LobbyID < 1 {
			return response.Write(c, http.StatusBadRequest, "Valid time, lobby and lobbyID are required", nil)
		}
		msg.Data = req
	default:
		return response.Write(c, http.StatusBadRequest, "Unknown message kind", nil)
	}
	msg.ID = utils.NewUUID()
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	h.Services.BSI.Broadcast(data)
	return response.Write(c, http.StatusAccepted, "ok", nil)
}
