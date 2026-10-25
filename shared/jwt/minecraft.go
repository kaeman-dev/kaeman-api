package jwt

import (
	"errors"
	"strings"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kaeman-dev/kaeman-api/shared/permission"
	"github.com/kaeman-dev/kaeman-api/shared/utils"
)

type Claims[T PublicAPIClaimsData | MinecraftClaimsData] struct {
	jwt.RegisteredClaims
	Data T `json:"data"`
}

type PublicAPIClaimsData struct {
	UID         string                 `json:"uid"`
	Permissions *permission.Permission `json:"permissions,omitempty"`
	Ratelimit   *int                   `json:"ratelimit,omitempty"`
}

type MinecraftClaimsData struct {
	Minecraft MinecraftIdentity `json:"minecraft,omitempty"`
}

type MinecraftIdentity struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

type Token struct {
	Secret []byte
}

func (t *Token) IssueMinecraft(mcIdentity MinecraftIdentity, lifetime time.Duration) (string, error) {
	id, err := utils.NormalizeUUID(mcIdentity.UUID)
	if err != nil {
		return "", err
	}
	mcIdentity.UUID = id

	now := time.Now().UTC()

	return jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims[MinecraftClaimsData]{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    "kaeman-api",
			Audience:  jwt.ClaimStrings{"minecraft-session"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(lifetime)),
		},
		Data: MinecraftClaimsData{
			Minecraft: mcIdentity,
		},
	}).SignedString(t.Secret)
}

func (t *Token) ParseMinecraft(value string) (*Claims[MinecraftClaimsData], error) {
	claims := Claims[MinecraftClaimsData]{}

	_, err := jwt.ParseWithClaims(
		value,
		&claims,
		func(*jwt.Token) (any, error) {
			return t.Secret, nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer("kaeman-api"),
		jwt.WithAudience("minecraft-session"),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil {
		return nil, err
	}

	if claims.ID == "" {
		return nil, errors.New("invalid minecraft identity")
	}

	if name := claims.Data.Minecraft.Name; name == "" || len(name) > 16 || strings.Trim(name, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_") != "" {
		return nil, errors.New("invalid minecraft name")
	}

	id, err := utils.NormalizeUUID(claims.Data.Minecraft.UUID)
	if err != nil {
		return nil, err
	}

	claims.Data.Minecraft.UUID = id

	return &claims, nil
}

func (t *Token) ParsePublicAPI(value string) (*Claims[PublicAPIClaimsData], error) {
	claims := Claims[PublicAPIClaimsData]{}

	_, err := jwt.ParseWithClaims(
		value,
		&claims,
		func(*jwt.Token) (any, error) {
			return t.Secret, nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithAudience("public-api"),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil {
		return nil, err
	}

	if claims.ID == "" {
		return nil, errors.New("invalid public API identity")
	}

	if claims.Data.Permissions == nil || *claims.Data.Permissions == 0 || claims.Data.Ratelimit == nil || *claims.Data.Ratelimit == 0 {
		return nil, errors.New("invalid public API permissions or ratelimit")
	}

	return &claims, nil
}
