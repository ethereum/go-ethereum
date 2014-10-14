package helper

import "github.com/ethereum/eth-go/ethutil"

func FromHex(h string) []byte {
	if ethutil.IsHex(h) {
		h = h[2:]
	}

	return ethutil.Hex2Bytes(h)
}
