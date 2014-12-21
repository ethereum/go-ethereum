package whisper

import "crypto/ecdsa"

type Filter struct {
	To     *ecdsa.PrivateKey
	From   *ecdsa.PublicKey
	Topics [][]byte
	Fn     func(*Message)
}
