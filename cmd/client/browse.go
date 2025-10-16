package main

import (
	"context"
	"fmt"
	"sort"
	"time"

	"encoding/json"

	"hillside/internal/models"
	"hillside/internal/p2p"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type BrowseScreen struct {
	*UI
	layout         *tview.Flex
	serverList     *tview.List
	serverPane     *tview.Flex
	infoView       *tview.TextView
	OnCreateServer func(request models.CreateServerRequest) (sid string, err error)
	OnJoinServer   func(serverID string, pass string) error
	servers        []models.ServerMeta
	modalForm      *tview.Form
	createBtn      *tview.Button
	noServersView  *tview.TextView
	Hub            string
	title          *tview.TextView
	joinForm       *tview.Form
	selectedServer *models.ServerMeta
}

func (b *BrowseScreen) NewBrowseScreen() {
	// left pane
	b.serverList = tview.NewList()

	b.serverList.SetSelectedBackgroundColor(b.UI.Theme.GetColor("background-light"))
	b.serverList.SetSelectedTextColor(b.UI.Theme.GetColor("primary")).
		SetHighlightFullLine(true)

	b.serverList.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		if index >= 0 && index < len(b.servers) {
			srv := b.servers[index]
			b.infoView.SetText(
				fmt.Sprintf("[yellow]Server:[white] %s\n[yellow]Description:[white] %s\n[yellow]Online:[white] %d\n",
					srv.Name, srv.Description, srv.Online,
				))
		}
	})
	b.serverList.
		SetTitleColor(b.Theme.GetColor("primary")).
		SetBackgroundColor(b.Theme.GetColor("background"))

	// info pane
	b.infoView = tview.NewTextView()
	b.infoView.
		SetText("Select a server to view details.").
		SetTextAlign(tview.AlignLeft).
		SetBackgroundColor(b.Theme.GetColor("background"))

	b.infoView.SetDynamicColors(true)

	b.infoView.SetBorder(true).
		SetTitle("[ Server Info ]").
		SetTitleColor(b.Theme.GetColor("primary")).
		SetBorderColor(b.Theme.GetColor("border")).
		SetBorderPadding(2, 2, 2, 2)

	// layout
	b.serverPane = tview.NewFlex()
	b.serverPane.AddItem(b.serverList, 0, 1, true)

	b.serverPane.SetDirection(tview.FlexRow)
	b.serverPane.SetBorder(true).
		SetTitle("[ Servers ]").
		SetTitleColor(b.Theme.GetColor("primary")).
		SetBorderColor(b.Theme.GetColor("border")).
		SetBackgroundColor(b.Theme.GetColor("background")).
		SetBorderPadding(2, 2, 2, 2)

	header := tview.NewFlex().
		SetDirection(tview.FlexColumn)

	b.createBtn = tview.NewButton("Create Server")
	b.createBtn.SetSelectedFunc(b.showCreateServerForm).
		SetLabelColor(b.Theme.GetColor("button-text")).
		SetBackgroundColor(b.Theme.GetColor("button-active"))

	b.title = tview.NewTextView().SetDynamicColors(true)
	b.title.SetTextAlign(tview.AlignLeft)

	header.AddItem(b.title, 0, 2, false).
		AddItem(b.createBtn, 15, 0, false)

	serverView := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(b.serverPane, 0, 2, true).
		AddItem(b.infoView, 0, 1, false)

	b.layout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 1, 0, false).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(serverView, 0, 1, true)
}

func formatBoolPasswordProtected(passwordProtected models.Visibility) string {
	if passwordProtected == models.PasswordProtected {
		return "🔒"
	}
	return "    "
}
func (b *BrowseScreen) UpdateServerList(servers []models.ServerMeta) {
	sort.SliceStable(servers, func(i, j int) bool {
		if servers[i].Online == servers[j].Online {
			return servers[i].Name < servers[j].Name
		}
		return servers[i].Online > servers[j].Online
	})

	b.servers = servers
	b.serverList.Clear()
	if len(servers) == 0 {
		if b.noServersView == nil {
			b.noServersView = tview.NewTextView().
				SetTextAlign(tview.AlignCenter).
				SetTextColor(b.Theme.GetColor("foreground")).
				SetText("No servers available. Create a new server to get started.")
			b.serverPane.AddItem(b.noServersView, 0, 1, false)
			b.infoView.SetText("")
		}
	} else {
		b.serverPane.RemoveItem(b.noServersView)

		for i, srv := range servers {
			line := fmt.Sprintf(
				"%-20s | %-30s | %s | %3d Online",
				srv.Name,
				srv.Description,
				formatBoolPasswordProtected(srv.Visibility),
				srv.Online,
			)

			b.serverList.AddItem(line, "", 0, func(idx int) func() {
				return func() {
					b.selectedServer = &b.servers[idx]
					if b.selectedServer.Visibility == models.Public {
						err := b.OnJoinServer(b.selectedServer.ID, "")
						if err != nil {
							b.UI.ShowToast("Join server failed: "+err.Error(), 3*time.Second, nil)
							return
						}
					} else {
						b.showJoinServerForm()
					}

				}
			}(i))
		}
	}
}

