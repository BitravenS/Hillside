package client

import (
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"hillside/internal/crypto"
	"hillside/internal/models"
	"hillside/internal/storage"
	"hillside/internal/utils"
)

type Current struct {
	Room         *RoomSession
	Server       *ServerSession
	GlobalTopics *TopicCollection
}

type Session struct {
	Servers   map[string]*ServerSession // key: server ID
	Rooms     map[string]*RoomSession   // key: room ID
	Current   Current
	Password  string
	SessionDB *storage.SessionDB
	Log       *utils.RemoteLogger
	Contexts  Contexts
}

type TopicCollection struct {
	topics map[models.TopicName]*pubsub.Topic
}

type RoomSession struct {
	RoomMeta      *models.RoomMeta
	RoomRatchet   *crypto.RoomRatchet
	BackupRatchet *crypto.RoomRatchet
	Members       []models.User
	Messages      []models.DecrypetMessage
	Topics        *TopicCollection
}

type ServerSession struct {
	ServerMeta *models.ServerMeta
	Topics     *TopicCollection
}
