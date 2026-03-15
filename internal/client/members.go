package client

import (
	"encoding/json"

	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"hillside/internal/models"
)

func (cli *Client) addNewMember(member models.Member) (err error) {
	if member.AddrInfo.ID == cli.Node.Host.ID() {
		return
	}
	if cli.isMemberInList(member.User.PeerID) {
		return nil
	}
	if err = cli.Node.Host.Connect(cli.Node.Ctx, member.AddrInfo); err != nil {
		return
	}
	cli.Session.Current.Room.Members = append(cli.Session.Current.Room.Members, member.User)
	err = cli.Session.SessionDB.Peers.EnqueueUserEntry(cli.Node.Ctx, &member.User)
	if err != nil {
		cli.Session.Log.Logf("Failed to enqueue user %s: %v", member.User.PeerID, err)
	}
	return
}

func (cli *Client) refreshMembersList(sub *pubsub.Subscription) error {

	for {
		msg, err := sub.Next(cli.Node.Ctx)
		if err != nil {
			return nil
		}
		if err := cli.processNewMembers(msg); err != nil {
			cli.Session.Log.Logf("Failed to process new members: %v", err)
			continue
		}
	}
}

func (cli *Client) processNewMembers(msg *pubsub.Message) error {
	var resp models.ListRoomMembersResponse
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return err
	}

	// Verify message is from hub
	if cli.Node.Hub.ID != msg.ReceivedFrom {
		return ErrSecurityIssue.WithDetails("Received message from unexpected peer: " + msg.ReceivedFrom.String())
	}

	cli.Session.Log.Logf("Received %d members", len(resp.Members))

	for _, member := range resp.Members {
		if err := cli.addNewMember(member); err != nil {
			cli.Session.Log.Logf("Failed to add member %s: %v", member.User.PeerID, err)
		}
	}

	return nil
}

func (cli *Client) isMemberInList(peerID string) bool {
	for _, member := range cli.Session.Current.Room.Members {
		if member.PeerID == peerID {
			return true
		}
	}
	return false
}