func (b *BrowseScreen) SetHub(hub string) {
	b.Hub = hub
	b.title.SetText(fmt.Sprintf("[yellow]Hub:[white] %s", b.Hub))
}

func (b *BrowseScreen) showCreateServerForm() {
	// 1) Build the form
	b.modalForm = tview.NewForm()

	bgColor, fieldBg, buttonBg, buttonText, fieldText := b.UI.Theme.FormColors()
	b.modalForm.SetBackgroundColor(bgColor)
	b.modalForm.SetButtonBackgroundColor(buttonBg)
	b.modalForm.SetButtonTextColor(buttonText)
	b.modalForm.SetFieldBackgroundColor(fieldBg)
	b.modalForm.SetFieldTextColor(fieldText)
	b.modalForm.SetLabelColor(b.UI.Theme.GetColor("primary"))
	b.modalForm.SetBorder(true)
	b.modalForm.SetBorderColor(b.UI.Theme.GetColor("border"))
	b.modalForm.SetBorderAttributes(tcell.AttrNone)

	visibilityDropdown := tview.NewDropDown().
		SetLabel("Visibility").
		SetOptions([]string{"Public", "Password Protected", "Private"}, nil)

	visibilityDropdown.SetBackgroundColor(b.UI.Theme.GetColor("background"))
	visibilityDropdown.SetFieldBackgroundColor(fieldBg)
	visibilityDropdown.SetFieldTextColor(fieldText)
	visibilityDropdown.SetPrefixTextColor(b.UI.Theme.GetColor("background-light"))
	visibilityDropdown.SetLabelColor(b.UI.Theme.GetColor("primary"))
	visibilityDropdown.SetListStyles(
		tcell.StyleDefault.
			Foreground(fieldText).
			Background(b.UI.Theme.GetColor("background")),
		tcell.StyleDefault.
			Foreground(fieldText).
			Background(b.UI.Theme.GetColor("background-light")),
	)
	visibilityDropdown.SetFocusedStyle(tcell.StyleDefault.
		Foreground(fieldText).
		Background(b.UI.Theme.GetColor("background")))

	b.modalForm.AddInputField("Name", "", 0, nil, nil).
		AddInputField("Description", "", 0, nil, nil).
		AddPasswordField("Password (opt)", "", 0, '*', nil).
		AddFormItem(visibilityDropdown).
		AddButton("Save", func() {
			name := b.modalForm.GetFormItemByLabel("Name").(*tview.InputField).GetText()
			description := b.modalForm.GetFormItemByLabel("Description").(*tview.InputField).GetText()
			pass := b.modalForm.GetFormItemByLabel("Password (opt)").(*tview.InputField).GetText()
			visibilityIndex, _ := b.modalForm.GetFormItemByLabel("Visibility").(*tview.DropDown).GetCurrentOption()
			visibility := models.Visibility(visibilityIndex)
			req := models.CreateServerRequest{
				Name:         name,
				Description:  description,
				Visibility:   visibility,
				PasswordHash: []byte(pass),
			}

			sid, err := b.OnCreateServer(req)
			if err != nil {
				b.UI.ShowError("Create server failed", err.Error(), "OK", 0, nil)
				return
			}
			if req.Visibility == models.Private {
				b.UI.ShowToast(fmt.Sprintf("Server created successfully! ID: %s\nThis ServerID will be the only way to access the server. It's been saved under ~/.hillside, encrypted with the server password. DON'T LOSE IT", sid), 0, nil)
				saveEncryptedSID(sid, pass)
			} else {
				b.UI.ShowToast("Server created successfully! ID: "+sid, 3*time.Second, nil)
			}
			b.UI.Pages.RemovePage("createServer")
		}).
		AddButton("Cancel", func() {
			b.UI.Pages.RemovePage("createServer")
		})

	b.modalForm.SetBorder(true).
		SetTitle("[ Create Server ]").
		SetTitleAlign(tview.AlignCenter).
		SetTitleColor(b.UI.Theme.GetColor("primary"))

	mf := func(p tview.Primitive, width, height int) tview.Primitive {
		return tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(p, height, 1, true).
				AddItem(nil, 0, 1, false), width, 1, true).
			AddItem(nil, 0, 1, false)
	}

	b.UI.Pages.AddPage("createServer", mf(b.modalForm, 40, 15), true, true)
	b.UI.App.SetFocus(b.modalForm)
}

