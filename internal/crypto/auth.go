// Package crypto provides cryptographic utilities for the application (e.g., password hashing, encryption, signatures...)
package crypto

import "crypto/sha256"

func HashPassword(password string) []byte {
	passHash := sha256.Sum256([]byte(password))
	return passHash[:]
}
