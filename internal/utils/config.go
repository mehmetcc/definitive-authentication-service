package utils

import (
	"os"

	"github.com/joho/godotenv"
)

type DatabaseConfig struct {
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
}

func (c *DatabaseConfig) DSN() string {
	return "host=localhost user=" + c.PostgresUser +
		" password=" + c.PostgresPassword +
		" dbname=" + c.PostgresDB +
		" port=5432 sslmode=disable TimeZone=UTC"
}

type ServerConfig struct {
	Port string
}

type Config struct {
	Database *DatabaseConfig
	Server   *ServerConfig
}

func LoadConfig(dotenvPath string) (*Config, error) {
	_ = godotenv.Load(dotenvPath)
	dbCfg := &DatabaseConfig{
		PostgresUser:     os.Getenv("POSTGRES_USER"),
		PostgresPassword: os.Getenv("POSTGRES_PASSWORD"),
		PostgresDB:       os.Getenv("POSTGRES_DB"),
	}
	serverCgf := &ServerConfig{
		Port: os.Getenv("SERVER_PORT"),
	}

	cfg := &Config{dbCfg, serverCgf}
	return cfg, nil
}
