package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LoginScreen struct {
	*UI
	layout        *tview.Flex
	form		  *tview.Form
	loginHandler func(username, password string, hub string)
	createUserHandler func(username, password string, hub string)
	Username	  string
	Password	  string
	Hub		  string
}

var Ascii string = `
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣀⣠⣤⣤⣄⣀⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀                                                                                     
⠀⠀⠀⠀⠀⠀⠀⣀⡴⠾⠛⠋⠉⠉⠁⠈⠉⠉⠙⠛⠷⢦⣄⠀⠀⠀⠀⠀⠀⠀                                                                                     
⠀⠀⠀⠀⢀⣴⠟⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠻⣦⡀⠀⠀⠀⠀                                                                                     
⠀⠀⠀⣠⠟⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣴⡀⠈⠻⣄⠀⠀⠀       ▄█    █▄     ▄█   ▄█        ▄█          ▄████████  ▄█  ████████▄     ▄████████
⠀⠀⣰⠏⠀⢀⣴⣄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣴⣿⣿⣿⣦⡀⠹⣆⠀⠀      ███    ███   ███  ███       ███         ███    ███ ███  ███   ▀███   ███    ███
⠀⢰⡟⢀⣴⣿⣿⣿⣷⡄⠀⠀⠀⢀⣴⣷⣄⠀⣠⡿⠟⡿⠻⢿⡟⠻⣦⣻⡆⠀      ███    ███   ███▌ ███       ███         ███    █▀  ███▌ ███    ███   ███    █▀ 
⠀⣾⣷⡿⢟⣿⠟⣿⠈⠙⢦⣀⣴⣿⣿⣿⣿⣿⣯⡀⠀⠀⠀⠀⠈⠀⠈⠻⣷⠀     ▄███▄▄▄▄███▄▄ ███▌ ███       ███         ███        ███▌ ███    ███  ▄███▄▄▄    
⠀⣿⠋⠀⠜⠁⠀⠈⠀⠀⣰⣿⣿⣿⣿⣿⣿⣿⣿⣿⣦⡀⠀⠀⠀⠀⠀⠀⣿⠀    ▀▀███▀▀▀▀███▀  ███▌ ███       ███       ▀███████████ ███▌ ███    ███ ▀▀███▀▀▀    
⠀⢿⡄⠀⠀⠀⠀⠀⣠⣾⡿⣿⣿⢿⡟⢿⣧⠙⣿⠉⠻⢿⣦⡀⠀⠀⠀⢠⣿⠀      ███    ███   ███  ███       ███                ███ ███  ███    ███   ███    █▄ 
⠀⠸⣧⠀⠀⠀⣠⡾⠋⠁⢠⡟⠁⠈⠀⠈⢻⡄⠈⠀⠀⠀⠉⠻⣦⠀⠀⣸⠇⠀      ███    ███   ███  ███▌    ▄ ███▌    ▄    ▄█    ███ ███  ███   ▄███   ███    ███
⠀⠀⠹⡆⣠⡾⠋⠀⠀⠀⠊⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠳⣴⡟⠀⠀      ███    █▀    █▀   █████▄▄██ █████▄▄██  ▄████████▀  █▀   ████████▀    ██████████
⠀⠀⠀⠹⣟⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣰⠏⠀⠀⠀                        ▀         ▀                                                  
⠀⠀⠀⠀⠈⠳⣄⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⣠⠞⠁⠀⠀⠀⠀                                                                                     
⠀⠀⠀⠀⠀⠀⠈⠙⠶⣤⣄⣀⡀⠀⠀⠀⠀⢀⣀⣠⣤⠶⠋⠁⠀⠀⠀⠀⠀⠀                                                                                     
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠉⠉⠛⠛⠛⠛⠛⠛⠉⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀                                                                                     
`
var AsciiMin string = `
   ▄█    █▄     ▄█   ▄█        ▄█          ▄████████  ▄█  ████████▄     ▄████████
  ███    ███   ███  ███       ███         ███    ███ ███  ███   ▀███   ███    ███
  ███    ███   ███▌ ███       ███         ███    █▀  ███▌ ███    ███   ███    █▀ 
 ▄███▄▄▄▄███▄▄ ███▌ ███       ███         ███        ███▌ ███    ███  ▄███▄▄▄    
▀▀███▀▀▀▀███▀  ███▌ ███       ███       ▀███████████ ███▌ ███    ███ ▀▀███▀▀▀    
  ███    ███   ███  ███       ███                ███ ███  ███    ███   ███    █▄ 
  ███    ███   ███  ███▌    ▄ ███▌    ▄    ▄█    ███ ███  ███   ▄███   ███    ███
  ███    █▀    █▀   █████▄▄██ █████▄▄██  ▄████████▀  █▀   ████████▀    ██████████
                    ▀         ▀                                                  
`
func (l *LoginScreen) NewLoginScreen() {
	l.layout = tview.NewFlex()
	l.layout.SetDirection(tview.FlexRow).
	SetBorder(false)
	_,_,width,_ := l.layout.GetRect()
	var header string
	if width < 60 {
		header = AsciiMin
	} else {
		header = Ascii
	}

	headerStyle := tcell.StyleDefault.
		Foreground(l.Theme.GetColor("accent")).
		Background(l.Theme.GetColor("background"))

    asciiTextView := tview.NewTextView().
        SetText(header).
        SetTextAlign(tview.AlignCenter)
	asciiTextView.SetTextStyle(headerStyle)

    asciiContainer := tview.NewFlex().
        SetDirection(tview.FlexRow).
        AddItem(nil, 0, 1, false).              // Top spacer
        AddItem(asciiTextView, 9, 0, false).    // ASCII art
        AddItem(nil, 0, 1, false)              // Bottom spacer


    l.layout.AddItem(asciiContainer, 0,2, false)

	l.form = tview.NewForm()
    bgColor, fieldBg, buttonBg, buttonText, fieldText := l.Theme.FormColors()
    l.form.SetBackgroundColor(bgColor)
    l.form.SetButtonBackgroundColor(buttonBg)
    l.form.SetButtonTextColor(buttonText)
    l.form.SetFieldBackgroundColor(fieldBg)
    l.form.SetFieldTextColor(fieldText)
    l.form.SetLabelColor(l.Theme.GetColor("primary"))
	l.form.SetBorder(true)
    l.form.SetBorderColor(l.Theme.GetColor("border"))
	l.form.SetBorderAttributes(tcell.AttrNone)
	l.form.SetButtonsAlign(tview.AlignCenter)

	l.form.AddInputField(
		"Hub   ", l.Hub, 0, nil,
		func(s string) { l.Hub = s },
	)
	l.form.AddInputField(
		"Username   ", l.Username, 0, nil,
		func(s string) { l.Username = s },
	)

	l.form.AddPasswordField(
		"Password", l.Password, 0, '*',
		func(s string) { l.Password = s },
	)

	l.form.AddButton("Login", func() {

		l.loginHandler(l.Username, l.Password, l.Hub)
	})
	l.form.AddButton("Create User", func() {

		l.createUserHandler(l.Username, l.Password, l.Hub)
	})

	formContainer := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 1, false).              // Left spacer
		AddItem(l.form, 0, 2, true).             // Form
		AddItem(nil, 0, 1, false)                // Right spacer
	l.layout.AddItem(nil, 0, 1, false)
	l.layout.AddItem(formContainer, 0, 2, true).SetBorder(false)




}
