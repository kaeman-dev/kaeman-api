package middleware

import (
	"errors"
	"net/http"
	"strings"

	"uuid"

	"github.com/kaeman-dev/kaeman-api/public/response"
	"github.com/kaeman-dev/kaeman-api/shared"
	"github.com/kaeman-dev/kaeman-api/shared/jwt"
	"github.com/kaeman-dev/kaeman-api/shared/permission"
	"github.com/kaeman-dev/kaeman-api/shared/storage"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"
)

func PublicAPIKey(deps *shared.Services, pms permission.Permission) echo.MiddlewareFunc {
	keyAuth := echomiddleware.KeyAuthWithConfig(echomiddleware.KeyAuthConfig{
		Validator: func(value string, c echo.Context) (bool, error) {
			claims, err := deps.Tokens.ParsePublicAPI(value)
			if err != nil {
				return false, err
			}
			c.Set("publicAPIKey", claims)
			return true, nil
		},
		ErrorHandler: func(err error, c echo.Context) error {
			var missing *echomiddleware.ErrKeyAuthMissing
			if errors.As(err, &missing) && strings.TrimSpace(c.Request().Header.Get(echo.HeaderAuthorization)) == "" {
				return response.Write(c, http.StatusUnauthorized, "Key required", nil)
			}
			return response.Write(c, http.StatusUnauthorized, "Invalid public api key", nil)
		},
	})
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return keyAuth(func(c echo.Context) error {
			claims := c.Get("publicAPIKey").(*jwt.Claims[jwt.PublicAPIClaimsData])
			if pms != 0 && *claims.Data.Permissions&pms != pms {
				return response.Write(c, http.StatusUnauthorized, "Required permission missing", nil)
			}
			return next(c)
		})
	}
}

func MinecraftToken(deps *shared.Services) echo.MiddlewareFunc {
	keyAuth := echomiddleware.KeyAuthWithConfig(echomiddleware.KeyAuthConfig{
		Validator: func(value string, c echo.Context) (bool, error) {
			claims, err := deps.Tokens.ParseMinecraft(value)
			if err != nil {
				return false, err
			}
			c.Set("minecraftKey", claims)
			return true, nil
		},
		ErrorHandler: func(err error, c echo.Context) error {
			var missing *echomiddleware.ErrKeyAuthMissing
			if errors.As(err, &missing) && strings.TrimSpace(c.Request().Header.Get(echo.HeaderAuthorization)) == "" {
				return response.Write(c, http.StatusUnauthorized, "Key required", nil)
			}
			return response.Write(c, http.StatusUnauthorized, "Invalid minecraft key", nil)
		},
	})
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return keyAuth(func(c echo.Context) error {
			claims := c.Get("minecraftKey").(*jwt.Claims[jwt.MinecraftClaimsData])
			if publicAPIKey := c.Get("publicAPIKey"); publicAPIKey != nil {
				pub, ok := publicAPIKey.(*jwt.Claims[jwt.PublicAPIClaimsData])
				if !ok {
					return errors.New("unexpected publicAPIKey claims type")
				}
				ownerUID, err := uuid.Parse(pub.Data.UID)
				if err != nil {
					return response.Write(c, http.StatusUnauthorized, "Invalid public api key", nil)
				}
				var bound storage.KaemanIdentity
				err = deps.DB.WithContext(c.Request().Context()).First(&bound, "platform = ? AND identity = ?", "minecraft", claims.Data.Minecraft.UUID).Error
				if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && bound.UID != ownerUID) {
					return response.Write(c, http.StatusUnauthorized, "Minecraft identity not bound to this public api key", nil)
				}
				if err != nil {
					return err
				}
			}
			return next(c)
		})
	}
}
