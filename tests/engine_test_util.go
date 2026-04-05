// Copyright 2025 The go-ethereum Authors
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

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	stdmath "math"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/params/forks"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/hashdb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
)

// EngineTest checks processing of engine API payloads.
type EngineTest struct {
	json                etJSON
	LastPayloadStatus   string // set during Run, exposed for the runner
	LastValidationError string // actual validation error from engine
}

func (t *EngineTest) UnmarshalJSON(in []byte) error {
	return json.Unmarshal(in, &t.json)
}

// Network returns the network/fork name for this test.
func (t *EngineTest) Network() string {
	return t.json.Network
}

type etJSON struct {
	Genesis   btHeader               `json:"genesisBlockHeader"`
	Pre       types.GenesisAlloc     `json:"pre"`
	Post      types.GenesisAlloc     `json:"postState"`
	PostHash  *common.UnprefixedHash `json:"postStateHash"`
	BestBlock common.UnprefixedHash  `json:"lastblockhash"`
	Network   string                 `json:"network"`
	Payloads  []etNewPayload         `json:"engineNewPayloads"`
}

// etNewPayload represents a single engine API new payload call from the fixture.
type etNewPayload struct {
	ExecutionPayload engine.ExecutableData
	VersionedHashes  []common.Hash
	BeaconRoot       *common.Hash
	Requests         [][]byte

	Version         int    // newPayloadVersion
	FcuVersion      int    // forkchoiceUpdatedVersion
	ValidationError string // expected validation error (empty = expect VALID)
	ErrorCode       *int   // expected JSON-RPC error code
}

