package client

import (
	"fmt"
	"time"

	"hillside/internal/crypto"
	"hillside/internal/models"
	"hillside/internal/utils"
)

func (cli *Client) validateChatMessageIntegrity(env *models.Envelope, msg *models.ChatMessage) error {
	if msg == nil {
		return utils.ValidationError("Chat message cannot be nil")
	}
	if string(msg.Ciphertext) == "" {
		return utils.ValidationError("Chat message content cannot be empty")
	}

	if len(msg.Ciphertext) > 10000 { // Arbitrary limit for message length
		return utils.ValidationError("Chat message content exceeds maximum length of 10000 characters")
	}
	if env.Timestamp == 0 {
		env.Timestamp = time.Now().Unix()
	}
	return nil
}

func (cli *Client) validateMessageSecurity(env *models.Envelope, strSenderID string) error {
	if env.Sender.PeerID != strSenderID {
		return utils.SecurityError("Sender ID does not match the message sender")
	}
	if env.Timestamp > time.Now().UnixMicro() {
		return utils.SecurityError(fmt.Sprintf("Message timestamp %d is in the future, now is %d", env.Timestamp, time.Now().UnixMicro()))
	}

	verify := crypto.ValidateSignature(env.Sender.DilithiumPub, env.Payload, env.Signature)
	if verify != nil {
		return utils.SecurityError(verify.Error())
	}

	return nil
}

func (cli *Client) validateChatMessage(env *models.Envelope, msg *models.ChatMessage, senderID string) error {
	if err := cli.validateChatMessageIntegrity(env, msg); err != nil {
		return err
	}
	if err := cli.validateMessageSecurity(env, senderID); err != nil {
		return err
	}
	return nil
}
