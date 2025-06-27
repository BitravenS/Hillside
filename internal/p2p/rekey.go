package p2p

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"hillside/internal/models"

	"github.com/cloudflare/circl/sign/dilithium/mode2"

	"github.com/cloudflare/circl/kem/kyber/kyber1024"
	"github.com/libp2p/go-libp2p/core/peer"
)

// RekeyEntry is one per-peer encapsulation of the new room key.





func (n *Node) RotateRoomKey(serverID, roomID string, memberPeers []models.User, sender models.User, sigPK *mode2.PrivateKey) error {
    // 1) generate a new room key


    msg := models.RekeyMessage{Entries: make([]models.RekeyEntry, 0, len(memberPeers))}
	seed := make([]byte, kyber1024.EncapsulationSeedSize)
		if _, err := rand.Read(seed[:]); err != nil {
			return fmt.Errorf("failed to generate random seed for rekey: %w", err)
	}

    for _, user := range memberPeers {
        // look up their Kyber public key in your Node (you must have stored it somewhere)
		pid := peer.ID(user.PeerID)
		pubkey := user.KyberPub
        kyberPub, err := kyber1024.Scheme().UnmarshalBinaryPublicKey(pubkey)
        if err != nil {
            return fmt.Errorf("failed to unmarshal kyber public key for %s: %w", pid, err)
        }


		ct, _, err := kyber1024.Scheme().EncapsulateDeterministically(kyberPub, seed)
		if err != nil {
			return fmt.Errorf("failed to encapsulate key for %s: %w", pid, err)
		}
        msg.Entries = append(msg.Entries, models.RekeyEntry{
            PeerID: pid.String(),
            Ciph:   ct,
        })
    }

    // 3) marshal and publish
    data, err := models.Marshal(&msg, sender, sigPK)
	if err != nil {
		return fmt.Errorf("failed to marshal rekey message: %w", err)
	}
    rkt, err := n.PS.Join(RekeyTopic(serverID, roomID))
	if err != nil {
		return fmt.Errorf("failed to join rekey topic: %w", err)
	}
    return rkt.Publish(n.Ctx, data)
}

// ListenForRekeys subscribes to the room’s rekey topic and updates n.RoomKey
// whenever there’s a RekeyMessage addressed to this node.
func (n *Node) ListenForRekeys(serverID, roomID string, pk *kyber1024.PrivateKey) error {

	sub, err := n.Topics.RekeyTopic.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to rekey topic: %w", err)
	}

    go func() {
        for {
            msg, err := sub.Next(n.Ctx)
            if err != nil {
                return
            }
            var rk models.RekeyMessage
            if err := json.Unmarshal(msg.Data, &rk); err != nil {
                continue
            }
            // find our entry
            ourID := n.Host.ID().String()
            for _, e := range rk.Entries {
                if e.PeerID == ourID {
                    _, err := kyber1024.Scheme().Decapsulate(pk, e.Ciph)
                    if err != nil {
						continue
                    }
                    // reset your RoomRatchet with this key
                    // TODO: n.RoomRatchet = NewRoomRatchet(newKey)
                    break
                }
            }
        }
    }()
    return nil
}
