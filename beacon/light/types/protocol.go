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
	"math/bits"

	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/common"
)

const MaxUpdateInfoLength = 128 // max number of advertised update scores of most recent periods

// LightClientUpdate is a proof of the next sync committee root based on a header
// signed by the sync committee of the given period. Optionally the update can
// prove quasi-finality by the signed header referring to a previous, finalized
// header from the same period, and the finalized header referring to the next
// sync committee root.
//
// See data structure definition here:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/light-client/sync-protocol.md#lightclientupdate
type LightClientUpdate struct {
	Header                  Header
	NextSyncCommitteeRoot   common.Hash
	NextSyncCommitteeBranch merkle.Values
	FinalizedHeader         Header
	FinalityBranch          merkle.Values
	SyncCommitteeBits       []byte
	SyncCommitteeSignature  []byte
	score                   UpdateScore // not part of the encoding, calculated after decoding
	scoreCalculated         bool
}

// Validate verifies the validity of the update
func (update *LightClientUpdate) Validate() error {
	if update.hasFinalizedHeader() {
		if update.FinalizedHeader.SyncPeriod() != update.Header.SyncPeriod() {
			return errors.New("finalizedHeader is from previous period") // proves the same committee it is signed by
		}
		if root, ok := merkle.VerifySingleProof(update.FinalityBranch, params.BsiFinalBlock, merkle.Value(update.FinalizedHeader.Hash())); !ok || root != update.Header.StateRoot {
			return errors.New("invalid FinalizedHeader merkle proof")
		}
	}
	if root, ok := merkle.VerifySingleProof(update.NextSyncCommitteeBranch, params.BsiNextSyncCommittee, merkle.Value(update.NextSyncCommitteeRoot)); !ok || root != update.Header.StateRoot {
		return errors.New("invalid NextSyncCommittee merkle proof")
	}
	return nil
}

// hasFinalizedHeader returns true if the update has a finalized header referred
// by the signed header and referring to the next sync committee.
// Note that in addition to this, a sufficient signer participation is also needed
// in order to fulfill the quasi-finality condition (see UpdateScore.isFinalized).
func (l *LightClientUpdate) hasFinalizedHeader() bool {
	return l.FinalizedHeader.BodyRoot != (common.Hash{}) && l.FinalizedHeader.SyncPeriod() == l.Header.SyncPeriod()
}

// Score returns the UpdateScore describing the proof strength of the update
// Note: thread safety can be ensured by always calling Score on a newly received
// or decoded update before making it potentially available for other threads
func (l *LightClientUpdate) Score() UpdateScore {
	if l.scoreCalculated {
		return l.score
	}
	l.score.SignerCount = 0
	for _, v := range l.SyncCommitteeBits {
		l.score.SignerCount += uint32(bits.OnesCount8(v))
	}
	l.score.SubPeriodIndex = uint32(l.Header.Slot & 0x1fff)
	l.score.FinalizedHeader = l.hasFinalizedHeader()
	l.scoreCalculated = true
	return l.score
}

// CommitteeRequest represents a request for fetching updates and committees at the given periods
type CommitteeRequest struct {
	UpdatePeriods    []uint64 // list of periods where LightClientUpdates are requested (not including full sync committee)
	CommitteePeriods []uint64 // list of periods where sync committees are requested
}

// IsEmpty returns true if the request does not request anything
func (req CommitteeRequest) IsEmpty() bool {
	return req.UpdatePeriods == nil && req.CommitteePeriods == nil
}

// CommitteeReply is an answer to a CommitteeRequest, contains the updates and
// committees corresponding to the period numbers in the request in the same order
type CommitteeReply struct {
	Updates    []LightClientUpdate // list of requested LightClientUpdates
	Committees [][]byte            // list of requested sync committees in serialized form
}

// UpdateInfo contains scores for an advertised update chain. Note that the most
// recent updates are always advertised but earliest ones might not because of
// length limitation.
type UpdateInfo struct {
	AfterLastPeriod uint64       // first period not covered by Scores
	Scores          UpdateScores // Scores[i] is the UpdateScore of period AfterLastPeriod-len(Scores)+i
}

func (u UpdateInfo) IsValid() bool {
	return uint64(len(u.Scores)) <= u.AfterLastPeriod
}

func (u UpdateInfo) Range() UpdateRange {
	l := uint64(len(u.Scores))
	if l > u.AfterLastPeriod {
		panic(nil)
	}
	return UpdateRange{First: u.AfterLastPeriod - l, AfterLast: u.AfterLastPeriod}
}

func (u UpdateInfo) HasScore(period uint64) bool {
	return period < u.AfterLastPeriod && period >= u.AfterLastPeriod-uint64(len(u.Scores))
}

func (u UpdateInfo) Score(period uint64) UpdateScore {
	return u.Scores[len(u.Scores)-int(u.AfterLastPeriod-period)]
}

type UpdateRange struct {
	First, AfterLast uint64
}

func (a UpdateRange) Shared(b UpdateRange) UpdateRange {
	if b.First > a.First {
		a.First = b.First
	}
	if b.AfterLast < a.AfterLast {
		a.AfterLast = b.AfterLast
	}
	return a
}

func (a UpdateRange) IsValid() bool {
	return a.AfterLast >= a.First
}

// UpdateScore allows the comparison between updates at the same period in order
// to find the best update chain that provides the strongest proof of being canonical.
//
// UpdateScores have a tightly packed binary encoding format for efficient p2p
// protocol transmission. Each UpdateScore is encoded in 3 bytes.
// When interpreted as a 24 bit little indian unsigned integer:
//  - the lowest 10 bits contain the number of signers in the header signature aggregate
//  - the next 13 bits contain the "sub-period index" which is he signed header's
//    slot modulo params.SyncPeriodLength (which is correlated with the risk of the chain being
//    re-orged before the previous period boundary in case of non-finalized updates)
//  - the highest bit is set when the update is finalized (meaning that the finality
//    header referenced by the signed header is in the same period as the signed
//    header, making reorgs before the period boundary impossible
type UpdateScore struct {
	SignerCount     uint32 // number of signers in the header signature aggregate
	SubPeriodIndex  uint32 // signed header's slot modulo params.SyncPeriodLength
	FinalizedHeader bool   // update is considered finalized if has finalized header from the same period and 2/3 signatures
}

type UpdateScores []UpdateScore

// isFinalized returns true if the update has a header signed by at least 2/3 of
// the committee, referring to a finalized header that refers to the next sync
// committee. This condition is a close approximation of the actual finality
// condition that can only be verified by full beacon nodes.
func (u *UpdateScore) isFinalized() bool {
	return u.FinalizedHeader && u.SignerCount >= params.SyncCommitteeSupermajority
}

// BetterThan returns true if update u is considered better than w.
func (u UpdateScore) BetterThan(w UpdateScore) bool {
	var (
		uFinalized = u.isFinalized()
		wFinalized = w.isFinalized()
	)
	if uFinalized != wFinalized {
		return uFinalized
	}
	return u.SignerCount > w.SignerCount
}
