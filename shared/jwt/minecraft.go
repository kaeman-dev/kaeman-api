package jwt

import (
	"errors"
	"strings"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kaeman-dev/kaeman-api/shared/utils"
)

const (
	Issuer                   = "kaeman-api"
	MinecraftSessionAudience = "minecraft-session"
)

type Claims[T any] struct {
	jwt.RegisteredClaims
	Data T `json:"data"`
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
			Issuer:    Issuer,
			Audience:  jwt.ClaimStrings{MinecraftSessionAudience},
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
		jwt.WithIssuer(Issuer),
		jwt.WithAudience(MinecraftSessionAudience),
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
