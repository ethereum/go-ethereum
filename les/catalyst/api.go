// Copyright 2021 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package catalyst implements the temporary eth1/eth2 RPC integration.
package catalyst

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/beacon"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

// Register adds catalyst APIs to the light client.
func Register(stack *node.Node, backend *les.LightEthereum) error {
	log.Warn("Engine API enabled", "protocol", "les")
	stack.RegisterAPIs([]rpc.API{
		{
			Namespace:     "engine",
			Service:       NewConsensusAPI(backend),
			Authenticated: true,
		},
	})
	return nil
}

type ConsensusAPI struct {
	les           *les.LightEthereum
	remoteHeaders *beacon.HeaderQueue
	// Lock for the forkChoiceUpdated method
	forkChoiceLock sync.Mutex
}

// NewConsensusAPI creates a new consensus api for the given backend.
// The underlying blockchain needs to have a valid terminal total difficulty set.
func NewConsensusAPI(les *les.LightEthereum) *ConsensusAPI {
	if les.BlockChain().Config().TerminalTotalDifficulty == nil {
		log.Warn("Engine API started but chain not configured for merge yet")
	}
	return &ConsensusAPI{
		les:           les,
		remoteHeaders: beacon.NewHeaderQueue(),
	}
}

// ForkchoiceUpdatedV1 has several responsibilities:
// If the method is called with an empty head block:
// 		we return success, which can be used to check if the catalyst mode is enabled
// If the total difficulty was not reached:
// 		we return INVALID
// If the finalizedBlockHash is set:
// 		we check if we have the finalizedBlockHash in our db, if not we start a sync
// We try to set our blockchain to the headBlock
// If there are payloadAttributes:
// 		we return an error since LES does not support payload creation.
func (api *ConsensusAPI) ForkchoiceUpdatedV1(update beacon.ForkchoiceStateV1, payloadAttributes *beacon.PayloadAttributesV1) (beacon.ForkChoiceResponse, error) {
	api.forkChoiceLock.Lock()
	defer api.forkChoiceLock.Unlock()

	log.Trace("Engine API request received", "method", "ForkchoiceUpdated", "head", update.HeadBlockHash, "finalized", update.FinalizedBlockHash, "safe", update.SafeBlockHash)
	if update.HeadBlockHash == (common.Hash{}) {
		log.Warn("Forkchoice requested update to zero hash")
		return beacon.STATUS_INVALID, nil // TODO(karalabe): Why does someone send us this?
	}

	// Check whether we have the header yet in our database or not.
	header := api.les.BlockChain().GetHeaderByHash(update.HeadBlockHash)
	if header == nil {
		// If not check whether we stored the header in our queue.
		header = api.remoteHeaders.Get(update.HeadBlockHash)
		if header == nil {
			log.Warn("Forkchoice requested unknown head", "hash", update.HeadBlockHash)
			// Post-Merge sync is not supported by LES yet, return syncing anyway
			return beacon.STATUS_SYNCING, nil
		}
	}
	// Block is known locally, just sanity check that the beacon client does not
	// attempt to push us back to before the merge.
	if header.Difficulty.BitLen() > 0 || header.Number.Uint64() == 0 {
		var (
			td  = api.les.BlockChain().GetTd(update.HeadBlockHash, header.Number.Uint64())
			ptd = api.les.BlockChain().GetTd(header.ParentHash, header.Number.Uint64()-1)
			ttd = api.les.BlockChain().Config().TerminalTotalDifficulty
		)
		if td == nil || (header.Number.Uint64() > 0 && ptd == nil) {
			log.Error("TDs unavailable for TTD check", "number", header.Number.Uint64(), "hash", update.HeadBlockHash, "td", td, "parent", header.ParentHash, "ptd", ptd)
			return beacon.STATUS_INVALID, errors.New("TDs unavailable for TDD check")
		}
		if td.Cmp(ttd) < 0 {
			log.Error("Refusing beacon update to pre-merge", "number", header.Number.Uint64(), "hash", update.HeadBlockHash, "diff", header.Difficulty, "age", common.PrettyAge(time.Unix(int64(header.Time), 0)))
			return beacon.ForkChoiceResponse{PayloadStatus: beacon.INVALID_TERMINAL_BLOCK, PayloadID: nil}, nil
		}
		if header.Number.Uint64() > 0 && ptd.Cmp(ttd) >= 0 {
			log.Error("Parent block is already post-ttd", "number", header.Number.Uint64(), "hash", update.HeadBlockHash, "diff", header.Difficulty, "age", common.PrettyAge(time.Unix(int64(header.Time), 0)))
			return beacon.ForkChoiceResponse{PayloadStatus: beacon.INVALID_TERMINAL_BLOCK, PayloadID: nil}, nil
		}
	}
	valid := func(id *beacon.PayloadID) beacon.ForkChoiceResponse {
		return beacon.ForkChoiceResponse{
			PayloadStatus: beacon.PayloadStatusV1{Status: beacon.VALID, LatestValidHash: &update.HeadBlockHash},
			PayloadID:     id,
		}
	}
	if rawdb.ReadCanonicalHash(api.les.ApiBackend.ChainDb(), header.Number.Uint64()) != update.HeadBlockHash {
		// Block is not canonical, set head.
		if err := api.les.BlockChain().SetCanonical(header); err != nil {
			return beacon.ForkChoiceResponse{PayloadStatus: beacon.PayloadStatusV1{Status: beacon.INVALID, LatestValidHash: nil}}, err
		}
	} else if api.les.BlockChain().CurrentHeader().Hash() == update.HeadBlockHash {
		// If the specified head matches with our local head, do nothing and keep
		// generating the payload. It's a special corner case that a few slots are
		// missing and we are requested to generate the payload in slot.
	} else {
		// If the head block is already in our canonical chain, the beacon client is
		// probably resyncing. Ignore the update.
		log.Info("Ignoring beacon update to old head", "number", header.Number.Uint64(), "hash", update.HeadBlockHash, "age", common.PrettyAge(time.Unix(int64(header.Time), 0)), "have", api.les.BlockChain().CurrentHeader().Number.Uint64())
		return valid(nil), nil
	}

	// If the beacon client also advertised a finalized block, mark the local
	// chain final and completely in PoS mode.
	if update.FinalizedBlockHash != (common.Hash{}) {
		if merger := api.les.Merger(); !merger.PoSFinalized() {
			merger.FinalizePoS()
		}
		// If the finalized block is not in our canonical tree, somethings wrong
		finalHeader := api.les.BlockChain().GetHeaderByHash(update.FinalizedBlockHash)
		if finalHeader == nil {
			log.Warn("Final block not available in database", "hash", update.FinalizedBlockHash)
			return beacon.STATUS_INVALID, beacon.InvalidForkChoiceState.With(errors.New("final block not available in database"))
		} else if rawdb.ReadCanonicalHash(api.les.ApiBackend.ChainDb(), finalHeader.Number.Uint64()) != update.FinalizedBlockHash {
			log.Warn("Final block not in canonical chain", "number", header.Number.Uint64(), "hash", update.HeadBlockHash)
			return beacon.STATUS_INVALID, beacon.InvalidForkChoiceState.With(errors.New("final block not in canonical chain"))
		}
		// Set the finalized block
		// TODO (MariusVanDerWijden): Enable this once Finalized is implemented in LES
		// api.les.BlockChain().SetFinalized(finalBlock)
	}
	// Check if the safe block hash is in our canonical tree, if not something is wrong
	if update.SafeBlockHash != (common.Hash{}) {
		safeHeader := api.les.BlockChain().GetHeaderByHash(update.SafeBlockHash)
		if safeHeader == nil {
			log.Warn("Safe block not available in database")
			return beacon.STATUS_INVALID, beacon.InvalidForkChoiceState.With(errors.New("safe block not available in database"))
		}
		if rawdb.ReadCanonicalHash(api.les.ApiBackend.ChainDb(), safeHeader.Number.Uint64()) != update.SafeBlockHash {
			log.Warn("Safe block not in canonical chain")
			return beacon.STATUS_INVALID, beacon.InvalidForkChoiceState.With(errors.New("safe block not in canonical chain"))
		}
	}
	// Payload generation is not supported in light mode.
	if payloadAttributes != nil {
		log.Error("Block production requested in light mode")
		return beacon.STATUS_INVALID, errors.New("not supported")
	}
	return valid(nil), nil
}

