package hub

import (
	"context"
	"fmt"
	"hillside/internal/hub"
	"hillside/internal/models"
	"testing"

	"bufio"
	"encoding/json"

	libp2p "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)


const protocolID = hub.HubProtocolID

func startTestHub(t *testing.T) (*hub.HubServer, []string) {
    ctx := context.Background()
    addr :=  "/ip4/127.0.0.1/tcp/12345"

    srv, err := hub.NewHubServer(ctx, addr)
    require.NoError(t, err)

    addrs := []string{}
    for _, a := range srv.Host.Addrs() {
        addrs = append(addrs, a.String()+"/p2p/"+srv.Host.ID().String())
    }
    return srv, addrs
}

// helper: create a libp2p host for testing
func newTestClient(t *testing.T) (host.Host, context.Context) {
    ctx := context.Background()
    h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
    require.NoError(t, err)
    d, err := dht.New(ctx, h)
    require.NoError(t, err)
    require.NoError(t, d.Bootstrap(ctx))
    return h, ctx
}

// helper: send one RPC and decode the reply
func sendRPC[T any, U any](t *testing.T, ctx context.Context, h host.Host, hubAddr string,
    method string, params T, out *U,
) {
    pi, err := peer.AddrInfoFromString(hubAddr)
    require.NoError(t, err)
    require.NoError(t, h.Connect(ctx, *pi))
    s, err := h.NewStream(ctx, pi.ID, protocolID)
    require.NoError(t, err)
    defer s.Close()

    // envelope
    env := struct {
        Method string      `json:"method"`
        Params interface{} `json:"params"`
    }{method, params}

    enc := json.NewEncoder(s)
    require.NoError(t, enc.Encode(env))

    rd := bufio.NewReader(s)
    dec := json.NewDecoder(rd)
    require.NoError(t, dec.Decode(out))
}

func TestHubServer_CRUD(t *testing.T) {
    srv, addrs := startTestHub(t)
    defer srv.Host.Close()

    // pick the first multiaddr
    hubAddr := addrs[0]

    // create a test client
    client, ctx := newTestClient(t)
    defer client.Close()

    // 1) ListServers → should be empty
    var listResp models.ListServersResponse
    sendRPC(t, ctx, client, hubAddr, "ListServers", models.ListServersRequest{}, &listResp)
    require.Len(t, listResp.Servers, 0)

    // 2) CreateServer
    createReq := models.CreateServerRequest{
        Name:       "TestServer",
		Description: "A cool server",
        Visibility: models.Public,
    }
    var createResp models.CreateServerResponse
    sendRPC(t, ctx, client, hubAddr, "CreateServer", createReq, &createResp)
	fmt.Printf("Created server with ID: %s\n", createResp.ServerID)
    require.NotEmpty(t, createResp.ServerID)

    // 3) ListServers → should contain one entry
    sendRPC(t, ctx, client, hubAddr, "ListServers", models.ListServersRequest{}, &listResp)
    require.Len(t, listResp.Servers, 1)
    srvMeta := listResp.Servers[0]
    require.Equal(t, "TestServer", srvMeta.Name)
    require.Equal(t, createResp.ServerID, srvMeta.ID)
    fmt.Printf("Listed server: %+v\n", srvMeta)

    // 4) ListRooms on new server → empty
    var roomsResp models.ListRoomsResponse
    sendRPC(t, ctx, client, hubAddr, "ListRooms", models.ListRoomsRequest{ServerID: createResp.ServerID}, &roomsResp)
    require.Len(t, roomsResp.Rooms, 0)

    // 5) CreateRoom
    roomReq := models.CreateRoomRequest{
        ServerID:   createResp.ServerID,
        RoomName:   "lobby",
        Visibility: models.Public,
    }
    var roomResp models.CreateRoomResponse
    sendRPC(t, ctx, client, hubAddr, "CreateRoom", roomReq, &roomResp)

    // 6) ListRooms → should contain “lobby”
    sendRPC(t, ctx, client, hubAddr, "ListRooms", models.ListRoomsRequest{ServerID: createResp.ServerID}, &roomsResp)
    require.Len(t, roomsResp.Rooms, 1)
    require.Equal(t, "lobby", roomsResp.Rooms[0].Name)
	fmt.Printf("Created room:%+v\n", roomsResp.Rooms[0])
}
