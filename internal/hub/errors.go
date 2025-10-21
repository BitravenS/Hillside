package hub

import "hillside/internal/utils"

var (
	ErrDuplicateID = utils.NewHillsideError("duplicate ID detected")
)
