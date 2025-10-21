// Package models defines the global data models used across the application (for both the client and server).
package models

import (
	"encoding/json"
)

// MessageType indicates which kind of payload lives inside the Envelope.
type MessageType string

const (
	MsgTypeChat        MessageType = "chat"
	MsgTypeJoin        MessageType = "join"
	MsgTypeLeave       MessageType = "leave"
	MsgTypeRekey       MessageType = "rekey"
	MsgTypeCatchUpReq  MessageType = "catchup_req"
	MsgTypeCatchUpResp MessageType = "catchup_resp"
	MsgTypeUserUpdate  MessageType = "user_update"
)

type DecrypetMessage struct {
	Sender    User   `json:"sender"`
	Timestamp int64  `json:"timestamp"` // unix nano
	Content   string `json:"content"`
	RoomID    string `json:"room_id"`
	ServerID  string `json:"server_id"`
}

type Message interface {
	Type() MessageType
}

type ChatMessage struct {
	ChainIndex uint64 `json:"chain_index"`
	Ciphertext []byte `json:"ciphertext"`
}

func (ChatMessage) Type() MessageType { return MsgTypeChat }

// JoinMessage signals a new member (and can carry their public keys)
type JoinMessage struct {
	User User `json:"user"`
	// maybe initial EncRoomKey blob
	EncKey []byte `json:"enc_key,omitempty"`
}

func (JoinMessage) Type() MessageType { return MsgTypeJoin }

// LeaveMessage signals a departure
type LeaveMessage struct {
	PeerID string `json:"peer_id"`
}

func (LeaveMessage) Type() MessageType { return MsgTypeLeave }

// RekeyMessage distributes new room‐key KEM ciphertexts
type RekeyMessage struct {
	Entries []RekeyEntry `json:"entries"`
}

type RekeyEntry struct {
	PeerID string `json:"peer_id"`
	Ciph   []byte `json:"ciphertext"`
}

func (RekeyMessage) Type() MessageType { return MsgTypeRekey }

type CatchUpRequest struct {
	SinceIndex uint64 `json:"since_index,omitempty"` // last chain index requester already has
}

func (CatchUpRequest) Type() MessageType { return MsgTypeCatchUpReq }

type CatchUpResponse struct {
	MasterRoomKey     []byte `json:"master_room_key"`
	MasterRoomKeyBase []byte `json:"master_room_key_base"` // base key, hashed becomes MasterRoomKey (for Proof of Work)
	ChainIndex        uint64 `json:"chain_index"`
	CatchUpMessages   []byte `json:"catchup_messages"` // serialized CatchUpMessages
	Error             string `json:"error,omitempty"`  // if any error occurred during catch-up
}

func (CatchUpResponse) Type() MessageType { return MsgTypeCatchUpResp }

type UserUpdate struct {
	User User `json:"user"`
}

func (UserUpdate) Type() MessageType { return MsgTypeUserUpdate }

// Envelope wraps any Message with metadata
type Envelope struct {
	Type      MessageType     `json:"type"`
	Sender    User            `json:"sender"`
	Timestamp int64           `json:"timestamp"` // unix micro
	Signature []byte          `json:"signature"` // signature of the payload
	Payload   json.RawMessage `json:"payload"`
}

type StoredMessage struct {
	ID         int64       `json:"id,omitempty"`
	RoomID     string      `json:"room_id"`
	ServerID   string      `json:"server_id,omitempty"`
	ChainIndex *uint64     `json:"chain_index"`
	MsgType    MessageType `json:"msg_type"`
	SenderID   string      `json:"sender_id"`
	Timestamp  int64       `json:"timestamp"`
	Signature  []byte      `json:"signature"`
	Payload    []byte      `json:"payload"`
}
