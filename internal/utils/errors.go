package utils

import (
	"errors"
	"fmt"
)

func ThemeError(message string) error {
	return fmt.Errorf("theme error: %s", message)
}

func CreateServerError(message string) error {
	return fmt.Errorf("create server error: %s", message)
}

func CreateRoomError(message string) error {
	return fmt.Errorf("create room error: %s", message)
}

func JoinServerError(message string) error {
	return fmt.Errorf("join server error: %s", message)
}
func JoinRoomError(message string) error {
	return fmt.Errorf("join room error: %s", message)
}

func ValidationError(message string) error {
	return fmt.Errorf("validation error: %s", message)
}

func SecurityError(message string) error {
	return fmt.Errorf("security error: %s", message)
}

func PQaeadError(message string) error {
	return fmt.Errorf("pq-aead error: %s", message)
}

func IsValidationError(err error) bool {
	return errors.Is(err, ValidationError(""))
}

func IsSecurityError(err error) bool {
	return errors.Is(err, SecurityError(""))
}

func SendMessageError(message string) error {
	return fmt.Errorf("send message error: %s", message)
}

type HillsideError struct {
	base    string
	details string
}

func NewHillsideError(base string) *HillsideError {
	return &HillsideError{base: "networking: " + base}
}

func (e *HillsideError) WithDetails(details string) *HillsideError {
	return &HillsideError{
		base:    e.base,
		details: details,
	}
}

func (e *HillsideError) Error() string {
	if e.details != "" {
		return fmt.Sprintf("%s: %s", e.base, e.details)
	}
	return e.base
}

func (e *HillsideError) Is(target error) bool {
	if target == nil {
		return false
	}
	return e.base == target.Error()
}
