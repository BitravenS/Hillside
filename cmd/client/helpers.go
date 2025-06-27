package main

import (
	"crypto/rand"
	"crypto/sha256"
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

func (cli *Client) requestCreateRoom(req models.CreateRoomRequest) (*models.CreateRoomResponse, error) {
	var resp models.CreateRoomResponse
	err := cli.Node.SendRPC("CreateRoom", req, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
	}
	return &resp, nil
}

func (cli *Client) requestRooms(serverID string) (*models.ListRoomsResponse, error) {
	var roomsResp models.ListRoomsResponse
	err := cli.Node.SendRPC("ListRooms", models.ListRoomsRequest{ServerID: serverID}, &roomsResp)
	if err != nil {
		return nil, err
	}
	return &roomsResp, nil
}

func (cli *Client) getServerName() string {
	if cli.Session == nil {
		return fmt.Errorf("Session is not initialized").Error()
	}
	return cli.Session.Server.Name
}

func (cli *Client) getServerId() string {
	if cli.Session == nil {
		return fmt.Errorf("Session is not initialized").Error()
	}
	return cli.Session.Server.ID
}

func (cli *Client) getRoomName() string {
	if cli.Session == nil {
		return fmt.Errorf("Session is not initialized").Error()
	}
	if cli.Session.Room == nil {
		return ""
	}
	return cli.Session.Room.Name
}

func (cli *Client) requestJoinServer(serverID string, pass string) error {
	var resp models.JoinServerResponse

	passwordHash := sha256.Sum256([]byte(pass))
	err := cli.Node.SendRPC("JoinServer", models.JoinServerRequest{ServerID: serverID, PasswordHash: passwordHash[:]}, &resp)
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return fmt.Errorf("failed to join server: %s", resp.Error)
	}

	cli.Session.Server = resp.Server

	return nil
}

func (cli *Client) requestJoinRoom(roomID, pass string) error {
	var resp models.JoinRoomResponse
	passwordHash := sha256.Sum256([]byte(pass))
	sid := cli.Session.Server.ID
	if sid == "" {
		return fmt.Errorf("no server joined, cannot join room")
	}
	err := cli.Node.SendRPC("JoinRoom", models.JoinRoomRequest{ServerID: sid, RoomID: roomID, PasswordHash: passwordHash[:], Sender: *cli.User}, &resp)
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}

	cli.Session.Room = resp.Room

	return nil
}

func (cli *Client) requestListRoomMembers() (*models.ListRoomMembersResponse, error) {
	if cli.Session == nil || cli.Session.Room == nil {
		return nil, fmt.Errorf("no room joined, cannot list members")
	}
	var resp models.ListRoomMembersResponse
	err := cli.Node.SendRPC("ListRoomMembers", models.ListRoomMembersRequest{ServerID: cli.Session.Server.ID, RoomID: cli.Session.Room.ID}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
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




func (cli *Client) Shutdown() error {
	if cli.Node.Host != nil {
		_ = cli.Node.Host.Close() // close the libp2p host
	}
	if cli.Node.DHT != nil {
		cli.Node.DHT.Close() // close the DHT
	}
	// Close individual topics if they exist
	if cli.Node.Topics.ChatTopic != nil {
		_ = cli.Node.Topics.ChatTopic.Close()
	}
	if cli.Node.Topics.RekeyTopic != nil {
		_ = cli.Node.Topics.RekeyTopic.Close()
	}
	// Add any other topics that need to be closed
	
	if cli.Node.Ctx != nil {
		cli.Node.Ctx.Done() // cancel the context
	}

	return nil
}