package person

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mehmetcc/definitive-authentication-service/config"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// CreatePersonRequest represents the payload to create a new Person.
// swagger:model CreatePersonRequest
type CreatePersonRequest struct {
	// Email address of the person
	// required: true
	// format: email
	Email string `json:"email"`

	// Plain-text password for the person
	// required: true
	// min length: 8
	Password string `json:"password"`
}

// UpdatePersonRequest represents the payload to update an existing Person.
// swagger:model UpdatePersonRequest
type UpdatePersonRequest struct {
	// New email address of the person
	// format: email
	Email *string `json:"email,omitempty"`

	// New plain-text password for the person
	// min length: 8
	Password *string `json:"password,omitempty"`

	// New role for the person (admin or user)
	// enum: [admin,user]
	// example: user
	Role *Role `json:"role,omitempty"`
}

// HTTPError represents an error response.
// swagger:model HTTPError
type HTTPError struct {
	// Error message
	Message string `json:"error"`
}

// PersonHandler handles HTTP requests for Person CRUD operations.
type PersonHandler struct {
	repo   PersonRepository
	logger *zap.Logger
}

// NewPersonHandler creates a new PersonHandler.
func NewPersonHandler(r PersonRepository, l *zap.Logger) *PersonHandler {
	return &PersonHandler{repo: r, logger: l}
}

// RegisterRoutes registers /persons endpoints (BasePath is applied at the router root).
func (h *PersonHandler) RegisterRoutes(r chi.Router) {
	r.Route("/persons", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/{id}", h.GetByID)
		r.Get("/email/{email}", h.GetByEmail)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
	})
}

func (h *PersonHandler) errorResponse(w http.ResponseWriter, status int, msg string) {
	h.logger.Warn(msg)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(HTTPError{Message: msg})
}

func (h *PersonHandler) writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Create creates a new Person.
// @Summary Create a new person
// @Description Create a new person with email, password, and role.
// @Tags persons
// @Accept json
// @Produce json
// @Param body body CreatePersonRequest true "Create person payload"
// @Success 201 {object} Person
// @Failure 400 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /persons [post]
func (h *PersonHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreatePersonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	p, err := NewPerson(req.Email, req.Password)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, fmt.Sprintf("validation error: %v", err))
		return
	}

	if err := h.repo.Create(r.Context(), p); err != nil {
		// special-case unique‐email
		if errors.Is(err, ErrEmailExists) {
			h.errorResponse(w, http.StatusConflict, "email already exists")
			return
		}
		h.logger.Error("failed to create person", zap.Error(err))
		h.errorResponse(w, http.StatusInternalServerError, "unable to create person")
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/persons/%d", p.ID))
	h.writeJSON(w, http.StatusCreated, p)
}

// GetByID retrieves a Person by ID.
// @Summary Retrieve a person by ID
// @Description Get a person by their unique ID.
// @Tags persons
// @Produce json
// @Param id path int true "Person ID"
// @Success 200 {object} Person
// @Failure 400 {object} HTTPError
// @Failure 404 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /persons/{id} [get]
func (h *PersonHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid id")
		return
	}

	p, err := h.repo.FindByID(r.Context(), id)
	if err == ErrNotFound {
		h.errorResponse(w, http.StatusNotFound, "person not found")
		return
	} else if err != nil {
		h.logger.Error("error fetching person", zap.Error(err))
		h.errorResponse(w, http.StatusInternalServerError, "error fetching person")
		return
	}

	h.writeJSON(w, http.StatusOK, p)
}

// GetByEmail retrieves a Person by email.
// @Summary Retrieve a person by email
// @Description Get a person by their email address.
// @Tags persons
// @Produce json
// @Param email path string true "Person Email"
// @Success 200 {object} Person
// @Failure 404 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /persons/email/{email} [get]
func (h *PersonHandler) GetByEmail(w http.ResponseWriter, r *http.Request) {
	email := chi.URLParam(r, "email")
	p, err := h.repo.FindByEmail(r.Context(), email)
	if err == ErrNotFound {
		h.errorResponse(w, http.StatusNotFound, "person not found")
		return
	} else if err != nil {
		h.logger.Error("error fetching person by email", zap.Error(err))
		h.errorResponse(w, http.StatusInternalServerError, "error fetching person")
		return
	}

	h.writeJSON(w, http.StatusOK, p)
}

// Update updates an existing Person.
// @Summary Update a person
// @Description Update a person's email or password.
// @Tags persons
// @Accept json
// @Produce json
// @Param id path int true "Person ID"
// @Param body body UpdatePersonRequest true "Update person payload"
// @Success 204 {nil} void
// @Failure 400 {object} HTTPError
// @Failure 404 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /persons/{id} [put]
func (h *PersonHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid id")
		return
	}

	existing, err := h.repo.FindByID(r.Context(), id)
	if err == ErrNotFound {
		h.errorResponse(w, http.StatusNotFound, "person not found")
		return
	} else if err != nil {
		h.logger.Error("error fetching person for update", zap.Error(err))
		h.errorResponse(w, http.StatusInternalServerError, "error fetching person")
		return
	}

	var req UpdatePersonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	if req.Email != nil {
		existing.Email = *req.Email
	}
	if req.Password != nil {
		hashed, err := bcrypt.GenerateFromPassword([]byte(*req.Password), config.AppConfig.Encryption.Cost)
		if err != nil {
			h.errorResponse(w, http.StatusInternalServerError, "error hashing password")
			return
		}
		existing.Password = hashed
	}
	if req.Role != nil {
		existing.Role = *req.Role
	}
	existing.UpdatedAt = time.Now().UTC()

	if err := h.repo.Update(r.Context(), existing); err != nil {
		if errors.Is(err, ErrEmailExists) {
			h.errorResponse(w, http.StatusConflict, "email already exists")
			return
		}
		h.logger.Error("error updating person", zap.Error(err))
		h.errorResponse(w, http.StatusInternalServerError, "error updating person")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Delete removes a Person by ID.
// @Summary Delete a person
// @Description Soft-delete a person by ID.
// @Tags persons
// @Produce json
// @Param id path int true "Person ID"
// @Success 204 {nil} void
// @Failure 400 {object} HTTPError
// @Failure 404 {object} HTTPError
// @Failure 500 {object} HTTPError
// @Router /persons/{id} [delete]
func (h *PersonHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err == ErrNotFound {
		h.errorResponse(w, http.StatusNotFound, "person not found")
		return
	} else if err != nil {
		h.logger.Error("error deleting person", zap.Error(err))
		h.errorResponse(w, http.StatusInternalServerError, "error deleting person")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
