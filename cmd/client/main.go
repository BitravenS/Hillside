package main

import (
	"context"
	"fmt"
	"log"

	"hillside/internal/models"
	"hillside/internal/p2p"
	"hillside/internal/storage"
	"hillside/internal/utils"
)

type Session struct {
	Server        *models.ServerMeta
	Room          *models.RoomMeta
	RoomRatchet   *p2p.RoomRatchet
	BackupRatchet *p2p.RoomRatchet // Ratchet for 5 epochs behind
	Members       []models.User
	Messages      []models.DecrypetMessage
	Password      string
	SessionDB     *storage.SessionDB
	Log           *utils.RemoteLogger
}
type Client struct {
	User    *models.User
	Keybag  *models.Keybag
	Node    *p2p.Node
	UI      *UI
	Session *Session
}

func main() {
	client := &Client{}
	ctx := context.Background()

	theme, err := LoadThemeFromDir("/home/bitraven/.hillside/", "default_theme")
	if err != nil {
		panic("Failed to load default theme: " + err.Error())
	}
	client.UI = NewUI(&UIConfig{
		Theme:               theme,
		loginHandler:        client.loginHandler,
		createUserHandler:   client.createUserHandler,
		createServerHandler: client.createServerHandler,
		joinServerHandler:   client.joinServerHandler,
		getServerName:       client.getServerName,
		getRoomName:         client.getRoomName,
		getServerId:         client.getServerId,
		createRoomHandler:   client.createRoomHandler,
		joinRoomHandler:     client.joinRoomHandler,
		sendMessageHandler:  client.sendMessageHandler,
		chatInputHandler:    client.chatInputHandler,
	})

	fmt.Println("Starting Hillside Client...")
	fmt.Println("Is ui chatscreen nil?", client.UI.ChatScreen == nil)
	client.UI.ChatScreen.inputHandler = client.chatInputHandler
	client.UI.ChatScreen.HookupInputHandler()
	node := &p2p.Node{
		Ctx: ctx,
	}
	client.Node = node
	rl, err := utils.NewRemoteLogger(4567)
	if err != nil {
		log.Printf("Failed to start remote logger: %v", err)
	}

	client.Session = &Session{
		Server:   nil,
		Room:     nil,
		Password: "",
		Log:      rl,
	}

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
