package middleware

import (
	"errors"
	"net/http"

	"uuid"

	"github.com/gin-gonic/gin"
	"github.com/kaeman-dev/kaeman-api/jwt"
	"github.com/kaeman-dev/kaeman-api/permission"
	"github.com/kaeman-dev/kaeman-api/response"
	"github.com/kaeman-dev/kaeman-api/server"
	"github.com/kaeman-dev/kaeman-api/storage"
	"github.com/kaeman-dev/kaeman-api/utils"
	"gorm.io/gorm"
)

func PublicAPIKey(deps *server.Services, pms permission.Permission) func(*gin.Context) error {
	return func(ctx *gin.Context) error {
		value := utils.BearerToken(ctx.GetHeader("Authorization"))
		if value == "" {
			return response.Write(ctx, http.StatusUnauthorized, "Key required", nil)
		}
		claims, err := deps.Tokens.ParsePublicAPI(value)
		if err != nil {
			return response.Write(ctx, http.StatusUnauthorized, "Invalid public api key", nil)
		}
		if pms != 0 && *claims.Data.Permissions&pms != pms {
			return response.Write(ctx, http.StatusUnauthorized, "Required permission missing", nil)
		}
		ctx.Set("publicAPIKey", claims)
		return nil
	}
}

func MinecraftToken(deps *server.Services) func(*gin.Context) error {
	return func(ctx *gin.Context) error {
		value := utils.BearerToken(ctx.GetHeader("Authorization"))
		if value == "" {
			return response.Write(ctx, http.StatusUnauthorized, "Key required", nil)
		}
		claims, err := deps.Tokens.ParseMinecraft(value)
		if err != nil {
			return response.Write(ctx, http.StatusUnauthorized, "Invalid minecraft key", nil)
		}

		if publicAPIKey, ok := ctx.Get("publicAPIKey"); ok {
			pub, ok := publicAPIKey.(*jwt.Claims[jwt.PublicAPIClaimsData])
			if !ok {
				return errors.New("unexpected publicAPIKey claims type")
			}
			ownerUID, err := uuid.Parse(pub.Data.UID)
			if err != nil {
				return response.Write(ctx, http.StatusUnauthorized, "Invalid public api key", nil)
			}
			var bound storage.KaemanIdentity
			err = deps.DB.WithContext(ctx.Request.Context()).First(&bound, "platform = ? AND identity = ?", "minecraft", claims.Data.Minecraft.UUID).Error
			if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && bound.UID != ownerUID) {
				return response.Write(ctx, http.StatusUnauthorized, "Minecraft identity not bound to this public api key", nil)
			}
			if err != nil {
				return err
			}
		}

		ctx.Set("minecraftKey", claims)
		return nil
	}
}
