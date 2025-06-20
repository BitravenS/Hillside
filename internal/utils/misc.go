package utils

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateRandomID() string {
    b := make([]byte, 8)
    rand.Read(b)
    return hex.EncodeToString(b)
}
