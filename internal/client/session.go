package client

import (
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"hillside/internal/crypto"
	"hillside/internal/models"
	"hillside/internal/storage"
	"hillside/internal/utils"
)

// NewTopicCollection creates a new empty topic collection
func NewTopicCollection() *TopicCollection {
	return &TopicCollection{
		topics: make(map[string]*pubsub.Topic),
	}
}

// NewRoomSession creates a new room session
func NewRoomSession() *RoomSession {
	return &RoomSession{
		RoomMeta:      nil,
		RoomRatchet:   nil,
		BackupRatchet: nil,
		Members:       []models.User{},
		Messages:      []models.DecrypetMessage{},
		Topics:        NewTopicCollection(),
	}
}

func (tc *TopicCollection) GetTopic(name string) *pubsub.Topic {
	return tc.topics[name]
}

// SetTopic adds or replaces a topic
func (tc *TopicCollection) SetTopic(name string, topic *pubsub.Topic) {
	tc.topics[name] = topic
}

// HasTopic checks if a topic exists
func (tc *TopicCollection) HasTopic(name string) bool {
	_, exists := tc.topics[name]
	return exists
}

// RemoveTopic removes a topic if it exists
func (tc *TopicCollection) RemoveTopic(name string) {
	delete(tc.topics, name)
}

// GetTopics returns all topics
func (tc *TopicCollection) GetTopics() map[string]*pubsub.Topic {
	return tc.topics
}

// Constants for standard topic names

// NewServerSession creates a new server session
func NewServerSession() *ServerSession {
	return &ServerSession{
		ServerMeta: nil,
		Topics:     NewTopicCollection(),
	}
}

// NewServerSessionWithMeta creates a new server session with metadata
func NewServerSessionWithMeta(meta *models.ServerMeta) *ServerSession {
	session := NewServerSession()
	session.ServerMeta = meta
	return session
}

// NewRoomSessionWithMeta creates a new room session with metadata
func NewRoomSessionWithMeta(meta *models.RoomMeta) *RoomSession {
	session := NewRoomSession()
	session.RoomMeta = meta
	return session
}

// NewCurrent creates a new empty Current struct
func NewCurrent() Current {
	return Current{Room: nil, Server: nil}
}

// NewSession creates a new Session
func NewSession(db *storage.SessionDB, logger *utils.RemoteLogger) *Session {
	return &Session{
		Servers:   make(map[string]*ServerSession),
		Rooms:     make(map[string]*RoomSession),
		Current:   NewCurrent(),
		SessionDB: db,
		Log:       logger,
	}
}

func (rs *RoomSession) SetInitialRatchet(ratchet *crypto.RoomRatchet) {
	rs.RoomRatchet = ratchet
	rs.BackupRatchet = ratchet.Clone()
}
