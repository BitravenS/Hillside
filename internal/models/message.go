package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudflare/circl/sign/dilithium/mode2"
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
	// … add more as needed
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
	SinceIndex uint64 `json:"since_index"` // last chain index requester already has
}

func (CatchUpRequest) Type() MessageType { return MsgTypeCatchUpReq }

type CatchUpResponse struct {
	EncState   []byte `json:"enc_state"` // encrypted (KEM+AEAD) gzipped framed envelopes
	ChainIndex uint64 `json:"chain_index"`
	Error      string `json:"error,omitempty"` // if any error occurred during catch-up
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

func Marshal(msg Message, sender User, sigPK *mode2.PrivateKey) ([]byte, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	sig := make([]byte, mode2.SignatureSize)
	if sigPK != nil {
		// Sign the payload with the sender's private key
		mode2.SignTo(sigPK, payload, sig)
	}
	env := Envelope{
		Type:      msg.Type(),
		Sender:    sender,
		Timestamp: time.Now().UnixMicro(),
		Signature: sig,
		Payload:   payload,
	}
	return json.Marshal(env)
}

// For storage and retrieval from the database
type StoredMessage struct {
	ID         int64
	RoomID     string
	ServerID   string
	ChainIndex *uint64
	MsgType    MessageType
	SenderID   string
	Timestamp  int64
	Signature  []byte
	Payload    []byte
}

func UnmarshalEnvelope(data []byte) (*Envelope, Message, error) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, nil, err
	}

	var msg Message
	switch env.Type {
	case MsgTypeChat:
		m := new(ChatMessage)
		msg = m
	case MsgTypeJoin:
		m := new(JoinMessage)
		msg = m
	case MsgTypeLeave:
		m := new(LeaveMessage)
		msg = m
	case MsgTypeRekey:
		m := new(RekeyMessage)
		msg = m
	case MsgTypeCatchUpReq:
		m := new(CatchUpRequest)
		msg = m
	case MsgTypeCatchUpResp:
		m := new(CatchUpResponse)
		msg = m
	case MsgTypeUserUpdate:
		m := new(UserUpdate)
		msg = m
	default:
		return &env, nil, fmt.Errorf("unknown message type: %s", env.Type)
	}

	if err := json.Unmarshal(env.Payload, msg); err != nil {
		return &env, nil, err
	}
	return &env, msg, nil
}
