package profile

import (
	"encoding/json"
	"os"

	"hillside/internal/crypto"
	"hillside/internal/models"
	"hillside/internal/utils"
)

func GenerateProfile(username string, pass string) (*Profile, error) {

	passKey, unlocker, salt, err := crypto.GenPasskeys(pass, nil) // idk what the unlocker is for tbh
	if err != nil {
		return nil, ErrProfileCreation.WithDetails(err.Error())
	}

	// Key generation
	_, dilPrivBytes, err := crypto.GenSignKey()
	if err != nil {
		return nil, ErrProfileCreation.WithDetails(err.Error())
	}

	_, kemPrivBytes, err := crypto.GenKEMKey()
	if err != nil {
		return nil, ErrProfileCreation.WithDetails(err.Error())
	}

	libPrivBytes, _, pid, err := crypto.GenP2PKey()
	if err != nil {
		return nil, ErrProfileCreation.WithDetails(err.Error())
	}

	aead, err := crypto.DeriveChaChaKey(passKey)
	if err != nil {
		return nil, ErrProfileCreation.WithDetails(err.Error())
	}

	dilEnc, err := crypto.SealAEAD(dilPrivBytes, aead)
	if err != nil {
		return nil, ErrProfileCreation.WithDetails(err.Error())
	}

	kemEnc, err := crypto.SealAEAD(kemPrivBytes, aead)
	if err != nil {
		return nil, ErrProfileCreation.WithDetails(err.Error())
	}

	libEnc, err := crypto.SealAEAD(libPrivBytes, aead)
	if err != nil {
		return nil, ErrProfileCreation.WithDetails(err.Error())
	}

	prof := &Profile{
		Username:         username,
		PasswordSalt:     salt,
		PasswordChecksum: unlocker,
		DilithiumPrivEnc: dilEnc,
		KyberPrivEnc:     kemEnc,
		Libp2pPrivEnc:    libEnc,
		PeerID:           pid,
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
	if err := enc.Encode(prof); err != nil {
		return nil, err
	}

	return prof, nil
}

func LoadProfile(usrname string, pass string, path string) (*models.Keybag, *models.User, error) {
	profilePath, err := getProfilePath(usrname, path)
	if err != nil {
		return nil, nil, ErrProfileLoad.WithDetails(err.Error())
	}

	file, err := os.Open(*profilePath)
	if err != nil {
		return nil, nil, ErrProfileNotFound
	}

	defer file.Close()

	var prof Profile
	dec := json.NewDecoder(file)
	if err := dec.Decode(&prof); err != nil {
		return nil, nil, err
	}
	passKey, _, _, err := crypto.GenPasskeys(pass, prof.PasswordSalt)
	if err != nil {
		return nil, nil, ErrProfileLoad.WithDetails(err.Error())
	}

	aead, err := crypto.DeriveChaChaKey(passKey)
	if err != nil {
		return nil, nil, ErrProfileLoad.WithDetails(err.Error())
	}

	dilPrivBytes, err := crypto.OpenAEAD(prof.DilithiumPrivEnc, aead)
	if err != nil {
		return nil, nil, ErrInvalidPassword.WithDetails(err.Error())
	}

	kemPrivBytes, err := crypto.OpenAEAD(prof.KyberPrivEnc, aead)
	if err != nil {
		return nil, nil, ErrInvalidPassword.WithDetails(err.Error())
	}

	libPrivBytes, err := crypto.OpenAEAD(prof.Libp2pPrivEnc, aead)
	if err != nil {
		return nil, nil, ErrInvalidPassword.WithDetails(err.Error())
	}

	_, _, dilPubBytes, err := crypto.DeriveSignKey(dilPrivBytes)
	if err != nil {
		return nil, nil, ErrProfileLoad.WithDetails("Profile file is corrupted: " + err.Error())
	}

	_, _, kyberPubBytes, err := crypto.DeriveKEMKey(kemPrivBytes)
	if err != nil {
		return nil, nil, ErrProfileLoad.WithDetails("Profile file is corrupted: " + err.Error())
	}

	libPriv, _, libPubBytes, err := crypto.DeriveP2PKey(libPrivBytes)
	if err != nil {
		return nil, nil, ErrProfileLoad.WithDetails("Profile file is corrupted: " + err.Error())
	}

	kb := &models.Keybag{
		DilithiumPriv: dilPrivBytes,
		KyberPriv:     kemPrivBytes,
		Libp2pPriv:    libPriv,
	}

	usr := &models.User{
		DilithiumPub:   dilPubBytes,
		KyberPub:       kyberPubBytes,
		Libp2pPub:      libPubBytes,
		PeerID:         prof.PeerID,
		Username:       prof.Username,
		PreferredColor: utils.GenerateRandomColor(),
	}

	return kb, usr, nil
}
