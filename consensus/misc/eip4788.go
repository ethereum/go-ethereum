package misc

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

func ApplyBeaconRoot(header *types.Header, state *state.StateDB) {
	// If EIP-4788 is enabled, we need to store the block root
	historicalStorageAddress := common.HexToAddress("0xfffffffffffffffffffffffffffffffffffffffd")
	key := header.Time
	value := header.BeaconRoot
	state.SetState(historicalStorageAddress, common.BigToHash(big.NewInt(int64(key))), *value)
}
