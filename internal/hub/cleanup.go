package hub

import (
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

func (s *HubServer) StartConnectionMonitor(interval time.Duration) {
	log.Printf("[CLEANUP] Starting connection monitor with interval %s", interval)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			s.cleanupDisconnectedMembers()
		}
	}()
}

func (s *HubServer) cleanupDisconnectedMembers() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, server := range s.Store.ListServers(false) {
		for _, room := range server.Rooms {
			for peerID := range room.Members {
				pid, err := peer.Decode(peerID)
				if err != nil || s.Host.Network().Connectedness(pid) != network.Connected {
					delete(room.Members, peerID)
					log.Printf("[CLEANUP] Removed disconnected member %s from room %s", peerID, room.ID)
				}
			}
		}
	}
}
