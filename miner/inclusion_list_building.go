package miner

import (
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// BuildInclusionListArgs contains the provided parameters for building inclusion list.
type BuildInclusionListArgs struct {
	Parent common.Hash // The parent block to build inclusion list on top
}

func (miner *Miner) BuildInclusionList(args *BuildInclusionListArgs) (engine.InclusionList, error) {
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
		// FOCIL currently doesn't support blob transactions in inclusion list.
		if tx.Type() == types.BlobTxType {
			continue
		}

		if inclusionListSize+tx.Size() > params.MaxBytesPerInclusionList {
			continue
		}

		inclusionListTxs = append(inclusionListTxs, tx)
		inclusionListSize += tx.Size()
	}

	return engine.TransactionsToInclusionList(inclusionListTxs), nil
}
