package ethcrypto

import (
	"crypto/sha256"

	"code.google.com/p/go.crypto/ripemd160"
	"code.google.com/p/go.crypto/sha3"
	"github.com/ethereum/eth-go/ethutil"
)

func Sha256Bin(data []byte) []byte {
	hash := sha256.Sum256(data)

	return hash[:]
}

func Ripemd160(data []byte) []byte {
	ripemd := ripemd160.New()
	ripemd.Write(data)

	return ripemd.Sum(nil)
}

func Sha3Bin(data []byte) []byte {
	d := sha3.New256()
	d.Write(data)

	return d.Sum(nil)
}

// Creates an ethereum address given the bytes and the nonce
func CreateAddress(b []byte, nonce uint64) []byte {
	return Sha3Bin(ethutil.NewValue([]interface{}{b, nonce}).Encode())[12:]
}
