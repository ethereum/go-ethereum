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

// PayloadAttributesAmsterdamFromEngine converts a JSON-RPC engine.PayloadAttributes
// into its Amsterdam SSZ form. The caller is responsible for ensuring the
// attributes carry every Amsterdam-required field.
func PayloadAttributesAmsterdamFromEngine(a *engine.PayloadAttributes) *PayloadAttributesAmsterdam {
	out := &PayloadAttributesAmsterdam{
		Timestamp:             a.Timestamp,
		PrevRandao:            a.Random,
		SuggestedFeeRecipient: a.SuggestedFeeRecipient,
		Withdrawals:           withdrawalsFromTypes(a.Withdrawals),
	}
	if a.BeaconRoot != nil {
		out.ParentBeaconBlockRoot = *a.BeaconRoot
	}
	if a.SlotNumber != nil {
		out.SlotNumber = *a.SlotNumber
	}
	return out
}

// PayloadAttributesAmsterdamToEngine is the inverse of the From helper. The
// target_gas_limit field has no representation in engine.PayloadAttributes
// today and is dropped on the way down; the caller can fish it out of the SSZ
// struct directly.
func PayloadAttributesAmsterdamToEngine(a *PayloadAttributesAmsterdam) *engine.PayloadAttributes {
	root := a.ParentBeaconBlockRoot
	slot := a.SlotNumber
	return &engine.PayloadAttributes{
		Timestamp:             a.Timestamp,
		Random:                a.PrevRandao,
		SuggestedFeeRecipient: a.SuggestedFeeRecipient,
		Withdrawals:           withdrawalsToTypes(a.Withdrawals),
		BeaconRoot:            &root,
		SlotNumber:            &slot,
	}
}

// ExecutionPayloadAmsterdamFromEngine converts an ExecutableData into the SSZ
// Amsterdam payload shape. The block_access_list and slot_number fields are
// expected to be present; missing values yield empty/zero defaults.
func ExecutionPayloadAmsterdamFromEngine(d *engine.ExecutableData) *ExecutionPayloadAmsterdam {
	var bloom [256]byte
	copy(bloom[:], d.LogsBloom)
	var fee *uint256.Int
	if d.BaseFeePerGas != nil {
		fee = new(uint256.Int)
		fee.SetFromBig(d.BaseFeePerGas)
	}
	out := &ExecutionPayloadAmsterdam{
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
		Withdrawals:   withdrawalsFromTypes(d.Withdrawals),
	}
	if d.BlobGasUsed != nil {
		out.BlobGasUsed = *d.BlobGasUsed
	}
	if d.ExcessBlobGas != nil {
		out.ExcessBlobGas = *d.ExcessBlobGas
	}
	if d.SlotNumber != nil {
		out.SlotNumber = *d.SlotNumber
	}
	// BlockAccessList is not yet wired through ExecutableData; the caller
	// passes it as a separate field on the envelope when constructing the
	// final wire form.
	return out
}

// ExecutionPayloadAmsterdamToEngine is the inverse helper. The caller is
// responsible for the BAL payload that ExecutableData doesn't yet carry.
func ExecutionPayloadAmsterdamToEngine(p *ExecutionPayloadAmsterdam) *engine.ExecutableData {
	bloom := append([]byte(nil), p.LogsBloom[:]...)
	out := &engine.ExecutableData{
		ParentHash:    p.ParentHash,
		FeeRecipient:  p.FeeRecipient,
		StateRoot:     p.StateRoot,
		ReceiptsRoot:  p.ReceiptsRoot,
		LogsBloom:     bloom,
		Random:        p.PrevRandao,
		Number:        p.BlockNumber,
		GasLimit:      p.GasLimit,
		GasUsed:       p.GasUsed,
		Timestamp:     p.Timestamp,
		ExtraData:     append([]byte(nil), p.ExtraData...),
		BlockHash:     p.BlockHash,
		Transactions:  p.Transactions,
		Withdrawals:   withdrawalsToTypes(p.Withdrawals),
		BlobGasUsed:   &p.BlobGasUsed,
		ExcessBlobGas: &p.ExcessBlobGas,
		SlotNumber:    &p.SlotNumber,
	}
	if p.BaseFeePerGas != nil {
		out.BaseFeePerGas = p.BaseFeePerGas.ToBig()
	}
	return out
}
