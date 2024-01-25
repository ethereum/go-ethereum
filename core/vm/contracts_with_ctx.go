package vm

import (
	"fmt"
	"github.com/holiman/uint256"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
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
	CanTransfer: func(db StateDB, addr common.Address, amount *uint256.Int) bool {
		return db.GetBalance(addr).Cmp(amount) >= 0
	},
	Transfer: func(StateDB, common.Address, common.Address, *uint256.Int) {
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

// Native token mint precompile to make bridging to native token possible.
type mint struct{}

func (c *mint) RequiredGas(input []byte) uint64 {
	// TODO: determine appropriate gas cost
	return 100
}

// Predetermined create2 address of whitelist contract with exclusive mint/burn privileges.
// This address assumes deployer is 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266.
const whitelistCreate2Addr = "0x5D1415C0973034d162F5FEcF19B50dA057057e29"

func (c *mint) Run(input []byte, ctx *precompileContext) ([]byte, error) {

	if ctx.caller != common.HexToAddress(whitelistCreate2Addr) {
		log.Error("Error parsing transfer: caller not whitelisted")
		return nil, fmt.Errorf("Error parsing transfer: caller not whitelisted")
	}

	mintTo := common.BytesToAddress(input[0:32])
	value, err := uint256.FromHex(hexutil.Encode(input[32:64]))
	if err != nil {
		log.Error("Error parsing transfer: unable to parse value from " + hexutil.Encode(input[32:64]))
		return nil, fmt.Errorf("Error parsing transfer: unable to parse value from " + hexutil.Encode(input[32:64]))
	}

	// Create native token out of thin air
	ctx.evm.StateDB.AddBalance(mintTo, value)

	return input, nil
}

// Native token burn precompile to make bridging back to L1 possible.
type burn struct{}

func (c *burn) RequiredGas(input []byte) uint64 {
	// TODO: determine appropriate gas cost
	return 100
}

// Note ctx.CanTransfer method obtains an incorrect balance w.r.t "burnFrom" address,
// specifically during estimateGas. The CanTransfer check was therefore removed,
// and the calling contract is responsible for checking balance.
func (c *burn) Run(input []byte, ctx *precompileContext) ([]byte, error) {

	if ctx.caller != common.HexToAddress(whitelistCreate2Addr) {
		log.Error("Error parsing transfer: caller not whitelisted")
		return nil, fmt.Errorf("Error parsing transfer: caller not whitelisted")
	}

	burnFrom := common.BytesToAddress(input[0:32])

	value, err := uint256.FromHex(hexutil.Encode(input[32:64]))
	if err != nil {
		log.Error("Error parsing transfer: unable to parse value from " + hexutil.Encode(input[32:64]))
		return nil, fmt.Errorf("Error parsing transfer: unable to parse value from " + hexutil.Encode(input[32:64]))
	}

	ctx.evm.StateDB.SubBalance(burnFrom, value)

	return input, nil
}
