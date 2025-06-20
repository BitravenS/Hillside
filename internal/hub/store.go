package hub

import (
	"hillside/internal/models"
	"hillside/internal/utils"
	"sync"
)

type HubStore struct {
	mu sync.RWMutex
	servers map[string]*models.ServerMeta
}

func NewHubStore() *HubStore {
	return &HubStore{
		servers: make(map[string]*models.ServerMeta),
	}
}

func (hs *HubStore) ListServers() []*models.ServerMeta {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	servers := make([]*models.ServerMeta, 0, len(hs.servers))
	for _, server := range hs.servers {
		servers = append(servers, server)
	}
	return servers
}

func (hs *HubStore) CreateServer(server *models.ServerMeta) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	if _, exists := hs.servers[server.ID]; exists {
		return utils.DuplicateID
	}
	hs.servers[server.ID] = server
	return nil
}

func (hs *HubStore) ListRooms(serverID string) ([]*models.RoomMeta, error) {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	server, exists := hs.servers[serverID]
	if !exists {
		return nil, utils.ServerNotFound
	}

	rooms := make([]*models.RoomMeta, 0, len(server.Rooms))
	for _, room := range server.Rooms {
		rooms = append(rooms, room)
	}
	return rooms, nil
}

func (hs *HubStore) CreateRoom(serverID string, room *models.RoomMeta) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	server, exists := hs.servers[serverID]
	if !exists {
		return utils.ServerNotFound
	}

	if _, exists := server.Rooms[room.ID]; exists {
		return utils.DuplicateID
	}

	server.Rooms[room.ID] = room
	return nil
}