package client

import (
	"fmt"

	"github.com/gdamore/tcell/v2"

	"hillside/internal/models"
	"hillside/internal/utils"
)

func (cli *Client) SwitchToBrowseScreen(hub string) {
	cli.UI.BrowseScreen.SetHub(hub)
	cli.UI.Pages.SwitchToPage("browse")
	go cli.StartServerAutoRefresh()
}

func (cli *Client) SwitchToChatScreen() {
	cli.UI.Pages.SwitchToPage("chat")
	go cli.StartRoomAutoRefresh()
}

func (cli *Client) ChatInputHandler() {
	cli.UI.ChatScreen.Layout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTAB {
			cli.UI.App.SetFocus(cli.UI.ChatScreen.ChatSection) // Focus back to chat section on Escape
			return nil
		} else if event.Key() == tcell.KeyESC {
			cli.SwitchToBrowseScreen(cli.UI.BrowseScreen.Hub)
			return nil
		}
		return event
	})
}

func (cli *Client) FormatMessage(timestamp int64, sender models.User, decMsg *models.DecrypetMessage) string {
	formattedTime := utils.FormatPrettyTime(timestamp)

	prefColor := sender.PreferredColor
	if !utils.Contains(utils.BaseXtermAnsiColorNames, prefColor) {
		prefColor = utils.GenerateRandomColor()
	}
	lineContent := fmt.Sprintf("[yellow][%s] [%s]%s:[white] %s", formattedTime, prefColor, sender.Username, decMsg.Content)
	return lineContent
}

func (cli *Client) DisplayMessage(timestamp int64, sender models.User, decMsg *models.DecrypetMessage) {
	lineContent := cli.FormatMessage(timestamp, sender, decMsg)
	go func() {

		cli.UI.App.QueueUpdateDraw(func() {
			cli.UI.App.Lock()
			cli.Session.Log.Logf("Draw locked for displaying message")
			defer func() {
				cli.UI.App.Unlock()
				cli.Session.Log.Logf("Draw unlocked after displaying message")
			}()
			cli.UI.ChatScreen.ChatSection.AddItem(lineContent, "", 0, nil)
		})

	}()
}
