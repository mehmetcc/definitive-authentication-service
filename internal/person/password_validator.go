package person

import (
	"errors"
	"fmt"
)

const PASSWORD_MINIMUM_LENGTH = 8

var (
	ErrPasswordNotAlphanumeric             = errors.New("password not alphanumeric")
	ErrPasswordDoesNotHaveSpecialCharacter = errors.New("password does not contain special characters")
	ErrPasswordShouldBeNCharacters         = fmt.Errorf("password should be at least %d characters", PASSWORD_MINIMUM_LENGTH)
)

func CheckPassword(password string) error {
	if !checkLength(password) {
		return ErrPasswordShouldBeNCharacters
	}
	if !checkAlphanumeric(password) {
		return ErrPasswordNotAlphanumeric
	}
	if !checkSpecialCharacter(password) {
		return ErrPasswordDoesNotHaveSpecialCharacter
	}
	return nil
}

func checkLength(password string) bool {
	return len(password) >= 8
}

func checkAlphanumeric(password string) bool {
	hasLetter := false
	hasDigit := false
	for _, c := range password {
		if ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') {
			hasLetter = true
		}
		if '0' <= c && c <= '9' {
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}

func checkSpecialCharacter(password string) bool {
	special := "!@#$%^&*()-_=+[]{}|;:'\",.<>?/`~"
	for _, c := range password {
		if containsRune(special, c) {
			return true
		}
	}
	return false
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}