// ExchangeTransitionConfigurationV1 checks the given configuration against
// the configuration of the node.
func (api *ConsensusAPI) ExchangeTransitionConfigurationV1(config beacon.TransitionConfigurationV1) (*beacon.TransitionConfigurationV1, error) {
	log.Trace("Engine API request received", "method", "ExchangeTransitionConfiguration", "ttd", config.TerminalTotalDifficulty)
	if config.TerminalTotalDifficulty == nil {
		return nil, errors.New("invalid terminal total difficulty")
	}
	ttd := api.les.BlockChain().Config().TerminalTotalDifficulty
	if ttd == nil || ttd.Cmp(config.TerminalTotalDifficulty.ToInt()) != 0 {
		log.Warn("Invalid TTD configured", "geth", ttd, "beacon", config.TerminalTotalDifficulty)
		return nil, fmt.Errorf("invalid ttd: execution %v consensus %v", ttd, config.TerminalTotalDifficulty)
	}

	if config.TerminalBlockHash != (common.Hash{}) {
		if hash := api.les.BlockChain().GetCanonicalHash(uint64(config.TerminalBlockNumber)); hash == config.TerminalBlockHash {
			return &beacon.TransitionConfigurationV1{
				TerminalTotalDifficulty: (*hexutil.Big)(ttd),
				TerminalBlockHash:       config.TerminalBlockHash,
				TerminalBlockNumber:     config.TerminalBlockNumber,
			}, nil
		}
		return nil, fmt.Errorf("invalid terminal block hash")
	}
	return &beacon.TransitionConfigurationV1{TerminalTotalDifficulty: (*hexutil.Big)(ttd)}, nil
}

