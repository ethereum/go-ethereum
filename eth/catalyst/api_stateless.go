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

package catalyst

import (
	"errors"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params/forks"
	"github.com/ethereum/go-ethereum/rlp"
)

// ForkchoiceUpdatedWithWitnessV1 is analogous to ForkchoiceUpdatedV1, only it
// generates an execution witness too if block building was requested.
func (api *ConsensusAPI) ForkchoiceUpdatedWithWitnessV1(update engine.ForkchoiceStateV1, payloadAttributes *engine.PayloadAttributes) (engine.ForkChoiceResponse, error) {
	if payloadAttributes != nil {
		if payloadAttributes.Withdrawals != nil || payloadAttributes.BeaconRoot != nil {
			return engine.STATUS_INVALID, engine.InvalidParams.With(errors.New("withdrawals and beacon root not supported in V1"))
		}
		if api.eth.BlockChain().Config().IsShanghai(api.eth.BlockChain().Config().LondonBlock, payloadAttributes.Timestamp) {
			return engine.STATUS_INVALID, engine.InvalidParams.With(errors.New("forkChoiceUpdateV1 called post-shanghai"))
		}
	}
	return api.forkchoiceUpdated(update, payloadAttributes, engine.PayloadV1, true)
}

// ForkchoiceUpdatedWithWitnessV2 is analogous to ForkchoiceUpdatedV2, only it
// generates an execution witness too if block building was requested.
func (api *ConsensusAPI) ForkchoiceUpdatedWithWitnessV2(update engine.ForkchoiceStateV1, params *engine.PayloadAttributes) (engine.ForkChoiceResponse, error) {
	if params != nil {
		if params.BeaconRoot != nil {
			return engine.STATUS_INVALID, engine.InvalidPayloadAttributes.With(errors.New("unexpected beacon root"))
		}
		switch api.eth.BlockChain().Config().LatestFork(params.Timestamp) {
		case forks.Paris:
			if params.Withdrawals != nil {
				return engine.STATUS_INVALID, engine.InvalidPayloadAttributes.With(errors.New("withdrawals before shanghai"))
			}
		case forks.Shanghai:
			if params.Withdrawals == nil {
				return engine.STATUS_INVALID, engine.InvalidPayloadAttributes.With(errors.New("missing withdrawals"))
			}
		default:
			return engine.STATUS_INVALID, engine.UnsupportedFork.With(errors.New("forkchoiceUpdatedV2 must only be called with paris and shanghai payloads"))
		}
	}
	return api.forkchoiceUpdated(update, params, engine.PayloadV2, true)
}

// ForkchoiceUpdatedWithWitnessV3 is analogous to ForkchoiceUpdatedV3, only it
// generates an execution witness too if block building was requested.
func (api *ConsensusAPI) ForkchoiceUpdatedWithWitnessV3(update engine.ForkchoiceStateV1, params *engine.PayloadAttributes) (engine.ForkChoiceResponse, error) {
	if params != nil {
		if params.Withdrawals == nil {
			return engine.STATUS_INVALID, engine.InvalidPayloadAttributes.With(errors.New("missing withdrawals"))
		}
		if params.BeaconRoot == nil {
			return engine.STATUS_INVALID, engine.InvalidPayloadAttributes.With(errors.New("missing beacon root"))
		}
		if !api.isLatestForkAt(params.Timestamp, forks.Cancun, forks.Prague) {
			return engine.STATUS_INVALID, engine.UnsupportedFork.With(errors.New("forkchoiceUpdatedV3 must only be called for cancun or prague payloads"))
		}
	}
	// TODO(matt): the spec requires that fcu is applied when called on a valid
	// hash, even if params are wrong. To do this we need to split up
	// forkchoiceUpdate into a function that only updates the head and then a
	// function that kicks off block construction.
	return api.forkchoiceUpdated(update, params, engine.PayloadV3, true)
}

// NewPayloadWithWitnessV1 is analogous to NewPayloadV1, only it also generates
// and returns a stateless witness after running the payload.
func (api *ConsensusAPI) NewPayloadWithWitnessV1(params engine.ExecutableData) (engine.PayloadStatusV1, error) {
	if params.Withdrawals != nil {
		return engine.PayloadStatusV1{Status: engine.INVALID}, engine.InvalidParams.With(errors.New("withdrawals not supported in V1"))
	}
	return api.newPayload(params, nil, nil, nil, true)
}

