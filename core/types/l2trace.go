package types

import (
	"runtime"
	"sync"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
)

var (
	loggerResPool = sync.Pool{
		New: func() interface{} {
			// init arrays here; other types are inited with default values
			return &StructLogRes{
				Stack:  []string{},
				Memory: []string{},
			}
		},
	}
)

// BlockResult contains block execution traces and results required for rollers.
type BlockResult struct {
	BlockTrace       *BlockTrace        `json:"blockTrace"`
	ExecutionResults []*ExecutionResult `json:"executionResults"`
}

// ExecutionResult groups all structured logs emitted by the EVM
// while replaying a transaction in debug mode as well as transaction
// execution status, the amount of gas used and the return value
type ExecutionResult struct {
	Gas         uint64 `json:"gas"`
	Failed      bool   `json:"failed"`
	ReturnValue string `json:"returnValue,omitempty"`
	// Sender's account proof.
	From *AccountProofWrapper `json:"from,omitempty"`
	// Receiver's account proof.
	To *AccountProofWrapper `json:"to,omitempty"`
	// It's exist only when tx is a contract call.
	CodeHash *common.Hash `json:"codeHash,omitempty"`
	// If it is a contract call, the contract code is returned.
	ByteCode   string          `json:"byteCode,omitempty"`
	StructLogs []*StructLogRes `json:"structLogs"`
}

// StructLogRes stores a structured log emitted by the EVM while replaying a
// transaction in debug mode
type StructLogRes struct {
	Pc            uint64            `json:"pc"`
	Op            string            `json:"op"`
	Gas           uint64            `json:"gas"`
	GasCost       uint64            `json:"gasCost"`
	Depth         int               `json:"depth"`
	Error         error             `json:"error,omitempty"`
	Stack         []string          `json:"stack,omitempty"`
	Memory        []string          `json:"memory,omitempty"`
	Storage       map[string]string `json:"storage,omitempty"`
	RefundCounter uint64            `json:"refund,omitempty"`
	ExtraData     *ExtraData        `json:"extraData,omitempty"`
}

// Basic StructLogRes skeleton, Stack&Memory&Storage&ExtraData are separated from it for GC optimization;
// still need to fill in with Stack&Memory&Storage&ExtraData
func NewStructLogResBasic(pc uint64, op string, gas, gasCost uint64, depth int, refundCounter uint64, err error) *StructLogRes {
	logRes := loggerResPool.Get().(*StructLogRes)
	logRes.Pc, logRes.Op, logRes.Gas, logRes.GasCost, logRes.Depth, logRes.RefundCounter = pc, op, gas, gasCost, depth, refundCounter
	logRes.Error = err
	runtime.SetFinalizer(logRes, func(logRes *StructLogRes) {
		logRes.Stack = logRes.Stack[:0]
		logRes.Memory = logRes.Memory[:0]
		logRes.Storage = nil
		logRes.ExtraData = nil
		loggerResPool.Put(logRes)
	})
	return logRes
}

type ExtraData struct {
	// CALL | CALLCODE | DELEGATECALL | STATICCALL: [tx.to address’s code, stack.nth_last(1) address’s code]
	CodeList [][]byte `json:"codeList,omitempty"`
	// SSTORE | SLOAD: [storageProof]
	// SELFDESTRUCT: [contract address’s accountProof, stack.nth_last(0) address’s accountProof]
	// SELFBALANCE: [contract address’s accountProof]
	// BALANCE | EXTCODEHASH: [stack.nth_last(0) address’s accountProof]
	// CREATE | CREATE2: [sender's accountProof, created contract address’s accountProof]
	// CALL | CALLCODE: [caller contract address’s accountProof, stack.nth_last(1) address’s accountProof]
	ProofList []*AccountProofWrapper `json:"proofList,omitempty"`
}

type AccountProofWrapper struct {
	Address  common.Address       `json:"address"`
	Nonce    uint64               `json:"nonce"`
	Balance  *hexutil.Big         `json:"balance"`
	CodeHash common.Hash          `json:"codeHash,omitempty"`
	Proof    []string             `json:"proof,omitempty"`
	Storage  *StorageProofWrapper `json:"storage,omitempty"` // StorageProofWrapper can be empty if irrelated to storage operation
}

// while key & value can also be retrieved from StructLogRes.Storage,
// we still stored in here for roller's processing convenience.
type StorageProofWrapper struct {
	Key   string   `json:"key,omitempty"`
	Value string   `json:"value,omitempty"`
	Proof []string `json:"proof,omitempty"`
}
