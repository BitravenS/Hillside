package client

import (
	"errors"
	"fmt"
	"time"

	"hillside/internal/crypto"
	"hillside/internal/models"
	"hillside/internal/p2p"
	"hillside/internal/storage"
)

func (cli *Client) CreateRoomHandler(req models.CreateRoomRequest) (string, error) {
	if req.RoomName == "" {
		return "", ErrCreateRoomFailed.WithDetails("Room name cannot be empty")
	}
	if req.Visibility == models.Private && len(req.PasswordHash) == 0 {
		return "", ErrCreateRoomFailed.WithDetails("Private rooms must have a password")
	}

	hash, salt, err := crypto.HashWithSalt(req.PasswordHash)
	if err != nil {
		return "", ErrCreateRoomFailed.WithDetails("Failed to hash password: " + err.Error())
	}
	req.PasswordHash = hash
	req.PasswordSalt = salt

	resp, err := cli.requestCreateRoom(req)
	if err != nil {
		return "", ErrCreateRoomFailed.WithDetails("Failed to create room: " + err.Error())
	}
	_, masterKey, err := crypto.GenerateRoomKey()
	if err != nil {
		return "", ErrCreateRoomFailed.WithDetails("Failed to generate room key: " + err.Error())
	}

	cli.Session.SessionDB.Store.SaveAuth(cli.Node.Ctx, resp.RoomID, 0, masterKey, time.Now())
	go cli.refreshRoomList()
	return resp.RoomID, nil
}

func (cli *Client) JoinRoomHandler(roomID string, pass string) (err error) {
	_, cached := cli.Session.Rooms[roomID]

	defer func() {
		if err != nil {
			cli.unsubscribeActiveSubs()
		}
	}()

	if roomID == "" {
		return ErrJoinRoomFailed.WithDetails("Server ID and Room ID cannot be empty")
	}
	if roomID == cli.GetRoomID() {
		return nil
	}
	// TODO: Leave previously joined rooms
	err = cli.requestJoinRoom(roomID, pass)
	if err != nil {
		return ErrJoinRoomFailed.WithDetails(err.Error())
	}

	if !cached {
		MembersTopic := p2p.MembersTopic(cli.GetServerID(), cli.GetRoomID())
		err = setupRoomTopics(cli, models.TopicMembers, MembersTopic)
		if err != nil {
			return ErrJoinRoomFailed.WithDetails("Failed to setup members topic: " + err.Error())
		}
		CatchupReq := p2p.CatchUpRequestTopic(cli.GetServerID(), roomID)
		err = setupRoomTopics(cli, models.TopicCatchUp, CatchupReq)
		if err != nil {
			return ErrJoinRoomFailed.WithDetails("Failed to setup catch-up topic: " + err.Error())
		}
	}

	membersSub, err := subToRoomTopic(cli, models.TopicMembers)
	if err != nil {
		return err
	}
	catchupSub, err := subToRoomTopic(cli, models.TopicCatchUp)
	if err != nil {
		return err
	}
	cli.Node.Subs = append(cli.Node.Subs, membersSub, catchupSub)
	go cli.refreshMembersList(membersSub)

	mbr, err := cli.requestListRoomMembers()
	if err != nil {
		return ErrJoinRoomFailed.WithDetails("Failed to list room members: " + err.Error())
	}

	for _, member := range mbr.Members {
		err = cli.addNewMember(member)
		if err != nil {
			return ErrJoinRoomFailed.WithDetails("Failed to add new member: " + err.Error())
		}
	}

	ratchet, err := cli.initializeRoomRatchet(roomID)
	if err != nil {
		return err
	}

	cli.Session.Current.Room.SetInitialRatchet(ratchet)
	cli.Session.Contexts.ChatCtx.Cancel()
	cli.Session.Contexts.ChatCtx = NewCtxWithCancel(cli.Node.Ctx)

	if err = cli.chatHandler(cli.Session.Contexts.ChatCtx.Ctx); err != nil {
		return ErrJoinRoomFailed.WithDetails("Failed to initialize chat handler: " + err.Error())
	}
	go cli.refreshRoomList()

	cli.UI.ChatScreen.ChatSection.SetTitle(fmt.Sprintf("[ %s ]", cli.GetRoomName()))
	go func() error {
		err = cli.helpCatchUp(catchupSub)
		if err != nil {
			return err
		}
		return nil
	}()
	return nil
}

func (cli *Client) initializeRoomRatchet(roomID string) (*crypto.RoomRatchet, error) {
	roomAuth, err := cli.Session.SessionDB.Store.GetAuth(cli.Node.Ctx, roomID)
	// If we have stored auth, use it
	if err == nil {
		return &crypto.RoomRatchet{
			Index:    roomAuth.ChainIndex,
			ChainKey: roomAuth.MasterRatchetKey,
		}, nil
	}

	// If not found, request catch-up
	if errors.Is(err, storage.ErrNoRows) {
		ratchet, err := cli.requestCatchUp(0, 0) // TODO: since, limit
		if err != nil {
			return nil, ErrJoinRoomFailed.WithDetails("Failed to catch up: " + err.Error())
		}

		if ratchet == nil {
			return nil, ErrJoinRoomFailed.WithDetails("Room ratchet is not initialized after catch-up")
		}

		return ratchet, nil
	}

	// Other error
	return nil, ErrJoinRoomFailed.WithDetails("Failed to get room auth: " + err.Error())
}
