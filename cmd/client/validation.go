package main

import (
	"hillside/internal/models"
	"hillside/internal/utils"
	"time"

	"github.com/cloudflare/circl/sign/dilithium/mode2"
	"github.com/libp2p/go-libp2p/core/peer"
)


func (cli *Client) validateChatMessageIntegrity(env *models.Envelope, msg *models.ChatMessage) error {
	if msg == nil {
		return utils.ValidationError("Chat message cannot be nil")
	}
	if string(msg.Ciphertext) == "" {
		return utils.ValidationError("Chat message content cannot be empty")
	}
	if msg.ChainIndex < 0 {
		return utils.ValidationError("Ambiguous chain index")
	}

	if len(msg.Ciphertext) > 10000 { // Arbitrary limit for message length
		return utils.ValidationError("Chat message content exceeds maximum length of 10000 characters")
	}
	if env.Timestamp == 0 {
		env.Timestamp = time.Now().Unix()
	}
	return nil
}


func (cli *Client) validateMessageSecurity(env *models.Envelope, senderID peer.ID) error {
	if env.Sender.PeerID != senderID.String() {
		return utils.SecurityError("Sender ID does not match the message sender")
	}
	if env.Timestamp > time.Now().Unix() {
		return utils.SecurityError("Message timestamp is in the future")
	}

	//verify signature
	if env.Sender.DilithiumPub == nil {
		return utils.SecurityError("Sender's public key is missing")
	}
	dilPub, err := mode2.Scheme().UnmarshalBinaryPublicKey(env.Sender.DilithiumPub)
	if err != nil {
		return utils.SecurityError("Invalid sender's public key format")
	}
	typedPub, ok := dilPub.(*mode2.PublicKey)
	if !ok {
		return utils.SecurityError("Failed to convert public key to correct type")
	}
	if !mode2.Verify(typedPub, env.Payload, env.Signature) {
		return utils.SecurityError("Invalid signature for the chat message")
	}

	return nil
}

func (cli *Client) validateChatMessage(env *models.Envelope, msg *models.ChatMessage, senderID peer.ID) error {
	if err := cli.validateChatMessageIntegrity(env, msg); err != nil {
		return err
	}
	if err := cli.validateMessageSecurity(env, senderID); err != nil {
		return err
	}
	return nil
}