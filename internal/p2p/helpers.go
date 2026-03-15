package p2p

import "github.com/libp2p/go-libp2p/core/peer"

func GetHubAddress(strAddr string) (*peer.AddrInfo, error) {
	return peer.AddrInfoFromString(strAddr)
}
