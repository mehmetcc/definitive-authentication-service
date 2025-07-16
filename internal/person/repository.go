package person

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgconn"
	"gorm.io/gorm"
)

var (
	ErrEmailAlreadyExists   = errors.New("email already exists")
	ErrPersonNotFound       = errors.New("person not found")
	ErrPersonNotCreated     = errors.New("person not created")
	ErrPersonNotUpdated     = errors.New("person not updated")
	ErrPersonNotDeleted     = errors.New("person not deleted")
	ErrUnresponsiveDatabase = errors.New("error occured during writing to persons table")
)

type PersonRepository interface {
	Create(ctx context.Context, person *Person) error
	ReadByEmail(ctx context.Context, email string) (*Person, error)
	ReadByID(ctx context.Context, id uint) (*Person, error)
	Update(ctx context.Context, person *Person) error
	Delete(ctx context.Context, id uint) error
}

type personRepository struct {
	db *gorm.DB
}

func NewPersonRepository(db *gorm.DB) PersonRepository {
	return &personRepository{db: db}
}

func (p *personRepository) ReadByID(ctx context.Context, id uint) (*Person, error) {
	var person Person
	err := p.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		First(&person, id).
		Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPersonNotFound
	}
	if err != nil {
		return nil, ErrUnresponsiveDatabase
	}
	return &person, nil
}

func (p *personRepository) Create(ctx context.Context, person *Person) error {
	err := p.db.WithContext(ctx).Create(person).Error
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" &&
			strings.Contains(pgErr.ConstraintName, "email") {
			return ErrEmailAlreadyExists
		}
		return ErrPersonNotCreated
	}
	return nil
}

func (p *personRepository) ReadByEmail(ctx context.Context, email string) (*Person, error) {
	var person Person
	err := p.db.WithContext(ctx).
		Where("email = ?", email).
		Where("deleted_at IS NULL").
		First(&person).
		Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPersonNotFound
	}
	if err != nil {
		return nil, ErrUnresponsiveDatabase
	}
	return &person, nil
}

func (p *personRepository) Update(ctx context.Context, person *Person) error {
	err := p.db.WithContext(ctx).Save(person).Error
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" &&
			strings.Contains(pgErr.ConstraintName, "email") {
			return ErrEmailAlreadyExists
		}
		return ErrPersonNotUpdated
	}
	return nil
}

func (p *personRepository) Delete(ctx context.Context, id uint) error {
	if err := p.db.WithContext(ctx).
		Delete(&Person{}, id).
		Error; err != nil {
		return ErrPersonNotDeleted
	}
	return nil
}
