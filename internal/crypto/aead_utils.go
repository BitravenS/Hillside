package crypto

import (
	"crypto/cipher"
	"crypto/rand"
)

func SealAEAD(data []byte, aead cipher.AEAD) ([]byte, error) {
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ct := aead.Seal(nonce, nonce, data, nil)
	return ct, nil
}

func OpenAEAD(encData, nonce []byte, aead cipher.AEAD) ([]byte, error) {
	if nonce == nil {
		nonce := encData[:aead.NonceSize()]
		return aead.Open(nil, nonce, encData[aead.NonceSize():], nil)
	}
	return aead.Open(nil, nonce, encData, nil)
}
