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
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"encoding/json"
	"errors"
	"math/bits"

	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/minio/sha256-simd"
	bls "github.com/protolambda/bls12-381-util"
)

const SerializedCommitteeSize = (params.SyncCommitteeSize + 1) * params.BlsPubkeySize

type SerializedCommittee [SerializedCommitteeSize]byte

// jsonSyncCommittee is the JSON representation of a sync committee
//
// See data structure definition here:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#syncaggregate
type jsonSyncCommittee struct {
	Pubkeys   []hexutil.Bytes `json:"pubkeys"`
	Aggregate hexutil.Bytes   `json:"aggregate_pubkey"`
}

// MarshalJSON marshals as JSON.
func (s *SerializedCommittee) MarshalJSON() ([]byte, error) {
	sc := jsonSyncCommittee{Pubkeys: make([]hexutil.Bytes, params.SyncCommitteeSize)}
	for i := range sc.Pubkeys {
		sc.Pubkeys[i] = make(hexutil.Bytes, params.BlsPubkeySize)
		copy(sc.Pubkeys[i][:], s[i*params.BlsPubkeySize:(i+1)*params.BlsPubkeySize])
	}
	sc.Aggregate = make(hexutil.Bytes, params.BlsPubkeySize)
	copy(sc.Aggregate[:], s[params.SyncCommitteeSize*params.BlsPubkeySize:])
	return json.Marshal(&sc)
}

// UnmarshalJSON unmarshals from JSON.
func (s *SerializedCommittee) UnmarshalJSON(input []byte) error {
	var sc jsonSyncCommittee
	if err := json.Unmarshal(input, &sc); err != nil {
		return err
	}
	if len(sc.Pubkeys) != params.SyncCommitteeSize {
		return errors.New("Invalid number of pubkeys")
	}
	for i, key := range sc.Pubkeys {
		if len(key) != params.BlsPubkeySize {
			return errors.New("Invalid pubkey size")
		}
		copy(s[i*params.BlsPubkeySize:(i+1)*params.BlsPubkeySize], key[:])
	}
	if len(sc.Aggregate) != params.BlsPubkeySize {
		return errors.New("Invalid pubkey size")
	}
	copy(s[params.SyncCommitteeSize*params.BlsPubkeySize:], sc.Aggregate[:])
	return nil
}

// SerializedCommitteeRoot calculates the root hash of the binary tree representation
// of a sync committee provided in serialized format
func (s *SerializedCommittee) Root() common.Hash {
	var (
		hasher  = sha256.New()
		padding [64 - params.BlsPubkeySize]byte
		data    [params.SyncCommitteeSize]common.Hash
		l       = params.SyncCommitteeSize
	)
	for i := range data {
		hasher.Reset()
		hasher.Write(s[i*params.BlsPubkeySize : (i+1)*params.BlsPubkeySize])
		hasher.Write(padding[:])
		hasher.Sum(data[i][:0])
	}
	for l > 1 {
		for i := 0; i < l/2; i++ {
			hasher.Reset()
			hasher.Write(data[i*2][:])
			hasher.Write(data[i*2+1][:])
			hasher.Sum(data[i][:0])
		}
		l /= 2
	}
	hasher.Reset()
	hasher.Write(s[SerializedCommitteeSize-params.BlsPubkeySize : SerializedCommitteeSize])
	hasher.Write(padding[:])
	hasher.Sum(data[1][:0])
	hasher.Reset()
	hasher.Write(data[0][:])
	hasher.Write(data[1][:])
	hasher.Sum(data[0][:0])
	return data[0]
}

func (s *SerializedCommittee) Deserialize() (*SyncCommittee, error) {
	sc := new(SyncCommittee)
	for i := 0; i <= params.SyncCommitteeSize; i++ {
		pk := new(bls.Pubkey)
		var sk [params.BlsPubkeySize]byte
		copy(sk[:], s[i*params.BlsPubkeySize:(i+1)*params.BlsPubkeySize])
		if err := pk.Deserialize(&sk); err != nil {
			return nil, err
		}
		if i < params.SyncCommitteeSize {
			sc.keys[i] = pk
		} else {
			sc.aggregate = pk
		}
	}
	return sc, nil
}

// SyncCommittee is a set of sync committee signer pubkeys
//
// See data structure definition here:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#syncaggregate
type SyncCommittee struct {
	keys      [params.SyncCommitteeSize]*bls.Pubkey
	aggregate *bls.Pubkey
}

func (sc *SyncCommittee) VerifySignature(signingRoot common.Hash, aggregate *SyncAggregate) bool {
	var (
		sig         bls.Signature
		signerKeys  [params.SyncCommitteeSize]*bls.Pubkey
		signerCount int
	)
	if err := sig.Deserialize(&aggregate.Signature); err != nil {
		return false
	}
	for i, key := range sc.keys {
		if aggregate.BitMask[i/8]&(byte(1)<<(i%8)) != 0 {
			signerKeys[signerCount] = key
			signerCount++
		}
	}
	return bls.FastAggregateVerify(signerKeys[:signerCount], signingRoot[:], &sig)
}

// SyncAggregate represents an aggregated BLS signature with BitMask referring
// to a subset of the corresponding sync committee
//
// See data structure definition here:
// https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/beacon-chain.md#syncaggregate
type SyncAggregate struct {
	BitMask   [params.SyncCommitteeBitmaskSize]byte
	Signature [params.BlsSignatureSize]byte
}

type jsonSyncAggregate struct {
	BitMask   hexutil.Bytes `json:"sync_committee_bits"`
	Signature hexutil.Bytes `json:"sync_committee_signature"`
}

// MarshalJSON marshals as JSON.
func (s *SyncAggregate) MarshalJSON() ([]byte, error) {
	return json.Marshal(&jsonSyncAggregate{
		BitMask:   hexutil.Bytes(s.BitMask[:]),
		Signature: hexutil.Bytes(s.Signature[:]),
	})
}

// UnmarshalJSON unmarshals from JSON.
func (s *SyncAggregate) UnmarshalJSON(input []byte) error {
	var sc jsonSyncAggregate
	if err := json.Unmarshal(input, &sc); err != nil {
		return err
	}
	if len(sc.BitMask) != params.SyncCommitteeBitmaskSize {
		return errors.New("Invalid aggregate bitmask size")
	}
	if len(sc.Signature) != params.BlsSignatureSize {
		return errors.New("Invalid signature size")
	}
	copy(s.BitMask[:], []byte(sc.BitMask))
	copy(s.Signature[:], []byte(sc.Signature))
	return nil
}

func (s *SyncAggregate) SignerCount() int {
	var count int
	for _, v := range s.BitMask {
		count += bits.OnesCount8(v)
	}
	return count
}
