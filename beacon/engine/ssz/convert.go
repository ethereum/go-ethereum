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

package ssz

import (
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/karalabe/ssz"
)

// statusToEnum maps the JSON-RPC PayloadStatusV1.Status string to the SSZ
// uint8. ACCEPTED is only valid on /payloads responses; the caller is
// responsible for downstream restrictions.
func statusToEnum(s string) uint8 {
	switch s {
	case engine.VALID:
		return StatusValid
	case engine.INVALID:
		return StatusInvalid
	case engine.SYNCING:
		return StatusSyncing
	case engine.ACCEPTED:
		return StatusAccepted
	}
	return StatusInvalid
}

// statusFromEnum is the inverse of statusToEnum.
func statusFromEnum(s uint8) string {
	switch s {
	case StatusValid:
		return engine.VALID
	case StatusInvalid:
		return engine.INVALID
	case StatusSyncing:
		return engine.SYNCING
	case StatusAccepted:
		return engine.ACCEPTED
	}
	return engine.INVALID
}

// PayloadStatusFromV1 converts the JSON-RPC PayloadStatusV1 into its SSZ wire form.
func PayloadStatusFromV1(s *engine.PayloadStatusV1) *PayloadStatus {
	out := &PayloadStatus{Status: statusToEnum(s.Status)}
	if s.LatestValidHash != nil {
		out.LatestValidHash = []common.Hash{*s.LatestValidHash}
	}
	if s.ValidationError != nil {
		out.ValidationError = [][]byte{[]byte(*s.ValidationError)}
	}
	return out
}

// PayloadStatusToV1 is the inverse of PayloadStatusFromV1.
func PayloadStatusToV1(s *PayloadStatus) engine.PayloadStatusV1 {
	out := engine.PayloadStatusV1{Status: statusFromEnum(s.Status)}
	if len(s.LatestValidHash) == 1 {
		h := s.LatestValidHash[0]
		out.LatestValidHash = &h
	}
	if len(s.ValidationError) == 1 {
		msg := string(s.ValidationError[0])
		out.ValidationError = &msg
	}
	return out
}

// ForkchoiceStateFromV1 converts ForkchoiceStateV1 to its SSZ form.
func ForkchoiceStateFromV1(s engine.ForkchoiceStateV1) *ForkchoiceState {
	return &ForkchoiceState{
		HeadBlockHash:      s.HeadBlockHash,
		SafeBlockHash:      s.SafeBlockHash,
		FinalizedBlockHash: s.FinalizedBlockHash,
	}
}

// ForkchoiceStateToV1 is the inverse of ForkchoiceStateFromV1.
func ForkchoiceStateToV1(s *ForkchoiceState) engine.ForkchoiceStateV1 {
	return engine.ForkchoiceStateV1{
		HeadBlockHash:      s.HeadBlockHash,
		SafeBlockHash:      s.SafeBlockHash,
		FinalizedBlockHash: s.FinalizedBlockHash,
	}
}

// withdrawalsFromTypes converts geth Withdrawals into SSZ Withdrawals.
func withdrawalsFromTypes(ws []*types.Withdrawal) []*Withdrawal {
	out := make([]*Withdrawal, len(ws))
	for i, w := range ws {
		out[i] = &Withdrawal{
			Index:          w.Index,
			ValidatorIndex: w.Validator,
			Address:        w.Address,
			Amount:         w.Amount,
		}
	}
	return out
}

// withdrawalsToTypes is the inverse of withdrawalsFromTypes.
func withdrawalsToTypes(ws []*Withdrawal) []*types.Withdrawal {
	out := make([]*types.Withdrawal, len(ws))
	for i, w := range ws {
		out[i] = &types.Withdrawal{
			Index:     w.Index,
			Validator: w.ValidatorIndex,
			Address:   w.Address,
			Amount:    w.Amount,
		}
	}
	return out
}

// PayloadAttributesFromEngine converts a JSON-RPC engine.PayloadAttributes into
// the monolithic SSZ form for the given fork. Only the fields the fork's wire
// shape carries are populated; the caller should pass the same fork to the
// codec. target_gas_limit has no representation in engine.PayloadAttributes
// today, so it is left nil for Amsterdam unless explicitly populated by the
// caller afterwards.
func PayloadAttributesFromEngine(a *engine.PayloadAttributes, fork ssz.Fork) *PayloadAttributes {
	out := &PayloadAttributes{
		Timestamp:             a.Timestamp,
		PrevRandao:            a.Random,
		SuggestedFeeRecipient: a.SuggestedFeeRecipient,
	}
	if fork >= ssz.ForkShapella {
		out.Withdrawals = withdrawalsFromTypes(a.Withdrawals)
	}
	if fork >= ssz.ForkDencun {
		root := common.Hash{}
		if a.BeaconRoot != nil {
			root = *a.BeaconRoot
		}
		out.ParentBeaconBlockRoot = &root
	}
	if fork >= forkAmsterdam {
		slot := uint64(0)
		if a.SlotNumber != nil {
			slot = *a.SlotNumber
		}
		out.SlotNumber = &slot
		// target_gas_limit not carried by engine.PayloadAttributes; default 0.
		tgl := uint64(0)
		out.TargetGasLimit = &tgl
	}
	return out
}

