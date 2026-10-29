package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/Sn0wo2/ordo"
	"github.com/Sn0wo2/ordo/format"
	"github.com/go-playground/validator/v10"
	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Server    Server    `toml:"server"`
	Database  Database  `toml:"database"`
	Redis     Redis     `toml:"redis,omitempty"`
	Auth      Auth      `toml:"auth"`
	Minecraft Minecraft `toml:"minecraft"`
}

var validate = validator.New()

func init() {
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		name, _, _ := strings.Cut(field.Tag.Get("toml"), ",")
		if name == "" {
			name = field.Name
		}

		return name
	})
}

type Server struct {
	Debug        bool     `toml:"debug"`
	ListenAddr   string   `toml:"listen_addr"             validate:"required"`
	TrustProxies []string `toml:"trust_proxies,omitempty" validate:"omitempty,dive,ip|cidr"`
}

type Database struct {
	DSN string `toml:"dsn" validate:"required"`
}

type Redis struct {
	Addr     string `toml:"addr,omitempty"`
	Password string `toml:"password,omitempty"`
}

type Auth struct {
	JWTSecret string `toml:"secret" validate:"omitempty,min=32"`
}

type Minecraft struct {
	MojangSessionBaseURL string `toml:"mojang_session_base_url" validate:"omitempty,url"`
}

func Load(path string) (*Config, error) {
	loader := &ordo.Loader[Config]{
		Formats: []format.Format[Config]{format.NewFormatter[Config](
			"toml", []string{".toml"}, 10,
			toml.Unmarshal,
			toml.Marshal,
		)},
		Default: &Config{
			Server: Server{
				Debug:      true,
				ListenAddr: ":38080",
			},
			Database: Database{
				// #nosec G101 -- development-only DSN used when no config is provided.
				DSN: "postgres://postgres:postgres@localhost/postgres",
			},
			Minecraft: Minecraft{
				MojangSessionBaseURL: "https://sessionserver.mojang.com",
			},
		},
		Validate: func(cfg *Config) error {
			cfg.Minecraft.MojangSessionBaseURL = strings.TrimRight(cfg.Minecraft.MojangSessionBaseURL, "/")
			if err := validate.Struct(cfg); err != nil {
				var fieldErrs validator.ValidationErrors
				if !errors.As(err, &fieldErrs) {
					return err
				}

				errs := make([]error, len(fieldErrs))
				for i, fe := range fieldErrs {
					errs[i] = fmt.Errorf("%s failed on the %q validation", strings.TrimPrefix(fe.Namespace(), "Config."), fe.Tag())
				}

				return errors.Join(errs...)
			}

			return nil
		},
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		secret := make([]byte, 32)
		rand.Read(secret)
		loader.Default.Auth.JWTSecret = base64.RawURLEncoding.EncodeToString(secret)

		if err := loader.Save(loader.Default, path); err != nil {
			return nil, fmt.Errorf("failed to create default config file %s: %w", path, err)
		}

		return nil, fmt.Errorf("default config file created at %s, please edit it and restart", path)
	}

	cfg, _, err := loader.Load(path)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
