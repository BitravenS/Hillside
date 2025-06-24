package hub

import (
	"hillside/internal/models"
	"hillside/internal/utils"
	"log"
	"sync"
)

type HubStore struct {
	mu      sync.RWMutex
	servers map[string]*models.ServerMeta
}

func NewHubStore() *HubStore {
	log.Printf("[STORE] Initializing new hub store")
	return &HubStore{
		servers: make(map[string]*models.ServerMeta),
	}
}

func (hs *HubStore) ListServers() []*models.ServerMeta {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	log.Printf("[STORE] ListServers called - %d servers in store", len(hs.servers))

	servers := make([]*models.ServerMeta, 0, len(hs.servers))
	for _, server := range hs.servers {
		servers = append(servers, server)
	}

	log.Printf("[STORE] ListServers returning %d servers", len(servers))
	return servers
}

func (hs *HubStore) CreateServer(server *models.ServerMeta) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	log.Printf("[STORE] CreateServer called - ID: %s, Name: '%s', Owner: %s",
		server.ID, server.Name, server.OwnerPeerID)

	if _, exists := hs.servers[server.ID]; exists {
		log.Printf("[STORE] CreateServer failed - Server ID %s already exists", server.ID)
		return utils.DuplicateID
	}

	hs.servers[server.ID] = server
	log.Printf("[STORE] Server created successfully - Total servers: %d", len(hs.servers))
	return nil
}

func (hs *HubStore) ListRooms(serverID string) ([]*models.RoomMeta, error) {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	log.Printf("[STORE] ListRooms called for server: %s", serverID)

	server, exists := hs.servers[serverID]
	if !exists {
		log.Printf("[STORE] ListRooms failed - Server %s not found", serverID)
		return nil, utils.ServerNotFound
	}

	rooms := make([]*models.RoomMeta, 0, len(server.Rooms))
	for _, room := range server.Rooms {
		rooms = append(rooms, room)
	}

	log.Printf("[STORE] ListRooms returning %d rooms for server %s", len(rooms), serverID)
	return rooms, nil
}

func (hs *HubStore) CreateRoom(serverID string, room *models.RoomMeta) error {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	log.Printf("[STORE] CreateRoom called - Server: %s, Room ID: %s, Name: '%s'",
		serverID, room.ID, room.Name)

	server, exists := hs.servers[serverID]
	if !exists {
		log.Printf("[STORE] CreateRoom failed - Server %s not found", serverID)
		return utils.ServerNotFound
	}

	if _, exists := server.Rooms[room.ID]; exists {
		log.Printf("[STORE] CreateRoom failed - Room ID %s already exists in server %s",
			room.ID, serverID)
		return utils.DuplicateID
	}

	server.Rooms[room.ID] = room
	log.Printf("[STORE] Room created successfully - Server %s now has %d rooms",
		serverID, len(server.Rooms))
	return nil
}