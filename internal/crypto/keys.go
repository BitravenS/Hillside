package crypto

import (
	"crypto/rand"
	"crypto/sha256"
)

func GenerateRoomKey() ([]byte, []byte, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return nil, nil, err
	}
	hash := sha256.Sum256(key)
	return key, hash[:], nil
}
