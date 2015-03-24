package helper

import "github.com/ethereum/go-ethereum/common"

func FromHex(h string) []byte {
	if common.IsHex(h) {
		h = h[2:]
	}

	return common.Hex2Bytes(h)
}
