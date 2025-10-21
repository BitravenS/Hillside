package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"hillside/internal/models"
	"hillside/internal/p2p"
	"hillside/internal/utils"

	"github.com/cloudflare/circl/sign/dilithium/mode2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"golang.org/x/crypto/argon2"
	chacha "golang.org/x/crypto/chacha20poly1305"
)

func (cli *Client) requestServers() (*models.ListServersResponse, error) {
	var listResp models.ListServersResponse
	err := cli.Node.SendRPC("ListServers", models.ListServersRequest{}, &listResp)
	if err != nil {
		return nil, err
	}
	return &listResp, nil

}

func (cli *Client) requestCreateServer(req models.CreateServerRequest) (*models.CreateServerResponse, error) {
	var resp models.CreateServerResponse
	err := cli.Node.SendRPC("CreateServer", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (cli *Client) requestCreateRoom(req models.CreateRoomRequest) (*models.CreateRoomResponse, error) {
	var resp models.CreateRoomResponse
	err := cli.Node.SendRPC("CreateRoom", req, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
	}
	return &resp, nil
}

func (cli *Client) requestRooms(serverID string) (*models.ListRoomsResponse, error) {
	var roomsResp models.ListRoomsResponse
	err := cli.Node.SendRPC("ListRooms", models.ListRoomsRequest{ServerID: serverID}, &roomsResp)
	if err != nil {
		return nil, err
	}
	return &roomsResp, nil
}

func (cli *Client) getServerName() string {
	if cli.Session == nil {
		return fmt.Errorf("SESSION IS NOT INITIALIZED").Error()
	}
	if cli.Session.Current.Server == nil {
		return fmt.Errorf("NO SERVER JOINED").Error()
	}
	return cli.Session.Current.Server.ServerMeta.Name
}

func (cli *Client) getServerID() string {
	if cli.Session == nil {
		return fmt.Errorf("SESSION IS NOT INITIALIZED").Error()
	}
	if cli.Session.Current.Server == nil {
		return fmt.Errorf("NO SERVER JOINED").Error()
	}
	return cli.Session.Current.Server.ServerMeta.ID
}

func (cli *Client) getRoomName() string {
	if cli.Session == nil {
		return fmt.Errorf("SESSION IS NOT INITIALIZED").Error()
	}
	if cli.Session.Current.Room == nil {
		return fmt.Errorf("NO SERVER JOINED").Error()
	}
	return cli.Session.Current.Room.RoomMeta.Name
}

func (cli *Client) getRoomID() string {
	if cli.Session == nil {
		return fmt.Errorf("SESSION IS NOT INITIALIZED").Error()
	}
	if cli.Session.Current.Room == nil {
		return fmt.Errorf("NO SERVER JOINED").Error()
	}
	return cli.Session.Current.Room.RoomMeta.ID
}

func (cli *Client) requestJoinServer(serverID string, pass string) error {
	var resp models.JoinServerResponse
	cli.Session.Log.Logf("Requesting to join server with ID: %s", serverID)

	passwordHash := sha256.Sum256([]byte(pass))
	err := cli.Node.SendRPC("JoinServer", models.JoinServerRequest{ServerID: serverID, PasswordHash: passwordHash[:]}, &resp)
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return fmt.Errorf("failed to join server: %s", resp.Error)
	}
	cli.Session.Log.Logf("Successfully joined server: %s", resp.Server.Name)
	cli.Session.Servers[resp.Server.ID] = p2p.NewServerSessionWithMeta(resp.Server)
	cli.Session.Log.Logf("Added server to session servers map: %s", resp.Server.Name)

	cli.Session.Current.Server = cli.Session.Servers[resp.Server.ID]
	cli.Session.Log.Logf("Set current server to: %s", cli.getServerName())

	return nil
}

func (cli *Client) requestJoinRoom(roomID, pass string) error {
	var resp models.JoinRoomResponse
	passwordHash := sha256.Sum256([]byte(pass))
	sid := cli.getServerID()
	if sid == "" {
		return fmt.Errorf("no server joined, cannot join room")
	}
	err := cli.Node.SendRPC("JoinRoom", models.JoinRoomRequest{ServerID: sid, RoomID: roomID, PasswordHash: passwordHash[:], Sender: *cli.User}, &resp)
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}
	if roomSession, ok := cli.Session.Rooms[resp.Room.ID]; ok {
		cli.Session.Current.Room = roomSession
		return nil
	}

	cli.Session.Rooms[resp.Room.ID] = p2p.NewRoomSessionWithMeta(resp.Room)
	cli.Session.Current.Room = cli.Session.Rooms[resp.Room.ID]

	return nil
}

