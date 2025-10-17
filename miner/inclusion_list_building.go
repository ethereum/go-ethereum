package miner

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// BuildInclusionListArgs contains the provided parameters for building inclusion list.
type BuildInclusionListArgs struct {
	Parent common.Hash // The hash of the parent block to build the inclusion list upon
}

func (miner *Miner) BuildInclusionList(args *BuildInclusionListArgs) (types.InclusionList, error) {
	genParams := &generateParams{
		timestamp:   uint64(time.Now().Unix()),
		forceTime:   false,
		parentHash:  args.Parent,
		coinbase:    miner.config.PendingFeeRecipient,
		random:      common.Hash{},
		withdrawals: []*types.Withdrawal{},
		beaconRoot:  nil,
		noTxs:       false,
	}
	env, err := miner.prepareWork(genParams, false)
	if err != nil {
		return nil, err
	}

	if err := miner.fillTransactions(nil, env); err != nil {
		return nil, err
	}

	inclusionListTxs := make([]*types.Transaction, 0)
	inclusionListSize := uint64(0)

	for _, tx := range env.txs {
		if inclusionListSize+tx.Size() > params.MaxBytesPerInclusionList {
			continue
		}

		inclusionListTxs = append(inclusionListTxs, tx)
		inclusionListSize += tx.Size()
	}

	return types.TransactionsToInclusionList(inclusionListTxs), nil
}
