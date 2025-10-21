package client

import (
	"encoding/json"
	"fmt"
	"time"

	"hillside/internal/crypto"
	"hillside/internal/models"
)

func MarshalEnvelope(msg models.Message, sender models.User, sigPKBlob []byte) ([]byte, *models.Envelope, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, nil, err
	}
	sig, err := crypto.Sign(payload, sigPKBlob)
	if err != nil {
		return nil, nil, err
	}

	env := models.Envelope{
		Type:      msg.Type(),
		Sender:    sender,
		Timestamp: time.Now().UnixMicro(),
		Signature: sig,
		Payload:   payload,
	}
	data, err := json.Marshal(env)
	return data, &env, err
}

func UnmarshalEnvelope(data []byte) (*models.Envelope, models.Message, error) {
	var env models.Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, nil, err
	}

	var msg models.Message
	switch env.Type {
	case models.MsgTypeChat:
		m := new(models.ChatMessage)
		msg = m
	case models.MsgTypeJoin:
		m := new(models.JoinMessage)
		msg = m
	case models.MsgTypeLeave:
		m := new(models.LeaveMessage)
		msg = m
	case models.MsgTypeRekey:
		m := new(models.RekeyMessage)
		msg = m
	case models.MsgTypeCatchUpReq:
		m := new(models.CatchUpRequest)
		msg = m
	case models.MsgTypeCatchUpResp:
		m := new(models.CatchUpResponse)
		msg = m
	case models.MsgTypeUserUpdate:
		m := new(models.UserUpdate)
		msg = m
	default:
		return &env, nil, fmt.Errorf("unknown message type: %s", env.Type)
	}

	if err := json.Unmarshal(env.Payload, msg); err != nil {
		return &env, nil, err
	}
	return &env, msg, nil
}
