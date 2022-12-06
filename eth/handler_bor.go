package eth

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/bor"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	// errCheckpoint is returned when we are unable to fetch the
	// latest checkpoint from the local heimdall.
	errCheckpoint = errors.New("failed to fetch latest checkpoint")

	// errMissingCheckpoint is returned when we don't have the
	// checkpoint blocks locally, yet.
	errMissingCheckpoint = errors.New("missing checkpoint blocks")

	// errRootHash is returned when we aren't able to calculate the root hash
	// locally for a range of blocks.
	errRootHash = errors.New("failed to get local root hash")

	// errCheckpointRootHashMismatch is returned when the local root hash
	// doesn't match with the root hash in checkpoint.
	errCheckpointRootHashMismatch = errors.New("checkpoint roothash mismatch")

	// errEndBlock is returned when we're unable to fetch a block locally.
	errEndBlock = errors.New("failed to get end block")
)

// fetchWhitelistCheckpoint fetched the latest checkpoint from it's local heimdall
// and verifies the data against bor data.
func (h *ethHandler) fetchWhitelistCheckpoint(bor *bor.Bor) (uint64, common.Hash, error) {
	// check for checkpoint whitelisting: bor
	checkpoint, err := bor.HeimdallClient.FetchLatestCheckpoint()
	if err != nil {
		log.Debug("Failed to fetch latest checkpoint for whitelisting")
		return 0, common.Hash{}, errCheckpoint
	}

	// check if we have the checkpoint blocks
	head := h.ethAPI.BlockNumber()
	if head < hexutil.Uint64(checkpoint.EndBlock.Uint64()) {
		log.Debug("Head block behind checkpoint block", "head", head, "checkpoint end block", checkpoint.EndBlock)
		return 0, common.Hash{}, errMissingCheckpoint
	}

	// verify the root hash of checkpoint
	roothash, err := h.ethAPI.GetRootHash(context.Background(), checkpoint.StartBlock.Uint64(), checkpoint.EndBlock.Uint64())
	if err != nil {
		log.Debug("Failed to get root hash of checkpoint while whitelisting")
		return 0, common.Hash{}, errRootHash
	}

	if roothash != checkpoint.RootHash.String()[2:] {
		log.Warn("Checkpoint root hash mismatch while whitelisting", "expected", checkpoint.RootHash.String()[2:], "got", roothash)
		return 0, common.Hash{}, errCheckpointRootHashMismatch
	}

	// fetch the end checkpoint block hash
	block, err := h.ethAPI.GetBlockByNumber(context.Background(), rpc.BlockNumber(checkpoint.EndBlock.Uint64()), false)
	if err != nil {
		log.Debug("Failed to get end block hash of checkpoint while whitelisting")
		return 0, common.Hash{}, errEndBlock
	}

	hash := fmt.Sprintf("%v", block["hash"])

	return checkpoint.EndBlock.Uint64(), common.HexToHash(hash), nil
}