func (p *etNewPayload) UnmarshalJSON(data []byte) error {
	var raw struct {
		Params                   []json.RawMessage `json:"params"`
		NewPayloadVersion        string            `json:"newPayloadVersion"`
		ForkchoiceUpdatedVersion string            `json:"forkchoiceUpdatedVersion"`
		ValidationError          string            `json:"validationError,omitempty"`
		ErrorCode                json.RawMessage   `json:"errorCode,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	p.ValidationError = raw.ValidationError
	// errorCode can be a string ("-32602") or int (-32602) in fixtures
	if len(raw.ErrorCode) > 0 && string(raw.ErrorCode) != "null" {
		s := string(raw.ErrorCode)
		// Strip quotes if it's a JSON string
		if len(s) >= 2 && s[0] == '"' {
			s = s[1 : len(s)-1]
		}
		code, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("invalid errorCode %s: %v", raw.ErrorCode, err)
		}
		p.ErrorCode = &code
	}

	var err error
	p.Version, err = strconv.Atoi(raw.NewPayloadVersion)
	if err != nil {
		return fmt.Errorf("invalid newPayloadVersion: %v", err)
	}
	p.FcuVersion, err = strconv.Atoi(raw.ForkchoiceUpdatedVersion)
	if err != nil {
		return fmt.Errorf("invalid forkchoiceUpdatedVersion: %v", err)
	}

	if len(raw.Params) < 1 {
		return errors.New("params must have at least one element")
	}
	// params[0] is always the ExecutableData
	if err := json.Unmarshal(raw.Params[0], &p.ExecutionPayload); err != nil {
		return fmt.Errorf("failed to unmarshal ExecutableData: %v", err)
	}
	// V3+: params[1] = versionedHashes, params[2] = beaconRoot
	if len(raw.Params) >= 3 {
		if err := json.Unmarshal(raw.Params[1], &p.VersionedHashes); err != nil {
			return fmt.Errorf("failed to unmarshal versionedHashes: %v", err)
		}
		var beaconRoot common.Hash
		if err := json.Unmarshal(raw.Params[2], &beaconRoot); err != nil {
			return fmt.Errorf("failed to unmarshal beaconRoot: %v", err)
		}
		p.BeaconRoot = &beaconRoot
	}
	// V4/V5+: params[3] = executionRequests
	if len(raw.Params) >= 4 {
		var hexRequests []hexutil.Bytes
		if err := json.Unmarshal(raw.Params[3], &hexRequests); err != nil {
			return fmt.Errorf("failed to unmarshal executionRequests: %v", err)
		}
		p.Requests = make([][]byte, len(hexRequests))
		for i, r := range hexRequests {
			p.Requests[i] = r
		}
	}
	return nil
}

// Run executes the engine test.
func (t *EngineTest) Run(scheme string, tracer *tracing.Hooks, postCheck func(error, *core.BlockChain)) (result error) {
	config, ok := Forks[t.json.Network]
	if !ok {
		return UnsupportedForkError{t.json.Network}
	}
	// Create genesis spec
	gspec := t.genesis(config)

	db := rawdb.NewMemoryDatabase()
	tconf := &triedb.Config{
		Preimages: true,
		IsVerkle:  gspec.Config.VerkleTime != nil && *gspec.Config.VerkleTime <= gspec.Timestamp,
	}
	if scheme == rawdb.PathScheme || tconf.IsVerkle {
		tconf.PathDB = pathdb.Defaults
	} else {
		tconf.HashDB = hashdb.Defaults
	}
	if gspec.Config.TerminalTotalDifficulty == nil {
		gspec.Config.TerminalTotalDifficulty = big.NewInt(stdmath.MaxInt64)
	}
	trieDb := triedb.NewDatabase(db, tconf)
	gblock, err := gspec.Commit(db, trieDb, nil)
	if err != nil {
		return err
	}
	trieDb.Close()

	if gblock.Hash() != t.json.Genesis.Hash {
		return fmt.Errorf("genesis block hash doesn't match test: computed=%x, test=%x", gblock.Hash().Bytes()[:6], t.json.Genesis.Hash[:6])
	}
	if gblock.Root() != t.json.Genesis.StateRoot {
		return fmt.Errorf("genesis block state root does not match test: computed=%x, test=%x", gblock.Root().Bytes()[:6], t.json.Genesis.StateRoot[:6])
	}
	eng := beacon.New(ethash.NewFaker())
	options := &core.BlockChainConfig{
		TrieCleanLimit: 0,
		StateScheme:    scheme,
		Preimages:      true,
		TxLookupLimit:  -1,
		VmConfig:       vm.Config{Tracer: tracer},
		NoPrefetch:     true,
	}
	chain, err := core.NewBlockChain(db, gspec, eng, options)
	if err != nil {
		return err
	}
	defer chain.Stop()

	if postCheck != nil {
		defer postCheck(result, chain)
	}

	// Create engine handler and execute payloads
	// Uses the same core functions as ConsensusAPI (ExecutableDataToBlock,
	// InsertBlockWithoutSetHead, SetCanonical) — different from blocktest's InsertChain.
	handler := newEngineHandler(chain)

	// Send initial forkchoiceUpdated to genesis (matching consume engine behavior)
	genesisHash := chain.Genesis().Hash()
	initialFcResp := handler.forkchoiceUpdated(engine.ForkchoiceStateV1{
		HeadBlockHash:      genesisHash,
		SafeBlockHash:      genesisHash,
		FinalizedBlockHash: genesisHash,
	})
	if initialFcResp.PayloadStatus.Status != engine.VALID {
		return fmt.Errorf("initial FCU to genesis returned %s", initialFcResp.PayloadStatus.Status)
	}

	for i, payload := range t.json.Payloads {
		status, err := handler.newPayloadVersioned(payload)
		// Check error code expectation
		if payload.ErrorCode != nil {
			var apiErr *engine.EngineAPIError
			if err == nil || !errors.As(err, &apiErr) {
				return fmt.Errorf("payload %d: expected error code %d, got err=%v", i, *payload.ErrorCode, err)
			}
			if apiErr.ErrorCode() != *payload.ErrorCode {
				return fmt.Errorf("payload %d: expected error code %d, got %d", i, *payload.ErrorCode, apiErr.ErrorCode())
			}
			continue // error code matched, move to next payload
		}
		if err != nil {
			return fmt.Errorf("payload %d: unexpected error: %v", i, err)
		}
		// Track last payload status and validation error for result reporting
		t.LastPayloadStatus = status.Status
		if status.ValidationError != nil {
			t.LastValidationError = *status.ValidationError
		}
		// Check validation error expectation
		if payload.ValidationError != "" {
			if status.Status != engine.INVALID {
				return fmt.Errorf("payload %d: expected INVALID status for validation error %q, got %s", i, payload.ValidationError, status.Status)
			}
			continue // invalid payload as expected, move to next
		}
		// Expect valid
		if status.Status != engine.VALID {
			errMsg := ""
			if status.ValidationError != nil {
				errMsg = *status.ValidationError
			}
			return fmt.Errorf("payload %d: expected VALID, got %s (err: %s)", i, status.Status, errMsg)
		}
		// Advance chain head via forkchoice update
		fcResp := handler.forkchoiceUpdated(engine.ForkchoiceStateV1{
			HeadBlockHash:      payload.ExecutionPayload.BlockHash,
			SafeBlockHash:      payload.ExecutionPayload.BlockHash,
			FinalizedBlockHash: common.Hash{}, // don't set finalized
		})
		if fcResp.PayloadStatus.Status != engine.VALID {
			return fmt.Errorf("payload %d: forkchoiceUpdated returned %s", i, fcResp.PayloadStatus.Status)
		}
	}

	// Validate final state
	cmlast := chain.CurrentBlock().Hash()
	if common.Hash(t.json.BestBlock) != cmlast {
		return fmt.Errorf("last block hash validation mismatch: want: %x, have: %x", t.json.BestBlock, cmlast)
	}
	if t.json.Post != nil {
		statedb, err := chain.State()
		if err != nil {
			return err
		}
		if err := validateEnginePostState(t.json.Post, statedb); err != nil {
			return fmt.Errorf("post state validation failed: %v", err)
		}
	} else if t.json.PostHash != nil {
		have := chain.CurrentBlock().Root
		want := common.Hash(*t.json.PostHash)
		if have != want {
			return fmt.Errorf("post state root mismatch: want %x, have %x", want, have)
		}
	}
	return nil
}

func (t *EngineTest) genesis(config *params.ChainConfig) *core.Genesis {
	return &core.Genesis{
		Config:        config,
		Nonce:         t.json.Genesis.Nonce.Uint64(),
		Timestamp:     t.json.Genesis.Timestamp,
		ParentHash:    t.json.Genesis.ParentHash,
		ExtraData:     t.json.Genesis.ExtraData,
		GasLimit:      t.json.Genesis.GasLimit,
		GasUsed:       t.json.Genesis.GasUsed,
		Difficulty:    t.json.Genesis.Difficulty,
		Mixhash:       t.json.Genesis.MixHash,
		Coinbase:      t.json.Genesis.Coinbase,
		Alloc:         t.json.Pre,
		BaseFee:       t.json.Genesis.BaseFeePerGas,
		BlobGasUsed:   t.json.Genesis.BlobGasUsed,
		ExcessBlobGas: t.json.Genesis.ExcessBlobGas,
	}
}

// validateEnginePostState verifies the post-state accounts match the expected values.
// Mirrors BlockTest.validatePostState.
func validateEnginePostState(post types.GenesisAlloc, statedb *state.StateDB) error {
	for addr, acct := range post {
		code := statedb.GetCode(addr)
		balance := statedb.GetBalance(addr).ToBig()
		nonce := statedb.GetNonce(addr)
		if !bytes.Equal(code, acct.Code) {
			return fmt.Errorf("account code mismatch for addr: %s want: %v have: %x", addr, acct.Code, code)
		}
		if balance.Cmp(acct.Balance) != 0 {
			return fmt.Errorf("account balance mismatch for addr: %s, want: %d, have: %d", addr, acct.Balance, balance)
		}
		if nonce != acct.Nonce {
			return fmt.Errorf("account nonce mismatch for addr: %s want: %d have: %d", addr, acct.Nonce, nonce)
		}
		for k, v := range acct.Storage {
			v2 := statedb.GetState(addr, k)
			if v2 != v {
				return fmt.Errorf("account storage mismatch for addr: %s, slot: %x, want: %x, have: %x", addr, k, v, v2)
			}
		}
	}
	return nil
}

// engineHandler is a lightweight Engine API handler that mirrors the core logic
// of eth/catalyst.ConsensusAPI but operates directly on a *core.BlockChain
// without requiring the full eth.Ethereum node stack.
type engineHandler struct {
	chain             *core.BlockChain
	invalidBlocksHits map[common.Hash]int
	invalidTipsets    map[common.Hash]*types.Header
}

func newEngineHandler(chain *core.BlockChain) *engineHandler {
	return &engineHandler{
		chain:             chain,
		invalidBlocksHits: make(map[common.Hash]int),
		invalidTipsets:    make(map[common.Hash]*types.Header),
	}
}

// newPayloadVersioned dispatches to the appropriate version-specific validation
// before calling the core newPayload logic. Mirrors NewPayloadV1-V5 in
// eth/catalyst/api.go.
func (h *engineHandler) newPayloadVersioned(p etNewPayload) (engine.PayloadStatusV1, error) {
	params := p.ExecutionPayload
	switch p.Version {
	case 1:
		if params.Withdrawals != nil {
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("withdrawals not supported in V1")
		}
		return h.newPayload(params, nil, nil, nil)

	case 2:
		cancun := h.config().IsCancun(h.config().LondonBlock, params.Timestamp)
		shanghai := h.config().IsShanghai(h.config().LondonBlock, params.Timestamp)
		switch {
		case cancun:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("can't use newPayloadV2 post-cancun")
		case shanghai && params.Withdrawals == nil:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("nil withdrawals post-shanghai")
		case !shanghai && params.Withdrawals != nil:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("non-nil withdrawals pre-shanghai")
		case params.ExcessBlobGas != nil:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("non-nil excessBlobGas pre-cancun")
		case params.BlobGasUsed != nil:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("non-nil blobGasUsed pre-cancun")
		}
		return h.newPayload(params, nil, nil, nil)

	case 3:
		switch {
		case params.Withdrawals == nil:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("nil withdrawals post-shanghai")
		case params.ExcessBlobGas == nil:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("nil excessBlobGas post-cancun")
		case params.BlobGasUsed == nil:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("nil blobGasUsed post-cancun")
		case p.VersionedHashes == nil:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("nil versionedHashes post-cancun")
		case p.BeaconRoot == nil:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("nil beaconRoot post-cancun")
		case !h.checkFork(params.Timestamp, forks.Cancun, forks.Prague, forks.Osaka, forks.BPO1, forks.BPO2, forks.BPO3, forks.BPO4, forks.BPO5):
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineUnsupportedForkErr("newPayloadV3 must only be called for cancun payloads")
		}
		return h.newPayload(params, p.VersionedHashes, p.BeaconRoot, nil)

	case 4:
		switch {
		case params.Withdrawals == nil:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("nil withdrawals post-shanghai")
		case params.ExcessBlobGas == nil:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("nil excessBlobGas post-cancun")
		case params.BlobGasUsed == nil:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("nil blobGasUsed post-cancun")
		case p.VersionedHashes == nil:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("nil versionedHashes post-cancun")
		case p.BeaconRoot == nil:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("nil beaconRoot post-cancun")
		case p.Requests == nil:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("nil executionRequests post-prague")
		case !h.checkFork(params.Timestamp, forks.Prague, forks.Osaka, forks.BPO1, forks.BPO2, forks.BPO3, forks.BPO4, forks.BPO5):
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineUnsupportedForkErr("newPayloadV4 must only be called for prague/osaka payloads")
		}
		if err := engineValidateRequests(p.Requests); err != nil {
			return engine.PayloadStatusV1{Status: engine.INVALID}, engine.InvalidParams.With(err)
		}
		return h.newPayload(params, p.VersionedHashes, p.BeaconRoot, p.Requests)

	case 5:
		switch {
		case params.Withdrawals == nil:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("nil withdrawals post-shanghai")
		case params.ExcessBlobGas == nil:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("nil excessBlobGas post-cancun")
		case params.BlobGasUsed == nil:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("nil blobGasUsed post-cancun")
		case p.VersionedHashes == nil:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("nil versionedHashes post-cancun")
		case p.BeaconRoot == nil:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("nil beaconRoot post-cancun")
		case p.Requests == nil:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("nil executionRequests post-prague")
		case params.SlotNumber == nil:
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineParamsErr("nil slotnumber post-amsterdam")
		case !h.checkFork(params.Timestamp, forks.Amsterdam):
			return engine.PayloadStatusV1{Status: engine.INVALID}, engineUnsupportedForkErr("newPayloadV5 must only be called for amsterdam payloads")
		}
		if err := engineValidateRequests(p.Requests); err != nil {
			return engine.PayloadStatusV1{Status: engine.INVALID}, engine.InvalidParams.With(err)
		}
		return h.newPayload(params, p.VersionedHashes, p.BeaconRoot, p.Requests)

	default:
		return engine.PayloadStatusV1{Status: engine.INVALID}, fmt.Errorf("unsupported newPayload version: %d", p.Version)
	}
}

// newPayload mirrors the core logic of ConsensusAPI.newPayload (api.go:766).
func (h *engineHandler) newPayload(params engine.ExecutableData, versionedHashes []common.Hash, beaconRoot *common.Hash, requests [][]byte) (engine.PayloadStatusV1, error) {
	block, err := engine.ExecutableDataToBlock(params, versionedHashes, beaconRoot, requests)
	if err != nil {
		return h.invalid(err, nil), nil
	}
	// If we already have the block locally, return VALID immediately
	if existing := h.chain.GetBlockByHash(params.BlockHash); existing != nil {
		hash := existing.Hash()
		return engine.PayloadStatusV1{Status: engine.VALID, LatestValidHash: &hash}, nil
	}
	// If this block was rejected previously, keep rejecting it
	if res := h.checkInvalidAncestor(block.Hash(), block.Hash()); res != nil {
		return *res, nil
	}
	// Check parent exists
	parent := h.chain.GetBlock(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		// In a test context with complete fixture data, missing parent is unexpected.
		// Return SYNCING to match the real engine API behavior.
		return engine.PayloadStatusV1{Status: engine.SYNCING}, nil
	}
	// Check timestamp
	if block.Time() <= parent.Time() {
		return h.invalid(errors.New("invalid timestamp"), parent.Header()), nil
	}
	// Check parent state exists
	if !h.chain.HasBlockAndState(block.ParentHash(), block.NumberU64()-1) {
		return engine.PayloadStatusV1{Status: engine.ACCEPTED}, nil
	}
	// Insert block without setting head (same as ConsensusAPI)
	if _, err := h.chain.InsertBlockWithoutSetHead(context.Background(), block, false); err != nil {
		h.invalidBlocksHits[block.Hash()] = 1
		h.invalidTipsets[block.Hash()] = block.Header()
		return h.invalid(err, parent.Header()), nil
	}
	hash := block.Hash()
	return engine.PayloadStatusV1{Status: engine.VALID, LatestValidHash: &hash}, nil
}

// forkchoiceUpdated mirrors the core logic of ConsensusAPI.forkchoiceUpdated (api.go:237).
func (h *engineHandler) forkchoiceUpdated(update engine.ForkchoiceStateV1) engine.ForkChoiceResponse {
	if update.HeadBlockHash == (common.Hash{}) {
		return engine.STATUS_INVALID
	}
	block := h.chain.GetBlockByHash(update.HeadBlockHash)
	if block == nil {
		if res := h.checkInvalidAncestor(update.HeadBlockHash, update.HeadBlockHash); res != nil {
			return engine.ForkChoiceResponse{PayloadStatus: *res}
		}
		return engine.ForkChoiceResponse{PayloadStatus: engine.PayloadStatusV1{Status: engine.SYNCING}}
	}
	// Set canonical head if not already the current head
	if h.chain.CurrentBlock().Hash() != update.HeadBlockHash {
		if latestValid, err := h.chain.SetCanonical(block); err != nil {
			return engine.ForkChoiceResponse{
				PayloadStatus: engine.PayloadStatusV1{Status: engine.INVALID, LatestValidHash: &latestValid},
			}
		}
	}
	// Set finalized block if specified
	if update.FinalizedBlockHash != (common.Hash{}) {
		finalBlock := h.chain.GetBlockByHash(update.FinalizedBlockHash)
		if finalBlock != nil {
			h.chain.SetFinalized(finalBlock.Header())
		}
	}
	// Set safe block if specified
	if update.SafeBlockHash != (common.Hash{}) {
		safeBlock := h.chain.GetBlockByHash(update.SafeBlockHash)
		if safeBlock != nil {
			h.chain.SetSafe(safeBlock.Header())
		}
	}
	return engine.ForkChoiceResponse{
		PayloadStatus: engine.PayloadStatusV1{
			Status:          engine.VALID,
			LatestValidHash: &update.HeadBlockHash,
		},
	}
}

// checkInvalidAncestor mirrors ConsensusAPI.checkInvalidAncestor (api.go:952).
func (h *engineHandler) checkInvalidAncestor(check common.Hash, head common.Hash) *engine.PayloadStatusV1 {
	invalid, ok := h.invalidTipsets[check]
	if !ok {
		return nil
	}
	badHash := invalid.Hash()
	h.invalidBlocksHits[badHash]++
	if h.invalidBlocksHits[badHash] >= 128 {
		delete(h.invalidBlocksHits, badHash)
		for descendant, badHeader := range h.invalidTipsets {
			if badHeader.Hash() == badHash {
				delete(h.invalidTipsets, descendant)
			}
		}
		return nil
	}
	if check != head {
		if len(h.invalidTipsets) >= 512 {
			for key := range h.invalidTipsets {
				delete(h.invalidTipsets, key)
				break
			}
		}
		h.invalidTipsets[head] = invalid
	}
	lastValid := &invalid.ParentHash
	if header := h.chain.GetHeader(invalid.ParentHash, invalid.Number.Uint64()-1); header != nil && header.Difficulty.Sign() != 0 {
		lastValid = &common.Hash{}
	}
	failure := "links to previously rejected block"
	return &engine.PayloadStatusV1{
		Status:          engine.INVALID,
		LatestValidHash: lastValid,
		ValidationError: &failure,
	}
}

// invalid mirrors ConsensusAPI.invalid (api.go:1002).
func (h *engineHandler) invalid(err error, latestValid *types.Header) engine.PayloadStatusV1 {
	var currentHash *common.Hash
	if latestValid != nil {
		if latestValid.Difficulty.BitLen() != 0 {
			currentHash = &common.Hash{}
		} else {
			hash := latestValid.Hash()
			currentHash = &hash
		}
	}
	errorMsg := err.Error()
	return engine.PayloadStatusV1{
		Status:          engine.INVALID,
		LatestValidHash: currentHash,
		ValidationError: &errorMsg,
	}
}

func (h *engineHandler) config() *params.ChainConfig {
	return h.chain.Config()
}

func (h *engineHandler) checkFork(timestamp uint64, allowedForks ...forks.Fork) bool {
	latest := h.config().LatestFork(timestamp)
	for _, fork := range allowedForks {
		if latest == fork {
			return true
		}
	}
	return false
}

// engineParamsErr creates an InvalidParams Engine API error.
func engineParamsErr(msg string) error {
	return engine.InvalidParams.With(errors.New(msg))
}

// engineUnsupportedForkErr creates an UnsupportedFork Engine API error.
func engineUnsupportedForkErr(msg string) error {
	return engine.UnsupportedFork.With(errors.New(msg))
}

// engineValidateRequests checks that requests are ordered by type and not empty.
// Mirrors validateRequests in eth/catalyst/api.go.
func engineValidateRequests(requests [][]byte) error {
	for i, req := range requests {
		if len(req) < 2 {
			return fmt.Errorf("empty request: %v", req)
		}
		if i > 0 && req[0] <= requests[i-1][0] {
			return fmt.Errorf("invalid request order: %v", req)
		}
	}
	return nil
}
