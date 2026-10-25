package api

import (
	"net/http"

	"github.com/kaeman-dev/kaeman-api/public/api/response"
	"github.com/kaeman-dev/kaeman-api/shared"
	"github.com/labstack/echo/v4"
)

type Handler struct{ Services *shared.Services }

// Check godoc
// @Summary 验证 API Key
// @Tags auth
// @Security BearerAuth
// @Success 200 {object} response.Response[any]
// @Failure 401 {object} response.Response[any] "Key required / Invalid API key"
// @Failure 429 {object} response.Response[any] "Too Many Requests"
// @Router /v1/auth/check [get]
func (h Handler) Check(c echo.Context) error {
	return response.Write(c, http.StatusOK, "ok", nil)
}
