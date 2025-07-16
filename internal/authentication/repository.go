package authentication

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

var (
	ErrRecordNotFoundByGivenToken    = errors.New("record not found by given token")
	ErrRecordNotFoundByGivenID       = errors.New("record not found by given id")
	ErrRecordNotDeleted              = errors.New("record not deleted")
	ErrRecordNotFoundByGivenPersonID = errors.New("no tokens found for given person")
	ErrUnresponsiveDatabase          = errors.New("error occurred during writing to records table")
	ErrRecordExpired                 = errors.New("token expired")
)

type RecordRepository interface {
	Create(ctx context.Context, record *RefreshTokenRecord) error
	ReadByToken(ctx context.Context, token string) (*RefreshTokenRecord, error)
	ReadByID(ctx context.Context, id uint) (*RefreshTokenRecord, error)
	Rotate(ctx context.Context, oldToken, newToken string, newExpiry time.Time) error
	Delete(ctx context.Context, id uint) error
	DeleteByToken(ctx context.Context, token string) error
	DeleteByPersonID(ctx context.Context, personID uint) error
}

type recordRepository struct {
	db *gorm.DB
}

func NewRecordRepository(db *gorm.DB) RecordRepository {
	return &recordRepository{db: db}
}

func (r *recordRepository) Create(ctx context.Context, record *RefreshTokenRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(record).Error; err != nil {
			return fmt.Errorf("failed to create refresh token record: %w", err)
		}
		return nil
	})
}

func (r *recordRepository) Rotate(
	ctx context.Context,
	oldToken, newToken string,
	newExpiry time.Time,
) error {
	return r.db.
		WithContext(ctx).
		Transaction(func(tx *gorm.DB) error {
			var rec RefreshTokenRecord
			err := tx.
				Joins("JOIN persons ON persons.id = refresh_token_records.person_id").
				Where("refresh_token_records.refresh_token = ?", oldToken).
				Where("persons.deleted_at IS NULL").
				First(&rec).
				Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRecordNotFoundByGivenToken
			}
			if err != nil {
				return ErrUnresponsiveDatabase
			}

			rec.RefreshToken = newToken
			rec.ExpiresAt = newExpiry
			if err := tx.Save(&rec).Error; err != nil {
				return ErrUnresponsiveDatabase
			}
			return nil
		})
}

func (r *recordRepository) ReadByToken(ctx context.Context, token string) (*RefreshTokenRecord, error) {
	var record RefreshTokenRecord
	err := r.db.WithContext(ctx).
		Joins("JOIN persons ON persons.id = refresh_token_records.person_id").
		Where("refresh_token_records.refresh_token = ?", token).
		Where("persons.deleted_at IS NULL").
		First(&record).
		Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRecordNotFoundByGivenToken
	}
	if err != nil {
		return nil, ErrUnresponsiveDatabase
	}

	return &record, nil
}

func (r *recordRepository) ReadByID(ctx context.Context, id uint) (*RefreshTokenRecord, error) {
	var record RefreshTokenRecord
	err := r.db.WithContext(ctx).
		Joins("JOIN persons ON persons.id = refresh_token_records.person_id").
		Where("refresh_token_records.id = ?", id).
		Where("persons.deleted_at IS NULL").
		First(&record).
		Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRecordNotFoundByGivenID
	}
	if err != nil {
		return nil, ErrUnresponsiveDatabase
	}
	return &record, nil
}

func (r *recordRepository) Delete(ctx context.Context, id uint) error {
	res := r.db.WithContext(ctx).
		Joins("JOIN persons ON persons.id = refresh_token_records.person_id").
		Where("refresh_token_records.id = ?", id).
		Where("persons.deleted_at IS NULL").
		Delete(&RefreshTokenRecord{})
	if res.Error != nil {
		return ErrUnresponsiveDatabase
	}
	if res.RowsAffected == 0 {
		return ErrRecordNotFoundByGivenID
	}
	return nil
}

func (r *recordRepository) DeleteByToken(ctx context.Context, token string) error {
	res := r.db.WithContext(ctx).
		Joins("JOIN persons ON persons.id = refresh_token_records.person_id").
		Where("refresh_token_records.refresh_token = ?", token).
		Where("persons.deleted_at IS NULL").
		Delete(&RefreshTokenRecord{})
	if res.Error != nil {
		return ErrUnresponsiveDatabase
	}
	if res.RowsAffected == 0 {
		return ErrRecordNotFoundByGivenToken
	}
	return nil
}

func (r *recordRepository) DeleteByPersonID(ctx context.Context, personID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.
			Joins("JOIN persons ON persons.id = refresh_token_records.person_id").
			Where("refresh_token_records.person_id = ?", personID).
			Where("persons.deleted_at IS NULL").
			Delete(&RefreshTokenRecord{})
		if res.Error != nil {
			return ErrUnresponsiveDatabase
		}
		if res.RowsAffected == 0 {
			return ErrRecordNotFoundByGivenPersonID
		}
		return nil
	})
}
