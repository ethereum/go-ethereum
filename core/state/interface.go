package state

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/holiman/uint256"
)

type BlockProcessingDB interface {
	CreateAccount(common.Address)
	CreateContract(common.Address)

	SubBalance(common.Address, *uint256.Int, tracing.BalanceChangeReason) uint256.Int
	AddBalance(common.Address, *uint256.Int, tracing.BalanceChangeReason) uint256.Int
	GetBalance(common.Address) *uint256.Int

	GetNonce(common.Address) uint64
	SetNonce(common.Address, uint64, tracing.NonceChangeReason)

	GetCodeHash(common.Address) common.Hash
	GetCode(common.Address) []byte

	// SetCode sets the new code for the address, and returns the previous code, if any.
	SetCode(addr common.Address, code []byte, reason tracing.CodeChangeReason) (prev []byte)
	GetCodeSize(common.Address) int

	AddRefund(uint64)
	SubRefund(uint64)
	GetRefund() uint64

	GetStateAndCommittedState(common.Address, common.Hash) (common.Hash, common.Hash)
	GetState(common.Address, common.Hash) common.Hash
	SetState(common.Address, common.Hash, common.Hash) common.Hash
	GetStorageRoot(addr common.Address) common.Hash

	GetTransientState(addr common.Address, key common.Hash) common.Hash
	SetTransientState(addr common.Address, key, value common.Hash)

	SelfDestruct(common.Address) uint256.Int
	HasSelfDestructed(common.Address) bool

	// SelfDestruct6780 is post-EIP6780 selfdestruct, which means that it's a
	// send-all-to-beneficiary, unless the contract was created in this same
	// transaction, in which case it will be destructed.
	// This method returns the prior balance, along with a boolean which is
	// true iff the object was indeed destructed.
	SelfDestruct6780(common.Address) (uint256.Int, bool)

	// Exist reports whether the given account exists in
	// Notably this also returns true for self-destructed accounts within the current transaction.
	Exist(common.Address) bool
	// Empty returns whether the given account is empty. Empty
	// is defined according to EIP161 (balance = nonce = code = 0).
	Empty(common.Address) bool

	AddressInAccessList(addr common.Address) bool
	SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool)
	// AddAddressToAccessList adds the given address to the access list. This operation is safe to perform
	// even if the feature/fork is not active yet
	AddAddressToAccessList(addr common.Address)
	// AddSlotToAccessList adds the given (address,slot) to the access list. This operation is safe to perform
	// even if the feature/fork is not active yet
	AddSlotToAccessList(addr common.Address, slot common.Hash)

	// PointCache returns the point cache used in computations
	PointCache() *utils.PointCache

	Prepare(rules params.Rules, sender, coinbase common.Address, dest *common.Address, precompiles []common.Address, txAccesses types.AccessList)

	RevertToSnapshot(int)
	Snapshot() int

	AddLog(*types.Log)
	AddPreimage(common.Hash, []byte)

	Witness() *stateless.Witness

	AccessEvents() *AccessEvents

	// Finalise must be invoked at the end of a transaction
	Finalise(bool) (*bal.StateDiff, *bal.StateAccesses)

	// These two methods are not used in the EVM.  however, I need them to be part of the interface
	// so that block processing/production can use instances of this interface so that the StateDB
	// wrapped with BAL creation functionality can be passed in case of BALs
	GetLogs(hash common.Hash, blockNumber uint64, blockHash common.Hash, blockTime uint64) []*types.Log

	IntermediateRoot(deleteEmpty bool) common.Hash

	Database() Database
	GetTrie() Trie
	SetTxContext(thash common.Hash, ti int)
	Error() error

	TxIndex() int

	Copy() BlockProcessingDB
}
