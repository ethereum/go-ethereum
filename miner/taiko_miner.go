package miner

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/beacon"
	"github.com/ethereum/go-ethereum/core/types"
)

// SealBlockWith mines and seals a block without changing the canonical chain.
func (miner *Miner) SealBlockWith(
	parent common.Hash,
	timestamp uint64,
	blkMeta *beacon.BlockMetadata,
) (*types.Block, error) {
	return miner.worker.sealBlockWith(parent, timestamp, blkMeta)
}
