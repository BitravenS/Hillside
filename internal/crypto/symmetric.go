package crypto

/* scrapped cunstom PQaead implementation
import (
	"hillside/internal/utils"
	"unsafe"

	"github.com/cloudflare/circl/kem"
	"github.com/cloudflare/circl/kem/kyber/kyber1024"
)


type PQaead struct {
	Key [32]byte
}

func (aead *PQaead) New(key []byte) (*PQaead, error) {
	if len(key) != 32 {
		return nil, utils.PQaeadError("bad key length passed to PQaead New")
	}
	var keyArray [32]byte
	copy(keyArray[:], key)
	return &PQaead{Key: keyArray}, nil
}

func (aead *PQaead) Seal(dst []byte, nonce []byte, plaintext []byte) ([]byte, error) {
	if len(nonce) != 12 {
		return nil, utils.PQaeadError("bad nonce length passed to PQaead Seal")
	}

	if uint64(len(plaintext)) > (1<<38)-64 {
		return nil, utils.PQaeadError("plaintext too large for PQaead Seal")
	}
	ret, out := sliceForAppend(dst, len(plaintext)+16)
	if InexactOverlap(out, plaintext) {
		return nil, utils.PQaeadError("invalid buffer overlap in PQaead Seal")
	}
	pub, priv := kyber1024.NewKeyFromSeed(aead.Key[:])
	pub.(kem.PublicKey).EncryptTo(ret, plaintext, nonce)




}



// Functions defined in golang.org/x/crypto/chacha20poly1305 package
func sliceForAppend(in []byte, n int) (head, tail []byte) {
	if total := len(in) + n; cap(in) >= total {
		head = in[:total]
	} else {
		head = make([]byte, total)
		copy(head, in)
	}
	tail = head[len(in):]
	return
}

func AnyOverlap(x, y []byte) bool {
	return len(x) > 0 && len(y) > 0 &&
		uintptr(unsafe.Pointer(&x[0])) <= uintptr(unsafe.Pointer(&y[len(y)-1])) &&
		uintptr(unsafe.Pointer(&y[0])) <= uintptr(unsafe.Pointer(&x[len(x)-1]))
}

func InexactOverlap(x, y []byte) bool {
	if len(x) == 0 || len(y) == 0 || &x[0] == &y[0] {
		return false
	}
	return AnyOverlap(x, y)
}
*/