package profile

type Profile struct {
	Username         string `json:"username"`
	PasswordSalt     []byte `json:"password_salt"`
	PasswordChecksum []byte `json:"password_checksum"`
	DilithiumPrivEnc []byte `json:"dilithium_priv_enc"` // encrypted w/ password
	KyberPrivEnc     []byte `json:"kyber_priv_enc"`     // encrypted w/ password
	Libp2pPrivEnc    []byte `json:"libp2p_priv_enc"`    // encrypted w/ password
	PeerID           string `json:"peer_id"`
}
