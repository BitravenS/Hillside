package main

import (
	"context"
	"hillside/internal/models"
	"hillside/internal/p2p"
)

type Client struct {
	User *models.User
	Keybag *models.Keybag
	Node *p2p.Node
	UI *UI
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
	})
	node := &p2p.Node{
		Ctx: ctx,
	}
	client.Node = node
	
	if err := client.UI.App.Run(); err != nil {
		panic(err)
	}


}