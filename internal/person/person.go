package person

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"time"

	"github.com/mehmetcc/definitive-authentication-service/config"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrRoleAdminNotAllowed = errors.New("cannot create admin user")
	ErrRoleInvalid         = errors.New("invalid role")
	ErrEmailInvalid        = errors.New("invalid format")
	ErrPasswordTooShort    = errors.New("must be at least 8 characters long")
)

// Person represents a user in the system.
// swagger:model Person
//
// Fields:
//   - id: unique identifier for the person
//   - email: email address of the person
//   - role: role of the person (admin or user)
//   - created_at: timestamp when the person was created
//   - updated_at: timestamp when the person was last updated
//   - is_active: indicates if the person is active
// swagger:parameters Person
// ---
// id: int64
// email: string
// role: string
// created_at: string
// updated_at: string
// is_active: boolean
//
// swagger:response PersonResponse
// schema: Person

type Person struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email" gorm:"uniqueIndex"`
	Password  []byte    `json:"-"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	IsActive  bool      `json:"is_active"`
}

// NewPerson creates a new Person with the given email, password, and role.
// swagger:operation POST /persons persons createPerson
// ---
// summary: Create a new person
// consumes:
// - application/json
// produces:
// - application/json
// parameters:
//   - name: body
//     in: body
//     description: Person to create
//     required: true
//     schema:
//     $ref: '#/definitions/Person'
//
// responses:
//
//	"201":
//	  description: person created successfully
//	  schema:
//	    $ref: '#/definitions/Person'
//	"400":
//	  description: invalid input
//	  schema:
//	    type: object
//	    properties:
//	      error:
//	        type: string
func NewPerson(email, password string) (*Person, error) {
	if err := validate(email, password); err != nil {
		return nil, err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), config.AppConfig.Encryption.Cost)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &Person{
		Email:     email,
		Password:  hashed,
		Role:      User, // default to user role
		CreatedAt: now,
		UpdatedAt: now,
		IsActive:  true,
	}, nil
}

func validate(email, password string) error {
	for _, err := range []error{
		ValidateEmail(email),
		ValidatePassword(password),
	} {
		if err != nil {
			return err
		}
	}
	return nil
}

// ValidateEmail ensures the email is in a valid format.
func ValidateEmail(email string) error {
	if _, err := mail.ParseAddress(email); err != nil {
		return ErrEmailInvalid
	}
	return nil
}

// ValidatePassword ensures the password meets minimum length requirements.
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}
	return nil
}

// Role defines the set of possible user roles.
// swagger:model Role
// enum: [admin, user]
type Role int

const (
	// Admin is the administrator role.
	Admin Role = iota
	// User is the regular user role.
	User
)

func (r Role) String() string {
	switch r {
	case Admin:
		return "admin"
	case User:
		return "user"
	default:
		return "unknown"
	}
}

func (r Role) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

func (r *Role) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	switch s {
	case "admin":
		*r = Admin
	case "user":
		*r = User
	default:
		return fmt.Errorf("invalid role %q", s)
	}
	return nil
}
