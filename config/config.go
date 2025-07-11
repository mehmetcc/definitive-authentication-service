package config

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Server struct {
	Port     int
	BasePath string
}

type Encryption struct {
	Cost int
}

type Database struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

type Config struct {
	Server     Server
	Encryption Encryption
	Database   Database
}

func LoadConfig() (*Config, error) {
	var err error
	cfg, err := generate()
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func generate() (*Config, error) {
	_ = godotenv.Load()

	viper.AutomaticEnv()
	viper.BindEnv("server.port", "APPLICATION_PORT")
	viper.BindEnv("server.base_path", "APPLICATION_BASE_PATH")
	viper.BindEnv("encryption.cost", "ENCRYPTION_COST")
	viper.BindEnv("database.host", "POSTGRES_HOST")
	viper.BindEnv("database.port", "POSTGRES_PORT")
	viper.BindEnv("database.user", "POSTGRES_USER")
	viper.BindEnv("database.password", "POSTGRES_PASSWORD")
	viper.BindEnv("database.name", "POSTGRES_DB")
	viper.BindEnv("database.sslmode", "POSTGRES_SSLMODE")

	// defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.base_path", "/api/v1")
	viper.SetDefault("encryption.cost", 12)
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "postgrespw")
	viper.SetDefault("database.name", "b2db")
	viper.SetDefault("database.sslmode", "disable")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	cfg := &Config{
		Server: Server{
			Port:     viper.GetInt("server.port"),
			BasePath: viper.GetString("server.base_path"),
		},
		Encryption: Encryption{
			Cost: viper.GetInt("encryption.cost"),
		},
		Database: Database{
			Host:     viper.GetString("database.host"),
			Port:     viper.GetInt("database.port"),
			User:     viper.GetString("database.user"),
			Password: viper.GetString("database.password"),
			Name:     viper.GetString("database.name"),
			SSLMode:  viper.GetString("database.sslmode"),
		},
	}

	return cfg, nil
}
