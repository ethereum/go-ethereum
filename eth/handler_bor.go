package eth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/bor"
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall"
	"github.com/ethereum/go-ethereum/log"
)

var (
	// errCheckpoint is returned when we are unable to fetch the
	// latest checkpoint from the local heimdall.
	errCheckpoint = errors.New("failed to fetch latest checkpoint")

	// errMilestone is returned when we are unable to fetch the
	// latest milestone from the local heimdall.
	errMilestone = errors.New("failed to fetch latest milestone")
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
	err = reportCommonErrors("latest checkpoint", err, errCheckpoint)
	if err != nil {
		return blockNum, blockHash, err
	}

	log.Debug("Got new checkpoint from heimdall", "start", checkpoint.StartBlock.Uint64(), "end", checkpoint.EndBlock.Uint64(), "rootHash", checkpoint.RootHash.String())

	// Verify if the checkpoint fetched can be added to the local whitelist entry or not
	// If verified, it returns the hash of the end block of the checkpoint. If not,
	// it will return appropriate error.
	hash, err := verifier.verify(ctx, eth, h, checkpoint.StartBlock.Uint64(), checkpoint.EndBlock.Uint64(), checkpoint.RootHash.String()[2:], true)
	if err != nil {
		if errors.Is(err, errChainOutOfSync) {
			log.Info("Whitelisting checkpoint deferred", "err", err)
		} else {
			log.Warn("Failed to whitelist checkpoint", "err", err)
		}
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
	err = reportCommonErrors("latest milestone", err, errMilestone)
	if err != nil {
		return num, hash, err
	}

	num = milestone.EndBlock.Uint64()
	hash = milestone.Hash

	log.Debug("Got new milestone from heimdall", "start", milestone.StartBlock.Uint64(), "end", milestone.EndBlock.Uint64(), "hash", milestone.Hash.String())

	// Verify if the milestone fetched can be added to the local whitelist entry or not. If verified,
	// the hash of the end block of the milestone is returned else appropriate error is returned.
	_, err = verifier.verify(ctx, eth, h, milestone.StartBlock.Uint64(), milestone.EndBlock.Uint64(), milestone.Hash.String()[2:], false)
	if err != nil {
		if errors.Is(err, errChainOutOfSync) {
			log.Info("Whitelisting milestone deferred", "err", err)
		} else {
			log.Warn("Failed to whitelist milestone", "err", err)
		}
		h.downloader.UnlockSprint(milestone.EndBlock.Uint64())
	}

	return num, hash, err
}

func (h *ethHandler) fetchNoAckMilestone(ctx context.Context, bor *bor.Bor) (string, error) {
	milestoneID, err := bor.HeimdallClient.FetchLastNoAckMilestone(ctx)
	err = reportCommonErrors("latest no-ack milestone", err, nil)

	return milestoneID, err
}

func (h *ethHandler) fetchNoAckMilestoneByID(ctx context.Context, bor *bor.Bor, milestoneID string) error {
	err := bor.HeimdallClient.FetchNoAckMilestone(ctx, milestoneID)
	if errors.Is(err, heimdall.ErrNotInRejectedList) {
		log.Debug("MilestoneID not in rejected list", "milestoneID", milestoneID)
	}
	err = reportCommonErrors("no-ack milestone by ID", err, nil, "milestoneID", milestoneID)
	return err
}

// reportCommonErrors reports common errors which can occur while fetching data from heimdall. It also
// returns back the wrapped erorr if required to the caller.
func reportCommonErrors(msg string, err error, wrapError error, ctx ...interface{}) error {
	if err == nil {
		return err
	}

	// We're skipping extra check to the `heimdall.ErrServiceUnavailable` error as it should not
	// occur post HF (in heimdall). If it does, we'll anyways warn below as a normal error.

	ctx = append(ctx, "err", err)

	if strings.Contains(err.Error(), "context deadline exceeded") {
		log.Warn(fmt.Sprintf("Failed to fetch %s, please check the heimdall endpoint and status of your heimdall node", msg), ctx...)
	} else {
		log.Warn(fmt.Sprintf("Failed to fetch %s", msg), ctx...)
	}

	if wrapError != nil {
		return fmt.Errorf("%w: %v", wrapError, err)
	}

	return err
}
