package miner

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// BuildInclusionListArgs contains the provided parameters for building inclusion list.
type BuildInclusionListArgs struct {
	Parent common.Hash // The hash of the parent block to build the inclusion list upon
}

func (miner *Miner) BuildInclusionList(args *BuildInclusionListArgs) (engine.InclusionList, error) {
	miner.confMu.RLock()
	tip := miner.config.GasPrice
	miner.confMu.RUnlock()

	// Get the parent block upon which the inclusion list will be built
	block := miner.chain.GetBlockByHash(args.Parent)
	if block == nil {
		return nil, errors.New("missing parent")
	}
	parent := block.Header()

	number := new(big.Int).Add(parent.Number, common.Big1)
	var baseFee *big.Int

	// Set baseFee if we are on an EIP-1559 chain
	if miner.chainConfig.IsLondon(number) {
		baseFee = eip1559.CalcBaseFee(miner.chainConfig, parent)
	}

	// Retrieve the pending transactions pre-filtered by the 1559 dynamic fees
	filter := txpool.PendingFilter{
		MinTip: uint256.MustFromBig(tip),
	}
	if baseFee != nil {
		filter.BaseFee = uint256.MustFromBig(baseFee)
	}
	filter.OnlyPlainTxs, filter.OnlyBlobTxs = true, false
	plainTxs := miner.txpool.Pending(filter)

	// Build the inclusion list
	inclusionListTxs := make([]*types.Transaction, 0)
	inclusionListSize := uint64(0)

	for _, txs := range plainTxs {
		for _, tx := range txs {
			tx := tx.Resolve()

			// EIP-7805 doesn't support blob transactions in inclusion list
			if tx.Type() == types.BlobTxType {
				continue
			}

			if inclusionListSize+tx.Size() > params.MaxBytesPerInclusionList {
				continue
			}

			inclusionListTxs = append(inclusionListTxs, tx)
			inclusionListSize += tx.Size()
		}
	}

	return engine.TransactionsToInclusionList(inclusionListTxs), nil
}
