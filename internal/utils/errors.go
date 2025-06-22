package utils

import (
	"errors"
	"fmt"
)

var (
    ProfileNotFound = errors.New("profile not found")
    InvalidPassword = errors.New("invalid password")
    ServerNotFound = errors.New("server not found")
    DuplicateID = errors.New("duplicate ID detected")
)

func ThemeError(message string) error {
    return fmt.Errorf("theme error: %s", message)
}