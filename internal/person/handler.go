package person

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ContextUserKey is the key under which the authenticated Person is stored in Gin context.
const ContextUserKey = "user"

// CreatePersonRequest represents the payload for creating a new person.
// @Description payload to register a new person
// @Property email body string true "unique email address"
// @Property password body string true "password (min 8 alphanumeric)"
type CreatePersonRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,alphanum"`
}

// UpdateEmailRequest represents the payload to update a person's email.
// @Description payload to change email address
// @Property email body string true "new unique email address"
type UpdateEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// UpdatePasswordRequest represents the payload to update a person's password.
// @Description payload to change password
// @Property password body string true "new password (min 8 alphanumeric)"
type UpdatePasswordRequest struct {
	Password string `json:"password" binding:"required,min=8,alphanum"`
}

// IDRequest represents a URI ID parameter.
// @Description contains the resource ID in path
// @Param id path int true "resource identifier"
type IDRequest struct {
	ID uint `uri:"id" binding:"required,min=1"`
}

// IDResponse returns a newly created resource ID.
// @Description response containing the ID of the created resource
// @Property id body integer true "new resource ID"
type IDResponse struct {
	ID uint `json:"id"`
}

// PersonHandler handles HTTP requests for person resources.
type PersonHandler struct {
	router  *gin.RouterGroup
	service PersonService
	logger  *zap.Logger
}

// NewPersonHandler registers person endpoints on the given router group.
func NewPersonHandler(router *gin.RouterGroup, service PersonService, logger *zap.Logger) *PersonHandler {
	h := &PersonHandler{router: router, service: service, logger: logger}
	h.router.POST("/persons", h.CreatePerson)
	h.router.GET("/persons/:id", h.ReadPersonByID)
	h.router.GET("/persons", h.ReadPersonByEmail)
	h.router.PUT("/persons/:id/email", h.UpdateEmail)
	h.router.PUT("/persons/:id/password", h.UpdatePassword)
	h.router.DELETE("/persons/:id", h.DeletePerson)
	return h
}

func (h *PersonHandler) bindID(c *gin.Context) (uint, bool) {
	var uri IDRequest
	if err := c.ShouldBindUri(&uri); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or missing id"})
		return 0, false
	}
	return uri.ID, true
}

// ReadCurrentPerson returns the authenticated user from context.
// @Summary      Get current user
// @Description  Fetch the “me” record for the authenticated user
// @Tags         persons
// @Produce      json
// @Success      200 {object} Person
// @Failure      401 {object} map[string]string
// @Router       /persons/me [get]
func (h *PersonHandler) ReadCurrentPerson(c *gin.Context) {
	raw, exists := c.Get(ContextUserKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	user := raw.(*Person)
	c.JSON(http.StatusOK, user)
}

// CreatePerson godoc
// @Summary      Create Person
// @Description  Register a new person
// @Tags         persons
// @Accept       json
// @Produce      json
// @Param        payload  body      CreatePersonRequest  true  "Person payload"
// @Success      201      {object}  IDResponse
// @Failure      400      {object}  map[string]string
// @Failure      409      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /persons [post]
func (h *PersonHandler) CreatePerson(c *gin.Context) {
	var req CreatePersonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid create payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email or password format"})
		return
	}
	p, err := h.service.CreatePerson(c.Request.Context(), req.Email, req.Password)
	switch {
	case err == nil:
		c.JSON(http.StatusCreated, IDResponse{ID: p.ID})
	case errors.Is(err, ErrEmailAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
	default:
		h.logger.Error("service.CreatePerson failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create person"})
	}
}

// ReadPersonByID godoc
// @Summary      Get Person by ID
// @Description  Fetch a person by their ID
// @Tags         persons
// @Produce      json
// @Param        id       path      int     true  "Person ID"
// @Success      200      {object}  Person
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /persons/{id} [get]
func (h *PersonHandler) ReadPersonByID(c *gin.Context) {
	id, ok := h.bindID(c)
	if !ok {
		return
	}
	p, err := h.service.ReadPersonByID(c.Request.Context(), id)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, p)
	case errors.Is(err, ErrPersonNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "person not found"})
	default:
		h.logger.Error("service.ReadPersonByID failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch person"})
	}
}

// ReadPersonByEmail godoc
// @Summary      Get Person by Email
// @Description  Fetch a person by their email
// @Tags         persons
// @Produce      json
// @Param        email    query     string  true  "Email address"
// @Success      200      {object}  Person
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /persons [get]
func (h *PersonHandler) ReadPersonByEmail(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email query parameter required"})
		return
	}
	p, err := h.service.ReadPersonByEmail(c.Request.Context(), email)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, p)
	case errors.Is(err, ErrPersonNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "person not found"})
	default:
		h.logger.Error("service.ReadPersonByEmail failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch person"})
	}
}

// UpdateEmail godoc
// @Summary      Update Person Email
// @Description  Change a person's email
// @Tags         persons
// @Accept       json
// @Param        id       path      int                 true  "Person ID"
// @Param        payload  body      UpdateEmailRequest  true  "New email payload"
// @Success      204
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Failure      409      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /persons/{id}/email [put]
func (h *PersonHandler) UpdateEmail(c *gin.Context) {
	id, ok := h.bindID(c)
	if !ok {
		return
	}
	var req UpdateEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid update email payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email format"})
		return
	}
	err := h.service.UpdateEmail(c.Request.Context(), id, req.Email)
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrInvalidEmailFormat):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email format"})
	case errors.Is(err, ErrPersonNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "person not found"})
	case errors.Is(err, ErrEmailAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
	default:
		h.logger.Error("service.UpdateEmail failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update email"})
	}
}

// UpdatePassword godoc
// @Summary      Update Person Password
// @Description  Change a person's password
// @Tags         persons
// @Accept       json
// @Param        id       path      int                    true  "Person ID"
// @Param        payload  body      UpdatePasswordRequest  true  "New password payload"
// @Success      204
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /persons/{id}/password [put]
func (h *PersonHandler) UpdatePassword(c *gin.Context) {
	id, ok := h.bindID(c)
	if !ok {
		return
	}
	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid update password payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid password format"})
		return
	}
	err := h.service.UpdatePassword(c.Request.Context(), id, req.Password)
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrPasswordTooShort), errors.Is(err, ErrPasswordNotAlphanumeric):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid password format"})
	case errors.Is(err, ErrPersonNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "person not found"})
	default:
		h.logger.Error("service.UpdatePassword failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update password"})
	}
}

// DeletePerson godoc
// @Summary      Delete Person
// @Description  Remove a person by ID
// @Tags         persons
// @Param        id       path      int   true  "Person ID"
// @Success      204
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /persons/{id} [delete]
func (h *PersonHandler) DeletePerson(c *gin.Context) {
	id, ok := h.bindID(c)
	if !ok {
		return
	}
	err := h.service.DeletePerson(c.Request.Context(), id)
	switch {
	case err == nil:
		c.Status(http.StatusNoContent)
	case errors.Is(err, ErrPersonNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "person not found"})
	default:
		h.logger.Error("service.DeletePerson failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete person"})
	}
}
