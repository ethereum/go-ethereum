package eth

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/checkpoint"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type checkpointVerifier struct {
	verify func(ctx context.Context, handler *ethHandler, checkpoint *checkpoint.Checkpoint) (string, error)
}

func newCheckpointVerifier(verifyFn func(ctx context.Context, handler *ethHandler, checkpoint *checkpoint.Checkpoint) (string, error)) *checkpointVerifier {
	if verifyFn != nil {
		return &checkpointVerifier{verifyFn}
	}

	verifyFn = func(ctx context.Context, handler *ethHandler, checkpoint *checkpoint.Checkpoint) (string, error) {
		var (
			startBlock = checkpoint.StartBlock.Uint64()
			endBlock   = checkpoint.EndBlock.Uint64()
		)

		// check if we have the checkpoint blocks
		//nolint:contextcheck
		head := handler.ethAPI.BlockNumber()
		if head < hexutil.Uint64(endBlock) {
			log.Debug("Head block behind checkpoint block", "head", head, "checkpoint end block", endBlock)
			return "", errMissingCheckpoint
		}

		// verify the root hash of checkpoint
		roothash, err := handler.ethAPI.GetRootHash(ctx, startBlock, endBlock)
		if err != nil {
			log.Debug("Failed to get root hash of checkpoint while whitelisting", "err", err)
			return "", errRootHash
		}

		if roothash != checkpoint.RootHash.String()[2:] {
			log.Warn("Checkpoint root hash mismatch while whitelisting", "expected", checkpoint.RootHash.String()[2:], "got", roothash)
			return "", errCheckpointRootHashMismatch
		}

		// fetch the end checkpoint block hash
		block, err := handler.ethAPI.GetBlockByNumber(ctx, rpc.BlockNumber(endBlock), false)
		if err != nil {
			log.Debug("Failed to get end block hash of checkpoint while whitelisting", "err", err)
			return "", errEndBlock
		}

		hash := fmt.Sprintf("%v", block["hash"])

		return hash, nil
	}

	return &checkpointVerifier{verifyFn}
}
