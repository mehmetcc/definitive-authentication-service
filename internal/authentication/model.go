package authentication

import (
	"time"

	"gorm.io/gorm"
)

type RefreshTokenRecord struct {
	gorm.Model
	PersonID     uint      `gorm:"index;not null"`
	RefreshToken string    `gorm:"uniqueIndex;not null"`
	ExpiresAt    time.Time `gorm:"index;not null"`
}
