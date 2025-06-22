package main

import (
	"hillside/internal/models"
	"strconv"
	"time"

	"github.com/rivo/tview"
)



type BrowseScreen struct {
	*UI
	layout        *tview.Flex
	servers []models.ServerMeta
	Hub		  string
}

func (b *BrowseScreen) NewBrowseScreen() {
	b.layout = tview.NewFlex()
	b.layout.SetDirection(tview.FlexColumn).
		SetBorder(false)

	b.layout.SetBackgroundColor(b.Theme.GetColor("background"))
	b.layout.SetFullScreen(true)
	servers := tview.NewFlex().SetDirection(tview.FlexRow)
	serverLabels := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(tview.NewTextView().SetText("Name"), 0,2, false).
		AddItem(tview.NewTextView().SetText("Description"), 0,3, false).
		AddItem(tview.NewTextView().SetText("Password"), 0,1, false).
		AddItem(tview.NewTextView().SetText("Online"), 0,1, false)

	servers.AddItem(serverLabels, 1, 0, false)

	serverList := tview.NewList().
		SetBorder(true).
		SetBorderColor(b.Theme.GetColor("border")).
		SetTitle("Servers").
		SetTitleColor(b.Theme.GetColor("foreground")).
		SetBackgroundColor(b.Theme.GetColor("background"))

	servers.AddItem(serverList, 0, 1, false)

	serverInfo := tview.NewTextView().
		SetText("Select a server to view details.").
		SetTextAlign(tview.AlignLeft)
	b.layout.AddItem(servers, 0, 2, true)
	b.layout.AddItem(serverInfo, 0, 1, false)
	
}

func formatBoolPasswordProtected(passwordProtected models.ServerVisibility) string {
	if passwordProtected == models.ServerPasswordProtected {
		return "🔒"
	}
	return "	"
}

func (b *BrowseScreen) UpdateServerList(servers []models.ServerMeta) {
	b.servers = servers
	serverList := tview.NewFlex()
	serverList.SetDirection(tview.FlexRow)

	for _, server := range b.servers {
		serverEntry := tview.NewFlex().
			SetDirection(tview.FlexColumn)
		serverEntry.AddItem(tview.NewTextView().SetText(server.Name), 0, 2, false).
			AddItem(tview.NewTextView().SetText(server.Description), 0, 3, false).
			AddItem(tview.NewTextView().SetText(formatBoolPasswordProtected(server.Visibility)), 0, 1, false).
			AddItem(tview.NewTextView().SetText(strconv.Itoa(int(server.Online))), 0, 1, false)

		serverList.AddItem(serverEntry, 1, 0, false)
	}
	serverList.
		SetBorder(true).
		SetBorderColor(b.Theme.GetColor("border")).
		SetTitle("Servers").
		SetTitleColor(b.Theme.GetColor("foreground")).
		SetBackgroundColor(b.Theme.GetColor("background"))

	b.layout.AddItem(serverList, 0, 1, false)
	b.layout.AddItem(tview.NewTextView().SetText("Select a server to view details."), 0, 1, false)

}




func (cli *Client) refreshServerList(){
	refresh := func() {
		serverResp, err := cli.requetServers()
		serverList := serverResp.Servers
		cli.UI.App.QueueUpdate(func() {
			if err != nil {
				cli.UI.ShowToast(err.Error(), 0,nil)
				cli.UI.App.Draw()
				return
			}
			cli.UI.BrowseScreen.UpdateServerList(serverList)
			cli.UI.App.Draw()
		})
	}

	refresh()
	// Update the chat rooms on every timer fire
	for range 2* time.Second {
		refresh()
	}
}