// GetPayloadV1 returns a cached payload by id.
// LES does not allow for payload creation so this calls always fails.
func (api *ConsensusAPI) GetPayloadV1(payloadID beacon.PayloadID) (*beacon.ExecutableDataV1, error) {
	log.Trace("Engine API request received", "method", "GetPayload", "id", payloadID)
	return nil, beacon.GenericServerError.With(errors.New("not supported in light client mode"))
}

// NewPayloadV1 creates an Eth1 block, inserts it in the chain, and returns the status of the chain.
func (api *ConsensusAPI) NewPayloadV1(params beacon.ExecutableDataV1) (beacon.PayloadStatusV1, error) {
	log.Trace("Engine API request received", "method", "ExecutePayload", "number", params.Number, "hash", params.BlockHash)
	block, err := beacon.ExecutableDataToBlock(params)
	if err != nil {
		log.Debug("Invalid NewPayload params", "params", params, "error", err)
		return beacon.PayloadStatusV1{Status: beacon.INVALIDBLOCKHASH}, nil
	}

	// If we already have the header locally, ignore the entire execution and just
	// return a fake success.
	if header := api.les.BlockChain().GetHeaderByHash(params.BlockHash); header != nil {
		log.Warn("Ignoring already known beacon payload", "number", params.Number, "hash", params.BlockHash, "age", common.PrettyAge(time.Unix(int64(header.Time), 0)))
		hash := header.Hash()
		return beacon.PayloadStatusV1{Status: beacon.VALID, LatestValidHash: &hash}, nil
	}
	// If the parent is missing, we - in theory - could trigger a sync, but that
	// would also entail a reorg. That is problematic if multiple sibling blocks
	// are being fed to us, and even more so, if some semi-distant uncle shortens
	// our live chain. As such, payload execution will not permit reorgs and thus
	// will not trigger a sync cycle. That is fine though, if we get a fork choice
	// update after legit payload executions.
	parent := api.les.BlockChain().GetHeader(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		api.remoteHeaders.Put(block.Hash(), block.Header())
		log.Warn("Ignoring payload with missing parent", "number", params.Number, "hash", params.BlockHash, "parent", params.ParentHash)
		return beacon.PayloadStatusV1{Status: beacon.ACCEPTED}, nil
	}
	// We have an existing parent, do some sanity checks to avoid the beacon client
	// triggering too early
	var (
		ptd  = api.les.BlockChain().GetTd(parent.Hash(), parent.Number.Uint64())
		ttd  = api.les.BlockChain().Config().TerminalTotalDifficulty
		gptd = api.les.BlockChain().GetTd(parent.ParentHash, parent.Number.Uint64()-1)
	)
	if ptd.Cmp(ttd) < 0 {
		log.Warn("Ignoring pre-merge payload", "number", params.Number, "hash", params.BlockHash, "td", ptd, "ttd", ttd)
		return beacon.INVALID_TERMINAL_BLOCK, nil
	}
	if parent.Difficulty.BitLen() > 0 && gptd != nil && gptd.Cmp(ttd) >= 0 {
		log.Error("Ignoring pre-merge parent block", "number", params.Number, "hash", params.BlockHash, "td", ptd, "ttd", ttd)
		return beacon.INVALID_TERMINAL_BLOCK, nil
	}
	if block.Time() <= parent.Time {
		log.Warn("Invalid timestamp", "parent", block.Time(), "block", block.Time())
		return api.invalid(errors.New("invalid timestamp"), parent), nil
	}
	log.Trace("Inserting header", "hash", block.Hash(), "number", block.Number)
	if err := api.les.BlockChain().InsertHeader(block.Header()); err != nil {
		log.Warn("NewPayloadV1: inserting header failed", "error", err)
		return api.invalid(err, parent), nil
	}
	// We've accepted a valid payload from the beacon client. Mark the local
	// chain transitions to notify other subsystems (e.g. downloader) of the
	// behavioral change.
	if merger := api.les.Merger(); !merger.TDDReached() {
		merger.ReachTTD()
		api.les.Downloader().Cancel()
	}
	hash := block.Hash()
	return beacon.PayloadStatusV1{Status: beacon.VALID, LatestValidHash: &hash}, nil
}

// invalid returns a response "INVALID" with the latest valid hash supplied by latest or to the current head
// if no latestValid block was provided.
func (api *ConsensusAPI) invalid(err error, latestValid *types.Header) beacon.PayloadStatusV1 {
	currentHash := api.les.BlockChain().CurrentHeader().Hash()
	if latestValid != nil {
		// Set latest valid hash to 0x0 if parent is PoW block
		currentHash = common.Hash{}
		if latestValid.Difficulty.BitLen() == 0 {
			// Otherwise set latest valid hash to parent hash
			currentHash = latestValid.Hash()
		}
	}
	errorMsg := err.Error()
	return beacon.PayloadStatusV1{Status: beacon.INVALID, LatestValidHash: &currentHash, ValidationError: &errorMsg}
}
