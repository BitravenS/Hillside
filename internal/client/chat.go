package client

import (
	"fmt"

	"hillside/internal/crypto"
	"hillside/internal/models"
	"hillside/internal/p2p"
	"hillside/internal/utils"
)

func (cli *Client) chatHandler() error {
	/*
		kyberPriv, ok := cli.Keybag.KyberPriv.(*kyber1024.PrivateKey)
		if !ok {
			return errors.New("invalid KyberPriv type")
		}
			if err := cli.Node.ListenForRekeys(cli.GetServerID(), cli.GetRoomID(), kyberPriv); err != nil {
				return err
			}
	*/
	if !cli.Session.Current.Room.Topics.HasTopic(models.TopicChat) {
		chatTopic := p2p.ChatTopic(cli.GetServerID(), cli.GetRoomID())
		topic, err := cli.Node.PS.Join(chatTopic)
		if err != nil {
			return err
		}
		cli.Session.Current.Room.Topics.SetTopic(models.TopicChat, topic)
	}
	sub, err := cli.Session.Current.Room.Topics.GetTopic(models.TopicChat).Subscribe()
	if err != nil {
		return err
	}
	err = cli.parseAndDisplayDBMessages(cli.GetRoomID())
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
			env, message, err := UnmarshalEnvelope(msg.Data)
			if err != nil {
				return err
			}
			senderID := msg.ReceivedFrom
			err = cli.validateChatMessage(env, message.(*models.ChatMessage), senderID.String())
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
					RoomID:    cli.GetRoomID(),
					ServerID:  cli.GetServerID(),
				}
				cli.Session.Current.Room.Messages = append(cli.Session.Current.Room.Messages, *decMsg)
				if err := cli.Session.SessionDB.History.EnqueueEnvelope(cli.Node.Ctx, env.Signature, env.Payload, env.Timestamp, env.Type, &castedMsg.ChainIndex, env.Sender.PeerID, cli.GetRoomID(), cli.GetServerID()); err != nil {
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
					cli.UI.ChatScreen.ChatSection.AddItem(lineContent, "", 0, nil)

				})

			}

		}
	}()
	return nil
}

func (cli *Client) DisplayMessage(timestamp int64, sender models.User, decMsg *models.DecrypetMessage) {
	formattedTime := utils.FormatPrettyTime(timestamp)

	prefColor := sender.PreferredColor
	if !utils.Contains(utils.BaseXtermAnsiColorNames, prefColor) {
		prefColor = utils.GenerateRandomColor()
	}
	lineContent := fmt.Sprintf("[yellow][%s] [%s]%s:[white] %s", formattedTime, prefColor, sender.Username, decMsg.Content)

	go func() {
		cli.UI.App.QueueUpdateDraw(func() {
			cli.UI.ChatScreen.ChatSection.AddItem(lineContent, "", 0, nil)
		})

	}()
}

func (cli *Client) SendMessageHandler(text string) error {
	if cli.Session.Current.Room.RoomRatchet == nil {
		cli.UI.ShowError("Error", "You must join a room before sending messages", "OK", 0, nil)
		return utils.SendMessageError("Room ratchet is not initialized. Join a room first.")
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
	return err

}
