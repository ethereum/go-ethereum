package blocknative

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

var Tracers = map[string]func(cfg json.RawMessage) (Tracer, error){
	"txnOpCodeTracer": NewTxnOpCodeTracer,
}

type Tracer interface {
	vm.EVMLogger
	GetResult() (json.RawMessage, error)
	Stop(err error)
}

// TracerOpts configure the tracer to save or ignore various aspects of a
// transaction execution.
type TracerOpts struct {
	Logs bool `json:"logs"`
}

// Trace contains all the accumulated details of a transaction execution.
type Trace struct {
	CallFrame
	Logs []Log  `json:"logs,omitempty"`
	Time string `json:"time,omitempty"`
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

// Log represents a single log entry from the receipt of a transaction.
type Log struct {
	// Address is the address of the contract that emitted the log.
	Address common.Address `json:"address"`

	// Data is the encoded memory provided with the log.
	Data string `json:"data"`

	// Topics is a slice of up to 4 32byte words provided with the log.
	Topics []common.Hash `json:"topics"`
}
