// Copyright 2022 The go-ethereum Authors
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

package types

import (
	"encoding/binary"
	"encoding/json"

	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/common"
)

// Header defines a beacon header
//
// See data structure definition here:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#beaconblockheader
type Header struct {
	Slot          uint64
	ProposerIndex uint64
	ParentRoot    common.Hash
	StateRoot     common.Hash
	BodyRoot      common.Hash
}

// Header defines a beacon header and supports JSON encoding according to the
// standard beacon API format
//
// See data structure definition here:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/beacon-chain.md#beaconblockheader
type jsonHeader struct {
	Slot          common.Decimal `json:"slot"`
	ProposerIndex common.Decimal `json:"proposer_index"`
	ParentRoot    common.Hash    `json:"parent_root"`
	StateRoot     common.Hash    `json:"state_root"`
	BodyRoot      common.Hash    `json:"body_root"`
}

// MarshalJSON marshals as JSON.
func (bh *Header) MarshalJSON() ([]byte, error) {
	return json.Marshal(&jsonHeader{
		Slot:          common.Decimal(bh.Slot),
		ProposerIndex: common.Decimal(bh.ProposerIndex),
		ParentRoot:    bh.ParentRoot,
		StateRoot:     bh.StateRoot,
		BodyRoot:      bh.BodyRoot,
	})
}

// UnmarshalJSON unmarshals from JSON.
func (bh *Header) UnmarshalJSON(input []byte) error {
	var dec jsonHeader
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	bh.Slot = uint64(dec.Slot)
	bh.ProposerIndex = uint64(dec.ProposerIndex)
	bh.ParentRoot = dec.ParentRoot
	bh.StateRoot = dec.StateRoot
	bh.BodyRoot = dec.BodyRoot
	return nil
}

// Hash calculates the block root of the header
func (bh *Header) Hash() common.Hash {
	var values [8]merkle.Value // values corresponding to indices 8 to 15 of the beacon header tree
	binary.LittleEndian.PutUint64(values[params.BhiSlot-8][:8], bh.Slot)
	binary.LittleEndian.PutUint64(values[params.BhiProposerIndex-8][:8], bh.ProposerIndex)
	values[params.BhiParentRoot-8] = merkle.Value(bh.ParentRoot)
	values[params.BhiStateRoot-8] = merkle.Value(bh.StateRoot)
	values[params.BhiBodyRoot-8] = merkle.Value(bh.BodyRoot)
	return merkle.MultiProof{Format: merkle.NewRangeFormat(8, 15, nil), Values: values[:]}.RootHash()
}

// Epoch returns the epoch the header belongs to
func (bh *Header) Epoch() uint64 {
	return bh.Slot >> params.Log2EpochLength
}

// SyncPeriod returns the sync period the header belongs to
func (bh *Header) SyncPeriod() uint64 {
	return bh.Slot >> params.Log2SyncPeriodLength
}

// PeriodStart returns the first slot of the given period
func PeriodStart(period uint64) uint64 {
	return period << params.Log2SyncPeriodLength
}

// PeriodOfSlot returns the sync period that the given slot belongs to
func PeriodOfSlot(slot uint64) uint64 {
	return slot >> params.Log2SyncPeriodLength
}

// HeaderWithoutState stores beacon header fields except the state root which can
// be reconstructed from a partial beacon state proof stored alongside the header
type HeaderWithoutState struct {
	Slot                 uint64
	ProposerIndex        uint64
	ParentRoot, BodyRoot common.Hash
}

// Hash calculates the block root of the header
func (bh *HeaderWithoutState) Hash(stateRoot common.Hash) common.Hash {
	return bh.Proof(stateRoot).RootHash()
}

// Proof returns a MultiProof of the header
func (bh *HeaderWithoutState) Proof(stateRoot common.Hash) merkle.MultiProof {
	var values [8]merkle.Value // values corresponding to indices 8 to 15 of the beacon header tree
	binary.LittleEndian.PutUint64(values[params.BhiSlot-8][:8], bh.Slot)
	binary.LittleEndian.PutUint64(values[params.BhiProposerIndex-8][:8], bh.ProposerIndex)
	values[params.BhiParentRoot-8] = merkle.Value(bh.ParentRoot)
	values[params.BhiStateRoot-8] = merkle.Value(stateRoot)
	values[params.BhiBodyRoot-8] = merkle.Value(bh.BodyRoot)
	return merkle.MultiProof{Format: merkle.NewRangeFormat(8, 15, nil), Values: values[:]}
}

// FullHeader reconstructs a full Header from a HeaderWithoutState and a state root
func (bh *HeaderWithoutState) FullHeader(stateRoot common.Hash) Header {
	return Header{
		Slot:          bh.Slot,
		ProposerIndex: bh.ProposerIndex,
		ParentRoot:    bh.ParentRoot,
		StateRoot:     stateRoot,
		BodyRoot:      bh.BodyRoot,
	}
}

// SignedHead represents a beacon header signed by a sync committee
//
// Note: this structure is created from either an optimistic update or an instant update:
//  https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/light-client/sync-protocol.md#lightclientoptimisticupdate
//  https://github.com/zsfelfoldi/beacon-APIs/blob/instant_update/apis/beacon/light_client/instant_update.yaml
type SignedHead struct {
	Header        Header        // signed beacon header
	SyncAggregate SyncAggregate // sync committee signature aggregate
	SignatureSlot uint64        // slot in which the signature has been created (newer than Header.Slot, determines the signing sync committee)
}

// SignerCount returns the number of individual signers in the signature aggregate
func (s *SignedHead) SignerCount() int {
	return s.SyncAggregate.SignerCount()
}
