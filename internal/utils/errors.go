package utils

import "errors"

var (
    ProfileNotFound = errors.New("profile not found")
    InvalidPassword = errors.New("invalid password")
)