// NewPayloadWithWitnessV2 is analogous to NewPayloadV2, only it also generates
// and returns a stateless witness after running the payload.
func (api *ConsensusAPI) NewPayloadWithWitnessV2(params engine.ExecutableData) (engine.PayloadStatusV1, error) {
	if api.eth.BlockChain().Config().IsCancun(api.eth.BlockChain().Config().LondonBlock, params.Timestamp) {
		return engine.PayloadStatusV1{Status: engine.INVALID}, engine.InvalidParams.With(errors.New("can't use newPayloadV2 post-cancun"))
	}
	if api.isLatestForkAt(params.Timestamp, forks.Shanghai) {
		if params.Withdrawals == nil {
			return engine.PayloadStatusV1{Status: engine.INVALID}, engine.InvalidParams.With(errors.New("nil withdrawals post-shanghai"))
		}
	} else {
		if params.Withdrawals != nil {
			return engine.PayloadStatusV1{Status: engine.INVALID}, engine.InvalidParams.With(errors.New("non-nil withdrawals pre-shanghai"))
		}
	}
	if params.ExcessBlobGas != nil {
		return engine.PayloadStatusV1{Status: engine.INVALID}, engine.InvalidParams.With(errors.New("non-nil excessBlobGas pre-cancun"))
	}
	if params.BlobGasUsed != nil {
		return engine.PayloadStatusV1{Status: engine.INVALID}, engine.InvalidParams.With(errors.New("non-nil blobGasUsed pre-cancun"))
	}
	return api.newPayload(params, nil, nil, nil, true)
}

// NewPayloadWithWitnessV3 is analogous to NewPayloadV3, only it also generates
// and returns a stateless witness after running the payload.
func (api *ConsensusAPI) NewPayloadWithWitnessV3(params engine.ExecutableData, versionedHashes []common.Hash, beaconRoot *common.Hash) (engine.PayloadStatusV1, error) {
	checks := []param{
		{params.Withdrawals, "withdrawals", "shanghai"},
		{params.ExcessBlobGas, "excessBlobGas", "cancun"},
		{params.BlobGasUsed, "blobGasUsed", "cancun"},
		{versionedHashes, "versionedHashes", "cancun"},
		{beaconRoot, "beaconRoot", "cancun"},
	}
	if err := verifyParamsSet(checks); err != nil {
		return engine.PayloadStatusV1{Status: engine.INVALID}, err
	}

	if !api.isLatestForkAt(params.Timestamp, forks.Cancun) {
		return engine.PayloadStatusV1{Status: engine.INVALID}, engine.UnsupportedFork.With(errors.New("newPayloadWithWitnessV3 must only be called for cancun payloads"))
	}
	return api.newPayload(params, versionedHashes, beaconRoot, nil, true)
}

// NewPayloadWithWitnessV4 is analogous to NewPayloadV4, only it also generates
// and returns a stateless witness after running the payload.
func (api *ConsensusAPI) NewPayloadWithWitnessV4(params engine.ExecutableData, versionedHashes []common.Hash, beaconRoot *common.Hash, executionRequests []hexutil.Bytes) (engine.PayloadStatusV1, error) {
	checks := []param{
		{params.Withdrawals, "withdrawals", "shanghai"},
		{params.ExcessBlobGas, "excessBlobGas", "cancun"},
		{params.BlobGasUsed, "blobGasUsed", "cancun"},
		{versionedHashes, "versionedHashes", "cancun"},
		{beaconRoot, "beaconRoot", "cancun"},
		{executionRequests, "executionRequests", "prague"},
	}
	if err := verifyParamsSet(checks); err != nil {
		return engine.PayloadStatusV1{Status: engine.INVALID}, err
	}

	if !api.isLatestForkAt(params.Timestamp, forks.Prague) {
		return engine.PayloadStatusV1{Status: engine.INVALID}, engine.UnsupportedFork.With(errors.New("newPayloadWithWitnessV4 must only be called for prague payloads"))
	}
	requests := convertRequests(executionRequests)
	if err := validateRequests(requests); err != nil {
		return engine.PayloadStatusV1{Status: engine.INVALID}, engine.InvalidParams.With(err)
	}
	return api.newPayload(params, versionedHashes, beaconRoot, requests, true)
}

