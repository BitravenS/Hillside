package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"

	"hillside/internal/models"
	"hillside/internal/p2p"
	"hillside/internal/profile"
	"hillside/internal/storage"
	"hillside/internal/utils"

	"github.com/cloudflare/circl/kem/kyber/kyber1024"
	"github.com/cloudflare/circl/sign/dilithium/mode2"
	"github.com/gdamore/tcell/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	chacha "golang.org/x/crypto/chacha20poly1305"
)

func (cli *Client) loginHandler(username string, password string, hub string) {

	if username == "" || password == "" {
		cli.UI.ShowError("Error", "Username and password cannot be empty", "OK", 0, nil)
		return
	}
	kb, usr, err := profile.LoadProfile(username, password, "")
	if err != nil {
		cli.UI.ShowError("Login failed", err.Error(), "Retry", 0, nil)
		return
	}
	cli.User = usr
	cli.Keybag = kb
	cli.Session.Log.Logf("Loaded profile for user %s", username)

	hubadrr, err := peer.AddrInfoFromString(hub)
	if err != nil {
		cli.UI.ShowError("Invalid Hub Address", "Failed to parse hub address: "+err.Error(), "OK", 0, nil)
		return
	}
	cli.Node.Hub = hubadrr

	cli.Node.PK = kb.Libp2pPriv

	go func() {
		db, err := storage.InitSessionDB(username, "", 1024)
		if err != nil {
			cli.UI.ShowError("Storage Init Failed", "Failed to initialize storage: "+err.Error(), "OK", 0, nil)
			return
		}
		cli.Session.SessionDB = db

		if err := cli.Node.InitNode(); err != nil {
			cli.UI.App.QueueUpdateDraw(func() {
				cli.UI.ShowError("Node init failed", err.Error(), "OK", 0, nil)
			})
			return
		}
		cli.UI.App.QueueUpdateDraw(func() {
			cli.SwitchToBrowseScreen(hub)
		})
	}()

}

func (cli *Client) createUserHandler(username string, password string, hub string) {

	if username == "" || password == "" {
		cli.UI.ShowError("Error", "Username and password cannot be empty", "OK", 0, nil)
		return
	}
	_, err := profile.GenerateProfile(username, password)
	if err != nil {
		cli.UI.ShowError("Create user failed", err.Error(), "OK", 0, nil)
		return
	}
	kb, usr, err := profile.LoadProfile(username, password, "")
	if err != nil {
		cli.UI.ShowError("Login failed", err.Error(), "Retry", 0, nil)
		return
	}
	cli.User = usr
	cli.Keybag = kb

	hubadrr, err := peer.AddrInfoFromString(hub)
	if err != nil {
		cli.UI.ShowError("Invalid Hub Address", "Failed to parse hub address: "+err.Error(), "OK", 0, nil)
		return
	}
	cli.Node.Hub = hubadrr

	cli.Node.PK = kb.Libp2pPriv

	go func() {
		db, err := storage.InitSessionDB(username, "", 1024)
		if err != nil {
			cli.UI.ShowError("Storage Init Failed", "Failed to initialize storage: "+err.Error(), "OK", 0, nil)
			return
		}
		cli.Session.SessionDB = db
		// cli.UI.ShowError("Storage Success", fmt.Sprintf("db: %p", db), "OK", 0, nil)

		if err := cli.Node.InitNode(); err != nil {
			cli.UI.App.QueueUpdateDraw(func() {
				cli.UI.ShowError("Node init failed", err.Error(), "OK", 0, nil)
			})
			return
		}

		cli.UI.App.QueueUpdateDraw(func() {
			cli.SwitchToBrowseScreen(hub)
		})
	}()

}

func (cli *Client) SwitchToBrowseScreen(hub string) {
	cli.UI.BrowseScreen.SetHub(hub)
	cli.UI.Pages.SwitchToPage("browse")
	go cli.StartAutoRefresh()
}

func (cli *Client) SwitchToChatScreen() {
	cli.UI.Pages.SwitchToPage("chat")
	go cli.StartRoomAutoRefresh()
}

func (cli *Client) chatInputHandler() {
	fmt.Printf("Checking if layout is nil: %v\n", cli.UI.ChatScreen.layout == nil)
	cli.UI.ChatScreen.layout.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTAB {
			cli.UI.App.SetFocus(cli.UI.ChatScreen.chatSection) // Focus back to chat section on Escape
			return nil
		} else if event.Key() == tcell.KeyESC {
			cli.SwitchToBrowseScreen(cli.UI.BrowseScreen.Hub)
			return nil
		}
		return event
	})
}

