// Copyright 2023 The go-ethereum Authors
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
// GNU Lesser General Public License for more detaiapi.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package api

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
)

var (
	ErrNotFound = errors.New("404 Not Found")
	ErrInternal = errors.New("500 Internal Server Error")
)

type CommitteeUpdate struct {
	Update            types.LightClientUpdate
	NextSyncCommittee types.SerializedSyncCommittee
}

type jsonBeaconHeader struct {
	Beacon types.Header `json:"beacon"`
}

type jsonHeaderWithExecProof struct {
	Beacon          types.Header    `json:"beacon"`
	Execution       json.RawMessage `json:"execution"`
	ExecutionBranch merkle.Values   `json:"execution_branch"`
}

// See data structure definition here:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/light-client/sync-protocol.md#lightclientupdate
type committeeUpdateJson struct {
	Version string              `json:"version"`
	Data    committeeUpdateData `json:"data"`
}

type committeeUpdateData struct {
	Header                  jsonHeaderWithExecProof       `json:"attested_header"`
	NextSyncCommittee       types.SerializedSyncCommittee `json:"next_sync_committee"`
	NextSyncCommitteeBranch merkle.Values                 `json:"next_sync_committee_branch"`
	FinalizedHeader         *jsonHeaderWithExecProof      `json:"finalized_header,omitempty"`
	FinalityBranch          merkle.Values                 `json:"finality_branch,omitempty"`
	SyncAggregate           types.SyncAggregate           `json:"sync_aggregate"`
	SignatureSlot           common.Decimal                `json:"signature_slot"`
}

