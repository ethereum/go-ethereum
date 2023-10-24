package native

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
)

//go:generate go run github.com/fjl/gencodec -type account -field-override accountMarshaling -out gen_account_json.go

func init() {
	tracers.DefaultDirectory.Register("zeroTracer", newZeroTracer, false)
}

type Account struct {
	Balance      *big.Int                    `json:"balance,omitempty"`
	Nonce        uint64                      `json:"nonce,omitempty"`
	ReadStorage  map[common.Hash]common.Hash `json:"storage_read,omitempty"`
	WriteStorage map[common.Hash]common.Hash `json:"storage_write,omitempty"`
	CodeUsage    string                      `json:"code_usage,omitempty"`
}

type TX struct {
	ByteCode string                      `json:"byte_code,omitempty"` // TX CallData
	GasUsed  uint64                      `json:"gas_used,omitempty"`
	Trace    map[common.Address]*Account `json:"trace,omitempty"`
}

type zeroTracer struct {
	noopTracer // stub struct to mock not used interface methods
	env        *vm.EVM
	tx         TX
	interrupt  atomic.Bool // Atomic flag to signal execution interruption
	reason     error       // Textual reason for the interruption
}

func newZeroTracer(ctx *tracers.Context, cfg json.RawMessage) (tracers.Tracer, error) {
	return &zeroTracer{
		tx: TX{
			Trace: make(map[common.Address]*Account),
		},
	}, nil
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *zeroTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.env = env
	t.tx.ByteCode = common.Bytes2Hex(input)

	t.addAccountToTrace(from, false)
	t.addAccountToTrace(to, false)
	t.addAccountToTrace(env.Context.Coinbase, false)
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *zeroTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	if err != nil {
		return
	}

	// Skip if tracing was interrupted
	if t.interrupt.Load() {
		return
	}

	stack := scope.Stack
	stackData := stack.Data()
	stackLen := len(stackData)
	caller := scope.Contract.Address()

	switch {
	case stackLen >= 1 && op == vm.SLOAD:
		slot := common.Hash(stackData[stackLen-1].Bytes32())
		t.addSLOADToAccount(caller, slot)
	case stackLen >= 1 && op == vm.SSTORE:
		slot := common.Hash(stackData[stackLen-1].Bytes32())
		t.addSSTOREToAccount(caller, slot)
	case stackLen >= 1 && (op == vm.EXTCODECOPY || op == vm.EXTCODEHASH || op == vm.EXTCODESIZE || op == vm.BALANCE || op == vm.SELFDESTRUCT):
		addr := common.Address(stackData[stackLen-1].Bytes20())
		t.addAccountToTrace(addr, false)
	case stackLen >= 5 && (op == vm.DELEGATECALL || op == vm.CALL || op == vm.STATICCALL || op == vm.CALLCODE):
		addr := common.Address(stackData[stackLen-2].Bytes20())
		t.addAccountToTrace(addr, false)
	case op == vm.CREATE:
		nonce := t.env.StateDB.GetNonce(caller)
		addr := crypto.CreateAddress(caller, nonce)
		t.addAccountToTrace(addr, true)
	case stackLen >= 4 && op == vm.CREATE2:
		offset := stackData[stackLen-2]
		size := stackData[stackLen-3]
		init, err := tracers.GetMemoryCopyPadded(scope.Memory, int64(offset.Uint64()), int64(size.Uint64()))
		if err != nil {
			log.Warn("failed to copy CREATE2 input", "err", err, "tracer", "prestateTracer", "offset", offset, "size", size)
			return
		}
		inithash := crypto.Keccak256(init)
		salt := stackData[stackLen-4]
		addr := crypto.CreateAddress2(caller, salt.Bytes32(), inithash)
		t.addAccountToTrace(addr, true)
	}
}

func (t *zeroTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	t.tx.GasUsed = gasUsed
}

// GetResult returns the json-encoded nested list of call traces, and any
// error arising from the encoding or forceful termination (via `Stop`).
func (t *zeroTracer) GetResult() (json.RawMessage, error) {
	var res []byte
	var err error
	res, err = json.Marshal(t.tx)

	if err != nil {
		return nil, err
	}

	return json.RawMessage(res), t.reason
}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *zeroTracer) Stop(err error) {
	t.reason = err
	t.interrupt.Store(true)
}

func (t *zeroTracer) addAccountToTrace(addr common.Address, created bool) {
	if _, ok := t.tx.Trace[addr]; ok {
		return
	}

	var code string
	if created {
		code = common.Bytes2Hex(t.env.StateDB.GetCode(addr))
	} else {
		code = common.Hash.String(t.env.StateDB.GetCodeHash(addr))
	}

	t.tx.Trace[addr] = &Account{
		Balance:      t.env.StateDB.GetBalance(addr),
		Nonce:        t.env.StateDB.GetNonce(addr),
		CodeUsage:    code,
		WriteStorage: make(map[common.Hash]common.Hash),
		ReadStorage:  make(map[common.Hash]common.Hash),
	}

}

func (t *zeroTracer) addSLOADToAccount(addr common.Address, key common.Hash) {
	t.tx.Trace[addr].ReadStorage[key] = t.env.StateDB.GetState(addr, key)
}

func (t *zeroTracer) addSSTOREToAccount(addr common.Address, key common.Hash) {
	t.tx.Trace[addr].WriteStorage[key] = t.env.StateDB.GetState(addr, key)
}
