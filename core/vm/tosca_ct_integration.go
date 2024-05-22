package vm

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
)

// Error
var ErrStopToken = errStopToken

// Gas table
func MemoryGasCost(mem *Memory, wordSize uint64) (uint64, error) {
	return memoryGasCost(mem, wordSize)
}

// Stack
func NewStack() *Stack {
	return newstack()
}

func (st *Stack) Len() int {
	return st.len()
}

func (st *Stack) Push(d *uint256.Int) {
	st.push(d)
}

// EVM
func (evm *EVM) GetDepth() int {
	return evm.depth
}

func (evm *EVM) SetDepth(depth int) {
	evm.depth = depth
}

// Interpreter
func (g *EVMInterpreter) SetLastCallReturnData(data []byte) {
	g.returnData = data
}

func (g *EVMInterpreter) GetLastCallReturnData() []byte {
	return g.returnData
}

func (g *EVMInterpreter) SetReadOnly(readOnly bool) {
	g.readOnly = readOnly
}

func (g *EVMInterpreter) IsReadOnly() bool {
	return g.readOnly
}

// CallContext Call interceptor

// CallContext provides a basic interface for the EVM calling conventions. The EVM
// depends on this context being implemented for doing subcalls and initialising new EVM contracts.
type CallContextInterceptor interface {
	// Call calls another contract.
	Call(env *EVM, me ContractRef, addr common.Address, data []byte, gas, value *big.Int) ([]byte, uint64, error)
	// CallCode takes another contracts code and execute within our own context
	CallCode(env *EVM, me ContractRef, addr common.Address, data []byte, gas, value *big.Int) ([]byte, uint64, error)
	// DelegateCall is same as CallCode except sender and value is propagated from parent to child scope
	DelegateCall(env *EVM, me ContractRef, addr common.Address, data []byte, gas *big.Int) ([]byte, uint64, error)
	// Create creates a new contract
	Create(env *EVM, me ContractRef, data []byte, gas, value *big.Int) ([]byte, common.Address, uint64, error)

	StaticCall(env *EVM, me ContractRef, addr common.Address, input []byte, gas *big.Int) ([]byte, uint64, error)
	Create2(env *EVM, me ContractRef, code []byte, gas *big.Int, value *big.Int, salt *uint256.Int) ([]byte, common.Address, uint64, error)
}

// Interpreter interface
// EVMInterpreter defines an interface for different interpreter implementations.
type GethEVMInterpreter interface {
	// Run the contract's code with the given input data and returns the return byte-slice
	// and an error if one occurred.
	Run(contract *Contract, input []byte, readOnly bool) (ret []byte, err error)
}

type InterpreterFactory func(evm *EVM, cfg Config) GethEVMInterpreter

var interpreter_registry = map[string]InterpreterFactory{}

func RegisterInterpreterFactory(name string, factory InterpreterFactory) {
	interpreter_registry[strings.ToLower(name)] = factory
}

func NewInterpreter(name string, evm *EVM, cfg Config) GethEVMInterpreter {
	factory, found := interpreter_registry[strings.ToLower(name)]
	if !found {
		log.Error("no factory for interpreter registered", "name", name)
	}
	return factory(evm, cfg)
}

func init() {
	factory := func(evm *EVM, cfg Config) GethEVMInterpreter {
		return NewEVMInterpreter(evm)
	}
	RegisterInterpreterFactory("", factory)
	RegisterInterpreterFactory("geth", factory)
}

// Abstracted interpreter with single step execution.

// GethState represents the internal state of the interpreter.
type GethState struct {
	Contract *Contract // processed contract
	Memory   *Memory   // bound memory
	Stack    *Stack    // local stack
	// For optimisation reason we're using uint64 as the program counter.
	// It's theoretically possible to go above 2^64. The YP defines the PC
	// to be uint256. Practically much less so feasible.
	Pc          uint64 // program counter
	Result      []byte // result of the opcode execution function
	Err         error
	CallContext *ScopeContext
	ReadOnly    bool
	Halted      bool

	op   OpCode // current opcode
	cost uint64

	// copies used by tracer
	pcCopy  uint64 // needed for the deferred Tracer
	gasCopy uint64 // for Tracer to log gas remaining before execution
	logged  bool   // deferred Tracer should ignore already logged steps
}

func NewGethState(contract *Contract, memory *Memory, stack *Stack, Pc uint64) *GethState {
	return &GethState{
		Contract: contract,
		Memory:   memory,
		Stack:    stack,
		Pc:       Pc,
		CallContext: &ScopeContext{
			Memory:   memory,
			Stack:    stack,
			Contract: contract,
		},
	}
}

type InterpreterState struct {
	Contract *Contract
	Stack    *Stack
	Memory   *Memory
	pc       uint64
	finished bool
}

func (in *EVMInterpreter) Run(contract *Contract, input []byte, readOnly bool) (ret []byte, err error) {
	state := InterpreterState{
		Contract: contract,
		Stack:    NewStack(),
		Memory:   NewMemory(),
	}
	defer returnStack(state.Stack)
	return in.run(&state, input, readOnly)
}
