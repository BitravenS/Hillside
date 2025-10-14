package p2p

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"io"

	"golang.org/x/crypto/hkdf"
)

type RoomRatchet struct {
	ChainKey []byte // secret state
	Index    uint64 // message count
}

func (r *RoomRatchet) NextKey() (msgKey, nonce []byte, err error) {

	info := make([]byte, 8)
	binary.BigEndian.PutUint64(info, r.Index)
	hk := hkdf.New(sha256.New, r.ChainKey, nil, info)

	msgKey = make([]byte, 32)
	if _, err = io.ReadFull(hk, msgKey); err != nil {
		return
	}
	nonce = make([]byte, 12)
	if _, err = io.ReadFull(hk, nonce); err != nil {
		return
	}

	// Advance chain key = HMAC(ChainKey, constant)
	mac := hmac.New(sha256.New, r.ChainKey)
	mac.Write([]byte("ratchet"))
	r.ChainKey = mac.Sum(nil)

	r.Index++
	return
}

