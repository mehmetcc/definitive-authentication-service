package authentication

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LoginRequest is the payload for logging in.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,alphanum"`
}

// RefreshRequest is the payload for refreshing an access token.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// LogoutRequest is the payload for logging out.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// TokenResponse contains both access and refresh tokens.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// AuthHandler handles authentication-related HTTP endpoints.
type AuthHandler struct {
	router  *gin.RouterGroup
	service AuthenticationService
	logger  *zap.Logger
}

// NewAuthHandler registers auth endpoints on the given router group.
func NewAuthHandler(router *gin.RouterGroup, service AuthenticationService, logger *zap.Logger) *AuthHandler {
	h := &AuthHandler{router: router, service: service, logger: logger}
	h.router.POST("/auth/login", h.Login)
	h.router.POST("/auth/refresh", h.Refresh)
	h.router.POST("/auth/logout", h.Logout)
	return h
}

// Login godoc
// @Summary      Login
// @Description  Authenticate user and issue tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload  body      LoginRequest  true  "Login credentials"
// @Success      200      {object}  TokenResponse
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid login payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email or password format"})
		return
	}
	access, refresh, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, TokenResponse{AccessToken: access, RefreshToken: refresh})
	case errors.Is(err, ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
	default:
		h.logger.Error("Login service failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not login"})
	}
}

// Refresh godoc
// @Summary      Refresh Token
// @Description  Rotate refresh token and issue new tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload  body      RefreshRequest  true  "Refresh token payload"
// @Success      200      {object}  TokenResponse
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid refresh payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh token required"})
		return
	}
	access, refresh, err := h.service.Refresh(c.Request.Context(), req.RefreshToken)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, TokenResponse{AccessToken: access, RefreshToken: refresh})
	case errors.Is(err, ErrInvalidRefreshToken):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
	default:
		h.logger.Error("Refresh service failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not refresh token"})
	}
}

// Logout godoc
// @Summary      Logout
// @Description  Revoke a refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        payload  body      LogoutRequest  true  "Logout payload"
// @Success      204      {object}  nil
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid logout payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh token required"})
		return
	}
	err := h.service.Logout(c.Request.Context(), req.RefreshToken)
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrInvalidRefreshToken):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
	default:
		h.logger.Error("Logout service failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not logout"})
	}
}
