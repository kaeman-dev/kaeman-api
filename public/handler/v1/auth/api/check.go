package api

import (
	"net/http"
	"time"

	"uuid"

	"github.com/kaeman-dev/kaeman-api/public/response"
	"github.com/kaeman-dev/kaeman-api/shared"
	"github.com/kaeman-dev/kaeman-api/shared/jwt"
	"github.com/kaeman-dev/kaeman-api/shared/permission"
	"github.com/kaeman-dev/kaeman-api/shared/storage"
	"github.com/labstack/echo/v4"
)

type Handler struct{ Services *shared.Services }

type CheckData struct {
	// Minecrafts public api key 归属人绑定的全部 Minecraft 身份，小写无横杠 UUID
	Minecrafts []string `json:"minecrafts" example:"069a79f444e94726a5befca90e38aaf5"`
	// Permissions public api key 的权限位集合，按位与判断
	Permissions permission.Permission `json:"permissions" example:"4"`
	// ExpiresAt public api key 过期时间
	ExpiresAt time.Time `json:"expiresAt" example:"2026-09-30T12:00:00Z"`
}

// Check godoc
// @Summary 查询 public api key 归属人绑定的 Minecraft 身份
// @Tags auth
// @Security BearerAuth
// @Success 200 {object} response.Response[api.CheckData]
// @Failure 401 {object} response.Response[any] "Key required / Invalid public api key"
// @Failure 429 {object} response.Response[any] "Too Many Requests"
// @Failure 500 {object} response.Response[any]
// @Router /v1/auth/check [get]
func (h Handler) Check(c echo.Context) error {
	claims := c.Get("publicAPIKey").(*jwt.Claims[jwt.PublicAPIClaimsData])
	uid, err := uuid.Parse(claims.Data.UID)
	if err != nil {
		return response.Write(c, http.StatusUnauthorized, "Invalid public api key", nil)
	}
	var identities []storage.KaemanIdentity
	if err := h.Services.DB.WithContext(c.Request().Context()).Find(&identities, "uid = ?", uid).Error; err != nil {
		return err
	}
	minecrafts := make([]string, 0, len(identities))
	for _, identity := range identities {
		if identity.Platform == "minecraft" {
			minecrafts = append(minecrafts, identity.Identity)
		}
	}
	return response.Write(c, http.StatusOK, "ok", CheckData{
		Minecrafts:  minecrafts,
		Permissions: *claims.Data.Permissions,
		ExpiresAt:   claims.ExpiresAt.Time,
	})
}
