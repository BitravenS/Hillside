package models

import (
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

type User struct {
	DilithiumPub   []byte `json:"dilithium_pub"`
	KyberPub       []byte `json:"kyber_pub"`
	Libp2pPub      []byte `json:"libp2p_pub"`
	PeerID         string `json:"peer_id"`
	Username       string `json:"username"`
	PreferredColor string `json:"preferred_color"`
}

type Keybag struct {
	DilithiumPriv []byte         `json:"dilithium_priv"`
	KyberPriv     []byte         `json:"kyber_priv"`
	Libp2pPriv    crypto.PrivKey `json:"libp2p_priv"`
}

type Member struct {
	AddrInfo peer.AddrInfo `json:"addr_info"`
	User     User          `json:"user"`
}