// ExecuteStatelessPayloadV1 is analogous to NewPayloadV1, only it operates in
// a stateless mode on top of a provided witness instead of the local database.
func (api *ConsensusAPI) ExecuteStatelessPayloadV1(params engine.ExecutableData, opaqueWitness hexutil.Bytes) (engine.StatelessPayloadStatusV1, error) {
	if params.Withdrawals != nil {
		return engine.StatelessPayloadStatusV1{Status: engine.INVALID}, engine.InvalidParams.With(errors.New("withdrawals not supported in V1"))
	}
	return api.executeStatelessPayload(params, nil, nil, nil, opaqueWitness)
}

// ExecuteStatelessPayloadV2 is analogous to NewPayloadV2, only it operates in
// a stateless mode on top of a provided witness instead of the local database.
func (api *ConsensusAPI) ExecuteStatelessPayloadV2(params engine.ExecutableData, opaqueWitness hexutil.Bytes) (engine.StatelessPayloadStatusV1, error) {
	if api.eth.BlockChain().Config().IsCancun(api.eth.BlockChain().Config().LondonBlock, params.Timestamp) {
		return engine.StatelessPayloadStatusV1{Status: engine.INVALID}, engine.InvalidParams.With(errors.New("can't use newPayloadV2 post-cancun"))
	}
	if api.isLatestForkAt(params.Timestamp, forks.Shanghai) {
		if params.Withdrawals == nil {
			return engine.StatelessPayloadStatusV1{Status: engine.INVALID}, engine.InvalidParams.With(errors.New("nil withdrawals post-shanghai"))
		}
	} else {
		if params.Withdrawals != nil {
			return engine.StatelessPayloadStatusV1{Status: engine.INVALID}, engine.InvalidParams.With(errors.New("non-nil withdrawals pre-shanghai"))
		}
	}
	if params.ExcessBlobGas != nil {
		return engine.StatelessPayloadStatusV1{Status: engine.INVALID}, engine.InvalidParams.With(errors.New("non-nil excessBlobGas pre-cancun"))
	}
	if params.BlobGasUsed != nil {
		return engine.StatelessPayloadStatusV1{Status: engine.INVALID}, engine.InvalidParams.With(errors.New("non-nil blobGasUsed pre-cancun"))
	}
	return api.executeStatelessPayload(params, nil, nil, nil, opaqueWitness)
}

// ExecuteStatelessPayloadV3 is analogous to NewPayloadV3, only it operates in
// a stateless mode on top of a provided witness instead of the local database.
func (api *ConsensusAPI) ExecuteStatelessPayloadV3(params engine.ExecutableData, versionedHashes []common.Hash, beaconRoot *common.Hash, opaqueWitness hexutil.Bytes) (engine.StatelessPayloadStatusV1, error) {
	checks := []param{
		{params.Withdrawals, "withdrawals", "shanghai"},
		{params.ExcessBlobGas, "excessBlobGas", "cancun"},
		{params.BlobGasUsed, "blobGasUsed", "cancun"},
		{versionedHashes, "versionedHashes", "cancun"},
		{beaconRoot, "beaconRoot", "cancun"},
	}
	if err := verifyParamsSet(checks); err != nil {
		return engine.StatelessPayloadStatusV1{Status: engine.INVALID}, err
	}

	if !api.isLatestForkAt(params.Timestamp, forks.Cancun) {
		return engine.StatelessPayloadStatusV1{Status: engine.INVALID}, engine.UnsupportedFork.With(errors.New("executeStatelessPayloadV3 must only be called for cancun payloads"))
	}
	return api.executeStatelessPayload(params, versionedHashes, beaconRoot, nil, opaqueWitness)
}

