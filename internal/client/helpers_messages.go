package client

import (
	"encoding/json"

	"hillside/internal/crypto"
	"hillside/internal/models"
)

func (cli *Client) fetchMessagesFromDB(roomID string) error {
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
		pt, err := crypto.DecryptMessage(cli.Session.Current.Room.RoomRatchet, cli.Session.Current.Room.BackupRatchet, cm)
		if err != nil {
			cli.Session.Log.Logf("Failed to decrypt message: %v", err)
			return err
		}

		var sender *models.User
		sender, err = cli.Session.SessionDB.Store.GetUserByID(cli.Node.Ctx, msg.SenderID)
		if err != nil {
			cli.Session.Log.Logf("Failed to Get sender %s: %v", msg.SenderID, err)
			sender = &models.User{
				PeerID:   msg.SenderID,
				Username: "Unknown",
			}
		}
		cli.Session.Log.Logf("Is sender nil? %v", sender == nil)
		decMsg := &models.DecrypetMessage{
			Sender:    *sender,
			Timestamp: msg.Timestamp,
			Content:   string(pt),
			RoomID:    cli.GetRoomID(),
			ServerID:  cli.GetServerID(),
		}
		exists := false
		for _, m := range cli.Session.Current.Room.Messages {
			if m.Timestamp == decMsg.Timestamp && m.Sender.PeerID == decMsg.Sender.PeerID {
				exists = true
				break
			}
		}
		if !exists {
			cli.Session.Current.Room.Messages = append(cli.Session.Current.Room.Messages, *decMsg)
		}
	}
	return nil
}

func (cli *Client) displayCachedMessages() {
	for _, msg := range cli.Session.Current.Room.Messages {
		cli.Session.Log.Logf("Displaying cached message from %s at %d", msg.Sender.Username, msg.Timestamp)
		cli.DisplayMessage(msg.Timestamp, msg.Sender, &msg) // FIX: Due to its async nature, make sure to queue all draw calls then execute them at once
	}
}
