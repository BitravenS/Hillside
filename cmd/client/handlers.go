package main

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"hillside/internal/models"
	"hillside/internal/profile"
	"hillside/internal/utils"
	"time"
)
type loginSession struct {
	Username string
	Password string
	Path string
}

func (cli *Client) loginHandler(username string, password string, hub string) {

	if username == "" || password == "" {
		cli.UI.ShowError("Error", "Username and password cannot be empty","OK", 0, nil)
		return
	}
	kb,usr, err := profile.LoadProfile(username, password, "")
	if err != nil {
			cli.UI.ShowError("Login failed", err.Error(), "Retry", 0, nil)
		return
	}
	cli.UI.ShowToast("Login successful", 3*time.Second,nil)
	cli.User = usr
	cli.Keybag = kb
	cli.Node.PK = kb.Libp2pPriv
	err = cli.Node.InitNode()
	if err != nil {
		cli.UI.ShowError("Node initialization failed", err.Error(),"OK", 0, nil)
		return
	}
	

	cli.SwitchToBrowseScreen(hub)

	return

}

func (cli *Client) createUserHandler(username string, password string, hub string) {

	if username == "" || password == "" {
		cli.UI.ShowError("Error", "Username and password cannot be empty","OK", 0, nil)
		return
	}
	prof, err := profile.GenerateProfile(username, password)
	if err != nil {
		cli.UI.ShowError("Create user failed", err.Error(), "OK", 0, nil)
		return
	}
	cli.UI.ShowToast("User created successfully! Welcome "+ prof.Username, 3*time.Second,nil)
	kb,usr, err := profile.LoadProfile(username, password, "")
	if err != nil {
			cli.UI.ShowError("Login failed", err.Error(), "Retry", 0, nil)
		return
	}
	cli.User = usr
	cli.Keybag = kb
	cli.Node.PK = kb.Libp2pPriv
	err = cli.Node.InitNode()
	if err != nil {
		cli.UI.ShowError("Node initialization failed", err.Error(),"OK", 0, nil)
		return
	}
	cli.SwitchToBrowseScreen(hub)


	return
}

func (cli *Client) SwitchToBrowseScreen(hub string) {
	cli.UI.Pages.SwitchToPage("browse")

	cli.Node.HubAddr = hub
	cli.UI.BrowseScreen.SetHub(hub)

	go cli.refreshServerList()
}

func (cli *Client) createServerHandler(request models.CreateServerRequest) (serverID string, err error) {
	if request.Name == "" {
		return "", utils.CreateServerError("Server name cannot be empty")
	}
	if request.Visibility == models.Private && len(request.PasswordHash) == 0 {
		return "" , utils.CreateServerError("Private servers must have a password")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", utils.CreateServerError("Failed to generate salt: " + err.Error())
	}
	request.PasswordSalt = salt
	hash := sha256.Sum256(request.PasswordHash)
	request.PasswordHash = hash[:]
	resp, err := cli.requestCreateServer(request)
	if err != nil {
		return "", utils.CreateServerError("Failed to create server: " + err.Error())
	}
	serverID = resp.ServerID
	go cli.refreshServerList()
	return serverID, nil
}

func (cli *Client) createRoomHandler(req models.CreateRoomRequest) (string, error) {
	if req.RoomName == "" {
		return "", utils.CreateRoomError("Room name cannot be empty")
	}
	if req.Visibility == models.Private && len(req.PasswordHash) == 0 {
		return "", utils.CreateRoomError("Private rooms must have a password")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", utils.CreateRoomError("Failed to generate salt: " + err.Error())
	}
	req.PasswordSalt = salt
	hash := sha256.Sum256(req.PasswordHash)
	req.PasswordHash = hash[:]
	resp, err := cli.requestCreateRoom(req)
	if err != nil {
		return "", utils.CreateRoomError("Failed to create room: " + err.Error())
	}
	go cli.refreshRoomList()
	return resp.RoomID, nil
}


func (cli *Client) joinServerHandler(serverID string, pass string) error {
	if serverID == "" {
		return utils.JoinServerError("Server ID cannot be empty")
	}


	err := cli.requestJoinServer(serverID, pass)
	if err != nil {
		return utils.JoinServerError(err.Error())
	}
	cli.UI.Pages.SwitchToPage("chat")
	cli.UI.ChatScreen.roomWrapper.SetTitle(fmt.Sprintf("[ %s ]", cli.Session.Server.Name))
	go cli.refreshRoomList()
	return nil
}

func (cli *Client) joinRoomHandler(roomID string, pass string) error {
	if roomID == "" {
		return utils.JoinRoomError("Server ID and Room ID cannot be empty")
	}
	err := cli.requestJoinRoom(roomID, pass)
	if err != nil {
		return utils.JoinRoomError(err.Error())
	}
	cli.UI.ChatScreen.chatSection.SetTitle(fmt.Sprintf("[ %s ]", cli.Session.Room.Name))
	go cli.refreshRoomList()
	return nil
}