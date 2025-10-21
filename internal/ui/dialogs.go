package ui

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (ui *UI) ShowToast(message string, duration time.Duration, onDismiss func()) {
	modal := tview.NewModal()
	buttonStyle := tcell.StyleDefault.
		Background(ui.Theme.GetColor("background")).
		Foreground(ui.Theme.GetColor("primary"))
	buttonStyleActive := tcell.StyleDefault.
		Background(ui.Theme.GetColor("primary")).
		Foreground(ui.Theme.GetColor("background"))
	modal.SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			ui.Pages.RemovePage("toast")
			if onDismiss != nil {
				onDismiss()
			}
		}).SetButtonStyle(buttonStyle).
		SetButtonActivatedStyle(buttonStyleActive)
	modal.SetBackgroundColor(ui.Theme.GetColor("background")).
		SetBorder(true).
		SetBorderColor(ui.Theme.GetColor("primary")).
		SetBackgroundColor(ui.Theme.GetColor("background"))

	ui.Pages.AddPage("toast", modal, true, true)
	ui.App.SetFocus(modal)

	// Auto-dismiss after duration
	if duration > 0 {
		go func() {
			time.Sleep(duration)
			ui.App.QueueUpdateDraw(func() {
				ui.Pages.RemovePage("toast")
				if onDismiss != nil {
					onDismiss()
				}
			})
		}()
	}
}

func (ui *UI) ShowError(title string, message string, actionName string, duration time.Duration, onDismiss func()) {
	modal := tview.NewModal()
	buttonStyle := tcell.StyleDefault.
		Background(ui.Theme.GetColor("background")).
		Foreground(ui.Theme.GetColor("red"))
	buttonStyleActive := tcell.StyleDefault.
		Background(ui.Theme.GetColor("red")).
		Foreground(ui.Theme.GetColor("background"))
	modal.SetText(message).
		AddButtons([]string{actionName}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			ui.Pages.RemovePage("error")
			if onDismiss != nil {
				onDismiss()
			}
		}).SetButtonStyle(buttonStyle).
		SetButtonActivatedStyle(buttonStyleActive)
	modal.SetBackgroundColor(ui.Theme.GetColor("background")).
		SetBorder(true).
		SetBorderColor(ui.Theme.GetColor("red")).
		SetBackgroundColor(ui.Theme.GetColor("background")).
		SetTitle(title).
		SetTitleColor(ui.Theme.GetColor("red")).
		SetTitleAlign(tview.AlignCenter)

	ui.Pages.AddPage("error", modal, true, true)
	ui.App.SetFocus(modal)

	// Auto-dismiss after duration
	if duration > 0 {
		go func() {
			time.Sleep(duration)
			ui.App.QueueUpdateDraw(func() {
				ui.Pages.RemovePage("error")
				if onDismiss != nil {
					onDismiss()
				}
			})
		}()
	}
}
