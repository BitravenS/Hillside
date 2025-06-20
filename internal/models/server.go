package models

type ServerVisibility int
const (
	ServerPublic ServerVisibility = iota
	ServerPasswordProtected
	ServerPrivate
)

type ServerMeta struct {
	ID string `json:"server_id"`
	Name string `json:"name"`
	Description string `json:"description"`
	Visibility ServerVisibility `json:"visibility"`
	OwnerPeerID string `json:"owner_peer_id"`
	CreatedAt int64 `json:"created_at"`
	PasswordHash []byte `json:"password_hash,omitempty"`
	PasswordSalt []byte `json:"password_salt,omitempty"`
	Rooms map[string]*RoomMeta `json:"rooms"`
}