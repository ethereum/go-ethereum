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

package params

const (
	EpochLength      = 32
	SyncPeriodLength = 8192

	BLSSignatureSize = 96
	BLSPubkeySize    = 48

	SyncCommitteeSize          = 512
	SyncCommitteeBitmaskSize   = SyncCommitteeSize / 8
	SyncCommitteeSupermajority = (SyncCommitteeSize*2 + 2) / 3
)

const (
	StateIndexFinalBlockLegacy  = 105
	StateIndexFinalBlockElectra = 169
	StateIndexFinalBlockGloas   = 735

	StateIndexSyncCommitteeLegacy  = 54
	StateIndexSyncCommitteeElectra = 86
	StateIndexSyncCommitteeGloas   = 2945

	StateIndexNextSyncCommitteeLegacy  = 55
	StateIndexNextSyncCommitteeElectra = 87
	StateIndexNextSyncCommitteeGloas   = 2946

	BodyIndexExecPayload        = 25
	BodyIndexExecBlockHashGloas = 2856
)

func StateIndexFinalBlock(forkName string) uint64 {
	switch forkName {
	case "bellatrix", "capella", "deneb":
		return StateIndexFinalBlockLegacy
	case "electra", "fulu":
		return StateIndexFinalBlockElectra
	case "gloas":
		return StateIndexFinalBlockGloas
	default:
		return 0 // unknown fork
	}
}
func StateIndexSyncCommittee(forkName string) uint64 {
	switch forkName {
	case "bellatrix", "capella", "deneb":
		return StateIndexSyncCommitteeLegacy
	case "electra", "fulu":
		return StateIndexSyncCommitteeElectra
	case "gloas":
		return StateIndexSyncCommitteeGloas
	default:
		return 0 // unknown fork
	}
}
func StateIndexNextSyncCommittee(forkName string) uint64 {
	switch forkName {
	case "bellatrix", "capella", "deneb":
		return StateIndexNextSyncCommitteeLegacy
	case "electra", "fulu":
		return StateIndexNextSyncCommitteeElectra
	case "gloas":
		return StateIndexNextSyncCommitteeGloas
	default:
		return 0 // unknown fork
	}
}
