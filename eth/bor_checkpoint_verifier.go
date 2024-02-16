// nolint
package eth

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	// errMissingBlocks is returned when we don't have the blocks locally, yet.
	errMissingBlocks = errors.New("missing blocks")

	// errRootHash is returned when we aren't able to calculate the root hash
	// locally for a range of blocks.
	errRootHash = errors.New("failed to get local root hash")

	// errHashMismatch is returned when the local hash doesn't match
	// with the hash of checkpoint/milestone. It is the root hash of blocks
	// in case of checkpoint and is end block hash in case of milestones.
	errHashMismatch = errors.New("hash mismatch")

	// errEndBlock is returned when we're unable to fetch a block locally.
	errEndBlock = errors.New("failed to get end block")

	// errEndBlock is returned when we're unable to fetch a block locally.
	errTipConfirmationBlock = errors.New("failed to get tip confirmation block")

	// errBlockNumberConversion is returned when we get err in parsing hexautil block number
	errBlockNumberConversion = errors.New("failed to parse the block number")

	//Metrics for collecting the rewindLength
	rewindLengthMeter = metrics.NewRegisteredMeter("chain/autorewind/length", nil)
)

const maxRewindLen uint64 = 126

type borVerifier struct {
	verify func(ctx context.Context, eth *Ethereum, handler *ethHandler, start uint64, end uint64, hash string, isCheckpoint bool) (string, error)
}

func newBorVerifier() *borVerifier {
	return &borVerifier{borVerify}
}

func borVerify(ctx context.Context, eth *Ethereum, handler *ethHandler, start uint64, end uint64, hash string, isCheckpoint bool) (string, error) {
	str := "milestone"
	if isCheckpoint {
		str = "checkpoint"
	}

	// check if we have the given blocks
	currentBlock := eth.BlockChain().CurrentBlock()
	if currentBlock == nil {
		log.Debug(fmt.Sprintf("Failed to fetch current block from blockchain while verifying incoming %s", str))
		return hash, errMissingBlocks
	}

	head := currentBlock.Number.Uint64()

	if head < end {
		log.Debug(fmt.Sprintf("Current head block behind incoming %s block", str), "head", head, "end block", end)
		return hash, errMissingBlocks
	}

	var localHash string

	// verify the hash
	if isCheckpoint {
		var err error

		// in case of checkpoint get the rootHash
		localHash, err = handler.ethAPI.GetRootHash(ctx, start, end)

		if err != nil {
			log.Debug("Failed to get root hash of given block range while whitelisting checkpoint", "start", start, "end", end, "err", err)
			return hash, errRootHash
		}
	} else {
		// in case of milestone(isCheckpoint==false) get the hash of endBlock
		block, err := handler.ethAPI.GetBlockByNumber(ctx, rpc.BlockNumber(end), false)
		if err != nil {
			log.Debug("Failed to get end block hash while whitelisting milestone", "number", end, "err", err)
			return hash, errEndBlock
		}

		localHash = fmt.Sprintf("%v", block["hash"])[2:]
	}

	//nolint
	if localHash != hash {

		if isCheckpoint {
			log.Warn("Root hash mismatch while whitelisting checkpoint", "expected", localHash, "got", hash)
		} else {
			log.Warn("End block hash mismatch while whitelisting milestone", "expected", localHash, "got", hash)
		}

		ethHandler := (*ethHandler)(eth.handler)

		var (
			rewindTo uint64
			doExist  bool
		)

		if doExist, rewindTo, _ = ethHandler.downloader.GetWhitelistedMilestone(); doExist {

		} else if doExist, rewindTo, _ = ethHandler.downloader.GetWhitelistedCheckpoint(); doExist {

		} else {
			if start <= 0 {
				rewindTo = 0
			} else {
				rewindTo = start - 1
			}
		}

		if head-rewindTo > maxRewindLen {
			rewindTo = head - maxRewindLen
		}

		if isCheckpoint {
			log.Warn("Rewinding chain due to checkpoint root hash mismatch", "number", rewindTo)
		} else {
			log.Warn("Rewinding chain due to milestone endblock hash mismatch", "number", rewindTo)
		}

		rewindBack(eth, head, rewindTo)

		return hash, errHashMismatch
	}

	// fetch the end block hash
	block, err := handler.ethAPI.GetBlockByNumber(ctx, rpc.BlockNumber(end), false)
	if err != nil {
		log.Debug("Failed to get end block hash while whitelisting", "err", err)
		return hash, errEndBlock
	}

	hash = fmt.Sprintf("%v", block["hash"])

	return hash, nil
}

// Stop the miner if the mining process is running and rewind back the chain
func rewindBack(eth *Ethereum, head uint64, rewindTo uint64) {
	if eth.Miner().Mining() {
		ch := make(chan struct{})
		eth.Miner().Stop(ch)

		<-ch
		rewind(eth, head, rewindTo)

		eth.Miner().Start()
	} else {
		rewind(eth, head, rewindTo)
	}
}

func rewind(eth *Ethereum, head uint64, rewindTo uint64) {
	err := eth.blockchain.SetHead(rewindTo)

	if err != nil {
		log.Error("Error while rewinding the chain", "to", rewindTo, "err", err)
	} else {
		rewindLengthMeter.Mark(int64(head - rewindTo))
	}

}
