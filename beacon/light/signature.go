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

package light

import (
	"math/rand"

	"github.com/ethereum/go-ethereum/beacon/light/types"
	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/common"
)

// syncCommittee holds either a blsSyncCommittee or a fake dummySyncCommittee used for testing
type syncCommittee interface{}

// committeeSigVerifier verifies sync committee signatures (either proper BLS
// signatures or fake signatures used for testing)
type committeeSigVerifier interface {
	deserializeSyncCommittee(s *types.SerializedCommittee) (syncCommittee, error)
	verifySignature(committee syncCommittee, signedRoot common.Hash, aggregate *types.SyncAggregate) bool
}

// BLSVerifier implements committeeSigVerifier
type BLSVerifier struct{}

// deserializeSyncCommittee implements committeeSigVerifier
func (BLSVerifier) deserializeSyncCommittee(s *types.SerializedCommittee) (syncCommittee, error) {
	return s.Deserialize()
}

// verifySignature implements committeeSigVerifier
func (BLSVerifier) verifySignature(committee syncCommittee, signingRoot common.Hash, aggregate *types.SyncAggregate) bool {
	return committee.(*types.SyncCommittee).VerifySignature(signingRoot, aggregate)
}

type dummySyncCommittee [32]byte

// dummyVerifier implements committeeSigVerifier
type dummyVerifier struct{}

// deserializeSyncCommittee implements committeeSigVerifier
func (dummyVerifier) deserializeSyncCommittee(s *types.SerializedCommittee) (syncCommittee, error) {
	var sc dummySyncCommittee
	copy(sc[:], s[:32])
	return sc, nil
}

// verifySignature implements committeeSigVerifier
func (dummyVerifier) verifySignature(committee syncCommittee, signingRoot common.Hash, aggregate *types.SyncAggregate) bool {
	return aggregate.Signature == makeDummySignature(committee.(dummySyncCommittee), signingRoot, aggregate.BitMask)
}

func randomDummySyncCommittee() dummySyncCommittee {
	var sc dummySyncCommittee
	rand.Read(sc[:])
	return sc
}

func serializeDummySyncCommittee(sc dummySyncCommittee) *types.SerializedCommittee {
	s := new(types.SerializedCommittee)
	copy(s[:32], sc[:])
	return s
}

func makeDummySignature(committee dummySyncCommittee, signingRoot common.Hash, bitmask [params.SyncCommitteeBitmaskSize]byte) (sig [params.BlsSignatureSize]byte) {
	for i, b := range committee[:] {
		sig[i] = b ^ signingRoot[i]
	}
	copy(sig[32:], bitmask[:])
	return
}
