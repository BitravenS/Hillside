package client

import (
	"fmt"

	"hillside/internal/crypto"
	"hillside/internal/models"
)

func (cli *Client) requestServers() (*models.ListServersResponse, error) {
	var listResp models.ListServersResponse
	err := cli.Node.SendRPC("ListServers", models.ListServersRequest{}, &listResp)
	if err != nil {
		return nil, err
	}
	return &listResp, nil

}

func (cli *Client) requestCreateServer(req models.CreateServerRequest) (*models.CreateServerResponse, error) {
	var resp models.CreateServerResponse
	err := cli.Node.SendRPC("CreateServer", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (cli *Client) requestCreateRoom(req models.CreateRoomRequest) (*models.CreateRoomResponse, error) {
	var resp models.CreateRoomResponse
	err := cli.Node.SendRPC("CreateRoom", req, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
	}
	return &resp, nil
}

func (cli *Client) requestRooms(serverID string) (*models.ListRoomsResponse, error) {
	var roomsResp models.ListRoomsResponse
	err := cli.Node.SendRPC("ListRooms", models.ListRoomsRequest{ServerID: serverID}, &roomsResp)
	if err != nil {
		return nil, err
	}
	return &roomsResp, nil
}

func (cli *Client) requestJoinServer(serverID string, pass string) error {
	var resp models.JoinServerResponse
	cli.Session.Log.Logf("Requesting to join server with ID: %s", serverID)

	passwordHash := crypto.HashPassword(pass)
	err := cli.Node.SendRPC("JoinServer", models.JoinServerRequest{ServerID: serverID, PasswordHash: passwordHash}, &resp)
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return fmt.Errorf("failed to join server: %s", resp.Error)
	}
	cli.Session.Log.Logf("Successfully joined server: %s", resp.Server.Name)
	cli.Session.Servers[resp.Server.ID] = NewServerSessionWithMeta(resp.Server)
	cli.Session.Log.Logf("Added server to session servers map: %s", resp.Server.Name)

	cli.Session.Current.Server = cli.Session.Servers[resp.Server.ID]
	cli.Session.Log.Logf("Set current server to: %s", cli.GetServerName())

	return nil
}

func (cli *Client) requestJoinRoom(roomID, pass string) error {
	var resp models.JoinRoomResponse
	passwordHash := crypto.HashPassword(pass)
	sid := cli.GetServerID()
	if sid == "" {
		return fmt.Errorf("no server joined, cannot join room")
	}
	err := cli.Node.SendRPC("JoinRoom", models.JoinRoomRequest{ServerID: sid, RoomID: roomID, PasswordHash: passwordHash, Sender: *cli.User}, &resp)
	if err != nil {
		return err
	}
	if resp.Error != "" {
		return fmt.Errorf("%s", resp.Error)
	}
	if roomSession, ok := cli.Session.Rooms[resp.Room.ID]; ok {
		cli.Session.Current.Room = roomSession
		return nil
	}

	cli.Session.Rooms[resp.Room.ID] = NewRoomSessionWithMeta(resp.Room)
	cli.Session.Current.Room = cli.Session.Rooms[resp.Room.ID]

	return nil
}

func (cli *Client) requestListRoomMembers() (*models.ListRoomMembersResponse, error) {
	if cli.Session == nil || cli.Session.Current.Room == nil {
		return nil, fmt.Errorf("no room joined, cannot list members")
	}
	var resp models.ListRoomMembersResponse
	err := cli.Node.SendRPC("ListRoomMembers", models.ListRoomMembersRequest{ServerID: cli.GetServerID(), RoomID: cli.GetRoomID()}, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("%s", resp.Error)
	}
	return &resp, nil
}
