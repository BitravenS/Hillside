package crypto

import "hillside/internal/utils"

var (
	ErrEncryptionFailed = utils.NewHillsideError("encryption failed")
	ErrSigningFailed    = utils.NewHillsideError("signing failed")
	ErrSignatureInvalid = utils.NewHillsideError("signature invalid")
	ErrBadKey           = utils.NewHillsideError("invalid key provided")
)
