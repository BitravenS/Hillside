package models

import (
	"github.com/cloudflare/circl/kem"
	kyber "github.com/cloudflare/circl/kem/kyber/kyber1024"
	"github.com/cloudflare/circl/sign"
	dil2 "github.com/cloudflare/circl/sign/dilithium/mode2"
	lib "github.com/libp2p/go-libp2p/core/crypto"
)

type User struct {
	DilithiumPub   *dil2.PublicKey `json:"dilithium_pub"`
	KyberPub        *kyber.PublicKey `json:"kyber_pub"`
	Libp2pPub       lib.PubKey `json:"libp2p_pub"`
	PeerID          string `json:"peer_id"`
	Username        string `json:"username"`
}

type Keybag struct {
	DilithiumPriv sign.PrivateKey `json:"dilithium_priv"`
	KyberPriv        kem.PrivateKey `json:"kyber_priv"`
	Libp2pPriv 	lib.PrivKey `json:"libp2p_priv"`
}

