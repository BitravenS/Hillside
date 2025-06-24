package models

type ListServersRequest struct{} 
type ListServersResponse struct {
	Servers []ServerMeta `json:"servers"`
}

// CreateServer
type CreateServerRequest struct {
	Name       string           `json:"name"`
	Visibility ServerVisibility `json:"visibility"`
	Description string `json:"description"`
	PasswordHash []byte `json:"password_hash,omitempty"`
	PasswordSalt []byte `json:"password_salt,omitempty"`
	Creator User `json:"creator"`
}
type CreateServerResponse struct {
	ServerID string `json:"server_id"`
}

// ListRooms
type ListRoomsRequest struct {
	ServerID string `json:"server_id"`
}
type ListRoomsResponse struct {
	Rooms []RoomMeta `json:"rooms"`
	Error string `json:"error,omitempty"`
}

// CreateRoom
type CreateRoomRequest struct {
	ServerID          string           `json:"server_id"`
	RoomName          string           `json:"room_name"`
	Visibility        RoomVisibility   `json:"visibility"`
	PasswordHash []byte `json:"password_hash,omitempty"`
	PasswordSalt []byte `json:"password_salt,omitempty"`
	EncRoomKey []byte `json:"enc_room_key,omitempty"`
}
type CreateRoomResponse struct{
	Error string `json:"error,omitempty"`
}