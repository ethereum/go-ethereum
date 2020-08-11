package parlia

import (
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	wiggleTimeBeforeFork       = 500 * time.Millisecond // Random delay (per signer) to allow concurrent signers
	fixedBackOffTimeBeforeFork = 200 * time.Millisecond
)

func (p *Parlia) delayForRamanujanFork(snap *Snapshot, header *types.Header) time.Duration {
	delay := time.Until(time.Unix(int64(header.Time), 0)) // nolint: gosimple
	if p.chainConfig.IsRamanujan(header.Number) {
		return delay
	}
	if header.Difficulty.Cmp(diffNoTurn) == 0 {
		// It's not our turn explicitly to sign, delay it a bit
		wiggle := time.Duration(len(snap.Validators)/2+1) * wiggleTimeBeforeFork
		delay += time.Duration(fixedBackOffTimeBeforeFork) + time.Duration(rand.Int63n(int64(wiggle)))
	}
	return delay
}

func (p *Parlia) blockTimeForRamanujanFork(snap *Snapshot, header, parent *types.Header) uint64 {
	blockTime := parent.Time + p.config.Period
	if p.chainConfig.IsRamanujan(header.Number) {
		blockTime = blockTime + backOffTime(snap, p.val)
	}
	return blockTime
}

func (p *Parlia) blockTimeVerifyForRamanujanFork(snap *Snapshot, header, parent *types.Header) error {
	if p.chainConfig.IsRamanujan(header.Number) {
		if header.Time < parent.Time+p.config.Period+backOffTime(snap, header.Coinbase) {
			return consensus.ErrFutureBlock
		}
	}
	return nil
}