func (cli *Client) createServerHandler(request models.CreateServerRequest) (serverID string, err error) {
	if request.Name == "" {
		return "", utils.CreateServerError("Server name cannot be empty")
	}
	if request.Visibility == models.Private && len(request.PasswordHash) == 0 {
		return "", utils.CreateServerError("Private servers must have a password")
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
	cli.SwitchToChatScreen()
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
	_, err = topic.Subscribe()
	if err != nil {
		return err
	}

	MembersTopic := p2p.MembersTopic(cli.Session.Server.ID, cli.Session.Room.ID)
	top, err := cli.Node.PS.Join(MembersTopic)
	if err != nil {
		return err
	}
	cli.Node.Topics.MembersTopic = top
	subs, err := cli.Node.Topics.MembersTopic.Subscribe()
	if err != nil {
		return err
	}
	go cli.refreshMembersList(subs)

	cli.UI.ChatScreen.chatSection.SetTitle(fmt.Sprintf("[ %s ]", cli.Session.Room.Name))
	/*
		if err = cli.Node.AdvertiseRoom(cli.Session.Server.ID, roomID); err != nil {
			return utils.JoinRoomError("Failed to advertise room: " + err.Error())
		}
	*/
	mbr, err := cli.requestListRoomMembers()
	if err != nil {
		return utils.JoinRoomError("Failed to list room members: " + err.Error())
	}

	members := mbr.Members
	for _, member := range members {
		if member.AddrInfo.ID == cli.Node.Host.ID() {
			// Skip self
			continue
		}
		if err := cli.Node.Host.Connect(cli.Node.Ctx, member.AddrInfo); err != nil {
			utils.JoinRoomError(
				fmt.Sprintf("connect %s failed: %s", member.AddrInfo.ID.String(), err))
		}
		cli.Session.Members = append(cli.Session.Members, member.User)

	}

	cli.Session.RoomRatchet = &p2p.RoomRatchet{
		Index:    0,
		ChainKey: make([]byte, 0),
	} //TODO: initialize the ratchet with the room's initial key
	cli.Session.BackupRatchet = &p2p.RoomRatchet{
		Index:    0,
		ChainKey: make([]byte, 0),
	}
	if err = cli.chatHandler(); err != nil {
		return utils.JoinRoomError("Failed to initialize chat handler: " + err.Error())
	}
	go cli.refreshRoomList()
	return nil
}

func (cli *Client) chatHandler() error {

	kyberPriv, ok := cli.Keybag.KyberPriv.(*kyber1024.PrivateKey)
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

	// Recieve messages from the chat topic
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
					//TODO: Notify others
				}
			}

			castedMsg, ok := message.(*models.ChatMessage)
			if ok {
				pt, err := cli.decryptMessage(castedMsg)
				if err != nil {
					cli.UI.ShowError("Decryption Error", "Failed to decrypt message: "+err.Error(), "OK", 0, nil)
					continue
				}
				decMsg := &models.DecrypetMessage{
					Sender:    env.Sender,
					Timestamp: env.Timestamp,
					Content:   string(pt),
					RoomID:    cli.Session.Room.ID,
					ServerID:  cli.Session.Server.ID,
				}
				cli.Session.Messages = append(cli.Session.Messages, *decMsg)
				if err := cli.Session.SessionDB.History.EnqueueEnvelope(cli.Node.Ctx, msg.Data, env, castedMsg, cli.Session.Room.ID, cli.Session.Server.ID); err != nil {
					cli.UI.ShowError("Storage Error", "Failed to store message: "+err.Error(), "OK", 0, nil)
				}
				//line := fmt.Sprintf("[%d] %s: %s", env.Timestamp, env.Sender.Username, decMsg.Content)
				formattedTime := utils.FormatPrettyTime(env.Timestamp)

				prefColor := env.Sender.PreferredColor
				if !utils.Contains(utils.BaseXtermAnsiColorNames, prefColor) {
					prefColor = utils.GenerateRandomColor()
				}

				lineContent := fmt.Sprintf("[yellow][%s] [%s]%s:[white] %s", formattedTime, prefColor, env.Sender.Username, decMsg.Content)
				cli.UI.App.QueueUpdateDraw(func() {
					cli.UI.ChatScreen.chatSection.AddItem(lineContent, "", 0, nil)

				})

			}

		}
	}()
	return nil

}

