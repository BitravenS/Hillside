package ui

import (
	"github.com/rivo/tview"
)

type LoginScreen struct {
    Form             *tview.Form
    DefaultHubAddr   string
    OnLogin          func(hubAddr, username string)
}

func NewLoginScreen() *LoginScreen {
    f := tview.NewForm().
        AddInputField("Hub:", "", 40, nil, nil).
        AddInputField("Username:", "", 20, nil, nil).
        AddPasswordField("Password:", "", 20, '*', nil).
        AddButton("Login", nil).
        AddButton("Quit", nil)

    screen := &LoginScreen{
        Form: f,
    }

    // TODO: set button actions
    f.GetButton(0).SetSelectedFunc(func() { // Login
        hub := f.GetFormItemByLabel("Hub:").(*tview.InputField).GetText()
        user := f.GetFormItemByLabel("Username:").(*tview.InputField).GetText()
        if screen.OnLogin != nil {
            screen.OnLogin(hub, user)
        }
    })
    f.GetButton(1).SetSelectedFunc(func() {
        // Exit application
        screen.Form.
    })

    return screen
}