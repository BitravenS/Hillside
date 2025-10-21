package crypto

type RoomRatchet struct {
	ChainKey []byte // secret state
	Index    uint64 // message count
}
