package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// IL constants taken from specs here: https://github.com/potuz/consensus-specs/blob/a6c55576de059a1b2cae69848dee827f6e26e72d/specs/_features/epbs/beacon-chain.md#execution
const (
	MAX_TRANSACTIONS_PER_INCLUSION_LIST = 16
	MAX_GAS_PER_INCLUSION_LIST          = 2_097_152 // 2^21
)

// VerifyInclusionList verifies the properties of the inclusion list and the
// transactions in it against the given `state` object.
//
// The verification involves actual execution of the transactions so
// it's the caller's responsibility to send a copy of the state object.
func verifyInclusionList(list types.InclusionList, parent *types.Header, state *state.StateDB, config *params.ChainConfig) bool {
	// Validate few basic things first in the inclusion list.
	if len(list.Summary) != len(list.Transactions) {
		log.Debug("Inclusion list summary and transactions length mismatch")
		return false
	}

	if len(list.Summary) > MAX_TRANSACTIONS_PER_INCLUSION_LIST {
		log.Debug("Inclusion list exceeds maximum number of transactions")
		return false
	}

	// As IL will be included in the next block, calculate the current block's base fee.
	// As the current block's payload isn't revealed yet (due to ePBS), calculate
	// it from parent block.
	currentBaseFee := eip1559.CalcBaseFee(config, parent)

	// 1.125 * currentBaseFee
	gasFeeThreshold := new(big.Float).Mul(new(big.Float).SetFloat64(0.125), new(big.Float).SetInt(currentBaseFee))

	// Prepare the signer object
	signer := types.LatestSigner(config)

	// Verify if the summary and transactions match. Also check if the txs
	// have at least 12.5% higher `maxFeePerGas` than parent block's base fee.
	for i, summary := range list.Summary {
		tx := list.Transactions[i]
		from, err := types.Sender(signer, tx)
		if err != nil {
			log.Debug("Failed to get sender from transaction", "err", err)
			return false
		}
		if summary.Address != from {
			log.Debug("Inclusion list summary and transaction address mismatch")
			return false
		}

		// tx.GasFeeCap > 1.125 * parent.BaseFee
		if new(big.Float).SetInt(tx.GasFeeCap()).Cmp(gasFeeThreshold) < 1 {
			return false
		}
	}

	// TODO: Execute txs. Mostly all required params are available except evm context.

	return false
}
