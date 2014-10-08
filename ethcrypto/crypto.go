package ethcrypto

import (
	//"code.google.com/p/go.crypto/sha3"
	"crypto/sha256"

	"code.google.com/p/go.crypto/ripemd160"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/obscuren/sha3"

	"github.com/obscuren/secp256k1-go"
)

// TODO refactor, remove (bin)
func Sha3Bin(data []byte) []byte {
	d := sha3.NewKeccak256()
	d.Write(data)

	return d.Sum(nil)
}

// Creates an ethereum address given the bytes and the nonce
func CreateAddress(b []byte, nonce uint64) []byte {
	return Sha3Bin(ethutil.NewValue([]interface{}{b, nonce}).Encode())[12:]
}

func Sha256(data []byte) []byte {
	hash := sha256.Sum256(data)

	return hash[:]
}

func Ripemd160(data []byte) []byte {
	ripemd := ripemd160.New()
	ripemd.Write(data)

	return ripemd.Sum(nil)
}

func Ecrecover(data []byte) []byte {
	var in = struct {
		hash []byte
		sig  []byte
	}{data[:32], data[32:]}

	r, _ := secp256k1.RecoverPubkey(in.hash, in.sig)

	return r
}
