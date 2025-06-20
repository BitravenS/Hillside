package hub

import (
	"context"
	"encoding/json"
	"fmt"
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
    h, err := libp2p.New(libp2p.ListenAddrStrings(listenAddr))
    if err != nil {
        return nil, err
    }
    dhtNode, err := dht.New(ctx, h)
    if err != nil {
        return nil, err
    }
    if err := dhtNode.Bootstrap(ctx); err != nil {
        return nil, err
    }

    // In-memory store
    st := NewHubStore()

    srv := &HubServer{
        Ctx:   ctx,
        Host:  h,
        DHT:   dhtNode,
        Store: st,
    }
    h.SetStreamHandler(HubProtocolID, srv.handleRPC)

    return srv, nil
}

// ListenAddrs prints the multiaddrs so clients can dial you.
func (s *HubServer) ListenAddrs() {
    fmt.Println("Hub listening on:")
    for _, a := range s.Host.Addrs() {
        fmt.Printf("  %s/p2p/%s\n", a, s.Host.ID().String())
    }
}

// handleRPC is invoked for every incoming stream on HubProtocolID.
// It expects a JSON envelope {method, params} and returns the JSON-encoded response.
func (s *HubServer) handleRPC(stream network.Stream) {
    defer stream.Close()
    decoder := json.NewDecoder(stream)
    encoder := json.NewEncoder(stream)

    // 1) read envelope
    var env struct {
        Method string          `json:"method"`
        Params json.RawMessage `json:"params"`
    }
    if err := decoder.Decode(&env); err != nil {
        return
    }

    // 2) dispatch on Method
    switch env.Method {

    case "ListServers":
        // no params
        serverPtrs := s.Store.ListServers()
        servers := make([]models.ServerMeta, len(serverPtrs))
		for i, serverPtr := range serverPtrs {
			servers[i] = *serverPtr
    	}
		out := make([]models.ServerMeta,0)
		for _, server := range servers {
			if server.Visibility != models.ServerPrivate {
				out = append(out, server)
			}
		}
		resp := models.ListServersResponse{Servers: out}
		encoder.Encode(resp)

    case "CreateServer":
        var req models.CreateServerRequest
        if err := json.Unmarshal(env.Params, &req); err != nil {
            return
        }
        var sm *models.ServerMeta
        for {
			serverID := utils.GenerateRandomID()
			sm = &models.ServerMeta{
				ID:    serverID,
				Name:        req.Name,
				Visibility:  req.Visibility,
				Description: req.Description,
				CreatedAt:   time.Now().Unix(),
				OwnerPeerID: stream.Conn().RemotePeer().String(),
				Rooms:       make(map[string]*models.RoomMeta),
				PasswordSalt: req.PasswordSalt,
				PasswordHash: req.PasswordHash,
			}
			err := s.Store.CreateServer(sm)
			if err == nil {
				break
				}
        }
        encoder.Encode(models.CreateServerResponse{ServerID: sm.ID})

    case "ListRooms":
        var req models.ListRoomsRequest
        if err := json.Unmarshal(env.Params, &req); err != nil {
            return
        }
        roomPtrs, err := s.Store.ListRooms(req.ServerID)
        if err != nil {
            // TODO: Send error response
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
        encoder.Encode(models.ListRoomsResponse{Rooms: out})

    case "CreateRoom":
        var req models.CreateRoomRequest
        if err := json.Unmarshal(env.Params, &req); err != nil {
            return
        }
		var rm *models.RoomMeta
		for {
			roomID := utils.GenerateRandomID()
			rm = &models.RoomMeta{
				ID:               roomID,
				Name:             req.RoomName,
				Visibility:       req.Visibility,
				PasswordSalt:     req.PasswordSalt,
				PasswordHash:     req.PasswordHash,
				EncRoomKey: req.EncRoomKey,
			}
			err := s.Store.CreateRoom(req.ServerID, rm)
			if err == utils.ServerNotFound {
				encoder.Encode(models.CreateRoomResponse{Error: "Server not found"})
				break
			} else if err == nil {
				break
			}
		}
        encoder.Encode(models.CreateRoomResponse{})

    default:
        // TODO: Send error response for unknown method
		fmt.Printf("Unknown method: %s\n", env.Method)
    }
}


