package models


type RoomVisibility int
const (
	RoomPublic RoomVisibility = iota
	RoomPasswordProtected
	RoomPrivate
)
type RoomMeta struct {
	ID string `json:"room_id"`
	Name string `json:"name"`
	Visibility RoomVisibility `json:"visibility"`
	PasswordHash []byte `json:"password_hash,omitempty"`
	PasswordSalt []byte `json:"password_salt,omitempty"`
	EncRoomKey []byte `json:"enc_room_key,omitempty"`
}