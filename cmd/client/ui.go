package main

import (
	"fmt"
	"os"

	"hillside/internal/models"

	"github.com/rivo/tview"
)

type UIConfig struct {
	Theme               *Theme
	loginHandler        func(username, password string, hub string)
	createUserHandler   func(username, password string, hub string)
	createServerHandler func(request models.CreateServerRequest) (sid string, err error)
	joinServerHandler   func(serverID string, pass string) error
	getServerName       func() string
	getRoomName         func() string
	getServerId         func() string
	createRoomHandler   func(req models.CreateRoomRequest) (string, error)
	joinRoomHandler     func(roomID string, pass string) error
	sendMessageHandler  func(message string) error
	chatInputHandler    func()
}

type UI struct {
	App   *tview.Application
	Theme *Theme
	Pages *tview.Pages

	// Screens
	LoginScreen  *LoginScreen
	BrowseScreen *BrowseScreen
	ChatScreen   *ChatScreen
}

func NewUI(cfg *UIConfig) *UI {
	app := tview.NewApplication().EnableMouse(true)
	tview.Borders.HorizontalFocus = tview.Borders.Horizontal
	tview.Borders.VerticalFocus = tview.Borders.Vertical

	tview.Borders.TopLeftFocus = '╭'
	tview.Borders.TopRightFocus = '╮'
	tview.Borders.BottomLeftFocus = '╰'
	tview.Borders.BottomRightFocus = '╯'

	tview.Borders.Horizontal = ' '
	tview.Borders.Vertical = ' '

	tview.Borders.TopLeft = ' '
	tview.Borders.TopRight = ' '
	tview.Borders.BottomLeft = ' '
	tview.Borders.BottomRight = ' '

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Failed to get user home directory: " + err.Error())
		panic(err)
	}
	DefaultTheme, err := LoadTheme(homeDir + "/.hillside/default_theme.yaml")
	if err != nil {
		panic("Failed to load default theme: " + err.Error())
	}
	if cfg.Theme == nil {
		cfg.Theme = DefaultTheme
	}
	ui := &UI{
		App:   app,
		Theme: cfg.Theme,
	}

	tview.Styles.PrimitiveBackgroundColor = ui.Theme.GetColor("background")
	tview.Styles.TitleColor = ui.Theme.GetColor("primary")

	ui.LoginScreen = &LoginScreen{
		UI:                ui,
		Hub:               "",
		loginHandler:      cfg.loginHandler,
		createUserHandler: cfg.createUserHandler}
	ui.LoginScreen.NewLoginScreen()
	ui.BrowseScreen = &BrowseScreen{
		UI:             ui,
		Hub:            "",
		OnCreateServer: cfg.createServerHandler,
		OnJoinServer:   cfg.joinServerHandler,
	}
	ui.BrowseScreen.NewBrowseScreen()

	ui.ChatScreen = &ChatScreen{
		UI:            ui,
		GetServerName: cfg.getServerName,
		GetRoomName:   cfg.getRoomName,
		GetServerId:   cfg.getServerId,
		OnCreateRoom:  cfg.createRoomHandler,
		OnJoinRoom:    cfg.joinRoomHandler,
		sendMessage:   cfg.sendMessageHandler,
	}

	ui.ChatScreen.NewChatScreen()

	ui.Pages = tview.NewPages().
		AddPage("login", ui.LoginScreen.layout, true, true).
		AddPage("browse", ui.BrowseScreen.layout, true, false).
		AddPage("chat", ui.ChatScreen.layout, true, false)

	ui.App.SetRoot(ui.Pages, true).
		SetFocus(ui.LoginScreen.form)
	return ui
}
