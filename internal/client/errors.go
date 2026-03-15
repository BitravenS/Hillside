package client

import "hillside/internal/utils"

var (
	ErrSendMessageFailed  = utils.NewHillsideError("send message failed")
	ErrNotInitialized     = utils.NewHillsideError("not initialized")
	ErrCreateServerFailed = utils.NewHillsideError("create server failed")
	ErrJoinServerFailed   = utils.NewHillsideError("join server failed")
	ErrCreateRoomFailed   = utils.NewHillsideError("create room failed")
	ErrJoinRoomFailed     = utils.NewHillsideError("join room failed")
	ErrSecurityIssue      = utils.NewHillsideError("security issue")
	ErrValidationIssue    = utils.NewHillsideError("validation issue")
)
