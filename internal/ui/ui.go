package ui

import "github.com/rivo/tview"

type UIConfig struct {
	Theme string
}

type UI struct {
	App *tview.Application
	Pages *tview.Pages

	// Screens
	LoginScreen *LoginScreen
	ServersScreen *ServersScreen
	ChatScreen *ChatScreen

}

func NewUI(cfg *UIConfig) *UI {
	app := tview.NewApplication()
	pages := tview.NewPages()

	login := NewLoginScreen()
	servers := NewServersScreen()
	chat := NewChatScreen()
	pages.AddPage("login", login.Layout(), true, true)
	pages.AddPage("servers", servers.Layout(), true, false)
	pages.AddPage("chat", chat.Layout(), true, false)

	ui := &UI{
		App: app,
		Pages: pages,
		LoginScreen: login,
		ServersScreen: servers,
		ChatScreen: chat,
	}
	app.SetRoot(pages, true).
		SetFocus(login.Form)
	return ui
}