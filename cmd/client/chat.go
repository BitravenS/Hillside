package main

import (
	"fmt"

	"github.com/rivo/tview"
)


type ChatScreen struct {
	*UI
	layout        *tview.Flex
	GetServerName func() string
	roomList	 *tview.List
	roomPane	 *tview.Flex
	chatView      *tview.Flex
}


func (c *ChatScreen) NewChatScreen() {
	c.layout = tview.NewFlex()
	c.layout.SetDirection(tview.FlexRow).
		SetBorder(false)

	c.roomList = tview.NewList()
    c.roomList.SetSelectedBackgroundColor(c.UI.Theme.GetColor("background-light"))
    c.roomList.SetSelectedTextColor(c.UI.Theme.GetColor("primary")).
        SetHighlightFullLine(true)

	c.roomList.
        SetTitleColor(c.Theme.GetColor("primary")).
        SetBackgroundColor(c.Theme.GetColor("background"))


	c.roomPane = tview.NewFlex()
    c.roomPane.AddItem(c.roomList, 0, 1, true)

    c.roomPane.SetDirection(tview.FlexRow)
    c.roomPane.SetBorder(true).
        SetTitle(fmt.Sprintf("[ %s ]", c.GetServerName())).
        SetTitleColor(c.Theme.GetColor("primary")).
        SetBorderColor(c.Theme.GetColor("border")).
        SetBackgroundColor(c.Theme.GetColor("background")).
        SetBorderPadding(2,2,2,2)

}