// ExecuteStatelessPayloadV4 is analogous to NewPayloadV4, only it operates in
// a stateless mode on top of a provided witness instead of the local database.
func (api *ConsensusAPI) ExecuteStatelessPayloadV4(params engine.ExecutableData, versionedHashes []common.Hash, beaconRoot *common.Hash, executionRequests []hexutil.Bytes, opaqueWitness hexutil.Bytes) (engine.StatelessPayloadStatusV1, error) {
	checks := []param{
		{params.Withdrawals, "withdrawals", "shanghai"},
		{params.ExcessBlobGas, "excessBlobGas", "cancun"},
		{params.BlobGasUsed, "blobGasUsed", "cancun"},
		{versionedHashes, "versionedHashes", "cancun"},
		{beaconRoot, "beaconRoot", "cancun"},
		{executionRequests, "executionRequests", "prague"},
	}
	if err := verifyParamsSet(checks); err != nil {
		return engine.StatelessPayloadStatusV1{Status: engine.INVALID}, err
	}

	if !api.isLatestForkAt(params.Timestamp, forks.Prague) {
		return engine.StatelessPayloadStatusV1{Status: engine.INVALID}, engine.UnsupportedFork.With(errors.New("executeStatelessPayloadV4 must only be called for prague payloads"))
	}
	requests := convertRequests(executionRequests)
	return api.executeStatelessPayload(params, versionedHashes, beaconRoot, requests, opaqueWitness)
}

func (api *ConsensusAPI) executeStatelessPayload(params engine.ExecutableData, versionedHashes []common.Hash, beaconRoot *common.Hash, requests [][]byte, opaqueWitness hexutil.Bytes) (engine.StatelessPayloadStatusV1, error) {
	log.Trace("Engine API request received", "method", "ExecuteStatelessPayload", "number", params.Number, "hash", params.BlockHash)
	block, err := engine.ExecutableDataToBlockNoHash(params, versionedHashes, beaconRoot, requests)
	if err != nil {
		bgu := "nil"
		if params.BlobGasUsed != nil {
			bgu = strconv.Itoa(int(*params.BlobGasUsed))
		}
		ebg := "nil"
		if params.ExcessBlobGas != nil {
			ebg = strconv.Itoa(int(*params.ExcessBlobGas))
		}
		log.Warn("Invalid ExecuteStatelessPayload params",
			"params.Number", params.Number,
			"params.ParentHash", params.ParentHash,
			"params.BlockHash", params.BlockHash,
			"params.StateRoot", params.StateRoot,
			"params.FeeRecipient", params.FeeRecipient,
			"params.LogsBloom", common.PrettyBytes(params.LogsBloom),
			"params.Random", params.Random,
			"params.GasLimit", params.GasLimit,
			"params.GasUsed", params.GasUsed,
			"params.Timestamp", params.Timestamp,
			"params.ExtraData", common.PrettyBytes(params.ExtraData),
			"params.BaseFeePerGas", params.BaseFeePerGas,
			"params.BlobGasUsed", bgu,
			"params.ExcessBlobGas", ebg,
			"len(params.Transactions)", len(params.Transactions),
			"len(params.Withdrawals)", len(params.Withdrawals),
			"beaconRoot", beaconRoot,
			"len(requests)", len(requests),
			"error", err)
		errorMsg := err.Error()
		return engine.StatelessPayloadStatusV1{Status: engine.INVALID, ValidationError: &errorMsg}, nil
	}
	witness := new(stateless.Witness)
	if err := rlp.DecodeBytes(opaqueWitness, witness); err != nil {
		log.Warn("Invalid ExecuteStatelessPayload witness", "err", err)
		errorMsg := err.Error()
		return engine.StatelessPayloadStatusV1{Status: engine.INVALID, ValidationError: &errorMsg}, nil
	}
	// Stash away the last update to warn the user if the beacon client goes offline
	api.lastNewPayloadLock.Lock()
	api.lastNewPayloadUpdate = time.Now()
	api.lastNewPayloadLock.Unlock()

	log.Trace("Executing block statelessly", "number", block.Number(), "hash", params.BlockHash)
	stateRoot, receiptRoot, err := core.ExecuteStateless(api.eth.BlockChain().Config(), vm.Config{}, block, witness)
	if err != nil {
		log.Warn("ExecuteStatelessPayload: execution failed", "err", err)
		errorMsg := err.Error()
		return engine.StatelessPayloadStatusV1{Status: engine.INVALID, ValidationError: &errorMsg}, nil
	}
	return engine.StatelessPayloadStatusV1{Status: engine.VALID, StateRoot: stateRoot, ReceiptsRoot: receiptRoot}, nil
}
