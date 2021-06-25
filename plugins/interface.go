package plugins

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)


type StateDB interface {
	Error() error
	GetLogs(hash common.Hash) []*types.Log
	Logs() []*types.Log
	Preimages() map[common.Hash][]byte
	Exist(addr common.Address) bool
	Empty(addr common.Address) bool
	GetBalance(addr common.Address) *big.Int
	GetNonce(addr common.Address) uint64
	TxIndex() int
	BlockHash() common.Hash
	GetCode(addr common.Address) []byte
	GetCodeSize(addr common.Address) int
	GetCodeHash(addr common.Address) common.Hash
	GetState(addr common.Address, hash common.Hash) common.Hash
	GetProof(addr common.Address) ([][]byte, error)
	GetProofByHash(addrHash common.Hash) ([][]byte, error)
	GetStorageProof(a common.Address, key common.Hash) ([][]byte, error)
	GetStorageProofByHash(a common.Address, key common.Hash) ([][]byte, error)
	GetCommittedState(addr common.Address, hash common.Hash) common.Hash
	HasSuicided(addr common.Address) bool
	ForEachStorage(addr common.Address, cb func(key, value common.Hash) bool) error
	GetRefund() uint64
	AddressInAccessList(addr common.Address) bool
	SlotInAccessList(addr common.Address, slot common.Hash) (addressPresent bool, slotPresent bool)
}
