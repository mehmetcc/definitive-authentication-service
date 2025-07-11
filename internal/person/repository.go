package person

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrNotFound    = errors.New("person not found")
	ErrEmailExists = errors.New("email already exists")
)

type PersonRepository interface {
	Create(ctx context.Context, p *Person) error
	FindByEmail(ctx context.Context, email string) (*Person, error)
	FindByID(ctx context.Context, id int64) (*Person, error)
	Update(ctx context.Context, p *Person) error
	Delete(ctx context.Context, id int64) error
}

type personRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewPersonRepository(db *gorm.DB, l *zap.Logger) PersonRepository {
	return &personRepository{db: db, logger: l}
}

func (r *personRepository) Create(ctx context.Context, p *Person) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&Person{}).
			Where("email = ? AND is_active = ?", p.Email, true).
			Count(&count).Error; err != nil {
			r.logger.Error("error checking existing email", zap.Error(err))
			return err
		}
		if count > 0 {
			return ErrEmailExists
		}

		p.IsActive = true
		now := time.Now().UTC()
		p.CreatedAt = now
		p.UpdatedAt = now

		if err := tx.Create(p).Error; err != nil {
			r.logger.Error("error creating person", zap.Error(err))
			return err
		}

		return nil
	})
}

func (r *personRepository) FindByEmail(ctx context.Context, email string) (*Person, error) {
	var p Person
	err := r.db.WithContext(ctx).
		Where("email = ? AND is_active = ?", email, true).
		First(&p).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *personRepository) FindByID(ctx context.Context, id int64) (*Person, error) {
	var p Person
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_active = ?", id, true).
		First(&p).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *personRepository) Update(ctx context.Context, p *Person) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&Person{}).
			Where("email = ? AND id <> ? AND is_active = ?", p.Email, p.ID, true).
			Count(&count).Error; err != nil {
			r.logger.Error("error checking email uniqueness", zap.Error(err))
			return err
		}
		if count > 0 {
			return ErrEmailExists
		}

		p.UpdatedAt = time.Now().UTC()
		if err := tx.Save(p).Error; err != nil {
			r.logger.Error("error updating person", zap.Error(err))
			return err
		}

		return nil
	})
}

func (r *personRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p Person
		if err := tx.Where("id = ? AND is_active = ?", id, true).
			First(&p).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}

		p.IsActive = false
		p.UpdatedAt = time.Now().UTC()
		if err := tx.Save(&p).Error; err != nil {
			r.logger.Error("error soft deleting person", zap.Error(err))
			return err
		}

		return nil
	})
}
