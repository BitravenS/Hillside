package p2p

import (
	"bufio"
	"context"
	"encoding/json"

	"hillside/internal/hub"

	libp2p "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	lib "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

var protocolID = hub.HubProtocolID

type Topics struct {
	RekeyTopic *pubsub.Topic
	ChatTopic  *pubsub.Topic
}

type Node struct {
	Host   host.Host
	DHT    *dht.IpfsDHT
	PS *pubsub.PubSub
	Ctx   context.Context
	PK lib.PrivKey
	Hub *peer.AddrInfo
	Topics Topics
}

func (n *Node) InitHost(listenAddrs []string) error{
	pk := n.PK
	host, err := libp2p.New(
		libp2p.Identity(pk),
		libp2p.ListenAddrStrings(listenAddrs...),
	)
	if err != nil {
		return err
	}
	n.Host = host
	return nil
}

func (n *Node) InitDHT() error {
	dhtOpts := []dht.Option{
        dht.Mode(dht.ModeServer),
        // include the Hub and public IPFS peers as bootstrap
        dht.BootstrapPeers(*n.Hub),
    }
	dht, err := dht.New(n.Ctx, n.Host, dhtOpts...)
	if err != nil {
		return err
	}
	n.DHT = dht
	if err := n.DHT.Bootstrap(n.Ctx); err != nil {
		return err
	}
	return nil
}

func (n *Node) InitPubSub() error {
	ps, err := pubsub.NewGossipSub(n.Ctx, n.Host)
	if err != nil {
		return err
	}
	n.PS = ps
	return nil
}

func (n *Node) InitNode() error {
	if err := n.InitHost([]string{"/ip4/0.0.0.0/tcp/0"}); err != nil {
		return err
	}

	if err := n.Host.Connect(n.Ctx, *n.Hub); err != nil {
		return err
	}
	if err := n.InitDHT(); err != nil {
		return err
	}
	if err := n.InitPubSub(); err != nil {
		return err
	}
	return nil
}

func (n *Node) SendRPC(method string, params interface{}, out interface{}) error {
	pi := n.Hub
	
	s, err := n.Host.NewStream(n.Ctx, pi.ID, protocol.ID(protocolID))
    defer s.Close()

    // envelope
    env := struct {
        Method string      `json:"method"`
        Params interface{} `json:"params"`
    }{method, params}

    enc := json.NewEncoder(s)
    err = enc.Encode(env)
	if err != nil {
		return err
	}

    rd := bufio.NewReader(s)
    dec := json.NewDecoder(rd)
    err = dec.Decode(out)
	if err != nil {
		return err
	}
	return nil
}