func (u *CommitteeUpdate) MarshalJSON() ([]byte, error) {
	execHeader, err := types.ExecutionHeaderToJSON(u.Update.Version, u.Update.AttestedHeader.PayloadHeader)
	if err != nil {
		return nil, err
	}
	enc := committeeUpdateJson{
		Version: u.Update.Version,
		Data: committeeUpdateData{
			Header: jsonHeaderWithExecProof{
				Beacon:          u.Update.AttestedHeader.Header,
				Execution:       execHeader,
				ExecutionBranch: u.Update.AttestedHeader.PayloadBranch,
			},
			NextSyncCommittee:       u.NextSyncCommittee,
			NextSyncCommitteeBranch: u.Update.NextSyncCommitteeBranch,
			SyncAggregate:           u.Update.Signature,
			SignatureSlot:           common.Decimal(u.Update.SignatureSlot),
		},
	}
	if u.Update.FinalizedHeader != nil {
		execHeader, err := types.ExecutionHeaderToJSON(u.Update.Version, u.Update.FinalizedHeader.PayloadHeader)
		if err != nil {
			return nil, err
		}
		enc.Data.FinalizedHeader = &jsonHeaderWithExecProof{
			Beacon:          u.Update.FinalizedHeader.Header,
			Execution:       execHeader,
			ExecutionBranch: u.Update.FinalizedHeader.PayloadBranch,
		}
		enc.Data.FinalityBranch = u.Update.FinalityBranch
	}
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (u *CommitteeUpdate) UnmarshalJSON(input []byte) error {
	var dec committeeUpdateJson
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	u.NextSyncCommittee = dec.Data.NextSyncCommittee
	execHeader, err := types.ExecutionHeaderFromJSON(dec.Version, dec.Data.Header.Execution)
	if err != nil {
		return err
	}
	u.Update = types.LightClientUpdate{
		Version: dec.Version,
		AttestedHeader: types.HeaderWithExecProof{
			Header:        dec.Data.Header.Beacon,
			PayloadHeader: execHeader,
			PayloadBranch: dec.Data.Header.ExecutionBranch,
		},
		Signature:               dec.Data.SyncAggregate,
		SignatureSlot:           uint64(dec.Data.SignatureSlot),
		NextSyncCommitteeRoot:   u.NextSyncCommittee.Root(),
		NextSyncCommitteeBranch: dec.Data.NextSyncCommitteeBranch,
		FinalityBranch:          dec.Data.FinalityBranch,
	}
	if dec.Data.FinalizedHeader != nil {
		execHeader, err := types.ExecutionHeaderFromJSON(dec.Version, dec.Data.FinalizedHeader.Execution)
		if err != nil {
			return err
		}
		u.Update.FinalizedHeader = &types.HeaderWithExecProof{
			Header:        dec.Data.FinalizedHeader.Beacon,
			PayloadHeader: execHeader,
			PayloadBranch: dec.Data.FinalizedHeader.ExecutionBranch,
		}
	}
	return nil
}

type jsonOptimisticUpdate struct {
	Version string `json:"version"`
	Data    struct {
		Attested      jsonHeaderWithExecProof `json:"attested_header"`
		Aggregate     types.SyncAggregate     `json:"sync_aggregate"`
		SignatureSlot common.Decimal          `json:"signature_slot"`
	} `json:"data"`
}

func encodeOptimisticUpdate(update types.OptimisticUpdate) ([]byte, error) {
	var data jsonOptimisticUpdate
	data.Version = update.Version
	attestedHeader, err := types.ExecutionHeaderToJSON(update.Version, update.Attested.PayloadHeader)
	if err != nil {
		return nil, err
	}
	data.Data.Attested = jsonHeaderWithExecProof{
		Beacon:          update.Attested.Header,
		Execution:       attestedHeader,
		ExecutionBranch: update.Attested.PayloadBranch,
	}
	data.Data.Aggregate = update.Signature
	data.Data.SignatureSlot = common.Decimal(update.SignatureSlot)
	return json.Marshal(&data)
}

func decodeOptimisticUpdate(enc []byte) (types.OptimisticUpdate, error) {
	var data jsonOptimisticUpdate
	if err := json.Unmarshal(enc, &data); err != nil {
		return types.OptimisticUpdate{}, err
	}
	// Decode the execution payload headers.
	attestedExecHeader, err := types.ExecutionHeaderFromJSON(data.Version, data.Data.Attested.Execution)
	if err != nil {
		return types.OptimisticUpdate{}, fmt.Errorf("invalid attested header: %v", err)
	}
	if data.Data.Attested.Beacon.StateRoot == (common.Hash{}) {
		// workaround for different event encoding format in Lodestar
		if err := json.Unmarshal(enc, &data.Data); err != nil {
			return types.OptimisticUpdate{}, err
		}
	}

	if len(data.Data.Aggregate.Signers) != params.SyncCommitteeBitmaskSize {
		return types.OptimisticUpdate{}, errors.New("invalid sync_committee_bits length")
	}
	if len(data.Data.Aggregate.Signature) != params.BLSSignatureSize {
		return types.OptimisticUpdate{}, errors.New("invalid sync_committee_signature length")
	}
	return types.OptimisticUpdate{
		Version: data.Version,
		Attested: types.HeaderWithExecProof{
			Header:        data.Data.Attested.Beacon,
			PayloadHeader: attestedExecHeader,
			PayloadBranch: data.Data.Attested.ExecutionBranch,
		},
		Signature:     data.Data.Aggregate,
		SignatureSlot: uint64(data.Data.SignatureSlot),
	}, nil
}

type jsonFinalityUpdate struct {
	Version string `json:"version"`
	Data    struct {
		Attested       jsonHeaderWithExecProof `json:"attested_header"`
		Finalized      jsonHeaderWithExecProof `json:"finalized_header"`
		FinalityBranch merkle.Values           `json:"finality_branch"`
		Aggregate      types.SyncAggregate     `json:"sync_aggregate"`
		SignatureSlot  common.Decimal          `json:"signature_slot"`
	} `json:"data"`
}

func encodeFinalityUpdate(update types.FinalityUpdate) ([]byte, error) {
	var data jsonFinalityUpdate
	data.Version = update.Version
	attestedHeader, err := types.ExecutionHeaderToJSON(update.Version, update.Attested.PayloadHeader)
	if err != nil {
		return nil, err
	}
	finalizedHeader, err := types.ExecutionHeaderToJSON(update.Version, update.Finalized.PayloadHeader)
	if err != nil {
		return nil, err
	}
	data.Data.Attested = jsonHeaderWithExecProof{
		Beacon:          update.Attested.Header,
		Execution:       attestedHeader,
		ExecutionBranch: update.Attested.PayloadBranch,
	}
	data.Data.Finalized = jsonHeaderWithExecProof{
		Beacon:          update.Finalized.Header,
		Execution:       finalizedHeader,
		ExecutionBranch: update.Finalized.PayloadBranch,
	}
	data.Data.FinalityBranch = update.FinalityBranch
	data.Data.Aggregate = update.Signature
	data.Data.SignatureSlot = common.Decimal(update.SignatureSlot)
	return json.Marshal(&data)
}

func decodeFinalityUpdate(enc []byte) (types.FinalityUpdate, error) {
	var data jsonFinalityUpdate
	if err := json.Unmarshal(enc, &data); err != nil {
		return types.FinalityUpdate{}, err
	}
	// Decode the execution payload headers.
	attestedExecHeader, err := types.ExecutionHeaderFromJSON(data.Version, data.Data.Attested.Execution)
	if err != nil {
		return types.FinalityUpdate{}, fmt.Errorf("invalid attested header: %v", err)
	}
	finalizedExecHeader, err := types.ExecutionHeaderFromJSON(data.Version, data.Data.Finalized.Execution)
	if err != nil {
		return types.FinalityUpdate{}, fmt.Errorf("invalid finalized header: %v", err)
	}
	// Perform sanity checks.
	if len(data.Data.Aggregate.Signers) != params.SyncCommitteeBitmaskSize {
		return types.FinalityUpdate{}, errors.New("invalid sync_committee_bits length")
	}
	if len(data.Data.Aggregate.Signature) != params.BLSSignatureSize {
		return types.FinalityUpdate{}, errors.New("invalid sync_committee_signature length")
	}

	return types.FinalityUpdate{
		Version: data.Version,
		Attested: types.HeaderWithExecProof{
			Header:        data.Data.Attested.Beacon,
			PayloadHeader: attestedExecHeader,
			PayloadBranch: data.Data.Attested.ExecutionBranch,
		},
		Finalized: types.HeaderWithExecProof{
			Header:        data.Data.Finalized.Beacon,
			PayloadHeader: finalizedExecHeader,
			PayloadBranch: data.Data.Finalized.ExecutionBranch,
		},
		FinalityBranch: data.Data.FinalityBranch,
		Signature:      data.Data.Aggregate,
		SignatureSlot:  uint64(data.Data.SignatureSlot),
	}, nil
}

type jsonHeadEvent struct {
	Slot  common.Decimal `json:"slot"`
	Block common.Hash    `json:"block"`
}

type jsonBeaconBlock struct {
	Data struct {
		Message capella.BeaconBlock `json:"message"`
	} `json:"data"`
}

// See data structure definition here:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/light-client/sync-protocol.md#lightclientbootstrap
type jsonBootstrapData struct {
	Version string `json:"version"`
	Data    struct {
		Header          jsonBeaconHeader               `json:"header"`
		Committee       *types.SerializedSyncCommittee `json:"current_sync_committee"`
		CommitteeBranch merkle.Values                  `json:"current_sync_committee_branch"`
	} `json:"data"`
}

type jsonHeaderData struct {
	ExecutionOptimistic bool `json:"execution_optimistic"`
	Finalized           bool `json:"finalized"`
	Data                struct {
		Root      common.Hash `json:"root"`
		Canonical bool        `json:"canonical"`
		Header    struct {
			Message   types.Header  `json:"message"`
			Signature hexutil.Bytes `json:"signature"`
		} `json:"header"`
	} `json:"data"`
}