// PayloadAttributesToEngine is the inverse of PayloadAttributesFromEngine. The
// target_gas_limit field has no representation in engine.PayloadAttributes and
// is dropped; the caller can read it from the SSZ struct directly.
func PayloadAttributesToEngine(a *PayloadAttributes) *engine.PayloadAttributes {
	out := &engine.PayloadAttributes{
		Timestamp:             a.Timestamp,
		Random:                a.PrevRandao,
		SuggestedFeeRecipient: a.SuggestedFeeRecipient,
		Withdrawals:           withdrawalsToTypes(a.Withdrawals),
	}
	if a.ParentBeaconBlockRoot != nil {
		root := *a.ParentBeaconBlockRoot
		out.BeaconRoot = &root
	}
	if a.SlotNumber != nil {
		slot := *a.SlotNumber
		out.SlotNumber = &slot
	}
	return out
}

// ExecutionPayloadFromEngine converts an ExecutableData into the monolithic SSZ
// payload for the given fork. Only the fork's active fields are populated so
// the codec (driven by the same fork) and Validate(fork) stay consistent.
// BlockAccessList is not yet wired through ExecutableData; the caller sets it
// separately for Amsterdam.
func ExecutionPayloadFromEngine(d *engine.ExecutableData, fork ssz.Fork) *ExecutionPayload {
	var bloom [256]byte
	copy(bloom[:], d.LogsBloom)
	var fee *uint256.Int
	if d.BaseFeePerGas != nil {
		fee = new(uint256.Int)
		fee.SetFromBig(d.BaseFeePerGas)
	}
	out := &ExecutionPayload{
		ParentHash:    d.ParentHash,
		FeeRecipient:  d.FeeRecipient,
		StateRoot:     d.StateRoot,
		ReceiptsRoot:  d.ReceiptsRoot,
		LogsBloom:     bloom,
		PrevRandao:    d.Random,
		BlockNumber:   d.Number,
		GasLimit:      d.GasLimit,
		GasUsed:       d.GasUsed,
		Timestamp:     d.Timestamp,
		ExtraData:     append([]byte(nil), d.ExtraData...),
		BaseFeePerGas: fee,
		BlockHash:     d.BlockHash,
		Transactions:  d.Transactions,
	}
	if fork >= ssz.ForkShapella {
		out.Withdrawals = withdrawalsFromTypes(d.Withdrawals)
	}
	if fork >= ssz.ForkDencun {
		blobGas := uint64(0)
		if d.BlobGasUsed != nil {
			blobGas = *d.BlobGasUsed
		}
		excess := uint64(0)
		if d.ExcessBlobGas != nil {
			excess = *d.ExcessBlobGas
		}
		out.BlobGasUsed = &blobGas
		out.ExcessBlobGas = &excess
	}
	if fork >= forkAmsterdam {
		slot := uint64(0)
		if d.SlotNumber != nil {
			slot = *d.SlotNumber
		}
		out.SlotNumber = &slot
	}
	return out
}

// ExecutionPayloadToEngine is the inverse helper. The caller is responsible for
// the BAL payload that ExecutableData doesn't yet carry.
func ExecutionPayloadToEngine(p *ExecutionPayload) *engine.ExecutableData {
	bloom := append([]byte(nil), p.LogsBloom[:]...)
	out := &engine.ExecutableData{
		ParentHash:   p.ParentHash,
		FeeRecipient: p.FeeRecipient,
		StateRoot:    p.StateRoot,
		ReceiptsRoot: p.ReceiptsRoot,
		LogsBloom:    bloom,
		Random:       p.PrevRandao,
		Number:       p.BlockNumber,
		GasLimit:     p.GasLimit,
		GasUsed:      p.GasUsed,
		Timestamp:    p.Timestamp,
		ExtraData:    append([]byte(nil), p.ExtraData...),
		BlockHash:    p.BlockHash,
		Transactions: p.Transactions,
		Withdrawals:  withdrawalsToTypes(p.Withdrawals),
	}
	if p.BaseFeePerGas != nil {
		out.BaseFeePerGas = p.BaseFeePerGas.ToBig()
	}
	if p.BlobGasUsed != nil {
		blobGas := *p.BlobGasUsed
		out.BlobGasUsed = &blobGas
	}
	if p.ExcessBlobGas != nil {
		excess := *p.ExcessBlobGas
		out.ExcessBlobGas = &excess
	}
	if p.SlotNumber != nil {
		slot := *p.SlotNumber
		out.SlotNumber = &slot
	}
	return out
}
