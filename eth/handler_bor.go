package eth

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/bor"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

// fetchWhitelistCheckpoint fetched the latest checkpoint from it's local heimdall
// and verifies the data against bor data.
func (h *ethHandler) fetchWhitelistCheckpoint() (uint64, common.Hash, error) {
	// check for checkpoint whitelisting: bor
	checkpoint, err := h.chain.Engine().(*bor.Bor).HeimdallClient.FetchLatestCheckpoint()
	if err != nil {
		log.Debug("Failed to fetch latest checkpoint for whitelisting")
		return 0, common.Hash{}, fmt.Errorf("failed to fetch latest checkpoint")
	}

	// check if we have the checkpoint blocks
	head := h.ethAPI.BlockNumber()
	if head < hexutil.Uint64(checkpoint.EndBlock.Uint64()) {
		log.Debug("Head block behing checkpoint block", "head", head, "checkpoint end block", checkpoint.EndBlock)
		return 0, common.Hash{}, fmt.Errorf("missing checkpoint blocks")
	}

	// verify the root hash of checkpoint
	roothash, err := h.ethAPI.GetRootHash(context.Background(), checkpoint.StartBlock.Uint64(), checkpoint.EndBlock.Uint64())
	if err != nil {
		log.Debug("Failed to get root hash of checkpoint while whitelisting")
		return 0, common.Hash{}, fmt.Errorf("failed to get local root hash")
	}
	if roothash != checkpoint.RootHash.String()[2:] {
		log.Warn("Checkpoint root hash mismatch while whitelisting", "expected", checkpoint.RootHash.String()[2:], "got", roothash)
		return 0, common.Hash{}, fmt.Errorf("checkpoint roothash mismatch")
	}

	// fetch the end checkpoint block hash
	block, err := h.ethAPI.GetBlockByNumber(context.Background(), rpc.BlockNumber(checkpoint.EndBlock.Uint64()), false)
	if err != nil {
		log.Debug("Failed to get end block hash of checkpoint while whitelisting")
		return 0, common.Hash{}, fmt.Errorf("failed to get end block")
	}
	hash := fmt.Sprintf("%v", block["hash"])
	return checkpoint.EndBlock.Uint64(), common.HexToHash(hash), nil
}
