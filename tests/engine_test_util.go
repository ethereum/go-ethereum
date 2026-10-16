// Copyright 2026 The go-ethereum Authors
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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	stdmath "math"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rpc"
)

// EngineTest checks processing of engine API payloads.
type EngineTest struct {
	json etJSON
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
// The params are kept raw and sent to the node as they appear in the fixture.
type etNewPayload struct {
	Params    []json.RawMessage
	BlockHash common.Hash

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
	p.Params = raw.Params
	// params[0] is always the execution payload
	var payload struct {
		BlockHash common.Hash `json:"blockHash"`
	}
	if err := json.Unmarshal(raw.Params[0], &payload); err != nil {
		return fmt.Errorf("failed to unmarshal execution payload: %v", err)
	}
	p.BlockHash = payload.BlockHash
	return nil
}

// EngineAPIFunc attaches the engine API to a test's freshly built chain and
// returns an in-process client for it, along with a function releasing both.
type EngineAPIFunc func(chain *core.BlockChain, db ethdb.Database) (*rpc.Client, func(), error)

// Run executes the engine test on a fresh in-memory chain built from the
// fixture's genesis. attach puts the engine API on that chain, and payloads and
// forkchoice updates go through it at the method versions the fixture names.
func (t *EngineTest) Run(scheme string, tracer *tracing.Hooks, attach EngineAPIFunc, postCheck func(error, *core.BlockChain)) (result error) {
	config, ok := Forks[t.json.Network]
	if !ok {
		return UnsupportedForkError{t.json.Network}
	}
	cpy := *config
	gspec := genesisFromHeader(&cpy, &t.json.Genesis, t.json.Pre)
	// if ttd is not specified, set an arbitrary huge value
	if gspec.Config.TerminalTotalDifficulty == nil {
		gspec.Config.TerminalTotalDifficulty = big.NewInt(stdmath.MaxInt64)
	}
	db := rawdb.NewMemoryDatabase()
	options := &core.BlockChainConfig{
		TrieCleanLimit: 0,
		StateScheme:    scheme,
		Preimages:      true,
		TxLookupLimit:  -1, // disable tx indexing
		VmConfig: vm.Config{
			Tracer: tracer,
		},
	}
	chain, err := core.NewBlockChain(db, gspec, beacon.New(ethash.NewFaker()), options)
	if err != nil {
		return err
	}
	defer chain.Stop()

	if postCheck != nil {
		defer func() { postCheck(result, chain) }()
	}
	genesis := chain.Genesis()
	if genesis.Hash() != t.json.Genesis.Hash {
		return fmt.Errorf("genesis block hash doesn't match test: computed=%x, test=%x", genesis.Hash().Bytes()[:6], t.json.Genesis.Hash[:6])
	}
	if genesis.Root() != t.json.Genesis.StateRoot {
		return fmt.Errorf("genesis block state root does not match test: computed=%x, test=%x", genesis.Root().Bytes()[:6], t.json.Genesis.StateRoot[:6])
	}
	client, release, err := attach(chain, db)
	if err != nil {
		return err
	}
	defer release()
	if len(t.json.Payloads) == 0 {
		return errors.New("no payloads")
	}
	// Send initial forkchoiceUpdated to genesis (matching consume engine behavior)
	fcResp, err := forkchoiceUpdated(client, t.json.Payloads[0].FcuVersion, genesis.Hash())
	if err != nil {
		return fmt.Errorf("initial forkchoiceUpdated to genesis: %v", err)
	}
	if fcResp.PayloadStatus.Status != engine.VALID {
		return fmt.Errorf("initial FCU to genesis returned %s", fcResp.PayloadStatus.Status)
	}

	for i, payload := range t.json.Payloads {
		var status engine.PayloadStatusV1
		params := make([]any, len(payload.Params))
		for j, param := range payload.Params {
			params[j] = param
		}
		err := client.CallContext(context.Background(), &status, fmt.Sprintf("engine_newPayloadV%d", payload.Version), params...)
		if payload.ErrorCode != nil {
			var rpcErr rpc.Error
			if !errors.As(err, &rpcErr) {
				return fmt.Errorf("payload %d: expected error code %d, got err=%v", i, *payload.ErrorCode, err)
			}
			if rpcErr.ErrorCode() != *payload.ErrorCode {
				return fmt.Errorf("payload %d: expected error code %d, got %d", i, *payload.ErrorCode, rpcErr.ErrorCode())
			}
			continue // error code matched, move to next payload
		}
		if err != nil {
			return fmt.Errorf("payload %d: unexpected error: %v", i, err)
		}
		if payload.ValidationError != "" {
			if status.Status != engine.INVALID {
				return fmt.Errorf("payload %d: expected INVALID status for validation error %q, got %s", i, payload.ValidationError, status.Status)
			}
			continue // invalid payload as expected, move to next
		}
		if status.Status != engine.VALID {
			errMsg := ""
			if status.ValidationError != nil {
				errMsg = *status.ValidationError
			}
			return fmt.Errorf("payload %d: expected VALID, got %s (err: %s)", i, status.Status, errMsg)
		}
		// Advance chain head via forkchoice update
		fcResp, err := forkchoiceUpdated(client, payload.FcuVersion, payload.BlockHash)
		if err != nil {
			return fmt.Errorf("payload %d: forkchoiceUpdated: %v", i, err)
		}
		if fcResp.PayloadStatus.Status != engine.VALID {
			return fmt.Errorf("payload %d: forkchoiceUpdated returned %s", i, fcResp.PayloadStatus.Status)
		}
	}

	cmlast := chain.CurrentBlock().Hash()
	if common.Hash(t.json.BestBlock) != cmlast {
		return fmt.Errorf("last block hash validation mismatch: want: %x, have: %x", t.json.BestBlock, cmlast)
	}
	if t.json.Post != nil {
		statedb, err := chain.State()
		if err != nil {
			return err
		}
		if err := validatePostState(t.json.Post, statedb); err != nil {
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

// forkchoiceUpdated moves the node's head to the given block, as consume
// engine does: no safe or finalized block and no payload attributes.
func forkchoiceUpdated(client *rpc.Client, version int, head common.Hash) (engine.ForkChoiceResponse, error) {
	var resp engine.ForkChoiceResponse
	update := engine.ForkchoiceStateV1{HeadBlockHash: head}
	err := client.CallContext(context.Background(), &resp, fmt.Sprintf("engine_forkchoiceUpdatedV%d", version), update, nil)
	return resp, err
}
