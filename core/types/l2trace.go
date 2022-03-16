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
	// It's exist only when tx is a contract call.
	CodeHash *common.Hash `json:"codeHash,omitempty"`
	// If it is a contract call, the contract code is returned.
	ByteCode string `json:"byteCode,omitempty"`
	// The account's proof.
	Proof      []string       `json:"proof,omitempty"`
	StructLogs []StructLogRes `json:"structLogs"`
}

// StructLogRes stores a structured log emitted by the EVM while replaying a
// transaction in debug mode
type StructLogRes struct {
	Pc        uint64             `json:"pc"`
	Op        string             `json:"op"`
	Gas       uint64             `json:"gas"`
	GasCost   uint64             `json:"gasCost"`
	Depth     int                `json:"depth"`
	Error     string             `json:"error,omitempty"`
	Stack     *[]string          `json:"stack,omitempty"`
	Memory    *[]string          `json:"memory,omitempty"`
	Storage   *map[string]string `json:"storage,omitempty"`
	ExtraData *ExtraData         `json:"extraData,omitempty"`
}

type ExtraData struct {
	// CREATE | CREATE2: sender address
	From *common.Address `json:"from,omitempty"`
	// CREATE: sender nonce
	Nonce *uint64 `json:"nonce,omitempty"`
	// CALL | CALLCODE | DELEGATECALL | STATICCALL: [tx.to address’s code_hash, stack.nth_last(1) address’s code_hash]
	CodeList [][]byte `json:"codeList,omitempty"`
	// SSTORE | SLOAD: [storageProof]
	// SELFDESTRUCT: [contract address’s accountProof, stack.nth_last(0) address’s accountProof]
	// SELFBALANCE: [contract address’s accountProof]
	// BALANCE | EXTCODEHASH: [stack.nth_last(0) address’s accountProof]
	// CREATE | CREATE2: [created contract address’s accountProof]
	// CALL | CALLCODE: [caller contract address’s accountProof, stack.nth_last(1) address’s accountProof]
	ProofList [][]string `json:"proofList,omitempty"`
}

// NewExtraData create, init and return ExtraData
func NewExtraData() *ExtraData {
	return &ExtraData{
		CodeList:  make([][]byte, 0),
		ProofList: make([][]string, 0),
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
	if e.From == nil && e.Nonce == nil && e.CodeList == nil && e.ProofList == nil {
		return nil
	}
	return e
}
