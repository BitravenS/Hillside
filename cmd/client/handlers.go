package main

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"hillside/internal/models"
	"hillside/internal/p2p"
	"hillside/internal/profile"
	"hillside/internal/utils"
	"log"
	"time"

	kyber "github.com/cloudflare/circl/kem/kyber/kyber1024"
	"github.com/cloudflare/circl/sign/dilithium/mode2"
	chacha "golang.org/x/crypto/chacha20poly1305"
)
type loginSession struct {
	Username string
	Password string
	Path string
}

func (cli *Client) loginHandler(username string, password string, hub string) {

	if username == "" || password == "" {
		cli.UI.ShowError("Error", "Username and password cannot be empty","OK", 0, nil)
		return
	}
	kb,usr, err := profile.LoadProfile(username, password, "")
	if err != nil {
			cli.UI.ShowError("Login failed", err.Error(), "Retry", 0, nil)
		return
	}
	cli.UI.ShowToast("Login successful", 3*time.Second,nil)
	cli.User = usr
	cli.Keybag = kb
	cli.Node.PK = kb.Libp2pPriv
	err = cli.Node.InitNode()
	if err != nil {
		cli.UI.ShowError("Node initialization failed", err.Error(),"OK", 0, nil)
		return
	}
	

	cli.SwitchToBrowseScreen(hub)

	return

}

func (cli *Client) createUserHandler(username string, password string, hub string) {

	if username == "" || password == "" {
		cli.UI.ShowError("Error", "Username and password cannot be empty","OK", 0, nil)
		return
	}
	prof, err := profile.GenerateProfile(username, password)
	if err != nil {
		cli.UI.ShowError("Create user failed", err.Error(), "OK", 0, nil)
		return
	}
	cli.UI.ShowToast("User created successfully! Welcome "+ prof.Username, 3*time.Second,nil)
	kb,usr, err := profile.LoadProfile(username, password, "")
	if err != nil {
			cli.UI.ShowError("Login failed", err.Error(), "Retry", 0, nil)
		return
	}
	cli.User = usr
	cli.Keybag = kb
	cli.Node.PK = kb.Libp2pPriv
	err = cli.Node.InitNode()
	if err != nil {
		cli.UI.ShowError("Node initialization failed", err.Error(),"OK", 0, nil)
		return
	}
	cli.SwitchToBrowseScreen(hub)


	return
}

func (cli *Client) SwitchToBrowseScreen(hub string) {
	cli.UI.Pages.SwitchToPage("browse")

	cli.Node.HubAddr = hub
	cli.UI.BrowseScreen.SetHub(hub)

	go cli.refreshServerList()
}

func (cli *Client) createServerHandler(request models.CreateServerRequest) (serverID string, err error) {
	if request.Name == "" {
		return "", utils.CreateServerError("Server name cannot be empty")
	}
	if request.Visibility == models.Private && len(request.PasswordHash) == 0 {
		return "" , utils.CreateServerError("Private servers must have a password")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", utils.CreateServerError("Failed to generate salt: " + err.Error())
	}
	request.PasswordSalt = salt
	hash := sha256.Sum256(request.PasswordHash)
	request.PasswordHash = hash[:]
	resp, err := cli.requestCreateServer(request)
	if err != nil {
		return "", utils.CreateServerError("Failed to create server: " + err.Error())
	}
	serverID = resp.ServerID
	go cli.refreshServerList()
	return serverID, nil
}

func (cli *Client) createRoomHandler(req models.CreateRoomRequest) (string, error) {
	if req.RoomName == "" {
		return "", utils.CreateRoomError("Room name cannot be empty")
	}
	if req.Visibility == models.Private && len(req.PasswordHash) == 0 {
		return "", utils.CreateRoomError("Private rooms must have a password")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", utils.CreateRoomError("Failed to generate salt: " + err.Error())
	}
	req.PasswordSalt = salt
	hash := sha256.Sum256(req.PasswordHash)
	req.PasswordHash = hash[:]
	resp, err := cli.requestCreateRoom(req)
	if err != nil {
		return "", utils.CreateRoomError("Failed to create room: " + err.Error())
	}
	go cli.refreshRoomList()
	return resp.RoomID, nil
}


func (cli *Client) joinServerHandler(serverID string, pass string) error {
	if serverID == "" {
		return utils.JoinServerError("Server ID cannot be empty")
	}


	err := cli.requestJoinServer(serverID, pass)
	if err != nil {
		return utils.JoinServerError(err.Error())
	}
	cli.UI.Pages.SwitchToPage("chat")
	cli.UI.ChatScreen.roomWrapper.SetTitle(fmt.Sprintf("[ %s ]", cli.Session.Server.Name))
	go cli.refreshRoomList()
	return nil
}

