package ui

import (
	"fmt"
	"os"

	"hillside/internal/models"

	"github.com/rivo/tview"
)

type UIConfig struct {
	Theme               *Theme
	LoginHandler        func(username, password string, hub string)
	CreateUserHandler   func(username, password string, hub string)
	CreateServerHandler func(request models.CreateServerRequest) (sid string, err error)
	JoinServerHandler   func(serverID string, pass string) error
	GetServerName       func() string
	GetRoomName         func() string
	GetServerID         func() string
	CreateRoomHandler   func(req models.CreateRoomRequest) (string, error)
	JoinRoomHandler     func(roomID string, pass string) error
	SendMessageHandler  func(message string) error
	ChatInputHandler    func()
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
		loginHandler:      cfg.LoginHandler,
		createUserHandler: cfg.CreateUserHandler}
	ui.LoginScreen.NewLoginScreen()
	ui.BrowseScreen = &BrowseScreen{
		UI:             ui,
		Hub:            "",
		OnCreateServer: cfg.CreateServerHandler,
		OnJoinServer:   cfg.JoinServerHandler,
	}
	ui.BrowseScreen.NewBrowseScreen()

	ui.ChatScreen = &ChatScreen{
		UI:            ui,
		GetServerName: cfg.GetServerName,
		GetRoomName:   cfg.GetRoomName,
		GetServerID:   cfg.GetServerID,
		OnCreateRoom:  cfg.CreateRoomHandler,
		OnJoinRoom:    cfg.JoinRoomHandler,
		sendMessage:   cfg.SendMessageHandler,
	}

	ui.ChatScreen.NewChatScreen()

	ui.Pages = tview.NewPages().
		AddPage("login", ui.LoginScreen.Layout, true, true).
		AddPage("browse", ui.BrowseScreen.Layout, true, false).
		AddPage("chat", ui.ChatScreen.Layout, true, false)

	ui.App.SetRoot(ui.Pages, true).
		SetFocus(ui.LoginScreen.form)
	return ui
}
