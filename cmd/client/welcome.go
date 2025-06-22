package main
/*
import (
	"fmt"
	"hillside/internal/profile"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type WelcomeUIButtons struct {
	*tview.Grid

	createUserButton *tview.Button
	loginButton *tview.Button
}

type WelcomeUI struct {
	*UI

	layout *tview.Grid
	header *tview.TextView
	infoText *tview.TextView
	Buttons *WelcomeUIButtons

}
var AsCII string = `
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣀⣠⣤⣤⣄⣀⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⣀⡴⠾⠛⠋⠉⠉⠁⠈⠉⠉⠙⠛⠷⢦⣄⠀⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⢀⣴⠟⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠻⣦⡀⠀⠀⠀⠀
⠀⠀⠀⣠⠟⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣴⡀⠈⠻⣄⠀⠀⠀   ▄█    █▄     ▄█   ▄█        ▄█          ▄████████  ▄█  ████████▄     ▄████████
⠀⠀⣰⠏⠀⢀⣴⣄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣴⣿⣿⣿⣦⡀⠹⣆⠀⠀  ███    ███   ███  ███       ███         ███    ███ ███  ███   ▀███   ███    ███
⠀⢰⡟⢀⣴⣿⣿⣿⣷⡄⠀⠀⠀⢀⣴⣷⣄⠀⣠⡿⠟⡿⠻⢿⡟⠻⣦⣻⡆⠀  ███    ███   ███▌ ███       ███         ███    █▀  ███▌ ███    ███   ███    █▀
⠀⣾⣷⡿⢟⣿⠟⣿⠈⠙⢦⣀⣴⣿⣿⣿⣿⣿⣯⡀⠀⠀⠀⠀⠈⠀⠈⠻⣷⠀ ▄███▄▄▄▄███▄▄ ███▌ ███       ███         ███        ███▌ ███    ███  ▄███▄▄▄
⠀⣿⠋⠀⠜⠁⠀⠈⠀⠀⣰⣿⣿⣿⣿⣿⣿⣿⣿⣿⣦⡀⠀⠀⠀⠀⠀⠀⣿⠀▀▀███▀▀▀▀███▀  ███▌ ███       ███       ▀███████████ ███▌ ███    ███ ▀▀███▀▀▀
⠀⢿⡄⠀⠀⠀⠀⠀⣠⣾⡿⣿⣿⢿⡟⢿⣧⠙⣿⠉⠻⢿⣦⡀⠀⠀⠀⢠⣿⠀  ███    ███   ███  ███       ███                ███ ███  ███    ███   ███    █▄
⠀⠸⣧⠀⠀⠀⣠⡾⠋⠁⢠⡟⠁⠈⠀⠈⢻⡄⠈⠀⠀⠀⠉⠻⣦⠀⠀⣸⠇⠀  ███    ███   ███  ███▌    ▄ ███▌    ▄    ▄█    ███ ███  ███   ▄███   ███    ███
⠀⠀⠹⡆⣠⡾⠋⠀⠀⠀⠊⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠳⣴⡟⠀⠀  ███    █▀    █▀   █████▄▄██ █████▄▄██  ▄████████▀  █▀   ████████▀    ██████████
⠀⠀⠀⠹⣟⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣰⠏⠀⠀⠀                    ▀         ▀
⠀⠀⠀⠀⠈⠳⣄⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣠⠞⠁⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠈⠙⠶⣤⣄⣀⡀⠀⠀⠀⠀⢀⣀⣠⣤⠶⠋⠁⠀⠀⠀⠀⠀⠀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠉⠉⠛⠛⠛⠛⠛⠛⠉⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
`

func (ui *WelcomeUI) Init() {
	ui.layout = tview.NewGrid().
		SetRows(-2, -1, -1).
		SetColumns(-1).
		SetBorders(true)

	ui.header = tview.NewTextView().
		SetText(ASCII).
		SetTextAlign(tview.AlignCenter)
	ui.header.SetBackgroundColor(tcell.ColorDefault)
	ui.infoText = tview.NewTextView().SetTextAlign(tview.AlignCenter)
	ui.infoText.SetBackgroundColor(tcell.ColorDefault)
	users, err := profile.CheckUsers()
	if err != nil {
		ui.App.Stop()
		return
	}
	if len(users) == 0 {
		ui.infoText.
			SetText("No users found. Please create a user to continue.")
	} else {
		ui.infoText.
			SetText(fmt.Sprintf("Welcome to Hillside!\n\nPlease select a user to login or create a new user.\n\nAvailable users: %s", users))
	}
	createUserButton := tview.NewButton("Create User")
	loginButton := tview.NewButton("Login")

	buttonsGrid := tview.NewGrid()
	ui.Buttons = &WelcomeUIButtons{
		Grid:             buttonsGrid,
		createUserButton: createUserButton,
		loginButton:      loginButton,
	}
	ui.Buttons.SetRows(1).SetColumns(-1, -1).
		SetBorders(false).
		AddItem(ui.Buttons.createUserButton, 0, 0, 1, 1, 0, 0, false).
		AddItem(ui.Buttons.loginButton, 0, 1, 1, 1, 0, 0, false)
	ui.Buttons.SetBackgroundColor(tcell.ColorDefault)

	ui.layout.AddItem(ui.header, 0, 0, 1, 1, 0, 0, true)
	ui.layout.AddItem(ui.Buttons, 1, 0, 1, 1, 0, 0, false)
	ui.layout.AddItem(ui.infoText, 2, 0, 1, 1, 0, 0, false)

	ui.layout.SetBorder(true).SetBackgroundColor(tcell.ColorDefault)
}*/