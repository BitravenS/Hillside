package utils

import "errors"

var (
    ProfileNotFound = errors.New("profile not found")
    InvalidPassword = errors.New("invalid password")
    ServerNotFound = errors.New("server not found")
    DuplicateID = errors.New("duplicate ID detected")
)