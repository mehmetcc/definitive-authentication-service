package db

import (
	"github.com/mehmetcc/definitive-authentication-service/internal/person"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(dsn string, logger zap.Logger) error {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Fatal("failed to connect to postgres", zap.Error(err))
		return err
	}

	DB = db
	return DB.AutoMigrate(
		&person.Person{})
}