func (cli *Client) requestListRoomMembers() (*models.ListRoomMembersResponse, error) {
	if cli.Session == nil || cli.Session.Current.Room == nil {
		return nil, fmt.Errorf("no room joined, cannot list members")
	}
	var resp models.ListRoomMembersResponse
	err := cli.Node.SendRPC("ListRoomMembers", models.ListRoomMembersRequest{ServerID: cli.getServerID(), RoomID: cli.getRoomID()}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
	}
	return &resp, nil
}

func saveEncryptedSID(sid string, password string) error {
	// Encrypt the SID with the password

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	passKey := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	aead, err := chacha.New(passKey)
	if err != nil {
		return err
	}
	n := make([]byte, aead.NonceSize())
	if _, err := rand.Read(n); err != nil {
		return err
	}
	encryptedSID := aead.Seal(n, n, []byte(sid), nil)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	encPath := homeDir + fmt.Sprintf("/.hillside/%ssid.enc", utils.GenerateRandomID())
	file, err := os.Create(encPath)
	if err != nil {
		return err
	}

	defer file.Close()
	data := struct {
		Salt  []byte `json:"salt"`
		Nonce []byte `json:"nonce"`
		SID   []byte `json:"sid"`
	}{
		Salt:  salt,
		Nonce: n,
		SID:   encryptedSID,
	}
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return err
	}
	return nil
}

func (cli *Client) requestCatchUp(since, limit uint64) (*p2p.RoomRatchet, error) {
	CatchupRespTopic := p2p.CatchUpResponseTopic(cli.getServerID(), cli.getRoomID(), cli.Node.Host.ID().String())
	resptop, err := cli.Node.PS.Join(CatchupRespTopic)
	if err != nil {
		return nil, err
	}
	sub, err := resptop.Subscribe()
	if err != nil {
		return nil, err
	}
	cli.Session.Log.Logf("Subscribed to catch-up response topic: %s", CatchupRespTopic)
	req := &models.CatchUpRequest{}
	dilithiumPriv, ok := cli.Keybag.DilithiumPriv.(*mode2.PrivateKey)
	if !ok {
		return nil, errors.New("invalid DilithiumPriv type")
	}
	data, _, err := models.Marshal(req, *cli.User, dilithiumPriv)
	if err != nil {
		return nil, err
	}
	if !cli.Session.Current.Room.Topics.HasTopic(models.TopicCatchUp) {
		return nil, errors.New("catch up topic is not initialized")
	}
	err = cli.Session.Current.Room.Topics.GetTopic(models.TopicCatchUp).Publish(cli.Node.Ctx, data)
	if err != nil {
		return nil, err
	}
	maxRetries := 5
	for attempt := range maxRetries { // TODO: add loading toast and verify with at least 1/3 of online peers in the room and verify PoW
		cli.Session.Log.Logf("Waiting for catch-up response (attempt %d/%d)...", attempt+1, maxRetries)
		attemptCtx, cancel := context.WithTimeout(cli.Node.Ctx, time.Second)
		resp, err := sub.Next(attemptCtx)
		cancel()
		if err == nil {
			cli.Session.Log.Logf("Received catch-up response from: %s", resp.ReceivedFrom.String())
			env, message, err := models.UnmarshalEnvelope(resp.Data)
			if err != nil {
				return nil, err
			}
			senderID := resp.ReceivedFrom
			err = cli.validateMessageSecurity(env, senderID.String())
			if err != nil {
				return nil, err
			}
			castMsg, ok := message.(*models.CatchUpResponse)
			if !ok {
				return nil, fmt.Errorf("expected CatchUpResponse, got %s", message.Type())
			}
			if castMsg.Error != "" {
				return nil, fmt.Errorf("catch-up error: %s", castMsg.Error)
			}
			r := &p2p.RoomRatchet{
				Index:    castMsg.ChainIndex,
				ChainKey: castMsg.MasterRoomKey, // TODO: derive from PoW key
			}
			if castMsg.Error != "" {
				return r, nil
			}
			catchUpMsgs, err := cli.Session.SessionDB.History.DecompressCatchUpPayload(cli.Node.Ctx, castMsg.CatchUpMessages, cli.getRoomID(), cli.Session.SessionDB.Store)
			cli.Session.Log.Logf("Recieved CatchUpMessages payload of length %d", len(castMsg.CatchUpMessages))
			cli.Session.Log.Logf("Recieved %d catch-up messages", len(catchUpMsgs.ReturnedMessages))
			if err != nil {
				return r, fmt.Errorf("failed to decompress catch-up payload: %v", err)
			}
			cli.Session.Log.Logf("Successfully decompressed %d catch-up messages", len(catchUpMsgs.ReturnedMessages))
			catchUpMsgs.SenderID = senderID.String()
			for _, msg := range catchUpMsgs.ReturnedMessages {
				valid := cli.validateCatchupMessageSecurity(&msg, msg.SenderID)
				if valid != nil {
					return r, fmt.Errorf("catch-up message security validation failed: %v", valid)
				}
				err = cli.Session.SessionDB.Store.SaveEnvelope(cli.Node.Ctx, msg.Signature, msg.Payload, msg.Timestamp, msg.MsgType, msg.ChainIndex, msg.SenderID, msg.RoomID, msg.ServerID)
				if err != nil {
					cli.Session.Log.Logf("Failed to save catch-up message index %d: %v", *msg.ChainIndex, err)
				}
				cli.Session.Log.Logf("Saved catch-up message ID %d of type %s from sender %s", msg.ID, msg.MsgType, msg.SenderID)
			}
			return r, nil
		}
		cli.Session.Log.Logf("No catch-up response received, retrying (%d/%d)...", attempt+1, maxRetries)
		err = cli.Session.Current.Room.Topics.GetTopic(models.TopicCatchUp).Publish(cli.Node.Ctx, data)
		if err != nil {
			cli.Session.Log.Logf("Failed to republish catch-up request: %v", err)
		}
	}
	return nil, fmt.Errorf("no catch-up response received after %d attempts", maxRetries)
}

