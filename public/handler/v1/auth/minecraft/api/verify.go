package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/eko/gocache/lib/v4/store"
	"github.com/kaeman-dev/kaeman-api/public/response"
	"github.com/kaeman-dev/kaeman-api/shared/minecraft"
	"github.com/kaeman-dev/kaeman-api/shared/utils"
	"github.com/labstack/echo/v4"
)

type VerifyInput struct {
	// ServerID challenge 返回的一次性挑战 ID，有/无横杠均可
	ServerID string `json:"serverID" binding:"required" example:"0f8f8c8e5d3c4b2a9e7d6c5b4a3f2e1d"`
	// UUID 玩家 Mojang UUID，有/无横杠均可
	UUID string `json:"uuid" binding:"required" example:"069a79f444e94726a5befca90e38aaf5"`
}

type VerifyData struct {
	// AccessToken Minecraft 会话 JWT，audience 为 minecraft-session，24 小时有效
	AccessToken string `json:"accessToken" example:"eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOlsibWluZWNyYWZ0LXNlc3Npb24iXSwiZGF0YSI6eyJtaW5lY3JhZnQiOnsibmFtZSI6Ik5vdGNoIiwidXVpZCI6IjA2OWE3OWY0NDRlOTQ3MjZhNWJlZmNhOTBlMzhhYWY1In19fQ.x"`
	// ExpiresAt 会话 token 过期时间
	ExpiresAt time.Time `json:"expiresAt" example:"2026-09-30T12:00:00Z"`
}

// Verify godoc
// @Summary 验证 Minecraft 会话并换取会话 token
// @Description 用 challenge 返回的 serverID 和玩家 UUID 调用，服务端校验玩家已在 Mojang 会话服务器 join 该 serverID。验证成功后签发 24 小时有效的 Minecraft 会话 JWT，并立即消费掉挑战。
// @Tags auth
// @Accept json
// @Param body body VerifyInput true "挑战与玩家 UUID，均支持带/不带横杠"
// @Success 200 {object} response.Response[api.VerifyData]
// @Header 200 {string} Cache-Control "no-store"
// @Failure 400 {object} response.Response[any] "Invalid request body / Invalid serverID / Invalid UUID"
// @Failure 401 {object} response.Response[any] "Invalid or expired challenge / Minecraft session verification failed / Challenge already used or expired"
// @Failure 429 {object} response.Response[any] "Too Many Requests"
// @Failure 500 {object} response.Response[any]
// @Router /v1/auth/minecraft/verify [post]
func (h Handler) Verify(c echo.Context) error {
	input := VerifyInput{}

	c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, 16<<10)
	if err := c.Echo().JSONSerializer.Deserialize(c, &input); err != nil {
		return response.Write(c, http.StatusBadRequest, "Invalid request body", nil)
	}
	serverID, err := utils.NormalizeUUID(input.ServerID)
	if err != nil {
		return response.Write(c, http.StatusBadRequest, "Invalid serverID", nil)
	}
	id, err := utils.NormalizeUUID(input.UUID)
	if err != nil {
		return response.Write(c, http.StatusBadRequest, "Invalid UUID", nil)
	}

	key := "kaeman:challenge:" + serverID
	_, err = h.Services.Cache.Get(c.Request().Context(), key)
	if errors.Is(err, store.NotFound{}) {
		return response.Write(c, http.StatusUnauthorized, "Invalid or expired challenge", nil)
	}
	if err != nil {
		return err
	}
	identity, err := h.Services.Minecraft.Profile(c.Request().Context(), id)
	if err != nil {
		return err
	}
	err = h.Services.Minecraft.HasJoined(c.Request().Context(), identity.Name, input.ServerID)
	if errors.Is(err, minecraft.ErrNotJoined) {
		return response.Write(c, http.StatusUnauthorized, "Minecraft session verification failed", nil)
	}
	if err != nil {
		return err
	}
	token, err := h.Services.Tokens.IssueMinecraft(*identity, 24*time.Hour)
	if err != nil {
		return err
	}
	_, err = h.Services.Cache.Get(c.Request().Context(), key)
	if errors.Is(err, store.NotFound{}) {
		return response.Write(c, http.StatusUnauthorized, "Challenge already used or expired", nil)
	}
	if err != nil {
		return err
	}
	if err = h.Services.Cache.Delete(c.Request().Context(), key); err != nil {
		return err
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return response.Write(c, http.StatusOK, "ok", VerifyData{AccessToken: token, ExpiresAt: time.Now().UTC().Add(24 * time.Hour)})
}
