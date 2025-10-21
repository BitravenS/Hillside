package models

import "hillside/internal/utils"

var (
	ErrServerNotFound = utils.NewHillsideError("server not found")
	ErrRoomNotFound   = utils.NewHillsideError("room not found")
)

