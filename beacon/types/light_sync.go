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
	"errors"
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/common"
	ctypes "github.com/ethereum/go-ethereum/core/types"
)

// HeadInfo represents an unvalidated new head announcement.
type HeadInfo struct {
	Slot      uint64
	BlockRoot common.Hash
}

// BootstrapData contains a sync committee where light sync can be started,
// together with a proof through a beacon header and corresponding state.
// Note: BootstrapData is fetched from a server based on a known checkpoint hash.
type BootstrapData struct {
	Version         string
	Header          Header
	CommitteeRoot   common.Hash
	Committee       *SerializedSyncCommittee `rlp:"-"`
	CommitteeBranch merkle.Values
}

// Validate verifies the proof included in BootstrapData. The proof is checked at the
// generalized index of the header's fork; the branch may be normalized to the depth of
// a later fork (data served in a later fork's format).
func (c *BootstrapData) Validate(config *params.ChainConfig) error {
	if c.CommitteeRoot != c.Committee.Root() {
		return errors.New("wrong committee root")
	}
	return merkle.VerifyNormalizedProof(c.Header.StateRoot, params.StateIndexSyncCommittee(config.ForkNameAtSlot(c.Header.Slot)), c.CommitteeBranch, merkle.Value(c.CommitteeRoot))
}

// LightClientUpdate is a proof of the next sync committee root based on a header
// signed by the sync committee of the given period. Optionally, the update can
// prove quasi-finality by the signed header referring to a previous, finalized
// header from the same period, and the finalized header referring to the next
// sync committee root.
//
// See data structure definition here:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/light-client/sync-protocol.md#lightclientupdate
type LightClientUpdate struct {
	Version                 string
	AttestedHeader          SignedHeader  // Arbitrary header out of the period signed by the sync committee
	NextSyncCommitteeRoot   common.Hash   // Sync committee of the next period advertised in the current one
	NextSyncCommitteeBranch merkle.Values // Proof for the next period's sync committee

	FinalizedHeader *Header       `rlp:"nil"` // Optional header to announce a point of finality
	FinalityBranch  merkle.Values // Proof for the announced finality

	score *UpdateScore // Weight of the update to compare between competing ones
}

// Validate verifies the validity of the update. The state proofs are checked at the
// generalized indices of the attested header's fork; the branches may be normalized to
// the depth of a later fork (an update served in a later fork's format).
func (update *LightClientUpdate) Validate(config *params.ChainConfig) error {
	period := update.AttestedHeader.Header.SyncPeriod()
	if SyncPeriod(update.AttestedHeader.SignatureSlot) != period {
		return errors.New("signature slot and signed header are from different periods")
	}
	fork := config.ForkNameAtSlot(update.AttestedHeader.Header.Slot)
	if update.FinalizedHeader != nil {
		if update.FinalizedHeader.SyncPeriod() != period {
			return errors.New("finalized header is from different period")
		}
		if err := merkle.VerifyNormalizedProof(update.AttestedHeader.Header.StateRoot, params.StateIndexFinalBlock(fork), update.FinalityBranch, merkle.Value(update.FinalizedHeader.Hash())); err != nil {
			return fmt.Errorf("invalid finalized header proof: %w", err)
		}
	}
	if err := merkle.VerifyNormalizedProof(update.AttestedHeader.Header.StateRoot, params.StateIndexNextSyncCommittee(fork), update.NextSyncCommitteeBranch, merkle.Value(update.NextSyncCommitteeRoot)); err != nil {
		return fmt.Errorf("invalid next sync committee proof: %w", err)
	}
	return nil
}

// Score returns the UpdateScore describing the proof strength of the update
// Note: thread safety can be ensured by always calling Score on a newly received
// or decoded update before making it potentially available for other threads
func (update *LightClientUpdate) Score() UpdateScore {
	if update.score == nil {
		update.score = &UpdateScore{
			SignerCount:     uint32(update.AttestedHeader.Signature.SignerCount()),
			SubPeriodIndex:  uint32(update.AttestedHeader.Header.Slot & 0x1fff),
			FinalizedHeader: update.FinalizedHeader != nil,
		}
	}
	return *update.score
}

