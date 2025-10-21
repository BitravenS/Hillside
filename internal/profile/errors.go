package profile

import (
	"hillside/internal/utils"
)

var (
	ErrProfileNotFound = utils.NewHillsideError("profile not found")
	ErrInvalidPassword = utils.NewHillsideError("invalid password")
)
