package models

type MessageType int

type Message struct {
	Type    MessageType
	Message interface{}
}