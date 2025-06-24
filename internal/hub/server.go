package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"hillside/internal/models"
	"hillside/internal/utils"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
)

const HubProtocolID = "/hillside/hub/1.0.0"

type HubServer struct {
	Ctx   context.Context
	Host  host.Host
	DHT   *dht.IpfsDHT
	Store *HubStore
	mu    sync.Mutex
}

func NewHubServer(ctx context.Context, listenAddr string) (*HubServer, error) {
	log.Printf("[HUB] Initializing hub server on %s", listenAddr)

	h, err := libp2p.New(libp2p.ListenAddrStrings(listenAddr))
	if err != nil {
		log.Printf("[HUB] ERROR: Failed to create libp2p host: %v", err)
		return nil, err
	}

	log.Printf("[HUB] Created libp2p host with ID: %s", h.ID().String())

	dhtNode, err := dht.New(ctx, h)
	if err != nil {
		log.Printf("[HUB] ERROR: Failed to create DHT: %v", err)
		return nil, err
	}

	log.Printf("[HUB] DHT initialized successfully")

	if err := dhtNode.Bootstrap(ctx); err != nil {
		log.Printf("[HUB] ERROR: Failed to bootstrap DHT: %v", err)
		return nil, err
	}

	log.Printf("[HUB] DHT bootstrap completed")

	// In-memory store
	st := NewHubStore()
	log.Printf("[HUB] Hub store initialized")

	srv := &HubServer{
		Ctx:   ctx,
		Host:  h,
		DHT:   dhtNode,
		Store: st,
	}

	// Set connection notification handlers
	h.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, c network.Conn) {
			log.Printf("[HUB] CONNECT: Peer %s connected from %s",
				c.RemotePeer().String(), c.RemoteMultiaddr().String())
		},
		DisconnectedF: func(n network.Network, c network.Conn) {
			log.Printf("[HUB] DISCONNECT: Peer %s disconnected",
				c.RemotePeer().String())
		},
	})

	h.SetStreamHandler(HubProtocolID, srv.handleRPC)
	log.Printf("[HUB] Stream handler set for protocol: %s", HubProtocolID)

	return srv, nil
}

// ListenAddrs prints the multiaddrs so clients can dial you.
func (s *HubServer) ListenAddrs() {
	log.Println("[HUB] Hub listening on:")
	for _, a := range s.Host.Addrs() {
		addr := fmt.Sprintf("%s/p2p/%s", a, s.Host.ID().String())
		log.Printf("[HUB]   %s", addr)
		fmt.Printf("  %s\n", addr) // Also print to stdout for easy copying
	}
}

