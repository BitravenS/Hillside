package models

type Visibility int

const (
	Public Visibility = iota
	PasswordProtected
	Private
)

type RoomMeta struct {
	ID           string     `json:"room_id"`
	Name         string     `json:"name"`
	Visibility   Visibility `json:"visibility"`
	PasswordHash []byte     `json:"password_hash,omitempty"`
	PasswordSalt []byte     `json:"password_salt,omitempty"`
	EncRoomKey   []byte     `json:"enc_room_key,omitempty"`

	Members map[string]Member `json:"members,omitempty"` // key: peer ID, value: Member
}

type RoomSync struct {
	ChainIndex        int64  `json:"chain_index"`
	MasterRoomKey     []byte `json:"master_room_key,omitempty"`
	MasterRoomKeyBase []byte `json:"master_room_key_base,omitempty"` // base key, hashed becomes MasterRoomKey (for Proof of Work)
}
