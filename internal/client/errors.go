package client

import "hillside/internal/utils"

var (
	ErrSendMessageFailed = utils.NewHillsideError("send message failed")
	ErrNotInitialized    = utils.NewHillsideError("not initialized")
)
