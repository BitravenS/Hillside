package p2p

import "fmt"

// Topic namespace root
const topicRoot = "/hillside"

// ServersTopic learn about active servers (unimplemented)
func ServersTopic() string { return topicRoot + "/servers" }

// ServerMetaTopic for server metadata
func ServerMetaTopic(sid string) string {
	return fmt.Sprintf("%s/servers/%s/meta", topicRoot, sid)
}

// RoomsTopic for info about rooms in a server
func RoomsTopic(sid string) string {
	return fmt.Sprintf("%s/servers/%s/rooms", topicRoot, sid)
}

// RoomMetaTopic for room metadata updates
func RoomMetaTopic(sid, rid string) string {
	return fmt.Sprintf("%s/servers/%s/rooms/%s/meta", topicRoot, sid, rid)
}

// ChatTopic for The actual encrypted chat stream for a room
func ChatTopic(sid, rid string) string {
	return fmt.Sprintf("%s/servers/%s/rooms/%s/chat", topicRoot, sid, rid)
}

// RekeyTopic for new room-key distribution when the key is rotated (unimplemented)
func RekeyTopic(sid, rid string) string {
	return fmt.Sprintf("%s/servers/%s/rooms/%s/rekey", topicRoot, sid, rid)
}

// MembersTopic to keep room members updated
func MembersTopic(sid, rid string) string {
	return fmt.Sprintf("%s/servers/%s/rooms/%s/members", topicRoot, sid, rid)
}

func UserUpdateTopic(sid, rid string) string {
	return fmt.Sprintf("%s/servers/%s/rooms/%s/user_update", topicRoot, sid, rid)
}

// HistoryReqTopic to allow late joiners to request past ciphertexts (unimplemented)
func HistoryReqTopic(sid, rid string) string {
	return fmt.Sprintf("%s/servers/%s/rooms/%s/history/req", topicRoot, sid, rid)
}

// HistoryRespTopic for responses to history requests (unimplemented)
func HistoryRespTopic(sid, rid, pid string) string {
	return fmt.Sprintf("%s/servers/%s/rooms/%s/history/resp/%s", topicRoot, sid, rid, pid)
}

// TypingTopic for typing notifications in a room (unimplemented)
func TypingTopic(sid, rid string) string {
	return fmt.Sprintf("%s/servers/%s/rooms/%s/typing", topicRoot, sid, rid)
}

// CatchUpRequestTopic for catch-up requests when a user joins a room to get the RoomRatchet
func CatchUpRequestTopic(sid, rid string) string {
	return fmt.Sprintf("%s/servers/%s/rooms/%s/catchup/request", topicRoot, sid, rid)
}

// CatchUpResponseTopic for catch-up responses containing the RoomRatchet encrypted with the user's key
func CatchUpResponseTopic(sid, rid, pid string) string {
	return fmt.Sprintf("%s/servers/%s/rooms/%s/catchup/%s", topicRoot, sid, rid, pid)
}
