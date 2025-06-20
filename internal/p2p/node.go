package p2p

import (
	"context"
	"hillside/internal/models"

	libp2p "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

type Node struct {
	Host   host.Host
	DHT    *dht.IpfsDHT
	PS *pubsub.PubSub
	Ctx   context.Context
	KB *models.Keybag
}

func (n *Node) InitHost(listenAddrs []string) error{
	pk := n.KB.Libp2pPriv
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
	dht, err := dht.New(n.Ctx, n.Host)
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


