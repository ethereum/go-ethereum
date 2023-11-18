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

package main

import (
	"fmt"

	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/log"
)

// store implements the light client state machine LightClientStore from the
// light client specification.
//
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/light-client/sync-protocol.md#lightclientstore
type store struct {
	config *ChainConfig

	finalized  *types.Header
	optimistic *types.Header

	current *types.SyncCommittee
	next    *types.SyncCommittee

	best *LightClientUpdate

	prevActive uint64
	currActive uint64
}

func (s *store) copy() *store {
	shallow := s
	return shallow
}

func (s *store) validate(update *LightClientUpdate) error {
	if update.SyncAggregate.SignerCount() <= MinSyncCommitteeParticipants {
		return errNotEnoughParticipants
	}
	var (
		updatePeriod = types.SyncPeriod(update.SignatureSlot)
		storedPeriod = types.SyncPeriod(s.finalized.Slot)
	)
	// Verify update does not skip a sync committee.
	if updatePeriod != storedPeriod && (s.next == nil || updatePeriod != storedPeriod+1) {
		log.Error("update not from current or next sync committee", "stored", storedPeriod, "update", updatePeriod)
		return errWrongPeriod
	}
	if !(update.AttestedHeader.Slot > s.finalized.Slot || update.NextSyncCommittee != nil) {
		return errUselessUpdate
	}

	// Verify finalized header update, if it exists.
	if update.FinalizedHeader != nil && update.FinalityBranch != nil {
		finalized := merkle.Value{}
		if update.FinalizedHeader.Slot != 0 {
			finalized = merkle.Value(update.FinalizedHeader.Hash())
		}
		if err := merkle.VerifyProof(update.AttestedHeader.StateRoot, params.StateIndexFinalBlock, *update.FinalityBranch, finalized); err != nil {
			return errInvalidFinalityBranch
		}
	}

	// Validate sync committee update, if it exsits.
	if update.NextSyncCommittee != nil && update.NextSyncCommitteeBranch != nil {
		if types.SyncPeriod(update.AttestedHeader.Slot) == storedPeriod && s.next != nil {
			// TODO: maybe check if current next equals this update's next?
		} else if err := merkle.VerifyProof(
			update.AttestedHeader.StateRoot,
			params.StateIndexNextSyncCommittee,
			*update.NextSyncCommitteeBranch,
			merkle.Value(update.NextSyncCommittee.Root()),
		); err != nil {
			return errInvalidNextSyncCommitteeBranch
		}
	}

	// Validate sync committee signature.
	var (
		domain      = s.config.Domain(SyncCommitteeDomain, update.SignatureSlot)
		signingRoot = computeSigningRoot(update.AttestedHeader.Hash(), domain)
	)
	committee := s.current
	if updatePeriod == storedPeriod+1 {
		committee = s.next
	}
	if !committee.VerifySignature(signingRoot, &update.SyncAggregate) {
		return errInvalidSyncCommitteeSignature
	}

	return nil
}

func (s *store) safetyThreshold() int {
	return int(max(s.currActive, s.prevActive) / 2)
}

func (s *store) finalizedPeriod() int {
	return int(types.SyncPeriod(s.finalized.Slot))
}

func (s *store) Insert(update *LightClientUpdate) error {
	if err := s.validate(update); err != nil {
		return err
	}
	if err := s.best.Compare(update); err == nil {
		s.best = update
	}
	s.currActive = max(s.currActive, uint64(update.SyncAggregate.SignerCount()))

	if update.SyncAggregate.SignerCount() > s.safetyThreshold() &&
		update.AttestedHeader.Slot > s.optimistic.Slot {
		s.optimistic = &update.AttestedHeader.Header
	}

	// Process sync committee update.
	if update.NextSyncCommittee != nil {
		var (
			storedPeriod = types.SyncPeriod(s.finalized.Slot)
			updatePeriod = types.SyncPeriod(update.FinalizedHeader.Slot)
		)
		if s.next == nil && storedPeriod == updatePeriod {
			committee, err := update.NextSyncCommittee.Deserialize()
			if err != nil {
				return fmt.Errorf("failed to deserialize next sync committee: %w", err)
			}
			s.next = committee
		} else if updatePeriod == storedPeriod+1 {
			committee, err := update.NextSyncCommittee.Deserialize()
			if err != nil {
				return fmt.Errorf("failed to deserialize next sync committee: %w", err)
			}
			s.current = s.next
			s.next = committee
			s.prevActive = s.currActive
			s.currActive = 0
		}
	}

	// Process finalized header update.
	if update.FinalizedHeader != nil {
		if update.FinalizedHeader.Header.Slot > s.finalized.Slot {
			s.finalized = &update.FinalizedHeader.Header
			var (
				storedPeriod = types.SyncPeriod(s.finalized.Slot)
				updatePeriod = types.SyncPeriod(update.FinalizedHeader.Slot)
			)
			// Shift over sync committee and active ratio when a finalized update
			// moves into a new period.
			if updatePeriod == storedPeriod+1 {
				s.current = s.next
				s.next = nil
				s.prevActive = s.currActive
				s.currActive = 0
			}
		}
	}

	return nil
}

func max(x, y uint64) uint64 {
	if x < y {
		return y
	}
	return x
}
