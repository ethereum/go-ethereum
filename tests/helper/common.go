package helper

import "github.com/ethereum/go-ethereum/ethutil"

func FromHex(h string) []byte {
	if ethutil.IsHex(h) {
		h = h[2:]
	}

	return ethutil.Hex2Bytes(h)
}
