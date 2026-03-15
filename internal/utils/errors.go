package utils

import (
	"fmt"
)

func ThemeError(message string) error {
	return fmt.Errorf("theme error: %s", message)
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
