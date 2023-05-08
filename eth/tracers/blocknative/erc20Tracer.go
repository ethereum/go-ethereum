package blocknative

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

type erc20Tracer struct {
	evm               *vm.EVM
	affectedAddresses map[common.Address]struct{}
}

func NewErc20Tracer(cfg json.RawMessage) (Tracer, error) {
	return &erc20Tracer{
		affectedAddresses: make(map[common.Address]struct{}),
	}, nil
}

func (t *erc20Tracer) CaptureTxStart(gasLimit uint64) {
}

func (t *erc20Tracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

func (t *erc20Tracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.evm = env
}

// func (t *erc20Tracer) CaptureState(env *vm.EVM, pc uint64, op vm.OpCode, gas, cost uint64, memory *vm.Memory, stack *vm.Stack, rData []byte, contract *vm.Contract, depth int, err error) error {

func (t *erc20Tracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
	if op == vm.SSTORE {
		contractAddress := scope.Contract.Address()
		t.affectedAddresses[contractAddress] = struct{}{}
	}
}

func (t *erc20Tracer) CaptureExit(output []byte, gasUsed uint64, err error) {
}

func (t *erc20Tracer) CaptureFault(pc uint64, op vm.OpCode, gas uint64, cost uint64, scope *vm.ScopeContext, depth int, err error) {
}

func (t *erc20Tracer) CaptureEnd(output []byte, gasUsed uint64, err error) {
	for addr := range t.affectedAddresses {
		symbol, err := t.GetTokenSymbol(t.evm, addr)
		if err == nil {
			fmt.Printf("ERC20 Token Detected: \nContract Address: %s\nToken Symbol: %s\n", addr.Hex(), symbol)
		}
		fmt.Printf("GetTokenSymbol error: ", err)
	}
}

func (t *erc20Tracer) CaptureTxEnd(restGas uint64) {
}

func (t *erc20Tracer) GetResult() (json.RawMessage, error) {
	return nil, nil
}

func (t *erc20Tracer) Stop(err error) {
}

func (t *erc20Tracer) GetTokenSymbol(evm *vm.EVM, contractAddress common.Address) (string, error) {
	erc20SymbolSelector := common.Hex2Bytes("95d89b41") // Function selector for the "symbol()" function
	gasLimit := uint64(5000000)
	// value := big.NewInt(0)

	sender := common.Address{}
	ret, _, err := evm.StaticCall(vm.AccountRef(sender), contractAddress, erc20SymbolSelector, gasLimit)
	if err != nil {
		return "", err
	}

	symbol, err := unpackERC20Symbol(ret)
	if err != nil {
		return "", err
	}

	return symbol, nil
}

func unpackERC20Symbol(data []byte) (string, error) {
	if len(data) < 32 {
		return "", fmt.Errorf("invalid input length")
	}

	strLen := int(new(big.Int).SetBytes(data[:32]).Uint64())
	if len(data) < 32+strLen {
		return "", fmt.Errorf("input too short")
	}

	return string(data[32 : 32+strLen]), nil
}
