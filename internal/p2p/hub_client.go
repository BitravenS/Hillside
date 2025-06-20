package p2p

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"hillside/internal/models"

	"github.com/libp2p/go-libp2p/core/peer"
)

const HubProtocolID = "/hillside/hub/1.0.0"


// sendHubRequest opens a libp2p stream to the hub, sends {method, params}, and decodes into out.
func (n *Node) sendHubRequest(method string, params interface{}, out interface{}) error {
    pi, err := peer.AddrInfoFromString(n.HubAddr)
    if err != nil {
        return err
    }
    if err := n.Host.Connect(n.Ctx, *pi); err != nil {
        return err
    }
    s, err := n.Host.NewStream(n.Ctx, pi.ID, HubProtocolID)
    if err != nil {
        return err
    }
    defer s.Close()

    env := struct {
        Method string      `json:"method"`
        Params interface{} `json:"params"`
    }{method, params}

    enc := json.NewEncoder(s)
    if err := enc.Encode(env); err != nil {
        return err
    }
    dec := json.NewDecoder(s)
    return dec.Decode(out)
}

func (n *Node) FetchServersFromHub(hubAddr string) ([]models.ServerMeta, error) {
    var resp models.ListServersResponse
    if err := n.sendHubRequest("ListServers", models.ListServersRequest{}, &resp); err != nil {
        return nil, err
    }
    return resp.Servers, nil
}
func (n *Node) CreateServer(name string, desc string, vis models.ServerVisibility, password []byte) (string, error) {

	req := models.CreateServerRequest{
        Name:       name,
		Description: desc,
        Visibility: vis,
    }
	if vis != models.ServerPublic {
		salt := make([]byte, 16)
		if _, err := rand.Read(salt); err != nil {
			return "", err
		}
        req.PasswordSalt = salt
        hash := sha256.Sum256(password)
        req.PasswordHash = hash[:]
        }
    var resp models.CreateServerResponse
    if err := n.sendHubRequest("CreateServer", req, &resp); err != nil {
        return "", err
    }
    return resp.ServerID, nil
}


func (n *Node) FetchRoomsFromHub(serverID string) ([]models.RoomMeta, error) {
    req := models.ListRoomsRequest{ServerID: serverID}
    var resp models.ListRoomsResponse
    if err := n.sendHubRequest("ListRooms", req, &resp); err != nil {
        return nil, err
    }
    return resp.Rooms, nil
}

func (n *Node) CreateRoom(serverID, roomName string, vis models.RoomVisibility, encryptedRoomKey []byte, password []byte) error {
    req := models.CreateRoomRequest{
        ServerID:         serverID,
        RoomName:         roomName,
        Visibility:       vis,
		EncRoomKey: encryptedRoomKey,
    }
	if vis != models.RoomPublic {
		salt := make([]byte, 16)
		if _, err := rand.Read(salt); err != nil {
			return err
		}
		req.PasswordSalt = salt
		hash := sha256.Sum256(password)
		req.PasswordHash = hash[:]
	}

    var resp models.CreateRoomResponse
    return n.sendHubRequest("CreateRoom", req, &resp)
}