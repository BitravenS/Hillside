// Package client provides the main client logic and application for Hillside.
package client

import (
	"context"
	"fmt"
	"log"
	"os"

	"hillside/internal/models"
	"hillside/internal/p2p"
	"hillside/internal/ui"
	"hillside/internal/utils"
)

type Client struct {
	User    *models.User
	Keybag  *models.Keybag
	Node    *p2p.Node
	UI      *ui.UI
	Session *Session
}

func StartClientApp(logPort int) {

	client := &Client{}
	ctx := context.Background()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Failed to Get user home directory: %v", err)
		panic(err)
	}
	theme, err := ui.LoadThemeFromDir(homeDir+"/.hillside/", "default_theme")
	if err != nil {
		panic("Failed to load default theme: " + err.Error())
	}
	client.UI = ui.NewUI(&ui.UIConfig{
		Theme:               theme,
		LoginHandler:        client.LoginHandler,
		CreateUserHandler:   client.CreateUserHandler,
		CreateServerHandler: client.CreateServerHandler,
		JoinServerHandler:   client.JoinServerHandler,
		GetServerName:       client.GetServerName,
		GetRoomName:         client.GetRoomName,
		GetServerID:         client.GetServerID,
		CreateRoomHandler:   client.CreateRoomHandler,
		JoinRoomHandler:     client.JoinRoomHandler,
		SendMessageHandler:  client.SendMessageHandler,
		ChatInputHandler:    client.ChatInputHandler,
	})

	fmt.Println("Starting Hillside Client...")
	client.UI.ChatScreen.InputHandler = client.ChatInputHandler
	client.UI.ChatScreen.HookupInputHandler()
	node := &p2p.Node{
		Ctx: ctx,
	}
	client.Node = node

	rl, err := utils.NewRemoteLogger(logPort)
	if err != nil {
		log.Printf("Failed to start remote logger: %v", err)
	}
	rl.Logf("Hillside Client started on port %d", logPort)

	client.Session = NewSession(nil, rl)

	defer func() {
		if err := client.Shutdown(); err != nil {
			log.Printf("[Shutdown] Failed to close node resources: %v", err)
		}
	}()

	if err := client.UI.App.Run(); err != nil {
		log.Printf("Failed to run UI: %v", err)
		panic(err)
	}

}