// handleRPC is invoked for every incoming stream on HubProtocolID.
// It expects a JSON envelope {method, params} and returns the JSON-encoded response.
func (s *HubServer) handleRPC(stream network.Stream) {
	remotePeer := stream.Conn().RemotePeer()
	log.Printf("[HUB] RPC: New stream from peer %s", remotePeer.String())

	defer func() {
		log.Printf("[HUB] RPC: Closing stream from peer %s", remotePeer.String())
		stream.Close()
	}()

	decoder := json.NewDecoder(stream)
	encoder := json.NewEncoder(stream)

	// 1) read envelope
	var env struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := decoder.Decode(&env); err != nil {
		log.Printf("[HUB] RPC ERROR: Failed to decode envelope from %s: %v",
			remotePeer.String(), err)
		return
	}

	log.Printf("[HUB] RPC: Method '%s' called by peer %s", env.Method, remotePeer.String())
	startTime := time.Now()

	// 2) dispatch on Method
	switch env.Method {

	case "ListServers":
		log.Printf("[HUB] RPC: ListServers called by %s", remotePeer.String())
		serverPtrs := s.Store.ListServers()
		servers := make([]models.ServerMeta, len(serverPtrs))
		for i, serverPtr := range serverPtrs {
			servers[i] = *serverPtr
		}
		out := make([]models.ServerMeta, 0)
		for _, server := range servers {
			if server.Visibility != models.ServerPrivate {
				out = append(out, server)
			}
		}
		log.Printf("[HUB] RPC: ListServers returning %d public servers to %s",
			len(out), remotePeer.String())
		resp := models.ListServersResponse{Servers: out}
		if err := encoder.Encode(resp); err != nil {
			log.Printf("[HUB] RPC ERROR: Failed to encode ListServers response: %v", err)
		}

	case "CreateServer":
		var req models.CreateServerRequest
		if err := json.Unmarshal(env.Params, &req); err != nil {
			log.Printf("[HUB] RPC ERROR: Failed to unmarshal CreateServer params: %v", err)
			return
		}

		log.Printf("[HUB] RPC: CreateServer called by %s - Name: '%s', Visibility: %v",
			remotePeer.String(), req.Name, req.Visibility)

		var sm *models.ServerMeta
		for {
			serverID := utils.GenerateRandomID()
			sm = &models.ServerMeta{
				ID:           serverID,
				Name:         req.Name,
				Visibility:   req.Visibility,
				Description:  req.Description,
				CreatedAt:    time.Now().Unix(),
				OwnerPeerID:  remotePeer.String(),
				Rooms:        make(map[string]*models.RoomMeta),
				PasswordSalt: req.PasswordSalt,
				PasswordHash: req.PasswordHash,
			}
			err := s.Store.CreateServer(sm)
			if err == nil {
				break
			}
			log.Printf("[HUB] RPC: Server ID collision, retrying with new ID")
		}

		log.Printf("[HUB] RPC: Server created successfully - ID: %s, Name: '%s', Owner: %s",
			sm.ID, sm.Name, remotePeer.String())

		if err := encoder.Encode(models.CreateServerResponse{ServerID: sm.ID}); err != nil {
			log.Printf("[HUB] RPC ERROR: Failed to encode CreateServer response: %v", err)
		}

	case "ListRooms":
		var req models.ListRoomsRequest
		if err := json.Unmarshal(env.Params, &req); err != nil {
			log.Printf("[HUB] RPC ERROR: Failed to unmarshal ListRooms params: %v", err)
			return
		}

		log.Printf("[HUB] RPC: ListRooms called by %s for server %s",
			remotePeer.String(), req.ServerID)

		roomPtrs, err := s.Store.ListRooms(req.ServerID)
		if err != nil {
			log.Printf("[HUB] RPC ERROR: ListRooms failed for server %s: %v",
				req.ServerID, err)
			return
		}

		rooms := make([]models.RoomMeta, len(roomPtrs))
		for i, roomPtr := range roomPtrs {
			rooms[i] = *roomPtr
		}
		out := make([]models.RoomMeta, 0)
		for _, room := range rooms {
			if room.Visibility != models.RoomPrivate {
				out = append(out, room)
			}
		}

		log.Printf("[HUB] RPC: ListRooms returning %d public rooms for server %s to %s",
			len(out), req.ServerID, remotePeer.String())

		if err := encoder.Encode(models.ListRoomsResponse{Rooms: out}); err != nil {
			log.Printf("[HUB] RPC ERROR: Failed to encode ListRooms response: %v", err)
		}

	case "CreateRoom":
		var req models.CreateRoomRequest
		if err := json.Unmarshal(env.Params, &req); err != nil {
			log.Printf("[HUB] RPC ERROR: Failed to unmarshal CreateRoom params: %v", err)
			return
		}

		log.Printf("[HUB] RPC: CreateRoom called by %s - Server: %s, Room: '%s', Visibility: %v",
			remotePeer.String(), req.ServerID, req.RoomName, req.Visibility)

		var rm *models.RoomMeta
		for {
			roomID := utils.GenerateRandomID()
			rm = &models.RoomMeta{
				ID:           roomID,
				Name:         req.RoomName,
				Visibility:   req.Visibility,
				PasswordSalt: req.PasswordSalt,
				PasswordHash: req.PasswordHash,
				EncRoomKey:   req.EncRoomKey,
			}
			err := s.Store.CreateRoom(req.ServerID, rm)
			if err == utils.ServerNotFound {
				log.Printf("[HUB] RPC ERROR: CreateRoom failed - Server %s not found", req.ServerID)
				encoder.Encode(models.CreateRoomResponse{Error: "Server not found"})
				break
			} else if err == nil {
				log.Printf("[HUB] RPC: Room created successfully - ID: %s, Name: '%s', Server: %s",
					rm.ID, rm.Name, req.ServerID)
				break
			}
			log.Printf("[HUB] RPC: Room ID collision, retrying with new ID")
		}

		if err := encoder.Encode(models.CreateRoomResponse{}); err != nil {
			log.Printf("[HUB] RPC ERROR: Failed to encode CreateRoom response: %v", err)
		}

	default:
		log.Printf("[HUB] RPC ERROR: Unknown method '%s' called by %s",
			env.Method, remotePeer.String())
	}

	duration := time.Since(startTime)
	log.Printf("[HUB] RPC: Method '%s' completed in %v for peer %s",
		env.Method, duration, remotePeer.String())
}


