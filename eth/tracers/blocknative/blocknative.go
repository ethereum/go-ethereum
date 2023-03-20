package blocknative

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// TracerOpts configure the tracer to save or ignore various aspects of a transaction execution.
type TracerOpts struct {
	Logs          bool `json:"logs"`
	NetBalChanges bool `json:"netBalChanges"`
}

// Trace contains all the accumulated details of a transaction execution.
type Trace struct {
	CallFrame
	BlockContext  BlockContext  `json:"blockContext"`
	Logs          []CallLog     `json:"logs,omitempty"`
	NetBalChanges NetBalChanges `json:"netBalChanges,omitempty"`
	Time          string        `json:"time,omitempty"`
}

// BlockContext contains information about the block we simulate transactions in.
type BlockContext struct {
	Number    uint64 `json:"number"`
	StateRoot string `json:"stateRoot,omitempty"`
	BaseFee   uint64 `json:"baseFee"`
	Time      uint64 `json:"time"`
	Coinbase  string `json:"coinbase"`
	GasLimit  uint64 `json:"gasLimit"`
	Random    string `json:"random,omitempty"`
}

type CallFrame struct {
	Type        string      `json:"type"`
	From        string      `json:"from"`
	To          string      `json:"to,omitempty"`
	Value       string      `json:"value,omitempty"`
	Gas         string      `json:"gas"`
	GasUsed     string      `json:"gasUsed"`
	Input       string      `json:"input"`
	Output      string      `json:"output,omitempty"`
	Error       string      `json:"error,omitempty"`
	ErrorReason string      `json:"errorReason,omitempty"`
	Calls       []CallFrame `json:"calls,omitempty"`
}

// CallLog represents a single log entry from the receipt of a transaction.
type CallLog struct {
	// Address is the address of the contract that emitted the log.
	Address common.Address `json:"address"`

	// Data is the encoded memory provided with the log.
	Data string `json:"data"`

	// Topics is a slice of up to 4 32byte words provided with the log.
	Topics []common.Hash `json:"topics"`
}

// NetBalChanges represents the difference of value (ETH, erc20, erc721) after the transaction for all addresses
type NetBalChanges struct {
	Pre  state `json:"pre"`
	Post state `json:"post"`
	// Todo alex: if we want to track contract creations / deletions we must add the idea here!
}

type state = map[common.Address]*account

// Todo alex: This account structure will need some additional post processing mapping for token names and decoding of values
type account struct {
	Balance *big.Int                    `json:"balance,omitempty"`
	Code    []byte                      `json:"code,omitempty"`
	Storage map[common.Hash]common.Hash `json:"storage,omitempty"`
}
