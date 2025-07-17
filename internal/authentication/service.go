package authentication

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
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
	Refresh(ctx context.Context, refreshJWT string) (newAccessToken, newRefreshToken string, err error)
	Logout(ctx context.Context, refreshJWT string) error
}

type authenticationService struct {
	personService   person.PersonService
	recordRepo      RecordRepository
	logger          *zap.Logger
	accessSecret    string
	refreshSecret   string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewAuthenticationService(
	personService person.PersonService,
	recordRepo RecordRepository,
	logger *zap.Logger,
	accessSecret string,
	accessTTL time.Duration,
	refreshSecret string,
	refreshTTL time.Duration,
) AuthenticationService {
	return &authenticationService{
		personService:   personService,
		recordRepo:      recordRepo,
		logger:          logger,
		accessSecret:    accessSecret,
		refreshSecret:   refreshSecret,
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}
}

func (a *authenticationService) Login(ctx context.Context, email, password string) (string, string, error) {
	// 1) Validate credentials
	user, err := a.personService.ReadPersonByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, person.ErrPersonNotFound) {
			return "", "", ErrInvalidCredentials
		}
		return "", "", ErrLoginFailed
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return "", "", ErrInvalidCredentials
	}

	// 2) Issue Access Token
	accessJWT, err := utils.IssueAccessToken(
		strconv.Itoa(int(user.ID)),
		user.Role,
		a.accessSecret,
		a.accessTokenTTL,
	)
	if err != nil {
		return "", "", err
	}

	// 3) Generate & store Refresh Token with retry-on-duplicate
	var refreshJWT string
	for {
		jti := uuid.NewString()
		sum := sha256.Sum256([]byte(jti))
		rec := &RefreshTokenRecord{
			PersonID:     user.ID,
			RefreshToken: hex.EncodeToString(sum[:]),
			ExpiresAt:    time.Now().Add(a.refreshTokenTTL),
		}

		if err := a.recordRepo.Create(ctx, rec); err != nil {
			if strings.Contains(err.Error(), "duplicate key") {
				// Collision—try a new JTI
				continue
			}
			return "", "", err
		}

		// Only issue the JWT once the DB row is secured
		refreshJWT, err = utils.IssueRefreshToken(
			strconv.Itoa(int(user.ID)),
			jti,
			a.refreshSecret,
			a.refreshTokenTTL,
		)
		if err != nil {
			return "", "", err
		}
		break
	}

	return accessJWT, refreshJWT, nil
}

func (a *authenticationService) Refresh(ctx context.Context, refreshJWT string) (string, string, error) {
	// 1) Parse & validate incoming refresh JWT
	claims, err := utils.ParseRefreshToken(refreshJWT, a.refreshSecret)
	if err != nil {
		return "", "", ErrInvalidRefreshToken
	}

	// 2) Look up JTI in DB
	hash := sha256.Sum256([]byte(claims.ID))
	rec, err := a.recordRepo.ReadByToken(ctx, hex.EncodeToString(hash[:]))
	if err != nil {
		return "", "", ErrInvalidRefreshToken
	}

	// 3) Check DB‐record expiry
	if time.Now().After(rec.ExpiresAt) {
		_ = a.recordRepo.DeleteByToken(ctx, hex.EncodeToString(hash[:]))
		return "", "", ErrInvalidRefreshToken
	}

	// 4) Issue new Access Token
	userID := rec.PersonID
	user, err := a.personService.ReadPersonByID(ctx, userID)
	if err != nil {
		return "", "", ErrLoginFailed
	}
	accessJWT, err := utils.IssueAccessToken(
		strconv.Itoa(int(user.ID)),
		user.Role,
		a.accessSecret,
		a.accessTokenTTL,
	)
	if err != nil {
		return "", "", err
	}

	// 5) Rotate Refresh Token: new JWT + DB update
	newJTI := uuid.NewString()
	newRefreshJWT, err := utils.IssueRefreshToken(
		strconv.Itoa(int(user.ID)),
		newJTI,
		a.refreshSecret,
		a.refreshTokenTTL,
	)
	if err != nil {
		return "", "", err
	}
	newHash := sha256.Sum256([]byte(newJTI))
	if err := a.recordRepo.Rotate(
		ctx,
		hex.EncodeToString(hash[:]),
		hex.EncodeToString(newHash[:]),
		time.Now().Add(a.refreshTokenTTL),
	); err != nil {
		return "", "", ErrLoginFailed
	}

	return accessJWT, newRefreshJWT, nil
}

func (a *authenticationService) Logout(ctx context.Context, refreshJWT string) error {
	claims, err := utils.ParseRefreshToken(refreshJWT, a.refreshSecret)
	if err != nil {
		return ErrInvalidRefreshToken
	}
	hash := sha256.Sum256([]byte(claims.ID))
	if err := a.recordRepo.DeleteByToken(ctx, hex.EncodeToString(hash[:])); err != nil {
		return ErrInvalidRefreshToken
	}
	return nil
}
