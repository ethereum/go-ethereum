package statefull

import (
	"bytes"
	"context"
	"github.com/ethereum/go-ethereum/core/tracing"
	"math"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

var systemAddress = common.HexToAddress("0xffffFFFfFFffffffffffffffFfFFFfffFFFfFFfE")

type ChainContext struct {
	Chain consensus.ChainHeaderReader
	Bor   consensus.Engine
}

func (c ChainContext) Engine() consensus.Engine {
	return c.Bor
}

func (c ChainContext) GetHeader(hash common.Hash, number uint64) *types.Header {
	return c.Chain.GetHeader(hash, number)
}

// callmsg implements core.Message to allow passing it as a transaction simulator.
type Callmsg struct {
	ethereum.CallMsg
}

func (m Callmsg) From() common.Address { return m.CallMsg.From }
func (m Callmsg) Nonce() uint64        { return 0 }
func (m Callmsg) CheckNonce() bool     { return false }
func (m Callmsg) To() *common.Address  { return m.CallMsg.To }
func (m Callmsg) GasPrice() *big.Int   { return m.CallMsg.GasPrice }
func (m Callmsg) Gas() uint64          { return m.CallMsg.Gas }
func (m Callmsg) Value() *big.Int      { return m.CallMsg.Value }
func (m Callmsg) Data() []byte         { return m.CallMsg.Data }

// get system message
func GetSystemMessage(toAddress common.Address, data []byte) Callmsg {
	return Callmsg{
		ethereum.CallMsg{
			From:     systemAddress,
			Gas:      math.MaxUint64 / 2,
			GasPrice: big.NewInt(0),
			Value:    big.NewInt(0),
			To:       &toAddress,
			Data:     data,
		},
	}
}

// apply message
func ApplyMessage(
	_ context.Context,
	msg Callmsg,
	state *state.StateDB,
	header *types.Header,
	chainConfig *params.ChainConfig,
	chainContext core.ChainContext,
	spanID int64,
	tracer *tracing.Hooks,
) (uint64, error) {

	initialGas := msg.Gas()

	blockContext := core.NewEVMBlockContext(header, chainContext, &header.Coinbase)

	// Create a new environment which holds all relevant information
	// about the transaction and calling mechanisms.
	msgForCtx := core.Message{
		To:               msg.To(),
		From:             msg.From(),
		Nonce:            msg.Nonce(),
		Value:            msg.Value(),
		GasLimit:         msg.Gas(),
		GasPrice:         msg.GasPrice(),
		SkipNonceChecks:  false,
		SkipFromEOACheck: false,
	}
	txContext := core.NewEVMTxContext(&msgForCtx)
	vmenv := vm.NewEVM(blockContext, txContext, state, chainConfig, vm.Config{Tracer: tracer})

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    msg.Nonce(),
		GasPrice: msg.GasPrice(),
		Gas:      msg.Gas(),
		To:       msg.To(),
		Value:    msg.Value(),
		Data:     msg.Data(),
	})
	state.SetTxContext(tx.Hash(), 0)

	// Notify tracers about transaction start (system call is already started at Bor level)
	if tracer != nil {
		if tracer.OnTxStart != nil {
			tracer.OnTxStart(vmenv.GetVMContext(), tx, msg.From())
		}
	}

	defer func() {
		if tracer != nil && tracer.OnTxEnd != nil {
			tracer.OnTxEnd(nil, nil)
		}
	}()

	// nolint : contextcheck
	// Apply the transaction to the current state (included in the env)
	ret, gasLeft, err := vmenv.Call(
		vm.AccountRef(msg.From()),
		*msg.To(),
		msg.Data(),
		msg.Gas(),
		uint256.NewInt(msg.Value().Uint64()),
		nil,
	)

	success := big.NewInt(5).SetBytes(ret)

	validatorContract := common.HexToAddress(chainConfig.Bor.ValidatorContract)

	// if success == 0 and msg.To() != validatorContractAddress, log Error
	// if msg.To() == validatorContractAddress, its committing a span and we don't get any return value
	if success.Cmp(big.NewInt(0)) == 0 && !bytes.Equal(msg.To().Bytes(), validatorContract.Bytes()) {
		log.Error("message execution failed on contract", "msgData", msg.Data)
	}

	// If there's error committing span, log it here. It won't be reported before because the return value is empty.
	if bytes.Equal(msg.To().Bytes(), validatorContract.Bytes()) && err != nil {
		log.Error("message execution failed on contract", "err", err)
	}

	// Update the state with pending changes
	if err != nil {
		state.Finalise(true)
	}

	gasUsed := initialGas - gasLeft

	return gasUsed, nil
}

func ApplyBorMessage(vmenv *vm.EVM, msg Callmsg) (*core.ExecutionResult, error) {
	initialGas := msg.Gas()

	// Apply the transaction to the current state (included in the env)
	ret, gasLeft, err := vmenv.Call(
		vm.AccountRef(msg.From()),
		*msg.To(),
		msg.Data(),
		msg.Gas(),
		uint256.NewInt(msg.Value().Uint64()),
		nil,
	)
	// Update the state with pending changes
	if err != nil {
		vmenv.StateDB.Finalise(true)
	}

	gasUsed := initialGas - gasLeft

	return &core.ExecutionResult{
		UsedGas:    gasUsed,
		Err:        err,
		ReturnData: ret,
	}, nil
}
