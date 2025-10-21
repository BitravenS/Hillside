package models

type ListServersRequest struct{}
type ListServersResponse struct {
	Servers []ServerMeta `json:"servers"`
}

type CreateServerRequest struct {
	Name         string     `json:"name"`
	Visibility   Visibility `json:"visibility"`
	Description  string     `json:"description"`
	PasswordHash []byte     `json:"password_hash,omitempty"`
	PasswordSalt []byte     `json:"password_salt,omitempty"`
	Creator      User       `json:"creator"`
}
type CreateServerResponse struct {
	ServerID string `json:"server_id"`
}

type ListRoomsRequest struct {
	ServerID string `json:"server_id"`
}
type ListRoomsResponse struct {
	Rooms []RoomMeta `json:"rooms"`
	Error string     `json:"error,omitempty"`
}

type CreateRoomRequest struct {
	ServerID     string     `json:"server_id"`
	RoomName     string     `json:"room_name"`
	Visibility   Visibility `json:"visibility"`
	PasswordHash []byte     `json:"password_hash,omitempty"`
	PasswordSalt []byte     `json:"password_salt,omitempty"`
	EncRoomKey   []byte     `json:"enc_room_key,omitempty"`
}
type CreateRoomResponse struct {
	RoomID string `json:"room_id"`
	Error  string `json:"error,omitempty"`
}

type JoinServerRequest struct {
	ServerID     string `json:"server_id"`
	PasswordHash []byte `json:"password_hash,omitempty"`
}

type JoinServerResponse struct {
	Server *ServerMeta `json:"server,omitempty"`
	Error  string      `json:"error,omitempty"`
}

type JoinRoomRequest struct {
	ServerID     string `json:"server_id"`
	RoomID       string `json:"room_id"`
	PasswordHash []byte `json:"password_hash,omitempty"`
	Sender       User   `json:"sender"`
}

type JoinRoomResponse struct {
	Room  *RoomMeta `json:"room,omitempty"`
	Error string    `json:"error,omitempty"`
}

type ListRoomMembersRequest struct {
	ServerID string `json:"server_id"`
	RoomID   string `json:"room_id"`
}

type ListRoomMembersResponse struct {
	Members []Member `json:"members"`
	Error   string   `json:"error,omitempty"`
}

