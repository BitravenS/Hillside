package ui

import "github.com/rivo/tview"

type UIConfig struct {
	Theme string
}

type UI struct {
	App *tview.Application
	pages *tview.Pages
	WelcomeUI *WelcomeUI

}

func NewUI(cfg *UIConfig) *UI {
	ui := &UI{
		App: tview.NewApplication().EnableMouse(true),
	}
	ui.WelcomeUI = &WelcomeUI{
		UI: ui,
	}
	ui.WelcomeUI.Init()
	ui.pages = tview.NewPages().
		AddPage("welcome", ui.WelcomeUI.layout, true, true)

	ui.App.SetRoot(ui.pages, true).
		SetFocus(ui.pages)
	return ui
}