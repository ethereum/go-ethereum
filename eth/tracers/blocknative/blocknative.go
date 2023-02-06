package blocknative

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/core/vm"
)

var Tracers = map[string]func() (Tracer, error){
	"txnOpCodeTracer": NewTxnOpCodeTracer,
}

type Tracer interface {
	vm.EVMLogger
	GetResult() (json.RawMessage, error)
	Stop(err error)
}

// Trace contains all the accumulated details of a transaction execution.
type Trace struct {
	CallFrame
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
