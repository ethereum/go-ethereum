// Ported verbatim from github.com/QuarkChain/goquarkchain/account (byte-compatible).

package account

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var (
	// ErrGenIdentityKey error info : err generate identity key
	ErrGenIdentityKey = errors.New("ErrGenIdentityKey")
)

// DefaultKeyStoreDirectory default keystore dir
const (
	DefaultKeyStoreDirectory = "./keystore/"
	kdfParamsPrf             = "prf"
	kdfParamsPrfValue        = "hmac-sha256"
	kdfParamsPrfDkLen        = "dklen"
	kdfParamsPrfDkLenValue   = 32
	kdfParamsC               = "c"
	kdfParamsCValue          = 262144
	kdfParamsSalt            = "salt"

	cryptoKDF     = "pbkdf2"
	cryptoCipher  = "aes-128-ctr"
	cryptoVersion = 1

	jsonVersion = 3
)

const (
	RecipientLength    = 20
	KeyLength          = 32
	FullShardKeyLength = 4
)

// Recipient recipient type
type Recipient = common.Address

// BytesToIdentityRecipient trans bytes to Recipient
func BytesToIdentityRecipient(b []byte) Recipient {
	return Recipient(common.BytesToAddress(b))
}

func IsSameReceipt(a, b Recipient) bool {
	for index := 0; index < 20; index++ {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func IsSameAddress(a, b Address) bool {
	return IsSameReceipt(a.Recipient, b.Recipient) && a.FullShardKey == b.FullShardKey
}

// Key key type
type Key [KeyLength]byte

// SetBytes set bytes to it's value
func (a *Key) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-KeyLength:]
	}
	copy(a[KeyLength-len(b):], b)
}

// Bytes return it's bytes
func (a Key) Bytes() []byte {
	return a[:]
}

// BytesToIdentityKey trans bytes to Key
func BytesToIdentityKey(b []byte) Key {
	var a Key
	a.SetBytes(b)
	return a
}

type CoinbaseStatses struct {
	CoinbaseStatsList []CoinbaseStats `json:"ReceiptCntList" gencodec:"required" bytesizeofslicelen:"4"`
}

type CoinbaseStats struct {
	Addr Recipient
	Cnt  uint32
}
