package storage

import (
	"log/slog"
	"time"

	"github.com/Sn0wo2/agnosco"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var registry = agnosco.NewRegistry[gorm.Dialector]()

func init() {
	registry.Register(agnosco.Driver[gorm.Dialector]{
		Name:    "postgres",
		Schemes: []string{"postgres", "postgresql"},
		Open:    postgres.Open,
	})
}

func Open(dsn string, log *slog.Logger) (*gorm.DB, error) {
	dialector, err := registry.Open(dsn)
	if err != nil {
		return nil, err
	}
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.New(slog.NewLogLogger(log.Handler(), slog.LevelError), logger.Config{
			LogLevel:                  logger.Error,
			IgnoreRecordNotFoundError: true,
		}),
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, err
	}
	connection, err := db.DB()
	if err != nil {
		return nil, err
	}
	connection.SetMaxOpenConns(1)
	return db, nil
}
