package middleware

import (
	"errors"
	"net/http"
	"strings"
	"uuid"

	"github.com/kaeman-dev/kaeman-api/public/api/response"
	"github.com/kaeman-dev/kaeman-api/shared"
	"github.com/kaeman-dev/kaeman-api/shared/jwt"
	"github.com/kaeman-dev/kaeman-api/shared/permission"
	"github.com/kaeman-dev/kaeman-api/shared/storage"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"
)

func APIKey(deps *shared.Services) echo.MiddlewareFunc {
	return echomiddleware.KeyAuthWithConfig(echomiddleware.KeyAuthConfig{
		Validator: func(value string, c echo.Context) (bool, error) {
			claims, err := deps.Tokens.ParseAPIKey(value)
			if err != nil {
				return false, err
			}

			c.Set("publicAPIKey", claims)
			c.SetRequest(c.Request().WithContext(jwt.WithAPIKey(c.Request().Context(), claims)))

			return true, nil
		},
		ErrorHandler: func(err error, c echo.Context) error {
			var missing *echomiddleware.ErrKeyAuthMissing
			if errors.As(err, &missing) && strings.TrimSpace(c.Request().Header.Get(echo.HeaderAuthorization)) == "" {
				return response.Write(c, http.StatusUnauthorized, "Key required", nil)
			}

			return response.Write(c, http.StatusUnauthorized, "Invalid API key", nil)
		},
	})
}

func PublicAPIKey(deps *shared.Services, pms permission.Permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return APIKey(deps)(func(c echo.Context) error {
			claims := c.Get("publicAPIKey").(*jwt.APIKeyClaims)
			if !claims.Data.HasRPC() {
				if claims.Data.Public == nil || claims.Data.Public.API == nil {
					return response.Write(c, http.StatusUnauthorized, "Not a public API key", nil)
				}

				api := claims.Data.Public.API
				if api.Permissions == nil || *api.Permissions == 0 || api.Ratelimit == nil || *api.Ratelimit == 0 {
					return response.Write(c, http.StatusUnauthorized, "Invalid public API permissions or ratelimit", nil)
				}

				if pms != 0 && *api.Permissions&pms != pms {
					return response.Write(c, http.StatusUnauthorized, "Required permission missing", nil)
				}
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
				pub, ok := publicAPIKey.(*jwt.APIKeyClaims)
				if !ok {
					return errors.New("unexpected publicAPIKey claims type")
				}

				if pub.Data.HasRPC() {
					return next(c)
				}

				ownerUID, err := uuid.Parse(pub.Data.Public.API.UID)
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
