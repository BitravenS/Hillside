package client

import (
	"context"
	"errors"

	"hillside/internal/crypto"
	"hillside/internal/models"
	"hillside/internal/p2p"
)

func (cli *Client) chatHandler(ctx context.Context) error {
	cli.Session.Log.Logf("Setting up chat handler for room %s on server %s", cli.GetRoomID(), cli.GetServerID())

	err := setupRoomTopics(cli, models.TopicChat, p2p.ChatTopic(cli.GetServerID(), cli.GetRoomID()))
	if err != nil {
		return err
	}
	sub, err := subToRoomTopic(cli, models.TopicChat)
	if err != nil {
		return err
	}
	cli.Node.Subs = append(cli.Node.Subs, sub)
	cli.UI.ChatScreen.ChatSection.Clear()
	err = cli.fetchMessagesFromDB(cli.GetRoomID())
	if err != nil {
		return err
	}
	cli.displayCachedMessages()

	go func() error {
		for {

			msg, err := sub.Next(ctx)
			if err != nil {
				return nil
			}
			env, message, err := UnmarshalEnvelope(msg.Data)
			if err != nil {
				return err
			}
			senderID := msg.ReceivedFrom
			err = cli.validateChatMessage(env, message.(*models.ChatMessage), senderID.String())
			if err != nil {

				if errors.Is(err, ErrValidationIssue) {
					cli.UI.ShowError("Validation Error", err.Error(), "OK", 0, nil)
				} else if errors.Is(err, ErrSecurityIssue) {
					cli.UI.ShowError("Security Error", err.Error(), "OK", 0, nil)
					//TODO: Notify others
				} else {
					cli.UI.ShowError("Unknown Error", "An unknown error occurred: "+err.Error(), "OK", 0, nil)
				}

			}

			castedMsg, ok := message.(*models.ChatMessage)
			if ok {
				pt, err := crypto.DecryptMessage(cli.Session.Current.Room.RoomRatchet, cli.Session.Current.Room.BackupRatchet, castedMsg)
				if err != nil {
					cli.UI.ShowError("Decryption Error", "Failed to decrypt message: "+err.Error(), "OK", 0, nil)
					continue
				}
				decMsg := &models.DecrypetMessage{
					Sender:    env.Sender,
					Timestamp: env.Timestamp,
					Content:   string(pt),
					RoomID:    cli.GetRoomID(),
					ServerID:  cli.GetServerID(),
				}
				cli.Session.Current.Room.Messages = append(cli.Session.Current.Room.Messages, *decMsg)
				if err := cli.Session.SessionDB.History.EnqueueEnvelope(cli.Node.Ctx, env.Signature, env.Payload, env.Timestamp, env.Type, &castedMsg.ChainIndex, env.Sender.PeerID, cli.GetRoomID(), cli.GetServerID()); err != nil {
					cli.UI.ShowError("Storage Error", "Failed to store message: "+err.Error(), "OK", 0, nil)
				}
				cli.DisplayMessage(env.Timestamp, env.Sender, decMsg)
			}
		}
	}()
	return nil
}

func (cli *Client) SendMessageHandler(text string) error {
	if cli.Session.Current.Room == nil {
		cli.UI.ShowError("Error", "You must join a room before sending messages", "OK", 0, nil)
		return ErrSendMessageFailed.WithDetails("no room joined")
	}
	if cli.Session.Current.Room.RoomRatchet == nil {
		cli.UI.ShowError("Error", "You must join a room before sending messages", "OK", 0, nil)
		return ErrSendMessageFailed.WithDetails("room ratchet is nil")
	}

	ct, _, err := crypto.EncryptMessage(cli.Session.Current.Room.RoomRatchet, []byte(text))
	if err != nil {
		return err
	}

	msg := &models.ChatMessage{
		ChainIndex: cli.Session.Current.Room.RoomRatchet.Index - 1,
		Ciphertext: ct,
	}

	data, env, err := MarshalEnvelope(msg, *cli.User, cli.Keybag.DilithiumPriv)
	if err != nil {
		return err
	}
	if !cli.Session.Current.Room.Topics.HasTopic(models.TopicChat) {
		return ErrNotInitialized.WithDetails("chat topic is not initialized")
	}

	err = cli.Session.Current.Room.Topics.GetTopic(models.TopicChat).Publish(cli.Node.Ctx, data)
	if err != nil {
		return err
	}
	err = cli.Session.SessionDB.History.EnqueueEnvelope(cli.Node.Ctx, env.Signature, env.Payload, env.Timestamp, env.Type, &msg.ChainIndex, env.Sender.PeerID, cli.GetRoomID(), cli.GetServerID())
	cli.Session.Log.Logf("Sent message at %d: %s", env.Timestamp, text)
	return err

}
