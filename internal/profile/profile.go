package profile

import (
	"crypto/hmac"
	"crypto/rand"
	"encoding/json"
	"os"

	"hillside/internal/models"
	"hillside/internal/utils"

	kyber "github.com/cloudflare/circl/kem/kyber/kyber1024"
	dil2 "github.com/cloudflare/circl/sign/dilithium/mode2"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/crypto/argon2"
	chacha "golang.org/x/crypto/chacha20poly1305"
)

type Profile struct {
    Username        string `json:"username"`
    PasswordSalt    []byte `json:"password_salt"`
    PasswordChecksum    []byte `json:"password_checksum"`
    DilithiumPrivEnc []byte `json:"dilithium_priv_enc"`	// encrypted w/ password
    KyberPrivEnc    []byte `json:"kyber_priv_enc"`    // encrypted w/ password
	Libp2pPrivEnc   []byte `json:"libp2p_priv_enc"`   // encrypted w/ password
    PeerID          string `json:"peer_id"`
}

func GenerateProfile(username string, pass string) (*Profile, error) {

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	passKey := argon2.IDKey([]byte(pass), salt, 1, 64*1024, 4, 32)
	unlocker := argon2.IDKey([]byte(pass), salt, 3, 8*1024, 2, 32)

	// Key generation
	_, dilPriv, err := dil2.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	_, kemPriv, err := kyber.GenerateKeyPair(rand.Reader)
    if err != nil {
        return nil, err
    }

	libPriv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
    if err != nil {
        return nil, err
    }

	pid, err := peer.IDFromPrivateKey(libPriv)
	if err != nil {
		return nil, err
	}


	aead, err := chacha.New(passKey)
	if err != nil {
		return nil, err
	}

	n1 := make([]byte, aead.NonceSize())
	if _, err := rand.Read(n1); err != nil {
		return nil, err
	}

	dilPrivBytes, err := dilPriv.MarshalBinary()
	if err != nil {
		return nil, err
	}
	dilEnc := aead.Seal(n1, n1, dilPrivBytes, nil)

	kemPrivBytes, err := kemPriv.MarshalBinary()
	if err != nil {
		return nil, err
	}
	n2 := make([]byte, aead.NonceSize())
	if _, err := rand.Read(n2); err != nil {
		return nil, err
	}

	kemEnc := aead.Seal(n2, n2, kemPrivBytes, nil)

	libPrivBytes, err := crypto.MarshalPrivateKey(libPriv)
	if err != nil {
		return nil, err
	}
	n3 := make([]byte, aead.NonceSize())
	if _, err := rand.Read(n3); err != nil {
		return nil, err
	}
	libEnc := aead.Seal(n3, n3, libPrivBytes, nil)


	prof := &Profile{
		Username:        username,
		PasswordSalt:    salt,
		PasswordChecksum:    unlocker,
		DilithiumPrivEnc: dilEnc,
		KyberPrivEnc:    kemEnc,
		Libp2pPrivEnc:   libEnc,
		PeerID:          pid.String(),
	}

	// Saving to disk
	
	profilePath, err := createProfilePath(username)
	if err != nil {
		return nil, err
	}
	file, err := os.Create(*profilePath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(prof); err != nil {
		return nil, err
	}

	return prof, nil
}

func LoadProfile(usrname string, pass string, path string) (*models.Keybag, *models.User, error) {
	profilePath, err := getProfilePath(usrname, path)
	if err != nil {
		return nil,nil, err
	}

	file, err := os.Open(*profilePath)
	if err != nil {
		return nil,nil, err
	}

	defer file.Close()

	var prof Profile
	dec := json.NewDecoder(file)
	if err := dec.Decode(&prof); err != nil {
		return nil,nil, err
	}
	passKey := argon2.IDKey([]byte(pass), prof.PasswordSalt, 1, 64*1024, 4, 32)
	check := argon2.IDKey([]byte(pass), prof.PasswordSalt, 3, 8*1024, 2, 32)
	if !hmac.Equal(check, prof.PasswordChecksum) {
		return nil,nil, utils.InvalidPassword
	}

	aead, err := chacha.New(passKey)
	if err != nil {
		return nil,nil, err
	}

    n1 := prof.DilithiumPrivEnc[:aead.NonceSize()]
	dilPrivBytes, err := aead.Open(nil, n1, prof.DilithiumPrivEnc[aead.NonceSize():], nil)
	if err != nil {
		return nil,nil, err
	}

    n2 := prof.KyberPrivEnc[:aead.NonceSize()]
	kemPrivBytes, err := aead.Open(nil, n2, prof.KyberPrivEnc[aead.NonceSize():], nil)
	if err != nil {
		return nil,nil, err
	}

    n3 := prof.Libp2pPrivEnc[:aead.NonceSize()]
	libPrivBytes, err := aead.Open(nil, n3, prof.Libp2pPrivEnc[aead.NonceSize():], nil)
	if err != nil {
		return nil,nil, err
	}
	
	dilPriv, err := dil2.Scheme().UnmarshalBinaryPrivateKey(dilPrivBytes)
	
	if err != nil {
		return nil,nil, err
	}

	kemPriv, err := kyber.Scheme().UnmarshalBinaryPrivateKey(kemPrivBytes)
	if err != nil {
		return nil,nil, err
	}

	libPriv, err := crypto.UnmarshalPrivateKey(libPrivBytes)
	if err != nil {
		return nil,nil, err
	}

	kb := &models.Keybag{
		DilithiumPriv: dilPriv,
		KyberPriv: kemPriv,
		Libp2pPriv: libPriv,
	}
	usr := &models.User{
		DilithiumPub: dilPriv.Public().(*dil2.PublicKey),
		KyberPub: kemPriv.Public().(*kyber.PublicKey),
		Libp2pPub: libPriv.GetPublic(),
		PeerID: prof.PeerID,
		Username: prof.Username,
	}

	return kb, usr, nil
}