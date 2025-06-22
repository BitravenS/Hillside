package main

import (
	"hillside/internal/profile"
	"time"
)
type loginSession struct {
	Username string
	Password string
	Path string
}

func (cli *Client) loginHandler(username string, password string, hub string) {

	if username == "" || password == "" {
		cli.UI.ShowToast("Username and password cannot be empty", 0, nil)
		return
	}
	kb,usr, err := profile.LoadProfile(username, password, "")
	if err != nil {
			cli.UI.ShowToast("Login failed: "+err.Error(),0, nil)
		return
	}
	cli.UI.ShowToast("Login successful", 3*time.Second,nil)
	cli.User = usr
	cli.Keybag = kb
	cli.Node.PK = kb.Libp2pPriv
	err = cli.Node.InitNode()
	if err != nil {
		cli.UI.ShowToast("Node initialization failed: "+err.Error(), 0, nil)
		return
	}
	

	cli.SwitchToBrowseScreen(hub)

	return

}

func (cli *Client) createUserHandler(username string, password string, hub string) {

	if username == "" || password == "" {
		cli.UI.ShowToast("Username and password cannot be empty", 0, nil)
		return
	}
	prof, err := profile.GenerateProfile(username, password)
	if err != nil {
		cli.UI.ShowToast("Create user failed: "+err.Error(),0, nil)
		return
	}
	cli.UI.ShowToast("User created successfully! Welcome "+ prof.Username, 3*time.Second,nil)
	kb,usr, err := profile.LoadProfile(username, password, "")
	if err != nil {
			cli.UI.ShowToast("Login failed: "+err.Error(),0, nil)
		return
	}
	cli.User = usr
	cli.Keybag = kb
	cli.Node.PK = kb.Libp2pPriv
	err = cli.Node.InitNode()
	if err != nil {
		cli.UI.ShowToast("Node initialization failed: "+err.Error(), 0, nil)
		return
	}
	cli.SwitchToBrowseScreen(hub)


	return
}

func (cli *Client) SwitchToBrowseScreen(hub string) {
	cli.UI.Pages.SwitchToPage("Browse")

	cli.Node.HubAddr = hub

	go cli.refreshServerList()
}