// UpdateScore allows the comparison between updates at the same period in order
// to find the best update chain that provides the strongest proof of being canonical.
//
// UpdateScores have a tightly packed binary encoding format for efficient p2p
// protocol transmission. Each UpdateScore is encoded in 3 bytes.
// When interpreted as a 24 bit little indian unsigned integer:
//   - the lowest 10 bits contain the number of signers in the header signature aggregate
//   - the next 13 bits contain the "sub-period index" which is he signed header's
//     slot modulo params.SyncPeriodLength (which is correlated with the risk of the chain being
//     re-orged before the previous period boundary in case of non-finalized updates)
//   - the highest bit is set when the update is finalized (meaning that the finality
//     header referenced by the signed header is in the same period as the signed
//     header, making reorgs before the period boundary impossible
type UpdateScore struct {
	SignerCount     uint32 // number of signers in the header signature aggregate
	SubPeriodIndex  uint32 // signed header's slot modulo params.SyncPeriodLength
	FinalizedHeader bool   // update is considered finalized if has finalized header from the same period and 2/3 signatures
}

// finalized returns true if the update has a header signed by at least 2/3 of
// the committee, referring to a finalized header that refers to the next sync
// committee. This condition is a close approximation of the actual finality
// condition that can only be verified by full beacon nodes.
func (u *UpdateScore) finalized() bool {
	return u.FinalizedHeader && u.SignerCount >= params.SyncCommitteeSupermajority
}

// BetterThan returns true if update u is considered better than w.
func (u UpdateScore) BetterThan(w UpdateScore) bool {
	var (
		uFinalized = u.finalized()
		wFinalized = w.finalized()
	)
	if uFinalized != wFinalized {
		return uFinalized
	}
	return u.SignerCount > w.SignerCount
}

// ExecutionProof proves the execution block hash committed in a beacon block
// body. Its concrete form is fork-specific.
type ExecutionProof interface {
	BlockHash() common.Hash
	// Validate checks the proof against the body root of a block of the given fork.
	Validate(bodyRoot common.Hash, fork string) error
}

// LegacyHeaderProof is the pre-Gloas proof format. It commits to the complete
// execution payload header at the execution_payload body field.
type LegacyHeaderProof struct {
	PayloadHeader *ExecutionHeader
	Branch        merkle.Values
}

func (p *LegacyHeaderProof) BlockHash() common.Hash {
	return p.PayloadHeader.BlockHash()
}

func (p *LegacyHeaderProof) Validate(bodyRoot common.Hash, _ string) error {
	return merkle.VerifyProof(bodyRoot, params.BodyIndexExecPayload, p.Branch, p.PayloadHeader.PayloadRoot())
}

// GloasExecutionProof is the Gloas proof format. It commits to the parent
// execution block hash in signed_execution_payload_bid.message.
type GloasExecutionProof struct {
	ExecutionBlockHash common.Hash
	Branch             merkle.Values
}

func (p *GloasExecutionProof) BlockHash() common.Hash {
	return p.ExecutionBlockHash
}

// Validate checks the execution block hash against the body root of a block of the
// given fork. Gloas headers carry blocks of earlier forks too (around the fork): those
// prove the block hash at their fork's index, with the branch normalized to the Gloas
// depth, and have none before Capella.
func (p *GloasExecutionProof) Validate(bodyRoot common.Hash, fork string) error {
	switch fork {
	case "genesis", "phase0", "altair", "bellatrix":
		if p.ExecutionBlockHash != (common.Hash{}) || slices.ContainsFunc(p.Branch, func(v merkle.Value) bool { return v != merkle.Value{} }) {
			return errors.New("execution block hash before Capella")
		}
		return nil
	}
	index := params.BodyIndexExecBlockHash(fork)
	if index == 0 {
		return fmt.Errorf("unknown fork %q", fork)
	}
	return merkle.VerifyNormalizedProof(bodyRoot, index, p.Branch, merkle.Value(p.ExecutionBlockHash))
}