func (b *BrowseScreen) showJoinServerForm() {
	b.joinForm = tview.NewForm()

	bgColor, fieldBg, buttonBg, buttonText, fieldText := b.UI.Theme.FormColors()
	b.joinForm.SetBackgroundColor(bgColor)
	b.joinForm.SetButtonBackgroundColor(buttonBg)
	b.joinForm.SetButtonTextColor(buttonText)
	b.joinForm.SetFieldBackgroundColor(fieldBg)
	b.joinForm.SetFieldTextColor(fieldText)
	b.joinForm.SetLabelColor(b.UI.Theme.GetColor("primary"))
	b.joinForm.SetBorder(true)
	b.joinForm.SetBorderColor(b.UI.Theme.GetColor("border"))
	b.joinForm.SetBorderAttributes(tcell.AttrNone)
	b.joinForm.SetButtonsAlign(tview.AlignCenter)

	b.joinForm.AddPasswordField("Password", "", 0, '*', nil).
		AddButton("Join", func() {
			pass := b.joinForm.GetFormItemByLabel("Password").(*tview.InputField).GetText()

			err := b.OnJoinServer(b.selectedServer.ID, pass)
			if err != nil {
				b.UI.ShowError("Join server failed", err.Error(), "OK", 0, nil)
				return
			}

			b.UI.Pages.RemovePage("joinServer")
		}).
		AddButton("Cancel", func() {
			b.UI.Pages.RemovePage("joinServer")
		})

	b.joinForm.SetBorder(true).
		SetTitle(fmt.Sprintf("[ Join %s ]", b.selectedServer.Name)).
		SetTitleAlign(tview.AlignCenter).
		SetTitleColor(b.UI.Theme.GetColor("primary"))

	mf := func(p tview.Primitive, width, height int) tview.Primitive {
		return tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(p, height, 1, true).
				AddItem(nil, 0, 1, false), width, 1, true).
			AddItem(nil, 0, 1, false)
	}

	b.UI.Pages.AddPage("joinServer", mf(b.joinForm, 40, 8), true, true)
	b.UI.App.SetFocus(b.joinForm)
}

func (cli *Client) refreshServerList() {
	serverResp, err := cli.requestServers()
	cli.UI.App.QueueUpdateDraw(func() {
		if err != nil {
			cli.UI.ShowError("Server Error", err.Error(), "Go back to Login", 0, func() {
				cli.UI.Pages.SwitchToPage("login")
			})
			return
		} else {
			cli.UI.BrowseScreen.UpdateServerList(serverResp.Servers)
		}
	})
}
func isBrowsePageActive(page string) bool {
	return page == "browse" || page == "createServer" || page == "toast"
}

func (cli *Client) StartAutoRefresh() {
	cli.refreshServerList()
	currentPage, _ := cli.UI.Pages.GetFrontPage()
	cli.Session.Log.Logf("Starting auto-refresh on page: %s", currentPage)

	// cancellable context for this auto-refresh session
	refreshCtx, cancelRefresh := context.WithCancel(cli.Node.Ctx)
	defer cancelRefresh()

	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				currentPage, _ := cli.UI.Pages.GetFrontPage()
				if !isBrowsePageActive(currentPage) {
					cli.Session.Log.Logf("Left browse page, stopping auto-refresh")
					cancelRefresh() // This will break the message loop
					return
				}
			case <-refreshCtx.Done():
				return
			}
		}
	}()

	ServersTopic := p2p.ServersTopic()
	top, err := cli.Node.PS.Join(ServersTopic)
	if err != nil {
		cli.Session.Log.Logf("Failed to join servers topic: %v", err)
		return
	}
	cli.Node.Topics.ServersTopic = top

	sub, err := cli.Node.Topics.ServersTopic.Subscribe()
	if err != nil {
		cli.Session.Log.Logf("Failed to subscribe to servers topic: %v", err)
		return
	}
	defer sub.Cancel()

	for {
		select {
		case <-refreshCtx.Done():
			cli.Session.Log.Logf("Auto-refresh cancelled")
			return
		default:
			msg, err := sub.Next(refreshCtx)
			if err != nil {
				if err == context.Canceled {
					cli.Session.Log.Logf("Auto-refresh stopped due to page change")

					currentPage, _ := cli.UI.Pages.GetFrontPage()
					cli.Session.Log.Logf("Current page: %s", currentPage)
				} else {
					cli.Session.Log.Logf("Error reading from servers topic: %v", err)
				}
				return
			}

			var listResp models.ListServersResponse
			err = json.Unmarshal(msg.Data, &listResp)
			if err != nil {
				cli.Session.Log.Logf("Error unmarshaling servers topic message: %v", err)

			}
			cli.Session.Log.Logf("Received server list update with %d servers", len(listResp.Servers))
			cli.UI.App.QueueUpdateDraw(func() {
				if err != nil {
					cli.UI.ShowError("Server Error", err.Error(), "Go back to Login", 0, func() {
						cli.UI.Pages.SwitchToPage("login")
					})
					return
				} else {
					cli.UI.BrowseScreen.UpdateServerList(listResp.Servers)
				}
			})
		}
	}
}
