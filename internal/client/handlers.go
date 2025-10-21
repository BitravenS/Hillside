package client

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"hillside/internal/crypto"
	"hillside/internal/models"
	"hillside/internal/p2p"
	"hillside/internal/profile"
	"hillside/internal/storage"
	"hillside/internal/utils"

	"github.com/gdamore/tcell/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	chacha "golang.org/x/crypto/chacha20poly1305"
)

func (cli *Client) LoginHandler(username string, password string, hub string) {

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

func (cli *Client) CreateUserHandler(username string, password string, hub string) {

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

func (cli *Client) CreateServerHandler(request models.CreateServerRequest) (serverID string, err error) {
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

func (cli *Client) CreateRoomHandler(req models.CreateRoomRequest) (string, error) {
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
	_, masterKey, err := crypto.GenerateRoomKey()
	if err != nil {
		return "", utils.CreateRoomError("Failed to generate room key: " + err.Error())
	}

	cli.Session.SessionDB.Store.SaveAuth(cli.Node.Ctx, resp.RoomID, 0, masterKey, time.Now())
	go cli.refreshRoomList()
	return resp.RoomID, nil
}

func (cli *Client) JoinServerHandler(serverID string, pass string) error {
	if serverID == "" {
		return utils.JoinServerError("Server ID cannot be empty")
	}

	err := cli.requestJoinServer(serverID, pass)
	if err != nil {
		return utils.JoinServerError(err.Error())
	}
	cli.SwitchToChatScreen()
	cli.UI.ChatScreen.RoomWrapper.SetTitle(fmt.Sprintf("[ %s ]", cli.GetServerName()))
	go cli.refreshRoomList()
	cli.Session.Log.Logf("Joined server %s", serverID)
	return nil
}

func (cli *Client) JoinRoomHandler(roomID string, pass string) error {
	if roomID == "" {
		return utils.JoinRoomError("Server ID and Room ID cannot be empty")
	}
	cli.Session.Log.Logf("Joining room %s", roomID)
	err := cli.requestJoinRoom(roomID, pass)
	if err != nil {
		return utils.JoinRoomError(err.Error())
	}
	cli.Session.Log.Logf("Requested to join room %s", roomID)
	/* TODO: Add rekeying
	*
				RekeyTopic := p2p.RekeyTopic(cli.GetServerID(), cli.GetRoomID())
				topic, err := cli.Node.PS.Join(RekeyTopic)
				if err != nil {
					return err
				}
				cli.Node.Topics.RekeyTopic = topic
				_, err = topic.Subscribe()
				if err != nil {
					return err
				}
				cli.Session.Log.Logf("Subscribed to rekey topic for room %s", roomID)
	*/
	if !cli.Session.Current.Room.Topics.HasTopic(models.TopicMembers) {

		MembersTopic := p2p.MembersTopic(cli.GetServerID(), cli.GetRoomID())
		top, err := cli.Node.PS.Join(MembersTopic)
		if err != nil {
			return err
		}
		cli.Session.Current.Room.Topics.SetTopic(models.TopicMembers, top)
	}
	subs, err := cli.Session.Current.Room.Topics.GetTopic(models.TopicMembers).Subscribe()
	if err != nil {
		return err
	}
	cli.Session.Log.Logf("Subscribed to members topic for room %s", roomID)
	go cli.refreshMembersList(subs)
	cli.Session.Log.Logf("Refreshing members list for room %s", roomID)

	mbr, err := cli.requestListRoomMembers()
	if err != nil {
		return utils.JoinRoomError("Failed to list room members: " + err.Error())
	}

	members := mbr.Members
	for _, member := range members {
		cli.Session.Log.Logf("Connecting to member %s for room %s", member.User.PeerID, roomID)
		if member.AddrInfo.ID == cli.Node.Host.ID() {
			// Skip self
			continue
		}
		if err := cli.Node.Host.Connect(cli.Node.Ctx, member.AddrInfo); err != nil {
			utils.JoinRoomError(
				fmt.Sprintf("connect %s failed: %s", member.AddrInfo.ID.String(), err))
		}
		cli.Session.Current.Room.Members = append(cli.Session.Current.Room.Members, member.User)
		err = cli.Session.SessionDB.Peers.EnqueueUserEntry(cli.Node.Ctx, &member.User)
		if err != nil {
			cli.Session.Log.Logf("Failed to enqueue user %s: %v", member.User.PeerID, err)
		}
		cli.Session.Log.Logf("Connected to member %s for room %s", member.User.PeerID, roomID)

	}

	if !cli.Session.Current.Room.Topics.HasTopic(models.TopicCatchUp) {

		CatchupReq := p2p.CatchUpRequestTopic(cli.GetServerID(), roomID)
		top, err := cli.Node.PS.Join(CatchupReq)
		if err != nil {
			return err
		}
		cli.Session.Current.Room.Topics.SetTopic(models.TopicCatchUp, top)
	}
	cli.Session.Log.Logf("Connected to %d members for room %s", len(members), roomID)

	// Check if we have room auth stored
	roomAuth, err := cli.Session.SessionDB.Store.GetAuth(cli.Node.Ctx, roomID)
	cli.Session.Log.Logf("Fetched room auth for room %s? %+v", roomID, err)
	var ratchet *crypto.RoomRatchet
	if err != nil {
		if errors.Is(err, storage.ErrNoRows) {
			cli.Session.Log.Logf("No room auth found for room %s, requesting catch-up", roomID)

			ratchet, err = cli.requestCatchUp(0, 0) // TODO: since, limit
			if err != nil {
				return utils.JoinRoomError("Failed to catch up: " + err.Error())
			}
		} else {

			return utils.JoinRoomError("Failed to Get room auth: " + err.Error())
		}

	} else {
		ratchet = &crypto.RoomRatchet{
			Index:    roomAuth.ChainIndex,
			ChainKey: roomAuth.MasterRatchetKey,
		}
	}

	if ratchet == nil {
		return utils.JoinRoomError("Room ratchet is not initialized after catch-up")
	}
	cli.Session.Current.Room.SetInitialRatchet(ratchet)

	if err = cli.chatHandler(); err != nil {
		cli.Session.Log.Logf("Failed to initialize chat handler for room %s: %+v", roomID, err)
		return utils.JoinRoomError("Failed to initialize chat handler: " + err.Error())
	}
	go cli.refreshRoomList()
	sub, err := cli.Session.Current.Room.Topics.GetTopic(models.TopicCatchUp).Subscribe()
	if err != nil {
		return err
	}
	cli.UI.ChatScreen.ChatSection.SetTitle(fmt.Sprintf("[ %s ]", cli.GetRoomName()))
	cli.Session.Log.Logf("Set title for chat section for room %s", roomID)
	go func() error {
		err = cli.helpCatchUp(sub)
		cli.Session.Log.Logf("Finished catch-up for room %s: %+v", roomID, cli.Session.Current.Room.RoomRatchet)
		if err != nil {
			cli.Session.Log.Logf("Catch-up error for room %s: %+v", roomID, err)
			return err
		}
		return nil
	}()
	return nil
}

func (cli *Client) parseAndDisplayDBMessages(roomID string) error {
	msgs, err := cli.Session.SessionDB.Store.GetLatestMessages(cli.Node.Ctx, roomID, 200)
	if err != nil {
		return err
	}
	cli.Session.Log.Logf("Fetched %d messages from DB for room %s", len(msgs), roomID)
	for _, msg := range msgs {
		var cm *models.ChatMessage
		err := json.Unmarshal(msg.Payload, &cm)
		if err != nil {
			return err
		}
		cli.Session.Log.Logf("Decrypting message with chain index %d", cm.ChainIndex)
		pt, err := cli.decryptMessage(cm)
		if err != nil {
			cli.Session.Log.Logf("Failed to decrypt message: %v", err)
			return err
		}
		cli.Session.Log.Logf("Decrypted message: %s", string(pt))

		var sender *models.User
		sender, err = cli.Session.SessionDB.Store.GetUserByID(cli.Node.Ctx, msg.SenderID)
		if err != nil {
			cli.Session.Log.Logf("Failed to Get sender %s: %v", msg.SenderID, err)
			sender = &models.User{
				PeerID:   msg.SenderID,
				Username: "Unknown",
			}
		}
		decMsg := &models.DecrypetMessage{
			Sender:    *sender,
			Timestamp: msg.Timestamp,
			Content:   string(pt),
			RoomID:    cli.GetRoomID(),
			ServerID:  cli.GetServerID(),
		}
		cli.Session.Log.Logf("Displaying message from %s: %s", sender.Username, decMsg.Content)
		cli.DisplayMessage(msg.Timestamp, *sender, decMsg)
		cli.Session.Current.Room.Messages = append(cli.Session.Current.Room.Messages, *decMsg)
	}
	return nil
}

func (cli *Client) decryptMessage(cm *models.ChatMessage) ([]byte, error) {
	// Advance ratchet to the message’s index
	var key, nonce []byte
	var err error
	if cli.Session.Current.Room.RoomRatchet.Index <= cm.ChainIndex {
		for cli.Session.Current.Room.RoomRatchet.Index <= cm.ChainIndex {
			key, nonce, err = cli.Session.Current.Room.RoomRatchet.NextKey()
			if err != nil {
				return nil, fmt.Errorf("failed to Get next key: %w", err)
			}
			for cli.Session.Current.Room.BackupRatchet.Index+10 <= cli.Session.Current.Room.RoomRatchet.Index {
				_, _, err = cli.Session.Current.Room.BackupRatchet.NextKey()
				if err != nil {
					return nil, fmt.Errorf("failed to Get next backup key: %w", err)
				}
			}
		}
	} else if cli.Session.Current.Room.RoomRatchet.Index > cm.ChainIndex {
		rr := cli.Session.Current.Room.BackupRatchet
		for rr.Index <= cm.ChainIndex {
			key, nonce, err = rr.NextKey()
			if err != nil {
				return nil, fmt.Errorf("failed to Get next key: %w", err)
			}
		}
	} else {
		return nil, fmt.Errorf("invalid chain index: %d, current index: %d", cm.ChainIndex, cli.Session.Current.Room.RoomRatchet.Index)
	}
	cli.Session.Log.Logf("Decrypting message at chain index %d with key %x and nonce %x", cm.ChainIndex, key, nonce)
	aead, err := chacha.New(key)
	if err != nil {

		return nil, fmt.Errorf("failed to create AEAD: %+v , %x", err, key)
	}
	return aead.Open(nil, nonce, cm.Ciphertext, nil)
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
			for _, member := range cli.Session.Current.Room.Members {
				if member.PeerID == m.User.PeerID {
					alreadyInList = true
					break
				}
			}
			if alreadyInList {
				// Already in the list
				continue
			} else {
				cli.Session.Current.Room.Members = append(cli.Session.Current.Room.Members, m.User)
				err = cli.Session.SessionDB.Peers.EnqueueUserEntry(cli.Node.Ctx, &m.User)
				if err != nil {
					cli.Session.Log.Logf("Failed to enqueue user %s: %v", m.User.PeerID, err)
				}
				cli.Session.Log.Logf("Added member %s", m.User.PeerID)
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

	replyTopic := p2p.HistoryRespTopic(cli.GetServerID(), cli.Session.Room.ID, cli.Node.Host.ID().String())

	req := models.CatchUpRequest{
		SinceIndex: since,
		Limit:      1000,
	}
	b, _ := json.Marshal(&req)
	return cli.Node.Topics.PublishToRoom(cli.Node.Ctx, p2p.HistoryReqTopic(cli.GetServerID(), cli.Session.Room.ID), b)
}

/*

func (cli *Client) refreshMessageList() {
	// Clear the current message list

	// Add all messages from the session
	for _, msg := range cli.Session.Messages {
		cli.UI.ChatScreen.chatSection.AddMessage(msg)
	}
}*/

func (cli *Client) refreshRoomList() {
	roomResp, err := cli.requestRooms(cli.GetServerID())
	cli.UI.App.QueueUpdateDraw(func() {
		if err != nil {
			cli.UI.ShowError("Server Error", err.Error(), "Go back to Browse view", 0, func() {
				cli.UI.Pages.SwitchToPage("browse")
			})
			return
		} else {
			cli.UI.ChatScreen.UpdateRoomList(roomResp.Rooms)
		}
	})
}
func (cli *Client) refreshServerList() {
	serverResp, err := cli.requestServers()
	cli.UI.App.QueueUpdateDraw(func() {
		if err != nil {
			cli.UI.ShowError("Server Error", err.Error(), "Go back to Login", 0, func() {
				cli.UI.Pages.SwitchToPage("login")
			})
			return
		} else {
			cli.UI.BrowseScreen.UpdateServerList(serverResp.Servers)
		}
	})
}

func (cli *Client) StartAutoRefresh() {
	cli.refreshServerList()
	currentPage, _ := cli.UI.Pages.GetFrontPage()
	cli.Session.Log.Logf("Starting auto-refresh on page: %s", currentPage)

	// cancellable context for this auto-refresh session
	refreshCtx, cancelRefresh := context.WithCancel(cli.Node.Ctx)
	defer cancelRefresh()

	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				currentPage, _ := cli.UI.Pages.GetFrontPage()
				if !utils.IsBrowsePageActive(currentPage) {
					cli.Session.Log.Logf("Left browse page, stopping auto-refresh")
					cancelRefresh() // This will break the message loop
					return
				}
			case <-refreshCtx.Done():
				return
			}
		}
	}()

	ServersTopic := p2p.ServersTopic()
	top, err := cli.Node.PS.Join(ServersTopic)
	if err != nil {
		cli.Session.Log.Logf("Failed to join servers topic: %v", err)
		return
	}

	sub, err := top.Subscribe()
	if err != nil {
		cli.Session.Log.Logf("Failed to subscribe to servers topic: %v", err)
		return
	}
	defer sub.Cancel()

	for {
		select {
		case <-refreshCtx.Done():
			cli.Session.Log.Logf("Auto-refresh cancelled")
			return
		default:
			msg, err := sub.Next(refreshCtx)
			if err != nil {
				if err == context.Canceled {
					cli.Session.Log.Logf("Auto-refresh stopped due to page change")

					currentPage, _ := cli.UI.Pages.GetFrontPage()
					cli.Session.Log.Logf("Current page: %s", currentPage)
				} else {
					cli.Session.Log.Logf("Error reading from servers topic: %v", err)
				}
				return
			}

			var listResp models.ListServersResponse
			err = json.Unmarshal(msg.Data, &listResp)
			if err != nil {
				cli.Session.Log.Logf("Error unmarshaling servers topic message: %v", err)

			}
			cli.Session.Log.Logf("Received server list update with %d servers", len(listResp.Servers))
			cli.UI.App.QueueUpdateDraw(func() {
				if err != nil {
					cli.UI.ShowError("Server Error", err.Error(), "Go back to Login", 0, func() {
						cli.UI.Pages.SwitchToPage("login")
					})
					return
				} else {
					cli.UI.BrowseScreen.UpdateServerList(listResp.Servers)
				}
			})
		}
	}
}

func (cli *Client) StartRoomAutoRefresh() {
	cli.refreshServerList()
	currentPage, _ := cli.UI.Pages.GetFrontPage()
	cli.Session.Log.Logf("Starting auto-refresh on page: %s", currentPage)

	refreshCtx, cancelRefresh := context.WithCancel(cli.Node.Ctx)
	defer cancelRefresh()

	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				currentPage, _ := cli.UI.Pages.GetFrontPage()
				if !utils.IsChatPageActive(currentPage) {
					cli.Session.Log.Logf("Left chat page, stopping auto-refresh")
					cancelRefresh()
					return
				}
			case <-refreshCtx.Done():
				return
			}
		}
	}()

	if !cli.Session.Current.Server.Topics.HasTopic(models.TopicRooms) {

		RoomsTopic := p2p.RoomsTopic(cli.GetServerID())
		top, err := cli.Node.PS.Join(RoomsTopic)
		if err != nil {
			cli.Session.Log.Logf("Failed to join rooms topic: %v", err)
			return
		}
		cli.Session.Current.Server.Topics.SetTopic(models.TopicRooms, top)
	}

	sub, err := cli.Session.Current.Server.Topics.GetTopic(models.TopicRooms).Subscribe()
	if err != nil {
		cli.Session.Log.Logf("Failed to subscribe to rooms topic: %v", err)
		return
	}
	defer sub.Cancel()

	for {
		select {
		case <-refreshCtx.Done():
			cli.Session.Log.Logf("Auto-refresh cancelled")
			return
		default:
			msg, err := sub.Next(refreshCtx)
			if err != nil {
				if err == context.Canceled {
					cli.Session.Log.Logf("Auto-refresh stopped due to page change")

					currentPage, _ := cli.UI.Pages.GetFrontPage()
					cli.Session.Log.Logf("Current page: %s", currentPage)
				} else {
					cli.Session.Log.Logf("Error reading from rooms topic: %v", err)
				}
				return
			}

			var listResp models.ListRoomsResponse
			err = json.Unmarshal(msg.Data, &listResp)
			if err != nil {
				cli.Session.Log.Logf("Error unmarshaling rooms topic message: %v", err)

			}
			cli.Session.Log.Logf("Received rooms list update with %d rooms", len(listResp.Rooms))
			cli.UI.App.QueueUpdateDraw(func() {
				if err != nil {
					cli.UI.ShowError("Rooms Error", err.Error(), "Go back to servers", 0, func() {
						cli.UI.Pages.SwitchToPage("browse")
					})
					return
				} else {
					cli.UI.ChatScreen.UpdateRoomList(listResp.Rooms)
				}
			})
		}
	}
}
