package models

type TopicName string

const (
	TopicChat       TopicName = "chat"
	TopicMembers    TopicName = "members"
	TopicCatchUp    TopicName = "catchup"
	TopicUserUpdate TopicName = "userupdate"
	TopicRooms      TopicName = "rooms"
	TopicServers    TopicName = "servers"
)
