package app

import (
	"time"

	"rankr/pkg/database"
	"rankr/pkg/httpserver"
	"rankr/pkg/logger"
)

type Config struct {
	HTTPServer           httpserver.Config `koanf:"http_server"`
	PostgresDB           database.Config   `koanf:"postgres_db"`
	Logger               logger.Config     `koanf:"logger"`
	TotalShutdownTimeout time.Duration     `koanf:"total_shutdown_timeout"`
	PathOfMigration      string            `koanf:"path_of_migration"`
}
