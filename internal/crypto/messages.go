package crypto

import (
	"github.com/cloudflare/circl/sign/dilithium/mode2"

	chacha "golang.org/x/crypto/chacha20poly1305"
)

func Sign(message, pkBlob []byte) ([]byte, error) {
	sPK, err := DilithiumScheme.UnmarshalBinaryPrivateKey(pkBlob)
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

func ValidateSignature(pubKeyBlob, message, signature []byte) error {
	if pubKeyBlob == nil {
		return ErrSignatureInvalid.WithDetails("public key is nil")
	}
	dilPub, err := DilithiumScheme.UnmarshalBinaryPublicKey(pubKeyBlob)
	if err != nil {
		return ErrSignatureInvalid.WithDetails(err.Error())
	}
	castedPub, ok := dilPub.(*mode2.PublicKey)
	if !ok {
		return ErrSignatureInvalid.WithDetails("invalid public key type, isn't Dilithium2")
	}
	if !mode2.Verify(castedPub, message, signature) {
		return ErrSignatureInvalid.WithDetails("INVALID SIGNATURE")
	}
	return nil
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
