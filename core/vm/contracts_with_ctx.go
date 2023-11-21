package vm

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/params"
)

// Wrapper type which allows PrecompiledContract to be used as PrecompiledContractWithCtx
type precompileWrapper struct {
	PrecompiledContract
}

func (pw precompileWrapper) Run(input []byte, ctx *precompileContext) ([]byte, error) {
	return pw.PrecompiledContract.Run(input)
}

// Interface for precompiled contract with ctx object allowing for writes to state.
type PrecompiledContractWithCtx interface {
	RequiredGas(input []byte) uint64
	Run(input []byte, ctx *precompileContext) ([]byte, error)
}

type precompileContext struct {
	*BlockContext
	*params.Rules

	caller common.Address
	evm    *EVM
}

func NewContext(caller common.Address, evm *EVM) *precompileContext {
	return &precompileContext{
		BlockContext: &evm.Context,
		Rules:        &evm.chainRules,
		caller:       caller,
		evm:          evm,
	}
}

var vmBlockCtx = BlockContext{
	CanTransfer: func(db StateDB, addr common.Address, amount *big.Int) bool {
		return db.GetBalance(addr).Cmp(amount) >= 0
	},
	Transfer: func(StateDB, common.Address, common.Address, *big.Int) {
		panic("transfer: not implemented")
	},
	GetHash: func(u uint64) common.Hash {
		panic("getHash: not implemented")
	},
	Coinbase:    common.Address{},
	BlockNumber: new(big.Int).SetUint64(10),
	Time:        uint64(time.Now().Unix()),
}

var vmTxCtx = TxContext{
	GasPrice: common.Big1,
	Origin:   common.HexToAddress("a11ce"),
}

// Create a global mock EVM for use in the following tests.
var mockEVM = &EVM{
	Context:   vmBlockCtx,
	TxContext: vmTxCtx,
}

// Native transfer precompile to make bridging to native token possible.
type transfer struct{}

func (c *transfer) RequiredGas(input []byte) uint64 {
	// TODO: determine appropriate gas cost
	return 100
}

func Run(input []byte, ctx *precompileContext) ([]byte, error) {

	// TODO: filter out non-allowed callers

	// From
	_ = common.BytesToAddress(input[0:32])

	to := common.BytesToAddress(input[32:64])

	var parsed bool
	value, parsed := math.ParseBig256(hexutil.Encode(input[64:96]))
	if !parsed {
		return nil, fmt.Errorf("Error parsing transfer: unable to parse value from " + hexutil.Encode(input[64:96]))
	}

	// Mint case: Create native token out of thin air
	ctx.evm.StateDB.AddBalance(to, value)

	// if !ctx.CanTransfer(ctx.evm.StateDB, from, value) {
	// 	return nil, ErrInsufficientBalance
	// }
	// ctx.Transfer(ctx.evm, from, to, value)

	return input, nil
}
