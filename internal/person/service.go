package person

import (
	"context"
	"errors"
	"net/mail"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrHashingPasswordFailed = errors.New("hashing password failed")
	ErrInvalidEmailFormat    = errors.New("invalid email format")
)

type validationTarget struct {
	email    string
	password string
}

type PersonService interface {
	CreatePerson(ctx context.Context, email, password string) (*Person, error)
	ReadPersonByEmail(ctx context.Context, email string) (*Person, error)
	ReadPersonByID(ctx context.Context, id uint) (*Person, error)
	UpdateEmail(ctx context.Context, id uint, email string) error
	UpdatePassword(ctx context.Context, id uint, password string) error
	UpdateLastSeen(ctx context.Context, id uint) error
	DeletePerson(ctx context.Context, id uint) error
}

type personService struct {
	repo   PersonRepository
	logger *zap.Logger
}

func NewPersonService(repo PersonRepository, logger *zap.Logger) PersonService {
	return &personService{
		repo:   repo,
		logger: logger,
	}
}

/** CREATE */
func (s *personService) CreatePerson(ctx context.Context, email, password string) (*Person, error) {
	err := s.validate(ctx, &validationTarget{email: email, password: password})
	if err != nil {
		s.logger.Error("validation failed", zap.String("email", email), zap.Error(err))
		return nil, err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to create person", zap.Error(err))
		return nil, ErrHashingPasswordFailed
	}

	person := NewPerson(email, string(hashed))

	if err := s.repo.Create(ctx, person); err != nil {
		s.logger.Error("failed to create person in repository", zap.Error(err))
		return nil, err
	}
	return person, nil
}

func (s *personService) validate(ctx context.Context, target *validationTarget) error {
	if err := s.validateEmail(target.email); err != nil {
		s.logger.Error("invalid email format", zap.String("email", target.email), zap.Error(err))
		return err
	}
	if err := s.validatePassword(target.password); err != nil {
		s.logger.Error("invalid password format", zap.Error(err))
		return err
	}
	return nil
}

func (s *personService) validateEmail(email string) error {
	_, err := mail.ParseAddress(email)
	if err != nil {
		return ErrInvalidEmailFormat
	}
	return nil
}

func (s *personService) validatePassword(password string) error {
	err := CheckPassword(password)
	if err != nil {
		return err
	}
	return nil
}

/** READ */
func (s *personService) ReadPersonByEmail(ctx context.Context, email string) (*Person, error) {
	person, err := s.repo.ReadByEmail(ctx, email)
	if err != nil {
		s.logger.Error("failed to get person by email", zap.String("email", email), zap.Error(err))
		return nil, err
	}
	return person, nil
}

func (s *personService) ReadPersonByID(ctx context.Context, id uint) (*Person, error) {
	person, err := s.repo.ReadByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get person by ID", zap.Uint("id", id), zap.Error(err))
		return nil, err
	}
	return person, nil
}

/** UPDATE */
func (s *personService) UpdateEmail(ctx context.Context, id uint, email string) error {
	if err := s.validateEmail(email); err != nil {
		s.logger.Error("invalid email format", zap.Uint("id", id), zap.String("email", email), zap.Error(err))
		return err
	}

	person, err := s.repo.ReadByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to update email, person not found", zap.Uint("id", id), zap.Error(err))
		return err
	}

	person.Email = email
	if err := s.repo.Update(ctx, person); err != nil {
		s.logger.Error("failed to update email in repository", zap.Uint("id", id), zap.String("email", email), zap.Error(err))
		return err
	}
	return nil
}

func (s *personService) UpdatePassword(ctx context.Context, id uint, password string) error {
	if err := s.validatePassword(password); err != nil {
		s.logger.Error("invalid password format", zap.Uint("id", id), zap.Error(err))
		return err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to hash password", zap.Error(err))
		return ErrHashingPasswordFailed
	}

	person, err := s.repo.ReadByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to update password, person not found", zap.Uint("id", id), zap.Error(err))
		return err
	}

	person.Password = string(hashed)
	if err := s.repo.Update(ctx, person); err != nil {
		s.logger.Error("failed to update password in repository", zap.Uint("id", id), zap.Error(err))
		return err
	}
	return nil
}

func (s *personService) UpdateLastSeen(ctx context.Context, id uint) error {
	person, err := s.repo.ReadByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to update last seen, person not found", zap.Uint("id", id), zap.Error(err))
		return err
	}

	person.LastSeen = time.Now().UTC()
	if err := s.repo.Update(ctx, person); err != nil {
		s.logger.Error("failed to update last seen in repository", zap.Uint("id", id), zap.Error(err))
		return err
	}
	return nil
}

/** DELETE */
func (s *personService) DeletePerson(ctx context.Context, id uint) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete person", zap.Uint("id", id), zap.Error(err))
		return err
	}
	return nil
}
