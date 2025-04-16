package wrappeda0gibase

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type Supply = struct {
	Cap           *big.Int "json:\"cap\""
	InitialSupply *big.Int "json:\"initialSupply\""
	Supply        *big.Int "json:\"supply\""
}

var (
	supplyKey = []byte{0x00}
)

func SupplyKey(account common.Address) common.Hash {
	return crypto.Keccak256Hash(append(supplyKey, account.Bytes()...))
}