func (cli *Client) validateCatchupMessageSecurity(msg *models.StoredMessage, senderID string) error {
	sender, err := cli.Session.SessionDB.Store.GetUserByID(cli.Node.Ctx, senderID)
	if err != nil {
		cli.Session.Log.Logf("Failed to get sender user by ID: %v", err)
		return err
	}
	if sender == nil {
		cli.Session.Log.Logf("Sender user not found for ID: %s", senderID)
		return fmt.Errorf("sender user not found for ID: %s", senderID)
	}
	ephemeralEnv := &models.Envelope{
		Type:      msg.MsgType,
		Sender:    *sender,
		Timestamp: msg.Timestamp,
		Signature: msg.Signature,
		Payload:   msg.Payload,
	}
	cli.Session.Log.Logf("Sender PeerID: '%s', Expected SenderID: '%s' | same ? %v", ephemeralEnv.Sender.PeerID, senderID, ephemeralEnv.Sender.PeerID == senderID)
	valid := cli.validateMessageSecurity(ephemeralEnv, senderID)
	if valid != nil {
		cli.Session.Log.Logf("Catch-up message security validation failed: %v", valid)
		return valid
	}

	return nil
}

func (cli *Client) helpCatchUp(sub *pubsub.Subscription) error {

	for {
		cli.Session.Log.Logf("Waiting for catch-up requests on topic: %s", cli.Session.Current.Room.Topics.GetTopic(models.TopicCatchUp).String())
		msg, err := sub.Next(cli.Node.Ctx)
		catchUpPayload, _, dberr, msgs := cli.Session.SessionDB.History.BuildCatchUpPayload(cli.Node.Ctx, cli.getRoomID(), 0, 100, cli.Session.SessionDB.Store)
		roomkey, rkerr := cli.Session.SessionDB.Store.GetAuth(cli.Node.Ctx, cli.getRoomID())
		if rkerr != nil {
			return rkerr
		}
		if dberr != nil {
			cli.Session.Log.Logf("Failed to build catch-up payload: %v", err)
		}
		cli.Session.Log.Logf("Built catch-up payload with %d messages", len(msgs))

		cli.Session.Log.Logf("Built catch-up payload of length %d", len(catchUpPayload))

		if err != nil {
			return err
		}
		cli.Session.Log.Logf("Received catch-up request from: %s", msg.ReceivedFrom.String())
		env, message, err := models.UnmarshalEnvelope(msg.Data)
		if err != nil {
			return err
		}
		senderID := msg.ReceivedFrom
		err = cli.validateMessageSecurity(env, senderID.String())
		if err != nil {
			return err
		}
		_, ok := message.(*models.CatchUpRequest)
		if !ok {
			return fmt.Errorf("expected CatchUpRequest, got %s", message.Type())
		}
		resp := &models.CatchUpResponse{
			ChainIndex:        0,
			MasterRoomKey:     roomkey.MasterRatchetKey,
			MasterRoomKeyBase: roomkey.MasterRatchetKey, // TODO: change to PoW derived key
			CatchUpMessages:   catchUpPayload,
			Error:             "",
		}
		if dberr != nil {
			resp.Error = fmt.Sprintf("failed to build catch-up payload: %s", dberr)
		}
		respTopic := p2p.CatchUpResponseTopic(cli.getServerID(), cli.getRoomID(), senderID.String())
		top, err := cli.Node.PS.Join(respTopic)
		if err != nil {
			return err
		}
		dilithiumPriv, ok := cli.Keybag.DilithiumPriv.(*mode2.PrivateKey)
		if !ok {
			return errors.New("invalid DilithiumPriv type")
		}
		data, _, err := models.Marshal(resp, *cli.User, dilithiumPriv)
		if err != nil {
			return err
		}
		if !cli.Session.Current.Room.Topics.HasTopic(models.TopicCatchUp) {
			return errors.New("catch up topic is not initialized")
		}
		err = top.Publish(cli.Node.Ctx, data)
		if err != nil {
			return err
		}
		cli.Session.Log.Logf("Published catch-up response to topic: %s", respTopic)
		err = top.Close()
		if err != nil {
			return err
		}
	}
}

