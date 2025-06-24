package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"hillside/internal/models"
	"hillside/internal/utils"
	"os"

	"golang.org/x/crypto/argon2"
	chacha "golang.org/x/crypto/chacha20poly1305"
)


func (cli *Client) requestServers() (*models.ListServersResponse, error) {
	var listResp models.ListServersResponse
	err := cli.Node.SendRPC("ListServers", models.ListServersRequest{}, &listResp)
	if err != nil {
		return nil, err
	}
	return &listResp, nil

}

func (cli *Client) requestCreateServer(req models.CreateServerRequest) (*models.CreateServerResponse, error) {
	var resp models.CreateServerResponse
	err := cli.Node.SendRPC("CreateServer", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func saveEncryptedSID(sid string, password string) error {
	// Encrypt the SID with the password

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	passKey := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	aead, err := chacha.New(passKey)
	if err != nil {
		return err
	}
	n := make([]byte, aead.NonceSize())
	if _, err := rand.Read(n); err != nil {
		return err
	}
	encryptedSID := aead.Seal(n, n, []byte(sid), nil)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	encPath := homeDir + fmt.Sprintf("/.hillside/%ssid.enc", utils.GenerateRandomID())
	file, err := os.Create(encPath)
	if err != nil {
		return err
	}
	
	defer file.Close()
	data := struct {
		Salt []byte `json:"salt"`
		Nonce []byte `json:"nonce"`
		SID []byte `json:"sid"`
	}{
		Salt: salt,
		Nonce: n,
		SID: encryptedSID,
	}
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return err
	}
	return nil
}