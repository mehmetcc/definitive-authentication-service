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

type AdminConfig struct {
	Username string
	Password string
}

type Config struct {
	Database *DatabaseConfig
	Server   *ServerConfig
	Admin    *AdminConfig
}

func LoadConfig(dotenvPath string) (*Config, error) {
	err := godotenv.Load(dotenvPath)
	if err != nil {
		return nil, err
	}

	dbCfg := &DatabaseConfig{
		PostgresUser:     os.Getenv("POSTGRES_USER"),
		PostgresPassword: os.Getenv("POSTGRES_PASSWORD"),
		PostgresDB:       os.Getenv("POSTGRES_DB"),
	}
	serverCgf := &ServerConfig{
		Port: os.Getenv("SERVER_PORT"),
	}
	adminCfg := &AdminConfig{
		Username: os.Getenv("ADMIN_USERNAME"),
		Password: os.Getenv("ADMIN_PASSWORD"),
	}

	cfg := &Config{dbCfg, serverCgf, adminCfg}
	return cfg, nil
}
