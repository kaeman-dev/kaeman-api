package api

import (
	"github.com/kaeman-dev/kaeman-api/public/api/response"
	"github.com/labstack/echo/v4"
)

// Listen godoc
// @Summary BSI 实时消息 WebSocket 网关
// @Description 建立 WebSocket 连接后作为只读订阅者接收服务端推送的 BSIMessage 文本帧（JSON）：type 为 chat 时 data 为 BSIChatRequest，type 为 splash 时 data 为 BSISplashRequest。客户端发送会被 CloseRead 忽略，服务端每 30 秒 ping 一次，空闲写超时 5 秒。OpenAPI 不支持 WebSocket，仅作说明用。
// @Tags services
// @Success 101 {string} string "Switching Protocols (WebSocket upgrade)"
// @Failure 503 {object} response.Response[any] "Service Unavailable（hub 已关闭）"
// @Router /v1/services/bsi/gateway [get]
func (h Handler) Listen(c echo.Context) error {
	status, err := h.Services.BSI.Listen(c.Response(), c.Request())
	if err != nil {
		if status != 0 {
			return echo.NewHTTPError(status).SetInternal(err)
		}

		return err
	}

	if status != 0 {
		return response.Write(c, status, "", nil)
	}

	return nil
}
