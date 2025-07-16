package utils

import (
	"errors"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var (
	ErrAccessTokenTooShort = errors.New("access token too short")
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

type TokenConfig struct {
	AccessTokenSecret  string
	RefreshTokenSecret string
	RefreshTokenExpiry int // in hours
	AccessTokenExpiry  int // in minutes
}

type Config struct {
	Database *DatabaseConfig
	Server   *ServerConfig
	Admin    *AdminConfig
	Token    *TokenConfig
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
	tokenCfg := &TokenConfig{
		AccessTokenSecret:  os.Getenv("ACCESS_TOKEN_SECRET"),
		RefreshTokenSecret: os.Getenv("REFRESH_TOKEN_SECRET"),
		RefreshTokenExpiry: func() int {
			expiry, err := strconv.Atoi(os.Getenv("REFRESH_TOKEN_EXPIRY"))
			if err != nil {
				return 24 // default to 24 hours if parsing fails
			}
			return expiry
		}(),
		AccessTokenExpiry: func() int {
			expiry, err := strconv.Atoi(os.Getenv("REFRESH_TOKEN_EXPIRY"))
			if err != nil {
				return 24 // default to 2 minutes if parsing fails
			}
			return expiry
		}(),
	}

	if len(tokenCfg.AccessTokenSecret) < 32 {
		panic("access token too short. must be at least 32 characters")
	}
	if len(tokenCfg.RefreshTokenSecret) < 32 {
		panic("refresh token too short. must be at least 32 characters")
	}

	cfg := &Config{dbCfg, serverCgf, adminCfg, tokenCfg}
	return cfg, nil
}
