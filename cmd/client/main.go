package main

import (
	"context"
	"hillside/internal/models"
	"hillside/internal/p2p"
)

type Session struct {
	Server *models.ServerMeta
	Room *models.RoomMeta
	RoomRatchet *p2p.RoomRatchet
	Members []models.User
	Messages []models.DecrypetMessage
	Password string
}
type Client struct {
	User *models.User
	Keybag *models.Keybag
	Node *p2p.Node
	UI *UI
	Session *Session
}



func main() {
	client := &Client{}
	ctx := context.Background()

	theme, err := LoadThemeFromDir("/home/bitraven/.hillside/","default_theme")
	if err != nil {
		panic("Failed to load default theme: " + err.Error())
	}
	client.UI = NewUI(&UIConfig{
		Theme: theme,
		loginHandler: client.loginHandler,
		createUserHandler: client.createUserHandler,
		createServerHandler: client.createServerHandler,
		joinServerHandler: client.joinServerHandler,
		getServerName: client.getServerName,
		getRoomName: client.getRoomName,
		getServerId: client.getServerId,
		createRoomHandler: client.createRoomHandler,
		joinRoomHandler: client.joinRoomHandler,
		sendMessageHandler: client.sendMessageHandler,

	})
	node := &p2p.Node{
		Ctx: ctx,
	}
	client.Node = node
	client.Session = &Session{
		Server: nil,
		Room: nil,
		Password: "",
	}
	if err := client.UI.App.Run(); err != nil {
		panic(err)
	}


}