func (cli *Client) decryptMessage(cm *models.ChatMessage) ([]byte, error) {
	// Advance ratchet to the message’s index
	var key, nonce []byte
	var err error
	if cli.Session.RoomRatchet.Index <= cm.ChainIndex {
		for cli.Session.RoomRatchet.Index <= cm.ChainIndex {
			key, nonce, err = cli.Session.RoomRatchet.NextKey()
			if err != nil {
				return nil, fmt.Errorf("failed to get next key: %w", err)
			}
			for cli.Session.BackupRatchet.Index+10 <= cli.Session.RoomRatchet.Index {
				_, _, err = cli.Session.BackupRatchet.NextKey()
				if err != nil {
					return nil, fmt.Errorf("failed to get next backup key: %w", err)
				}
			}
		}
	} else if cli.Session.RoomRatchet.Index > cm.ChainIndex {
		rr := cli.Session.BackupRatchet
		for rr.Index <= cm.ChainIndex {
			key, nonce, err = rr.NextKey()
			if err != nil {
				return nil, fmt.Errorf("failed to get next key: %w", err)
			}
		}
	} else {
		return nil, fmt.Errorf("invalid chain index: %d, current index: %d", cm.ChainIndex, cli.Session.RoomRatchet.Index)
	}
	aead, err := chacha.New(key)
	if err != nil {

		return nil, fmt.Errorf("failed to create AEAD: %w | key: %x", err, key)
	}
	return aead.Open(nil, nonce, cm.Ciphertext, nil)
}

func (cli *Client) sendMessageHandler(text string) error {
	if cli.Session.RoomRatchet == nil {
		cli.UI.ShowError("Error", "You must join a room before sending messages", "OK", 0, nil)
		return utils.SendMessageError("Room ratchet is not initialized. Join a room first.")
	}
	key, nonce, err := cli.Session.RoomRatchet.NextKey()
	if err != nil {
		return fmt.Errorf("failed to get next key: %w", err)
	}
	aead, err := chacha.New(key)
	if err != nil {
		return fmt.Errorf("failed to create AEAD: %w", err)
	}
	ct := aead.Seal(nil, nonce, []byte(text), nil)

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
	err = cli.Node.Topics.ChatTopic.Publish(cli.Node.Ctx, data)
	if err != nil {
		return err
	}
	return nil

}

func (cli *Client) refreshMembersList(sub *pubsub.Subscription) error {

	for {
		msg, err := sub.Next(cli.Node.Ctx)
		if err != nil {
			return nil
		}
		var resp models.ListRoomMembersResponse
		if err = json.Unmarshal(msg.Data, &resp); err != nil {
			return err
		}

		if cli.Node.Hub.ID != msg.ReceivedFrom {
			return utils.SecurityError("Received message from unexpected peer: " + msg.ReceivedFrom.String())
		}

		member := resp.Members
		cli.Session.Log.Logf("Received %d members", len(member))

		for _, m := range member {
			if m.AddrInfo.ID == cli.Node.Host.ID() {
				// Skip self
				continue
			}
			cli.Session.Log.Logf("Member: %s", m.User.PeerID)
			alreadyInList := false
			for _, member := range cli.Session.Members {
				if member.PeerID == m.User.PeerID {
					alreadyInList = true
					break
				}
			}
			if alreadyInList {
				// Already in the list
				continue
			} else {
				cli.Session.Members = append(cli.Session.Members, m.User)
				cli.Session.SessionDB.Peers.EnqueueUserEntry(cli.Node.Ctx, &m.User)
				if err = cli.Node.Host.Connect(cli.Node.Ctx, m.AddrInfo); err != nil {
					return fmt.Errorf("connect %s failed: %w", m.AddrInfo.ID.String(), err)
				}
			}
		}

	}

}

/*
func (cli *Client) requestHistoryOnJoin() error {
	// compute sinceIndex (what you already have)
	var since uint64 = 0
	if cli.Session.HM != nil {
		if idx, err := cli.Session.HM.GetLastIndex(cli.Session.Room.ID); err == nil {
			since = idx
		}
	}

	replyTopic := p2p.HistoryRespTopic(cli.Session.Server.ID, cli.Session.Room.ID, cli.Node.Host.ID().String())

	req := models.CatchUpRequest{
		SinceIndex: since,
		Limit:      1000,
	}
	b, _ := json.Marshal(&req)
	return cli.Node.Topics.PublishToRoom(cli.Node.Ctx, p2p.HistoryReqTopic(cli.Session.Server.ID, cli.Session.Room.ID), b)
}

/*

func (cli *Client) refreshMessageList() {
	// Clear the current message list

	// Add all messages from the session
	for _, msg := range cli.Session.Messages {
		cli.UI.ChatScreen.chatSection.AddMessage(msg)
	}
}*/
