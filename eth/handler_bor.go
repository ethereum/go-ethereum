package eth

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/bor"
	"github.com/ethereum/go-ethereum/log"
)

var (
	// errCheckpoint is returned when we are unable to fetch the
	// latest checkpoint from the local heimdall.
	errCheckpoint = errors.New("failed to fetch latest checkpoint")

	// errMilestone is returned when we are unable to fetch the
	// latest milestone from the local heimdall.
	errMilestone = errors.New("failed to fetch latest milestone")

	ErrNotInRejectedList = errors.New("MilestoneID not in rejected list")
)

// fetchWhitelistCheckpoint fetches the latest checkpoint from it's local heimdall
// and verifies the data against bor data.
func (h *ethHandler) fetchWhitelistCheckpoint(ctx context.Context, bor *bor.Bor, eth *Ethereum, verifier *borVerifier) (uint64, common.Hash, error) {
	var (
		blockNum  uint64
		blockHash common.Hash
	)

	// fetch the latest checkpoint from Heimdall
	checkpoint, err := bor.HeimdallClient.FetchCheckpoint(ctx, -1)
	if err != nil {
		log.Debug("Failed to fetch latest checkpoint for whitelisting", "err", err)
		return blockNum, blockHash, errCheckpoint
	}

	log.Info("Got new checkpoint from heimdall", "start", checkpoint.StartBlock.Uint64(), "end", checkpoint.EndBlock.Uint64(), "rootHash", checkpoint.RootHash.String())

	// Verify if the checkpoint fetched can be added to the local whitelist entry or not
	// If verified, it returns the hash of the end block of the checkpoint. If not,
	// it will return appropriate error.
	hash, err := verifier.verify(ctx, eth, h, checkpoint.StartBlock.Uint64(), checkpoint.EndBlock.Uint64(), checkpoint.RootHash.String()[2:], true)
	if err != nil {
		log.Warn("Failed to whitelist checkpoint", "err", err)
		return blockNum, blockHash, err
	}

	blockNum = checkpoint.EndBlock.Uint64()
	blockHash = common.HexToHash(hash)

	return blockNum, blockHash, nil
}

// fetchWhitelistMilestone fetches the latest milestone from it's local heimdall
// and verifies the data against bor data.
func (h *ethHandler) fetchWhitelistMilestone(ctx context.Context, bor *bor.Bor, eth *Ethereum, verifier *borVerifier) (uint64, common.Hash, error) {
	var (
		num  uint64
		hash common.Hash
	)

	// fetch latest milestone
	milestone, err := bor.HeimdallClient.FetchMilestone(ctx)
	if err != nil {
		log.Error("Failed to fetch latest milestone for whitelisting", "err", err)
		return num, hash, errMilestone
	}

	num = milestone.EndBlock.Uint64()
	hash = milestone.Hash

	log.Info("Got new milestone from heimdall", "start", milestone.StartBlock.Uint64(), "end", milestone.EndBlock.Uint64(), "hash", milestone.Hash.String())

	// Verify if the milestone fetched can be added to the local whitelist entry or not
	// If verified, it returns the hash of the end block of the milestone. If not,
	// it will return appropriate error.
	_, err = verifier.verify(ctx, eth, h, milestone.StartBlock.Uint64(), milestone.EndBlock.Uint64(), milestone.Hash.String()[2:], false)
	if err != nil {
		h.downloader.UnlockSprint(milestone.EndBlock.Uint64())
		return num, hash, err
	}

	return num, hash, nil
}

func (h *ethHandler) fetchNoAckMilestone(ctx context.Context, bor *bor.Bor) (string, error) {
	var (
		milestoneID string
	)

	// fetch latest milestone
	milestoneID, err := bor.HeimdallClient.FetchLastNoAckMilestone(ctx)
	if err != nil {
		log.Error("Failed to fetch latest no-ack milestone", "err", err)

		return milestoneID, errMilestone
	}

	return milestoneID, nil
}

func (h *ethHandler) fetchNoAckMilestoneByID(ctx context.Context, bor *bor.Bor, milestoneID string) error {
	// fetch latest milestone
	err := bor.HeimdallClient.FetchNoAckMilestone(ctx, milestoneID)

	// fixme: handle different types of errors
	if errors.Is(err, ErrNotInRejectedList) {
		log.Warn("MilestoneID not in rejected list", "milestoneID", milestoneID, "err", err)
		return err
	}

	if err != nil {
		log.Error("Failed to fetch no-ack milestone by ID ", "milestoneID", milestoneID, "err", err)

		return errMilestone
	}

	return nil
}