/*
	func (cli *Client) requestCatchUp(since, limit uint64) (*models.CatchUpResponse, error) {
		if cli.Session == nil || cli.Session.Room == nil {
			return nil, fmt.Errorf("no room joined, cannot request catch-up")
		}
		var resp models.CatchUpResponse
		err := cli.Node.SendRPC("CatchUp", models.CatchUpRequest{}, &resp)
		if err != nil {
			return nil, err
		}
		if resp.Error != "" {
			return nil, fmt.Errorf("%s", resp.Error)
		}
		return &resp, nil
	}

	func (cli *Client) responseCatchUp(ps *pubsub.Message, top *pubsub.Topic) error {
		if cli.Session == nil || cli.Session.Room == nil {
			return fmt.Errorf("no room joined, cannot process catch-up response")
		}
		var resp models.CatchUpResponse
		var ct []byte
		var kyberPub kem.PublicKey
		rat := cli.Session.BackupRatchet
		var reqPub []byte
		senderID := ps.ReceivedFrom
		env, message, err := models.UnmarshalEnvelope(ps.Data)
		if err != nil {
			resp.Error = fmt.Sprintf("security validation failed: %s", err)
			goto send
		}
		if err := cli.validateMessageSecurity(env, senderID); err != nil {
			resp.Error = fmt.Sprintf("security validation failed: %s", err)
			goto send
		}
		if message.Type() != models.MsgTypeCatchUpResp {
			resp.Error = fmt.Sprintf("expected CatchUpResponse, got %s", message.Type())
			goto send
		}
		reqPub = env.Sender.KyberPub
		kyberPub, err = kyber1024.Scheme().UnmarshalBinaryPublicKey(reqPub)
		if err != nil {
			resp.Error = fmt.Sprintf("failed to unmarshal kyber public key: %s", err)
			goto send
		}
		ct, _, err = kyber1024.Scheme().EncapsulateDeterministically(kyberPub, rat.ChainKey)
		if err != nil {
			resp.Error = fmt.Sprintf("failed to encapsulate key: %s", err)
			goto send
		}
		resp.EncState = ct
		resp.ChainIndex = rat.Index

send:

		priv, ok := cli.Keybag.DilithiumPriv.(*mode2.PrivateKey)
		if !ok {
			return fmt.Errorf("invalid type for DilithiumPriv, expected *mode2.PrivateKey")
		}
		data, marshalErr := models.Marshal(resp, *cli.User, priv)
		if marshalErr != nil {
			return marshalErr
		}
		err = top.Publish(cli.Node.Ctx, data)
		if err != nil {
			return fmt.Errorf("failed to publish catch-up response: %s", err)
		}
		top.Close()

		return nil
	}

	func (t *Topics) PublishToRoom(ctx context.Context, topicName string, data []byte) error {
		topic, err := t.Pubsub.Join(topicName)
		if err != nil {
			return err
		}
		return topic.Publish(ctx, data)
	}
*/

func (cli *Client) Shutdown() error {
	if cli.Node.Host != nil {
		_ = cli.Node.Host.Close() // close the libp2p host
	}
	if cli.Node.DHT != nil {
		cli.Node.DHT.Close() // close the DHT
	}

	/* TODO: graceful pubsub shutdown

	// Close individual topics if they exist
	if cli.Node.Topics.ChatTopic != nil {
		_ = cli.Node.Topics.ChatTopic.Close()
	}
	if cli.Node.Topics.RekeyTopic != nil {
		_ = cli.Node.Topics.RekeyTopic.Close()
	}
	// Add any other topics that need to be closed
	*/
	if cli.Node.Ctx != nil {
		cli.Node.Ctx.Done() // cancel the context
	}

	return nil

}
