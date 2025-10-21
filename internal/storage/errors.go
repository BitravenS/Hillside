package storage

import "hillside/internal/utils"

var (
	ErrNoRows         = utils.NewHillsideError("no rows in result set")
	ErrDBNotConnected = utils.NewHillsideError("database not connected")
	ErrCannotConnect  = utils.NewHillsideError("cannot connect to database")
)
