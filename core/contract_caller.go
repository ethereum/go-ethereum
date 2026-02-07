package core

import (
	"context"
	"math"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

type contractCaller struct {
	evm  *vm.EVM
	addr common.Address
}

func (c contractCaller) CodeAt(ctx context.Context, addr common.Address, blockNumber *big.Int) ([]byte, error) {
	return c.evm.StateDB.GetCode(addr), nil
}

func (c contractCaller) CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	value := new(uint256.Int)
	if call.Value != nil {
		value.SetFromBig(call.Value)
	}

	from := call.From
	if from == (common.Address{}) {
		from = params.SystemAddress
	}

	// when gas is set to zero, it should use nearly infinite gas,
	// according to the CallMsg definition
	gas := call.Gas
	if gas == 0 {
		gas = uint64(math.MaxUint64)
	}

	res, _, err := c.evm.Call(from, c.addr, call.Data, gas, value)

	return res, err
}
