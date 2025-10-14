package utils

import (
	"errors"
	"fmt"
)

var (
	ProfileNotFound   = errors.New("profile not found")
	HistoryDBNotFound = errors.New("history db not found")
	InvalidPassword   = errors.New("invalid password")
	ServerNotFound    = errors.New("server not found")
	RoomNotFound      = errors.New("room not found")
	DuplicateID       = errors.New("duplicate ID detected")
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
