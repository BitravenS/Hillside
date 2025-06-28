package p2p

import "fmt"

// Topic namespace root
const topicRoot = "/hillside"

// Topic learn about active servers (unimplemented)
func ServersTopic() string      { return topicRoot + "/servers" }

// Topic for server metadata
func ServerMetaTopic(sid string) string {
    return fmt.Sprintf("%s/servers/%s/meta", topicRoot, sid)
}

// Topic for info about rooms in a server
func RoomsTopic(sid string) string {
    return fmt.Sprintf("%s/servers/%s/rooms", topicRoot, sid)
}

// Topic for room metadata updates
func RoomMetaTopic(sid, rid string) string {
    return fmt.Sprintf("%s/servers/%s/rooms/%s/meta", topicRoot, sid, rid)
}

// The actual encrypted chat stream for a room
func ChatTopic(sid, rid string) string {
    return fmt.Sprintf("%s/servers/%s/rooms/%s/chat", topicRoot, sid, rid)
}

// Topic for new room-key distribution when the key is rotated
func RekeyTopic(sid, rid string) string {
    return fmt.Sprintf("%s/servers/%s/rooms/%s/rekey", topicRoot, sid, rid)
}

// Topic to keep room members updated
func MembersTopic(sid, rid string) string {
    return fmt.Sprintf("%s/servers/%s/rooms/%s/members", topicRoot, sid, rid)
}

// Topic to allow late joiners to request past ciphertexts
func HistoryReqTopic(sid, rid string) string {
    return fmt.Sprintf("%s/servers/%s/rooms/%s/history/req", topicRoot, sid, rid)
}

// Topic for responses to history requests
func HistoryRespTopic(sid, rid, pid string) string {
    return fmt.Sprintf("%s/servers/%s/rooms/%s/history/resp/%s", topicRoot, sid, rid, pid)
}

// Topic for typing notifications in a room (unimplemented)
func TypingTopic(sid, rid string) string {
    return fmt.Sprintf("%s/servers/%s/rooms/%s/typing", topicRoot, sid, rid)
}
