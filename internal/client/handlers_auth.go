package client

import (
	"hillside/internal/p2p"
	"hillside/internal/profile"
	"hillside/internal/storage"
)

func (cli *Client) LoginHandler(username string, password string, hub string) {

	if validateUsernameAndPassword(username, password) {
		cli.UI.ShowError("Error", "Username and password cannot be empty", "OK", 0, nil)
		return
	}

	cli.AuthFlow(username, password, hub)

}

func (cli *Client) CreateUserHandler(username string, password string, hub string) {

	if username == "" || password == "" {
		cli.UI.ShowError("Error", "Username and password cannot be empty", "OK", 0, nil)
		return
	}
	_, err := profile.GenerateProfile(username, password)
	if err != nil {
		cli.UI.ShowError("Create user failed", err.Error(), "OK", 0, nil)
		return
	}

	cli.AuthFlow(username, password, hub)
}

func validateUsernameAndPassword(username, password string) bool {
	return username == "" || password == ""
}

func (cli *Client) AuthFlow(username, password, hub string) {
	kb, usr, err := profile.LoadProfile(username, password, "")
	if err != nil {
		cli.UI.ShowError("Login failed", err.Error(), "Retry", 0, nil)
		return
	}
	cli.User = usr
	cli.Keybag = kb
	cli.Session.Log.Logf("Loaded profile for user %s", cli.User.Username)

	hubadrr, err := p2p.GetHubAddress(hub)
	if err != nil {
		cli.UI.ShowError("Invalid Hub Address", "Failed to parse hub address: "+err.Error(), "OK", 0, nil)
		return
	}
	cli.Node.Hub = hubadrr

	cli.Node.PK = kb.Libp2pPriv

	go func() {
		db, err := storage.InitSessionDB(cli.User.Username, "", 1024)
		if err != nil {
			cli.UI.ShowError("Storage Init Failed", "Failed to initialize storage: "+err.Error(), "OK", 0, nil)
			return
		}
		cli.Session.SessionDB = db
		cli.Session.SessionDB.Peers.EnqueueUserEntry(cli.Node.Ctx, cli.User)

		if err := cli.Node.InitNode(); err != nil {
			cli.UI.App.QueueUpdateDraw(func() {
				cli.UI.ShowError("Node init failed", err.Error(), "OK", 0, nil)
			})
			return
		}
		cli.UI.App.QueueUpdateDraw(func() {
			cli.SwitchToBrowseScreen(hub)
		})
	}()
}
