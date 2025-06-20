package models

import (
	"github.com/cloudflare/circl/kem"
	kyber "github.com/cloudflare/circl/kem/kyber/kyber1024"
	"github.com/cloudflare/circl/sign"
	dil2 "github.com/cloudflare/circl/sign/dilithium/mode2"
	"github.com/libp2p/go-libp2p/core/crypto"
)

type User struct {
	Username        string `json:"username"`
	DilithiumPub   *dil2.PublicKey `json:"dilithium_pub"`
	//DilithiumPriv *dil2.PrivateKey `json:"dilithium_priv_enc"`
	KyberPub        *kyber.PublicKey `json:"kyber_pub"`
	//KyberPriv        *kyber.PrivateKey `json:"kyber_priv_enc"`
	Libp2pPub       *crypto.PubKey `json:"libp2p_pub"`
	//Libp2pPriv 	*crypto.PrivKey `json:"libp2p_priv_enc"`
	PeerID          string `json:"peer_id"`
}

type Keybag struct {
	Username        string `json:"username"`
	DilithiumPriv sign.PrivateKey `json:"dilithium_priv"`
	KyberPriv        kem.PrivateKey `json:"kyber_priv"`
	Libp2pPriv 	crypto.PrivKey `json:"libp2p_priv"`
	PeerID		  string `json:"peer_id"`
}