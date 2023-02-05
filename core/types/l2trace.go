package types

import (
	"encoding/json"
	"math/big"
	"runtime"
	"sync"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/params"
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

// BlockTrace contains block execution traces and results required for rollers.
type BlockTrace struct {
	Coinbase         *AccountWrapper    `json:"coinbase"`
	FeeVault         *AccountWrapper    `json:"feeVault"`
	Header           *Header            `json:"header"`
	Transactions     []*TransactionData `json:"transactions"`
	StorageTrace     *StorageTrace      `json:"storageTrace"`
	ExecutionResults []*ExecutionResult `json:"executionResults"`
	MPTWitness       *json.RawMessage   `json:"mptwitness,omitempty"`
}

// StorageTrace stores proofs of storage needed by storage circuit
type StorageTrace struct {
	// Root hash before block execution:
	RootBefore common.Hash `json:"rootBefore,omitempty"`
	// Root hash after block execution, is nil if execution has failed
	RootAfter common.Hash `json:"rootAfter,omitempty"`

	// All proofs BEFORE execution, for accounts which would be used in tracing
	Proofs map[string][]hexutil.Bytes `json:"proofs"`

	// All storage proofs BEFORE execution
	StorageProofs map[string]map[string][]hexutil.Bytes `json:"storageProofs,omitempty"`
}

// ExecutionResult groups all structured logs emitted by the EVM
// while replaying a transaction in debug mode as well as transaction
// execution status, the amount of gas used and the return value
type ExecutionResult struct {
	Gas         uint64 `json:"gas"`
	Failed      bool   `json:"failed"`
	ReturnValue string `json:"returnValue"`
	// Sender's account state (before Tx)
	From *AccountWrapper `json:"from,omitempty"`
	// Receiver's account state (before Tx)
	To *AccountWrapper `json:"to,omitempty"`
	// AccountCreated record the account if the tx is "create"
	// (for creating inside a contract, we just handle CREATE op)
	AccountCreated *AccountWrapper `json:"accountCreated,omitempty"`

	// Record all accounts' state which would be affected AFTER tx executed
	// currently they are just `from` and `to` account
	AccountsAfter []*AccountWrapper `json:"accountAfter"`

	// `CodeHash` only exists when tx is a contract call.
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
	Error         string            `json:"error,omitempty"`
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
	if err != nil {
		logRes.Error = err.Error()
	}
	runtime.SetFinalizer(logRes, func(logRes *StructLogRes) {
		logRes.Stack = logRes.Stack[:0]
		logRes.Memory = logRes.Memory[:0]
		logRes.Storage = nil
		logRes.ExtraData = nil
		logRes.Error = ""
		loggerResPool.Put(logRes)
	})
	return logRes
}

type ExtraData struct {
	// Indicate the call succeeds or not for CALL/CREATE op
	CallFailed bool `json:"callFailed,omitempty"`
	// CALL | CALLCODE | DELEGATECALL | STATICCALL: [tx.to address’s code, stack.nth_last(1) address’s code]
	// CREATE | CREATE2: [created contract’s code]
	// CODESIZE | CODECOPY: [contract’s code]
	// EXTCODESIZE | EXTCODECOPY: [stack.nth_last(0) address’s code]
	CodeList []string `json:"codeList,omitempty"`
	// SSTORE | SLOAD: [storageProof]
	// SELFDESTRUCT: [contract address’s account, stack.nth_last(0) address’s account]
	// SELFBALANCE: [contract address’s account]
	// BALANCE | EXTCODEHASH: [stack.nth_last(0) address’s account]
	// CREATE | CREATE2: [created contract address’s account (before constructed),
	// 					  created contract address's account (after constructed)]
	// CALL | CALLCODE: [caller contract address’s account,
	// 					stack.nth_last(1) (i.e. callee) address’s account,
	//					callee contract address's account (value updated, before called)]
	// STATICCALL: [stack.nth_last(1) (i.e. callee) address’s account,
	//					  callee contract address's account (before called)]
	StateList []*AccountWrapper `json:"proofList,omitempty"`
	// The status of caller, it would be captured twice:
	// 1. before execution and 2. updated in CaptureEnter (for CALL/CALLCODE it duplicated with StateList[0])
	Caller []*AccountWrapper `json:"caller,omitempty"`
}

type AccountWrapper struct {
	Address  common.Address  `json:"address"`
	Nonce    uint64          `json:"nonce"`
	Balance  *hexutil.Big    `json:"balance"`
	CodeHash common.Hash     `json:"codeHash,omitempty"`
	Storage  *StorageWrapper `json:"storage,omitempty"` // StorageWrapper can be empty if irrelated to storage operation
}

// while key & value can also be retrieved from StructLogRes.Storage,
// we still stored in here for roller's processing convenience.
type StorageWrapper struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type TransactionData struct {
	Type     uint8           `json:"type"`
	Nonce    uint64          `json:"nonce"`
	TxHash   string          `json:"txHash"`
	Gas      uint64          `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	ChainId  *hexutil.Big    `json:"chainId"`
	Value    *hexutil.Big    `json:"value"`
	Data     string          `json:"data"`
	IsCreate bool            `json:"isCreate"`
	V        *hexutil.Big    `json:"v"`
	R        *hexutil.Big    `json:"r"`
	S        *hexutil.Big    `json:"s"`
}

// NewTransactionData returns a transaction that will serialize to the trace
// representation, with the given location metadata set (if available).
func NewTransactionData(tx *Transaction, blockNumber uint64, config *params.ChainConfig) *TransactionData {
	signer := MakeSigner(config, big.NewInt(0).SetUint64(blockNumber))
	from, _ := Sender(signer, tx)
	v, r, s := tx.RawSignatureValues()
	result := &TransactionData{
		Type:     tx.Type(),
		TxHash:   tx.Hash().String(),
		Nonce:    tx.Nonce(),
		ChainId:  (*hexutil.Big)(tx.ChainId()),
		From:     from,
		Gas:      tx.Gas(),
		GasPrice: (*hexutil.Big)(tx.GasPrice()),
		To:       tx.To(),
		Value:    (*hexutil.Big)(tx.Value()),
		Data:     hexutil.Encode(tx.Data()),
		IsCreate: tx.To() == nil,
		V:        (*hexutil.Big)(v),
		R:        (*hexutil.Big)(r),
		S:        (*hexutil.Big)(s),
	}
	return result
}
