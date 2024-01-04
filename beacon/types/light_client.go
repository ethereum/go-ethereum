// Copyright 2024 The go-ethereum Authors
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
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/common"
)

// LightClientHeader is a wrapper around a beacon header with special, nested
// marshalling.
type LightClientHeader struct {
	Header
}

type lightClientHeaderMarshaling struct {
	Beacon Header `json:"beacon"`
}

// MarshalJSON marshals as JSON.
func (h LightClientHeader) MarshalJSON() ([]byte, error) {
	var enc lightClientHeaderMarshaling
	enc.Beacon = h.Header
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (h *LightClientHeader) UnmarshalJSON(input []byte) error {
	var dec lightClientHeaderMarshaling
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	*h = LightClientHeader{dec.Beacon}
	return nil
}

// Boostrap is response to the bootstap endpoint in the beacon API.
type Bootstrap struct {
	Header          LightClientHeader        `json:"header"`
	Committee       *SerializedSyncCommittee `json:"current_sync_committee"`
	CommitteeBranch merkle.Values            `json:"current_sync_committee_branch"`
}

// Valid verifies the current committee root is correctly encoded in the beacon
// state of the weak-subjectivity checkpoint.
func (b *Bootstrap) Valid() error {
	root := merkle.Value(b.Committee.Root())
	if err := merkle.VerifyProof(b.Header.StateRoot, params.StateIndexSyncCommittee, b.CommitteeBranch, root); err != nil {
		return err
	}
	return nil
}

type bootstrapMarshaling struct {
	Data struct {
		Header          LightClientHeader        `json:"header"`
		Committee       *SerializedSyncCommittee `json:"current_sync_committee"`
		CommitteeBranch merkle.Values            `json:"current_sync_committee_branch"`
	} `json:"data"`
}

// MarshalJSON marshals as JSON.
func (b Bootstrap) MarshalJSON() ([]byte, error) {
	var enc bootstrapMarshaling
	enc.Data.Header = b.Header
	enc.Data.Committee = b.Committee
	enc.Data.CommitteeBranch = b.CommitteeBranch
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (b *Bootstrap) UnmarshalJSON(input []byte) error {
	var dec bootstrapMarshaling
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	b.Header = dec.Data.Header
	b.Committee = dec.Data.Committee
	b.CommitteeBranch = dec.Data.CommitteeBranch
	return nil
}

// LightClientUpdate represents the possible light client updates the beacon api
// may respond with.
type LightClientUpdate struct {
	AttestedHeader          LightClientHeader        // Arbitrary header out of the period signed by the sync committee
	SyncAggregate           SyncAggregate            // BLS aggregate signature from sync committee
	SignatureSlot           uint64                   // Slot at which the signature is computed
	NextSyncCommittee       *SerializedSyncCommittee // Sync committee of the next period advertised in the current one
	NextSyncCommitteeBranch *merkle.Values           // Proof for the next period's sync committee
	FinalizedHeader         *LightClientHeader       // Optional header to announce a point of finality
	FinalityBranch          *merkle.Values           // Proof for the announced finality
}

type lightClientUpdateMarshaling struct {
	Data struct {
		AttestedHeader          LightClientHeader        `json:"attested_header"`
		SyncAggregate           SyncAggregate            `json:"sync_aggregate"`
		SignatureSlot           common.Decimal           `json:"signature_slot"`
		NextSyncCommittee       *SerializedSyncCommittee `json:"next_sync_committee"`
		NextSyncCommitteeBranch *merkle.Values           `json:"next_sync_committee_branch"`
		FinalizedHeader         *LightClientHeader       `json:"finalized_header"`
		FinalityBranch          *merkle.Values           `json:"finality_branch"`
	} `json:"data"`
}

// MarshalJSON marshals to JSON.
func (u LightClientUpdate) MarshalJSON() ([]byte, error) {
	var enc lightClientUpdateMarshaling
	enc.Data.AttestedHeader = u.AttestedHeader
	enc.Data.SyncAggregate = u.SyncAggregate
	enc.Data.SignatureSlot = common.Decimal(u.SignatureSlot)
	enc.Data.NextSyncCommittee = u.NextSyncCommittee
	enc.Data.NextSyncCommitteeBranch = u.NextSyncCommitteeBranch
	enc.Data.FinalizedHeader = u.FinalizedHeader
	enc.Data.FinalityBranch = u.FinalityBranch
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (u *LightClientUpdate) UnmarshalJSON(input []byte) error {
	var dec lightClientUpdateMarshaling
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	u.AttestedHeader = dec.Data.AttestedHeader
	u.SyncAggregate = dec.Data.SyncAggregate
	u.SignatureSlot = uint64(dec.Data.SignatureSlot)
	u.NextSyncCommittee = dec.Data.NextSyncCommittee
	u.NextSyncCommitteeBranch = dec.Data.NextSyncCommitteeBranch
	u.FinalizedHeader = dec.Data.FinalizedHeader
	u.FinalityBranch = dec.Data.FinalityBranch
	return nil
}

// Compare will compare two light client updates and determine the better one.
// If next is not better than curr, it will error.
func (curr *LightClientUpdate) Compare(next *LightClientUpdate) error {
	if curr == nil {
		// Nothing to compare.
		return nil
	}
	var (
		maxActiveParticipants    = params.SyncCommitteeSize
		newNumActiveParticipants = next.SyncAggregate.SignerCount()
		oldNumActiveParticipants = curr.SyncAggregate.SignerCount()
		newHasSupermajority      = newNumActiveParticipants*3 >= maxActiveParticipants*2
		oldHasSupermajority      = oldNumActiveParticipants*3 >= maxActiveParticipants*2
	)
	if newHasSupermajority && !oldHasSupermajority {
		return nil
	} else if !newHasSupermajority && oldHasSupermajority {
		return fmt.Errorf("new update does not have supermajority while old does")
	}
	if !newHasSupermajority && newNumActiveParticipants > oldNumActiveParticipants {
		return nil
	} else if !newHasSupermajority && newNumActiveParticipants <= oldNumActiveParticipants {
		return fmt.Errorf("more active participants in old update")
	}

	// TODO: implement all tie breakers from spec
	// var (
	// 	sigPeriod                         = slotToSyncCommitteePeriod(next.SignatureSlot)
	// 	newHasRelevantSyncCommitteeUpdate = next.NextSyncCommittee != nil && (slotToSyncCommitteePeriod(next.AttestedHeader.Slot) == sigPeriod)
	// 	oldHasRelevantSyncCommitteeUpdate = curr.NextSyncCommittee != nil && (slotToSyncCommitteePeriod(curr.AttestedHeader.Slot) == sigPeriod)
	// )
	// if !newHasRelevantSyncCommitteeUpdate && oldHasRelevantSyncCommitteeUpdate {
	// 	return fmt.Errorf("old update also includes sync committee update")
	// }

	return nil
}
