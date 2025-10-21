package crypto

import (
	"github.com/cloudflare/circl/sign/dilithium/mode2"
	"github.com/cloudflare/circl/sign/schemes"

	chacha "golang.org/x/crypto/chacha20poly1305"
)

func Sign(message, pkBlob []byte) ([]byte, error) {
	sch := schemes.ByName("Dilithium2")
	sPK, err := sch.UnmarshalBinaryPrivateKey(pkBlob)
	if err != nil {
		return nil, ErrSigningFailed.WithDetails(err.Error())
	}
	sigPK, ok := sPK.(*mode2.PrivateKey)
	if !ok {
		return nil, ErrSigningFailed.WithDetails("invalid private key type")
	}
	sig := make([]byte, mode2.SignatureSize)
	if sigPK != nil {
		mode2.SignTo(sigPK, message, sig)
	}
	return sig, nil
}

func EncryptMessage(r *RoomRatchet, plaintext []byte) (ciphertext, nonce []byte, err error) {
	key, nonce, err := r.NextKey()
	if err != nil {
		return nil, nil, ErrEncryptionFailed.WithDetails(err.Error())
	}
	aead, err := chacha.New(key)
	if err != nil {
		return nil, nil, ErrEncryptionFailed.WithDetails(err.Error())
	}
	ct := aead.Seal(nil, nonce, plaintext, nil)
	return ct, nonce, nil

}
