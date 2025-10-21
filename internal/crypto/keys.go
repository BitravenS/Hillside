package crypto

import (
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"

	"github.com/cloudflare/circl/kem"
	kyber "github.com/cloudflare/circl/kem/kyber/kyber1024"
	"github.com/cloudflare/circl/sign/dilithium/mode2"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/crypto/argon2"
	chacha "golang.org/x/crypto/chacha20poly1305"
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

func GenPasskeys(pass string, salt []byte) ([]byte, []byte, []byte, error) {
	if salt == nil {
		salt := make([]byte, 16)
		if _, err := rand.Read(salt); err != nil {
			return nil, nil, nil, err
		}
	}
	passkey := argon2.IDKey([]byte(pass), salt, 1, 64*1024, 4, 32)
	unlocker := argon2.IDKey([]byte(pass), salt, 3, 8*1024, 2, 32)
	return passkey, unlocker, salt, nil
}

func GenKEMKey() ([]byte, []byte, error) {
	pub, priv, err := kyber.GenerateKeyPair(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	pubBytes, err := pub.MarshalBinary()
	if err != nil {
		return nil, nil, err
	}
	privBytes, err := priv.MarshalBinary()
	if err != nil {
		return nil, nil, err
	}
	return pubBytes, privBytes, nil
}

func GenSignKey() ([]byte, []byte, error) {
	pub, priv, err := mode2.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	pubBytes, err := pub.MarshalBinary()
	if err != nil {
		return nil, nil, err
	}
	privBytes, err := priv.MarshalBinary()
	if err != nil {
		return nil, nil, err
	}
	return pubBytes, privBytes, nil
}

func GenP2PKey() ([]byte, []byte, string, error) {
	priv, pub, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
	if err != nil {
		return nil, nil, "", err
	}
	privBytes, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, nil, "", err
	}
	pubBytes, err := crypto.MarshalPublicKey(pub)
	if err != nil {
		return nil, nil, "", err
	}
	peerID, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		return nil, nil, "", err
	}
	return privBytes, pubBytes, peerID.String(), nil
}

func DeriveChaChaKey(passkey []byte) (cipher.AEAD, error) {
	aead, err := chacha.New(passkey)
	if err != nil {
		return nil, err
	}

	return aead, err
}

func DeriveSignKey(privBlob []byte) (*mode2.PrivateKey, *mode2.PublicKey, []byte, error) {
	privKey, err := DilithiumScheme.UnmarshalBinaryPrivateKey(privBlob)
	if err != nil {
		return nil, nil, nil, err
	}
	pubKey, ok := privKey.Public().(*mode2.PublicKey)
	if !ok {
		return nil, nil, nil, ErrBadKey.WithDetails("invalid dilithium private key")
	}
	privKeyCast, ok := privKey.(*mode2.PrivateKey)
	if !ok {
		return nil, nil, nil, ErrBadKey.WithDetails("invalid dilithium private key")
	}
	pubKeyBytes, err := pubKey.MarshalBinary()
	if err != nil {
		return nil, nil, nil, err
	}

	return privKeyCast, pubKey, pubKeyBytes, nil

}

func DeriveKEMKey(privBlob []byte) (*kem.PrivateKey, *kyber.PublicKey, []byte, error) {
	privKey, err := KyberScheme.UnmarshalBinaryPrivateKey(privBlob)
	if err != nil {
		return nil, nil, nil, err
	}
	pubKey, ok := privKey.Public().(*kyber.PublicKey)
	if !ok {
		return nil, nil, nil, ErrBadKey
	}
	pubKeyBytes, err := pubKey.MarshalBinary()
	if err != nil {
		return nil, nil, nil, err
	}
	return &privKey, pubKey, pubKeyBytes, nil
}

func DeriveP2PKey(privBlob []byte) (crypto.PrivKey, crypto.PubKey, []byte, error) {
	privKey, err := crypto.UnmarshalPrivateKey(privBlob)
	if err != nil {
		return nil, nil, nil, err
	}
	pubKey := privKey.GetPublic()
	pubKeyBytes, err := crypto.MarshalPublicKey(pubKey)
	return privKey, pubKey, pubKeyBytes, err
}
