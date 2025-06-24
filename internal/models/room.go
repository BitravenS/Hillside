package models


type Visibility int
const (
	Public Visibility = iota
	PasswordProtected
	Private
)
type RoomMeta struct {
	ID string `json:"room_id"`
	Name string `json:"name"`
	Visibility Visibility `json:"visibility"`
	PasswordHash []byte `json:"password_hash,omitempty"`
	PasswordSalt []byte `json:"password_salt,omitempty"`
	EncRoomKey []byte `json:"enc_room_key,omitempty"`
}