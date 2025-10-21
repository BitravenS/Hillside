package p2p

import (
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
)

func RendezvousString(serverID, roomID string) string {
	return fmt.Sprintf("/hillside/rendezvous/%s/%s", serverID, roomID)
}

func (n *Node) AdvertiseRoom(serverID, roomID string) error {
	routingDiscovery := routing.NewRoutingDiscovery(n.DHT)
	rendezvous := RendezvousString(serverID, roomID)
	_, err := routingDiscovery.Advertise(n.Ctx, rendezvous)
	return err
}

func (n *Node) DiscoverPeers(serverID, roomID string) ([]peer.AddrInfo, error) {
	routingDiscovery := routing.NewRoutingDiscovery(n.DHT)
	rendezvous := RendezvousString(serverID, roomID)
	peerChan, err := routingDiscovery.FindPeers(n.Ctx, rendezvous)
	if err != nil {
		return nil, err
	}
	fmt.Printf("[DiscoverPeers] Discovering peers for server %s, room %s", serverID, roomID)
	var peers []peer.AddrInfo
	for p := range peerChan {
		// Don't try to connect to self
		if p.ID == n.Host.ID() {
			continue
		}
		if err := n.Host.Connect(n.Ctx, p); err == nil {
			peers = append(peers, p)
		}
	}
	fmt.Printf("[DiscoverPeers] Found %d peers for server %s, room %s", len(peers), serverID, roomID)
	return peers, nil
}
