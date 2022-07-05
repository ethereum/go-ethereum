package ethapi

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type MulticallRunlist map[common.Address][]hexutil.Bytes

type MulticallResult map[common.Address][]*MulticallExecutionResult

type MulticallExecutionResult struct {
	UsedGas    uint64
	ReturnData hexutil.Bytes `json:",omitempty"`
	Err        string        `json:",omitempty"`
}

func (s *BlockChainAPI) Multicall(ctx context.Context, commonCallArgs TransactionArgs, contractsWithPayloads MulticallRunlist, blockNrOrHash rpc.BlockNumberOrHash) (MulticallResult, error) {
	start := time.Now()

	// result stores
	execResults := make(MulticallResult)

	// get state once from the API client
	state, header, err := s.b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}

	globalGasCap := s.b.RPCGasCap()
	// add max gas once for the EVM contract call (this should suffice for the entire batch call)
	gp := new(core.GasPool).AddGas(math.MaxUint64)

	callArgsBuf := commonCallArgs
	callArgsBuf.MaxFeePerGas = (*hexutil.Big)(header.BaseFee)

	for contractAddr, payloads := range contractsWithPayloads {
		callArgsBuf.To = &contractAddr // nolint: exportloopref

		execResultsForContract := make([]*MulticallExecutionResult, 0, len(payloads))

		for _, payload := range payloads {
			// assign the correct values to args
			callArgsBuf.Input = &payload // nolint: exportloopref,gosec

			// get a new Message to be used once
			msg, err := callArgsBuf.ToMessage(globalGasCap, header.BaseFee)
			if err != nil {
				return nil, err
			}

			// get a new instance of the EVM to be used once
			// ethapi's vmError callback always returns nil, so it is dropped here
			evm, _, getVmErr := s.b.GetEVM(ctx, msg, state, header, nil)

			if getVmErr != nil {
				// if we cannot retrieve the an EVM for any message, that failure
				// implies a fault in the node as a whole, so we should give up on
				// processing the entire request
				return nil, getVmErr
			}

			execResult, applyMsgErr := core.ApplyMessage(evm, msg, gp)

			var effectiveErrDesc string
			if applyMsgErr != nil {
				effectiveErrDesc = applyMsgErr.Error()
			} else if execResult.Err != nil {
				effectiveErrDesc = execResult.Err.Error()
			}

			mcExecResult := &MulticallExecutionResult{
				UsedGas: execResult.UsedGas,
				Err:     effectiveErrDesc,
			}

			if len(execResult.ReturnData) > 0 {
				mcExecResult.ReturnData = execResult.ReturnData
			}

			execResultsForContract = append(execResultsForContract, mcExecResult)
		}

		execResults[contractAddr] = execResultsForContract
	}

	log.Info("Executing EVM multicall finished", "runtime μs", time.Since(start).Microseconds())

	return execResults, nil
}
