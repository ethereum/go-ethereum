package wrappeda0gibase

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	supplyKey = []byte{0x00}
)

func SupplyKey(account common.Address) common.Hash {
	return crypto.Keccak256Hash(append(supplyKey, account.Bytes()...))
}
