package client

import (
	"fmt"

	"hillside/internal/crypto"
	"hillside/internal/models"
)

func (cli *Client) CreateServerHandler(req models.CreateServerRequest) (serverID string, err error) {
	if req.Name == "" {
		return "", ErrCreateServerFailed.WithDetails("Server name cannot be empty")
	}
	if req.Visibility == models.Private && len(req.PasswordHash) == 0 {
		return "", ErrCreateServerFailed.WithDetails("Private servers must have a password")
	}
	hash, salt, err := crypto.HashWithSalt(req.PasswordHash)
	if err != nil {
		return "", ErrCreateServerFailed.WithDetails("Failed to hash password: " + err.Error())
	}
	req.PasswordHash = hash
	req.PasswordSalt = salt

	resp, err := cli.requestCreateServer(req)
	if err != nil {
		return "", ErrCreateServerFailed.WithDetails("Failed to create server: " + err.Error())
	}
	serverID = resp.ServerID
	go cli.refreshServerList()
	return serverID, nil
}

func (cli *Client) JoinServerHandler(serverID string, pass string) error {
	if serverID == "" {
		return ErrJoinServerFailed.WithDetails("Server ID cannot be empty")
	}

	err := cli.requestJoinServer(serverID, pass)
	if err != nil {
		return ErrJoinServerFailed.WithDetails(err.Error())
	}
	cli.SwitchToChatScreen()
	cli.UI.ChatScreen.RoomWrapper.SetTitle(fmt.Sprintf("[ %s ]", cli.GetServerName()))
	go cli.refreshRoomList()
	cli.Session.Log.Logf("Joined server %s", serverID)
	return nil
}
