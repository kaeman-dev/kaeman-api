package minecraft

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/kaeman-dev/kaeman-api/shared/jwt"
	"github.com/kaeman-dev/kaeman-api/shared/utils"
)

type Client struct {
	HTTP           *http.Client
	SessionBaseURL string
}

func (m *Client) Profile(ctx context.Context, id string) (*jwt.MinecraftIdentity, error) {
	id, err := utils.NormalizeUUID(id)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.SessionBaseURL+"/session/minecraft/profile/"+id, nil)
	if err != nil {
		return nil, err
	}

	resp, err := m.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Mojang profile HTTP status: %d (uuid %s)", resp.StatusCode, id)
	}

	var profile struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&profile); err != nil {
		return nil, err
	}

	return &jwt.MinecraftIdentity{
		UUID: id,
		Name: profile.Name,
	}, nil
}

var ErrNotJoined = errors.New("player has not joined the server")

// HasJoined err 是 nil 就是已经进入了的
func (m *Client) HasJoined(ctx context.Context, username, serverID string) error {
	u, err := url.Parse(m.SessionBaseURL + "/session/minecraft/hasJoined")
	if err != nil {
		return fmt.Errorf("invalid session URL: %w", err)
	}

	query := u.Query()
	query.Set("username", username)
	query.Set("serverId", serverID)
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := m.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNoContent:
		return ErrNotJoined
	default:
		return fmt.Errorf("Mojang hasJoined HTTP status: %d (username %s)", resp.StatusCode, username)
	}
}
