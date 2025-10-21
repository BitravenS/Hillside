package crypto

import (
	ks "github.com/cloudflare/circl/kem/schemes"
	ss "github.com/cloudflare/circl/sign/schemes"
)

var (
	DilithiumScheme = ss.ByName("Dilithium2")
	KyberScheme     = ks.ByName("Kyber1024")
)
