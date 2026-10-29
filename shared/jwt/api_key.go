package jwt

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kaeman-dev/kaeman-api/shared/permission"
)

const APIKeyAudience = "api-key"

type APIKeyData struct {
	Public   *APIKeyPublicData   `json:"public,omitempty"`
	Internal *APIKeyInternalData `json:"internal,omitempty"`
}

type APIKeyPublicData struct {
	API *PublicAPIClaimsData `json:"api,omitempty"`
}

type APIKeyInternalData struct {
	RPC bool `json:"rpc"`
}

func (d APIKeyData) HasRPC() bool {
	return d.Internal != nil && d.Internal.RPC
}

type PublicAPIClaimsData struct {
	UID         string                 `json:"uid"`
	Permissions *permission.Permission `json:"permissions,omitempty"`
	Ratelimit   *int                   `json:"ratelimit,omitempty"`
}

type APIKeyClaims = Claims[APIKeyData]

type apiKeyContextKey struct{}

func WithAPIKey(ctx context.Context, claims *APIKeyClaims) context.Context {
	return context.WithValue(ctx, apiKeyContextKey{}, claims)
}

func APIKeyFromContext(ctx context.Context) (*APIKeyClaims, bool) {
	claims, ok := ctx.Value(apiKeyContextKey{}).(*APIKeyClaims)

	return claims, ok && claims != nil
}

func (t *Token) IssueAPIKey(subject string, data APIKeyData, lifetime time.Duration) (string, error) {
	if subject == "" {
		return "", errors.New("API key subject is required")
	}

	if lifetime <= 0 {
		return "", errors.New("API key lifetime must be positive")
	}

	now := time.Now().UTC()

	return jwt.NewWithClaims(jwt.SigningMethodHS256, &APIKeyClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    Issuer,
			Subject:   subject,
			Audience:  jwt.ClaimStrings{APIKeyAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(lifetime)),
		},
		Data: data,
	}).SignedString(t.Secret)
}

func (t *Token) ParseAPIKey(value string) (*APIKeyClaims, error) {
	claims := APIKeyClaims{}

	_, err := jwt.ParseWithClaims(
		value,
		&claims,
		func(*jwt.Token) (any, error) {
			return t.Secret, nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(Issuer),
		jwt.WithAudience(APIKeyAudience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil {
		return nil, err
	}

	if claims.Issuer == "" || claims.Subject == "" || claims.ID == "" || len(claims.Audience) == 0 || claims.IssuedAt == nil || claims.ExpiresAt == nil {
		return nil, errors.New("invalid API key registered claims")
	}

	return &claims, nil
}
