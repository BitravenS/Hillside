package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"hillside/internal/models"
	"hillside/internal/p2p"
	"hillside/internal/utils"
)

type Client struct {
	User    *models.User
	Keybag  *models.Keybag
	Node    *p2p.Node
	UI      *UI
	Session *p2p.Session
}

func main() {
	logFile, err := os.OpenFile("panic.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	run()
}

func run() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic: %v\nStack trace:\n%s", r, debug.Stack())
			fmt.Println("A fatal error occurred. Please check panic.log for details.")
			os.Exit(2)
		}
	}()

	client := &Client{}
	ctx := context.Background()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Failed to get user home directory: %v", err)
		panic(err)
	}
	theme, err := LoadThemeFromDir(homeDir+"/.hillside/", "default_theme")
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
		getServerId:         client.getServerID,
		createRoomHandler:   client.createRoomHandler,
		joinRoomHandler:     client.joinRoomHandler,
		sendMessageHandler:  client.sendMessageHandler,
		chatInputHandler:    client.chatInputHandler,
	})

	fmt.Println("Starting Hillside Client...")
	client.UI.ChatScreen.inputHandler = client.chatInputHandler
	client.UI.ChatScreen.HookupInputHandler()
	node := &p2p.Node{
		Ctx: ctx,
	}
	client.Node = node
	var logPort int
	flag.IntVar(&logPort, "logport", 4567, "Port for remote logger")
	flag.Parse()
	rl, err := utils.NewRemoteLogger(logPort)
	if err != nil {
		log.Printf("Failed to start remote logger: %v", err)
	}
	rl.Logf("Hillside Client started on port %d", logPort)

	client.Session = p2p.NewSession(nil, rl)

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
