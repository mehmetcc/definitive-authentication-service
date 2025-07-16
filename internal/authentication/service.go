package authentication

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mehmetcc/definitive-authentication-service/internal/person"
	"github.com/mehmetcc/definitive-authentication-service/internal/utils"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials  = errors.New("invalid username or password")
	ErrLoginFailed         = errors.New("login failed")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)

type AuthenticationService interface {
	Login(ctx context.Context, email, password string) (accessToken, refreshToken string, err error)
	Refresh(ctx context.Context, oldRefreshToken string) (newAccessToken, newRefreshToken string, err error)
	Logout(ctx context.Context, refreshToken string) error
}

type authenticationService struct {
	personService   person.PersonService
	recordRepo      RecordRepository
	logger          *zap.Logger
	jwtSecret       string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewAuthenticationService(personService person.PersonService,
	recordRepo RecordRepository,
	logger *zap.Logger,
	jwtSecret string,
	accessTokenTTL, refreshTokenTTL time.Duration) AuthenticationService {
	return &authenticationService{
		personService:   personService,
		recordRepo:      recordRepo,
		logger:          logger,
		jwtSecret:       jwtSecret,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}
}

func (a *authenticationService) Login(ctx context.Context, email string, password string) (accessToken string, refreshToken string, err error) {
	user, err := a.personService.ReadPersonByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, person.ErrPersonNotFound) {
			a.logger.Error("ReadPersonByEmail failed: person not found", zap.Error(err), zap.String("email", email))
			return "", "", ErrInvalidCredentials
		}
		a.logger.Error("ReadPersonByEmail failed", zap.Error(err), zap.String("email", email))
		return "", "", ErrLoginFailed
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		a.logger.Error("password comparison failed", zap.String("email", email))
		return "", "", ErrInvalidCredentials
	}

	accessToken, err = utils.IssueToken(
		strconv.Itoa(int(user.ID)),
		user.Role,
		a.jwtSecret,
		a.accessTokenTTL,
	)
	if err != nil {
		a.logger.Error("IssueToken failed", zap.Error(err), zap.Uint("userID", user.ID))
		return "", "", err
	}

	rawRefresh := uuid.NewString()
	sum := sha256.Sum256([]byte(rawRefresh))
	hashed := hex.EncodeToString(sum[:])
	rec := &RefreshTokenRecord{
		PersonID:     user.ID,
		RefreshToken: hashed,
		ExpiresAt:    time.Now().Add(a.refreshTokenTTL),
	}
	if err := a.recordRepo.Create(ctx, rec); err != nil {
		a.logger.Error("create refresh token record failed", zap.Error(err), zap.Uint("userID", user.ID))
		return "", "", err
	}

	go func(userID uint) {
		ctx := context.Background()
		for i := 0; i < 3; i++ {
			err := a.personService.UpdateLastSeen(ctx, userID)
			if err == nil {
				break
			}
			a.logger.Error("last seen update failed", zap.Error(err), zap.Int("attempt", i+1), zap.Uint("userID", userID))
			time.Sleep(100 * time.Millisecond)
		}
	}(user.ID)

	return accessToken, rawRefresh, nil
}

func (a *authenticationService) Logout(ctx context.Context, refreshToken string) error {
	sum := sha256.Sum256([]byte(refreshToken))
	hashed := hex.EncodeToString(sum[:])

	if err := a.recordRepo.DeleteByToken(ctx, hashed); err != nil {
		if errors.Is(err, ErrRecordNotFoundByGivenToken) {
			a.logger.Warn("token not found", zap.String("token", refreshToken))
			return ErrInvalidRefreshToken
		}
		a.logger.Error("logout failed", zap.Error(err))
		return err
	}
	return nil
}

func (a *authenticationService) Refresh(
	ctx context.Context,
	oldRefreshToken string,
) (newAccessToken, newRefreshToken string, err error) {
	sumOld := sha256.Sum256([]byte(oldRefreshToken))
	hashedOld := hex.EncodeToString(sumOld[:])

	rec, err := a.recordRepo.ReadByToken(ctx, hashedOld)
	if err != nil {
		if errors.Is(err, ErrRecordNotFoundByGivenToken) {
			a.logger.Warn("token not found", zap.String("token", oldRefreshToken))
			return "", "", ErrInvalidRefreshToken
		}
		a.logger.Error("repo error", zap.Error(err))
		return "", "", ErrLoginFailed
	}

	if time.Now().After(rec.ExpiresAt) {
		a.logger.Warn("token expired", zap.String("token", oldRefreshToken))
		_ = a.recordRepo.DeleteByToken(ctx, hashedOld)
		return "", "", ErrInvalidRefreshToken
	}

	user, err := a.personService.ReadPersonByID(ctx, rec.PersonID)
	if err != nil {
		a.logger.Error("could not load person", zap.Error(err), zap.Uint("userID", rec.PersonID))
		return "", "", ErrLoginFailed
	}

	newAccessToken, err = utils.IssueToken(
		strconv.Itoa(int(user.ID)),
		user.Role,
		a.jwtSecret,
		a.accessTokenTTL,
	)
	if err != nil {
		a.logger.Error("token can't be issued", zap.Error(err), zap.Uint("userID", user.ID))
		return "", "", ErrLoginFailed
	}

	rawNew := uuid.NewString()
	sumNew := sha256.Sum256([]byte(rawNew))
	hashedNew := hex.EncodeToString(sumNew[:])
	newExpiry := time.Now().Add(a.refreshTokenTTL)

	if err := a.recordRepo.Rotate(ctx, hashedOld, hashedNew, newExpiry); err != nil {
		a.logger.Error("token rotate error", zap.Error(err))
		return "", "", ErrLoginFailed
	}
	return newAccessToken, rawNew, nil
}
