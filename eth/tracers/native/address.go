package native

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"math/big"
	"strings"
)

func init() {
	tracers.DefaultDirectory.Register("addressTracer", newAddressTracer, false)
}

var (
	// mapping between opcodes and the location of the address argument on the stack
	// producers: after the instruction, consumers: before the instruction is executed
	// creates: special case
	producers = map[vm.OpCode]int{vm.ADDRESS: 0, vm.ORIGIN: 0, vm.CALLER: 0, vm.COINBASE: 0}
	creates   = map[vm.OpCode]int{vm.CREATE: 0, vm.CREATE2: 0}
	consumers = map[vm.OpCode]int{
		vm.BALANCE: 0, vm.EXTCODESIZE: 0, vm.EXTCODECOPY: 0, vm.EXTCODEHASH: 0,
		vm.CALL: 1, vm.CALLCODE: 1, vm.DELEGATECALL: 1, vm.STATICCALL: 1, vm.SELFDESTRUCT: 0,
	}
)

type myCallFrame struct {
	iid         uint
	opcode      vm.OpCode
	resultIndex int
}

type addressFrame struct {
	MessageId     uint      `json:"mid"`
	InstructionId uint      `json:"iid"`
	Opcode        vm.OpCode `json:"opcode"`
	Mnemonic      string    `json:"mnemonic"`
	PC            uint64    `json:"pc"`
	From          string    `json:"from"`
	Address       string    `json:"address"`
	ResultIndex   int       `json:"-"`
}

type addressTracer struct {
	env          *vm.EVM
	mid          uint
	producerFlag bool
	res          []addressFrame
	reason       error
	callstack    []myCallFrame
}

func newAddressTracer(ctx *tracers.Context, _ json.RawMessage) (tracers.Tracer, error) {
	return &addressTracer{}, nil
}

func (t *addressTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.callstack = append(t.callstack, myCallFrame{iid: 0, resultIndex: -1})
}
func (t *addressTracer) CaptureEnd(output []byte, gasUsed uint64, err error) {}
func (t *addressTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	size := len(t.callstack)
	if depth == size-1 {
		// if we just returned from a call, pop from the callstack before anything else...
		call := t.callstack[size-1]
		t.callstack = t.callstack[:size-1]

		if call.resultIndex > 0 {
			// this is how we check if we came back from a create(2)
			// apparently, if the address is 0, it means that the create has failed...
			// todo, find one instance and cross check it with gimli db
			size = len(scope.Stack.Data())
			t.res[call.resultIndex].Address = strings.ToLower(scope.Stack.Data()[size-1-creates[call.opcode]].String())
		}
	}

	obj := addressFrame{
		MessageId:     t.mid,
		InstructionId: t.callstack[len(t.callstack)-1].iid,
		Opcode:        op,
		Mnemonic:      op.String(),
		PC:            pc,
		From:          strings.ToLower(scope.Contract.Address().String()),
		Address:       "out of gas?",
	}

	t.callstack[len(t.callstack)-1].iid++

	if t.producerFlag {
		previousOp := t.res[len(t.res)-1].Opcode
		if len(scope.Stack.Data()) <= producers[previousOp] {
			t.res[len(t.res)-1].Address = "producer-stack-too-small"
		} else {
			size = len(scope.Stack.Data())
			addrBytes := scope.Stack.Data()[size-1-producers[previousOp]].Bytes()
			t.res[len(t.res)-1].Address = strings.ToLower(common.BytesToAddress(addrBytes).String())
		}
		t.producerFlag = false
	}

	if _, ok := producers[op]; ok {
		// true iff the previous instruction was an address producing opcode; that means the address has just now
		// has become available, so we add it to the last result object
		t.producerFlag = true
		t.res = append(t.res, obj)
		// fmt.Println("added and element from producer op")
	} else if val, ok := consumers[op]; ok {
		if len(scope.Stack.Data()) <= val {
			obj.Address = "consumer-stack-too-small"
		} else {
			size = len(scope.Stack.Data())
			addrBytes := scope.Stack.Data()[size-1-val].Bytes()
			obj.Address = strings.ToLower(common.BytesToAddress(addrBytes).String())
		}
		t.res = append(t.res, obj)
		// fmt.Println("added and element from consumer op")
	} else if _, ok := creates[op]; ok {
		t.res = append(t.res, obj)
		// fmt.Println("added and element from creates op to callstack")
		t.callstack = append(t.callstack, myCallFrame{
			iid:         0,
			opcode:      op,
			resultIndex: len(t.res) - 1,
		})
	}

	// AFAIK, message creating calls are: CREATE, CALL, CALLCODE, DELEGATECALL, CREATE2, STATICALL and SELFDESTRUCT
	// This means that REVERT and RETURN dont create messages. So we don't want to count them.
	// TODO check with gernot if my AFAIK is correct
	if _, ok := map[vm.OpCode]bool{
		vm.CREATE:       true,
		vm.CREATE2:      true,
		vm.CALL:         true,
		vm.CALLCODE:     true,
		vm.DELEGATECALL: true,
		vm.STATICCALL:   true,
		vm.SELFDESTRUCT: true,
	}[op]; ok {
		t.mid = t.mid + 1

		// if it's a CREATE opcode, we've already added one above, now also do it for CALL, CALLCODE etc.
		if _, ok := creates[op]; ok {
			t.callstack = append(t.callstack, myCallFrame{
				iid:         0,
				resultIndex: -1,
			})
		}
	}
}
func (t *addressTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
}
func (t *addressTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {

}
func (t *addressTracer) CaptureExit(output []byte, gasUsed uint64, err error) {}
func (*addressTracer) CaptureTxStart(gasLimit uint64)                         {}
func (*addressTracer) CaptureTxEnd(restGas uint64)                            {}

// GetResult returns an empty json object.
func (t *addressTracer) GetResult() (json.RawMessage, error) {
	res, err := json.Marshal(t.res)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(res), t.reason
}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *addressTracer) Stop(err error) {
}