func (cli *Client) joinRoomHandler(roomID string, pass string) error {
	if roomID == "" {
		return utils.JoinRoomError("Server ID and Room ID cannot be empty")
	}
	err := cli.requestJoinRoom(roomID, pass)
	if err != nil {
		return utils.JoinRoomError(err.Error())
	}
	RekeyTopic := p2p.RekeyTopic(cli.Session.Server.ID, cli.Session.Room.ID)
	topic, err := cli.Node.PS.Join(RekeyTopic)
	if err != nil {
		return err
	}
	cli.Node.Topics.RekeyTopic = topic
	cli.UI.ChatScreen.chatSection.SetTitle(fmt.Sprintf("[ %s ]", cli.Session.Room.Name))

	cli.Session.RoomRatchet = &p2p.RoomRatchet{
		Index: 0,
		ChainKey:  make([]byte, 0),
	} //TODO: initialize the ratchet with the room's initial key
	if err = cli.chatHandler(); err != nil {
		return utils.JoinRoomError("Failed to initialize chat handler: " + err.Error())
	}
	go cli.refreshRoomList()
	return nil
}



func (cli *Client) chatHandler() error {
	kyberPriv, ok := cli.Keybag.KyberPriv.(*kyber.PrivateKey)
	if !ok {
		return errors.New("invalid KyberPriv type")
	}
	if err := cli.Node.ListenForRekeys(cli.Session.Server.ID, cli.Session.Room.ID, kyberPriv); err != nil {
		return err
	}
	chatTopic := p2p.ChatTopic(cli.Session.Server.ID, cli.Session.Room.ID)
	topic, err := cli.Node.PS.Join(chatTopic)
	if err != nil {
		return err
	}
	cli.Node.Topics.ChatTopic = topic
	sub, err := topic.Subscribe()
	if err != nil {
		return err
	}


	go func() error {
		for {
			msg, err := sub.Next(cli.Node.Ctx)
			if err != nil {
				return nil
			}
			env, message, err := models.UnmarshalEnvelope(msg.Data)
			if err != nil {
				return err
			}
			senderID := msg.ReceivedFrom
			err = cli.validateChatMessage(env, message.(*models.ChatMessage), senderID)
			if err != nil {

				if utils.IsValidationError(err) {
					cli.UI.ShowError("Validation Error", err.Error(), "OK", 0, nil)
				}
				 if utils.IsSecurityError(err) {
					cli.UI.ShowError("Security Error", err.Error(), "OK", 0, nil)
					// notify other clients
				}
			}

			castedMsg, ok := message.(*models.ChatMessage)
			if ok {
				pt, err := cli.decryptMessage(castedMsg)
				if err != nil {
					cli.UI.ShowError("Decryption Error", "Failed to decrypt message: "+err.Error(), "OK", 0, nil)
					continue
				}
				log.Fatal("Decrypted message: ", string(pt))
				decMsg := &models.DecrypetMessage{
					Sender : env.Sender,
					Timestamp: env.Timestamp,
					Content: string(pt),
					RoomID: cli.Session.Room.ID,
					ServerID: cli.Session.Server.ID,
				}
				cli.Session.Messages = append(cli.Session.Messages, *decMsg)
				line := fmt.Sprintf("[%d] %s: %s", env.Timestamp, env.Sender.Username, decMsg.Content)
				cli.UI.ChatScreen.chatSection.AddItem(line, "", 0, nil)
				
			}



		}
	}()
	return nil
   
}


func (cli *Client) decryptMessage(cm *models.ChatMessage) ([]byte, error) {
    // Advance ratchet to the message’s index
    for cli.Session.RoomRatchet.Index <= cm.ChainIndex {
        cli.Session.RoomRatchet.NextKey()
    }
    key, nonce, _ := cli.Session.RoomRatchet.NextKey()
    aead, _ := chacha.New(key)
    return aead.Open(nil, nonce, cm.Ciphertext, nil)
}

func (cli *Client) sendMessageHandler(text string) error {
    key, nonce, _ := cli.Session.RoomRatchet.NextKey()
    aead, _ := chacha.New(key)
    ct := aead.Seal(nil, nonce, []byte(text), nil)

	// 3b) Wrap in your unified envelope
	msg := &models.ChatMessage{
		ChainIndex: cli.Session.RoomRatchet.Index - 1,
		Ciphertext: ct,
	}

	dilithiumPriv, ok := cli.Keybag.DilithiumPriv.(*mode2.PrivateKey)
	if !ok {
		return errors.New("invalid DilithiumPriv type")
	}
	data, _ := models.Marshal(msg, *cli.User, dilithiumPriv)
	if cli.Node.Topics.ChatTopic == nil {
		return errors.New("chat topic is not initialized")
	}
	err := cli.Node.Topics.ChatTopic.Publish(cli.Node.Ctx, data)
	if err != nil {
		return err
	}
	return nil

}
/*

func (cli *Client) refreshMessageList() {
	// Clear the current message list

	// Add all messages from the session
	for _, msg := range cli.Session.Messages {
		cli.UI.ChatScreen.chatSection.AddMessage(msg)
	}
}*/