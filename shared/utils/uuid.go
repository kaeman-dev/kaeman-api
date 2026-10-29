package utils

import (
	"encoding/hex"
	"uuid"
)

func NewUUID() string {
	id := uuid.New()

	return hex.EncodeToString(id[:])
}

func NormalizeUUID(id string) (string, error) {
	u, err := uuid.Parse(id)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(u[:]), nil
}
