package types

import (
	"github.com/scroll-tech/go-ethereum/common"
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
	Sender *AccountProofWrapper `json:"sender,omitempty"`
	// It's exist only when tx is a contract call.
	CodeHash *common.Hash `json:"codeHash,omitempty"`
	// If it is a contract call, the contract code is returned.
	ByteCode   string         `json:"byteCode,omitempty"`
	StructLogs []StructLogRes `json:"structLogs"`
}

// StructLogRes stores a structured log emitted by the EVM while replaying a
// transaction in debug mode
type StructLogRes struct {
	Pc            uint64             `json:"pc"`
	Op            string             `json:"op"`
	Gas           uint64             `json:"gas"`
	GasCost       uint64             `json:"gasCost"`
	Depth         int                `json:"depth"`
	Error         string             `json:"error,omitempty"`
	Stack         *[]string          `json:"stack,omitempty"`
	Memory        *[]string          `json:"memory,omitempty"`
	Storage       *map[string]string `json:"storage,omitempty"`
	RefundCounter uint64             `json:"refund,omitempty"`
	ExtraData     *ExtraData         `json:"extraData,omitempty"`
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
	Balance  string               `json:"balance"` // balance big.Int string
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

// NewExtraData create, init and return ExtraData
func NewExtraData() *ExtraData {
	return &ExtraData{
		CodeList:  make([][]byte, 0),
		ProofList: make([]*AccountProofWrapper, 0),
	}
}

// SealExtraData doesn't show empty fields.
func (e *ExtraData) SealExtraData() *ExtraData {
	if len(e.CodeList) == 0 {
		e.CodeList = nil
	}
	if len(e.ProofList) == 0 {
		e.ProofList = nil
	}
	if e.CodeList == nil && e.ProofList == nil {
		return nil
	}
	return e
}
