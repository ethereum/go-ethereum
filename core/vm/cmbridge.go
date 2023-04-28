package vm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
)

var (
	//cmBridgeContractAddress is 0xDb327e55CA2C68b23f83a0fbe29b592702e1d4d7
	cmBridgeContractAddress = common.BytesToAddress(crypto.Keccak256([]byte("cm_bridge"))[12:])
)

type CallToWasmByPrecompile func(ctx OKContext, caller, to common.Address, value *big.Int, input []byte, remainGas uint64) ([]byte, uint64, error)

type OKContext interface {
	GetEVMStateDB() StateDB
}

// cmBridge implemented as a native contract.
type cmBridge struct {
	context OKContext //OK chain add new Context
	// Context provides auxiliary blockchain related information
	EvmContext BlockContext
	callToWasm CallToWasmByPrecompile
	caller     common.Address
	to         common.Address
	value      *big.Int
}

// RequiredGas returns the gas required to execute the pre-compiled contract.
//
// This method does not require any overflow checking as the input size gas costs
// required for anything significant is so high it's impossible to pay for.
// we can cost some gas
func (c *cmBridge) RequiredGas(input []byte) uint64 {
	panic("cmBridge not support <Run> of implement")
}
func (c *cmBridge) Run(in []byte) ([]byte, error) {
	panic("cmBridge not support <Run> of implement")
}

func (c *cmBridge) CustomRun(in []byte, remainGas uint64) ([]byte, uint64, error) {
	// cmBridge can not got coin, when can cmBridgeContract may be send coin to cmBridgeContractAddress, so we must send coin back to caller.
	if c.value.Sign() != 0 && !c.EvmContext.CanTransfer(c.context.GetEVMStateDB(), cmBridgeContractAddress, c.value) {
		return nil, 0, ErrCMBirdgeInsufficientBalance
	}

	c.EvmContext.Transfer(c.context.GetEVMStateDB(), cmBridgeContractAddress, c.caller, c.value)
	// after send coin back to caller, we send coin
	return c.callToWasm(c.context, c.caller, c.to, c.value, in, remainGas)
}

func NewCMBridge(context OKContext, evmContext BlockContext, callToWasm CallToWasmByPrecompile, caller, to common.Address, value *big.Int) *cmBridge {
	return &cmBridge{context: context, EvmContext: evmContext, callToWasm: callToWasm, caller: caller, to: to, value: value}
}
