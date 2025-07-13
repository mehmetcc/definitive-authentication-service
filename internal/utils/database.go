package utils

import (
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDatabase(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), gormConfig())
	if err != nil {
		return nil, err
	}

	return db, nil
}

func gormConfig() *gorm.Config {
	return &gorm.Config{
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}
}
