package withdrawtrie

import (
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/rollup/rcfg"
)

// StateDB represents the StateDB interface
// required to get withdraw trie root
type StateDB interface {
	GetState(common.Address, common.Hash) common.Hash
}

// ReadWTRSlot reads WithdrawTrieRoot slot in L2MessageQueue predeploy, i.e., `messageRoot`
// in contracts/src/libraries/common/AppendOnlyMerkleTree.sol
func ReadWTRSlot(addr common.Address, state StateDB) common.Hash {
	return state.GetState(addr, rcfg.WithdrawTrieRootSlot)
}
