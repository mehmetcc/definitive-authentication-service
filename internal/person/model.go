package person

import (
	"time"

	"gorm.io/gorm"
)

// Role represents the set of possible user roles.
// @Description user role type: "admin" or "user"
type Role string

const (
	// Admin has full access
	Admin Role = "admin"
	// User has limited access
	User Role = "user"
)

// Person represents a user in the system.
// swagger:model PersonResponse
// @Description person model
// @Property ID         body integer true  "unique identifier"
// @Property CreatedAt  body string  true  "record creation timestamp"
// @Property UpdatedAt  body string  true  "record update timestamp"
// @Property DeletedAt  body string  false "record deletion timestamp (soft delete)"
// @Property email      body string  true  "unique email address"
// @Property last_seen  body string  true  "last seen timestamp"
// @Property role       body string  true  "user role"
// Person represents a user in the system.
// swagger:model PersonResponse
type Person struct {
	gorm.Model
	// Email address (unique)
	Email string `json:"email" gorm:"uniqueIndex;not null"`
	// Password hash (hidden from JSON)
	Password string `json:"-"`
	// LastSeen indicates last activity time
	LastSeen time.Time `json:"last_seen"`
	// Role of the person
	Role Role `json:"role" gorm:"type:text;default:'user'"`
}

// NewPerson initializes a new Person with default role.
// @Description factory to create Person with default User role
// @Param email path string true "email address"
// @Param password path string true "plaintext password"
// @Success 200 {object} Person
func NewPerson(email, password string) *Person {
	return &Person{
		Email:    email,
		Password: password,
		LastSeen: time.Now().UTC(),
		Role:     User,
	}
}