// HeaderWithExecProof contains a beacon header and its fork-specific execution
// proof.
type HeaderWithExecProof struct {
	Header
	Proof ExecutionProof
}

// BlockHash returns the execution hash committed by the header proof.
func (h *HeaderWithExecProof) BlockHash() common.Hash {
	if h.Proof == nil {
		return common.Hash{}
	}
	return h.Proof.BlockHash()
}

// Validate verifies the fork-specific execution proof.
func (h *HeaderWithExecProof) Validate(config *params.ChainConfig) error {
	if h.Proof == nil {
		return errors.New("missing execution proof")
	}
	return h.Proof.Validate(h.BodyRoot, config.ForkNameAtSlot(h.Slot))
}

// OptimisticUpdate proves sync committee commitment on the attested beacon header.
// It also proves the belonging execution payload header with a Merkle proof.
//
// See data structure definition here:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/light-client/sync-protocol.md#lightclientoptimisticupdate
type OptimisticUpdate struct {
	Attested HeaderWithExecProof

	// Sync committee BLS signature aggregate
	Signature SyncAggregate

	// Slot in which the signature has been created (newer than Header.Slot,
	// determines the signing sync committee)
	SignatureSlot uint64
}

// SignedHeader returns the signed attested header of the update.
func (u *OptimisticUpdate) SignedHeader() SignedHeader {
	return SignedHeader{
		Header:        u.Attested.Header,
		Signature:     u.Signature,
		SignatureSlot: u.SignatureSlot,
	}
}

// Validate verifies the Merkle proof proving the execution payload header.
// Note that the sync committee signature of the attested header should be
// verified separately by a synced committee chain.
func (u *OptimisticUpdate) Validate(config *params.ChainConfig) error {
	return u.Attested.Validate(config)
}

// FinalityUpdate proves a finalized beacon header by a sync committee commitment
// on an attested beacon header, referring to the latest finalized header with a
// Merkle proof.
// It also proves the execution payload header belonging to both the attested and
// the finalized beacon header with Merkle proofs.
//
// See data structure definition here:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/light-client/sync-protocol.md#lightclientfinalityupdate
type FinalityUpdate struct {
	Version             string
	Attested, Finalized HeaderWithExecProof
	FinalityBranch      merkle.Values
	// Sync committee BLS signature aggregate
	Signature SyncAggregate
	// Slot in which the signature has been created (newer than Header.Slot,
	// determines the signing sync committee)
	SignatureSlot uint64
}

// SignedHeader returns the signed attested header of the update.
func (u *FinalityUpdate) SignedHeader() SignedHeader {
	return SignedHeader{
		Header:        u.Attested.Header,
		Signature:     u.Signature,
		SignatureSlot: u.SignatureSlot,
	}
}

// Validate verifies the Merkle proofs proving the finalized beacon header and
// the execution payload headers belonging to the attested and finalized headers.
// The finality proof is checked at the index of the attested header's fork; its
// branch may be normalized to the depth of a later fork.
// Note that the sync committee signature of the attested header should be
// verified separately by a synced committee chain.
func (u *FinalityUpdate) Validate(config *params.ChainConfig) error {
	if err := u.Attested.Validate(config); err != nil {
		return err
	}
	if err := u.Finalized.Validate(config); err != nil {
		return err
	}
	return merkle.VerifyNormalizedProof(u.Attested.StateRoot, params.StateIndexFinalBlock(config.ForkNameAtSlot(u.Attested.Slot)), u.FinalityBranch, merkle.Value(u.Finalized.Hash()))
}

// ChainHeadEvent returns an authenticated execution payload associated with the
// latest accepted head of the beacon chain, along with the hash of the latest
// finalized execution block.
type ChainHeadEvent struct {
	BeaconHead   Header
	Block        *ctypes.Block
	ExecRequests [][]byte    // execution layer requests (added in Electra)
	Finalized    common.Hash // latest finalized block hash
}
