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

package light

import (
	"crypto/rand"
	"crypto/sha256"
	mrand "math/rand"

	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
)

func GenerateTestCommittee() *types.SerializedSyncCommittee {
	s := new(types.SerializedSyncCommittee)
	rand.Read(s[:32])
	return s
}

func GenerateTestUpdate(config *params.ChainConfig, period uint64, committee, nextCommittee *types.SerializedSyncCommittee, signerCount int, finalizedHeader bool) *types.LightClientUpdate {
	update := new(types.LightClientUpdate)
	update.NextSyncCommitteeRoot = nextCommittee.Root()
	var attestedHeader types.Header
	if finalizedHeader {
		update.FinalizedHeader = new(types.Header)
		*update.FinalizedHeader, update.NextSyncCommitteeBranch = makeTestHeaderWithMerkleProof(types.SyncPeriodStart(period)+100, params.StateIndexNextSyncCommittee(""), merkle.Value(update.NextSyncCommitteeRoot))
		attestedHeader, update.FinalityBranch = makeTestHeaderWithMerkleProof(types.SyncPeriodStart(period)+200, params.StateIndexFinalBlock(""), merkle.Value(update.FinalizedHeader.Hash()))
	} else {
		attestedHeader, update.NextSyncCommitteeBranch = makeTestHeaderWithMerkleProof(types.SyncPeriodStart(period)+2000, params.StateIndexNextSyncCommittee(""), merkle.Value(update.NextSyncCommitteeRoot))
	}
	update.AttestedHeader = GenerateTestSignedHeader(attestedHeader, config, committee, attestedHeader.Slot+1, signerCount)
	return update
}

func GenerateTestSignedHeader(header types.Header, config *params.ChainConfig, committee *types.SerializedSyncCommittee, signatureSlot uint64, signerCount int) types.SignedHeader {
	bitmask := makeBitmask(signerCount)
	signingRoot, _ := config.Forks.SigningRoot(header.Epoch(), header.Hash())
	c, _ := dummyVerifier{}.deserializeSyncCommittee(committee)
	return types.SignedHeader{
		Header: header,
		Signature: types.SyncAggregate{
			Signers:   bitmask,
			Signature: makeDummySignature(c.(dummySyncCommittee), signingRoot, bitmask),
		},
		SignatureSlot: signatureSlot,
	}
}

func GenerateTestCheckpoint(period uint64, committee *types.SerializedSyncCommittee) *types.BootstrapData {
	header, branch := makeTestHeaderWithMerkleProof(types.SyncPeriodStart(period)+200, params.StateIndexSyncCommittee(""), merkle.Value(committee.Root()))
	return &types.BootstrapData{
		Header:          header,
		Committee:       committee,
		CommitteeRoot:   committee.Root(),
		CommitteeBranch: branch,
	}
}

func makeBitmask(signerCount int) (bitmask [params.SyncCommitteeBitmaskSize]byte) {
	for i := 0; i < params.SyncCommitteeSize; i++ {
		if mrand.Intn(params.SyncCommitteeSize-i) < signerCount {
			bitmask[i/8] += byte(1) << (i & 7)
			signerCount--
		}
	}
	return
}

func makeTestHeaderWithMerkleProof(slot, index uint64, value merkle.Value) (types.Header, merkle.Values) {
	var branch merkle.Values
	hasher := sha256.New()
	for index > 1 {
		var proofHash merkle.Value
		rand.Read(proofHash[:])
		hasher.Reset()
		if index&1 == 0 {
			hasher.Write(value[:])
			hasher.Write(proofHash[:])
		} else {
			hasher.Write(proofHash[:])
			hasher.Write(value[:])
		}
		hasher.Sum(value[:0])
		index >>= 1
		branch = append(branch, proofHash)
	}
	return types.Header{Slot: slot, StateRoot: common.Hash(value)}, branch
}

// syncCommittee holds either a blsSyncCommittee or a fake dummySyncCommittee used for testing
type syncCommittee interface{}

// committeeSigVerifier verifies sync committee signatures (either proper BLS
// signatures or fake signatures used for testing)
type committeeSigVerifier interface {
	deserializeSyncCommittee(s *types.SerializedSyncCommittee) (syncCommittee, error)
	verifySignature(committee syncCommittee, signedRoot common.Hash, aggregate *types.SyncAggregate) bool
}

// blsVerifier implements committeeSigVerifier
type blsVerifier struct{}

// deserializeSyncCommittee implements committeeSigVerifier
func (blsVerifier) deserializeSyncCommittee(s *types.SerializedSyncCommittee) (syncCommittee, error) {
	return s.Deserialize()
}

// verifySignature implements committeeSigVerifier
func (blsVerifier) verifySignature(committee syncCommittee, signingRoot common.Hash, aggregate *types.SyncAggregate) bool {
	return committee.(*types.SyncCommittee).VerifySignature(signingRoot, aggregate)
}

type dummySyncCommittee [32]byte

// dummyVerifier implements committeeSigVerifier
type dummyVerifier struct{}

// deserializeSyncCommittee implements committeeSigVerifier
func (dummyVerifier) deserializeSyncCommittee(s *types.SerializedSyncCommittee) (syncCommittee, error) {
	var sc dummySyncCommittee
	copy(sc[:], s[:32])
	return sc, nil
}

// verifySignature implements committeeSigVerifier
func (dummyVerifier) verifySignature(committee syncCommittee, signingRoot common.Hash, aggregate *types.SyncAggregate) bool {
	return aggregate.Signature == makeDummySignature(committee.(dummySyncCommittee), signingRoot, aggregate.Signers)
}

func makeDummySignature(committee dummySyncCommittee, signingRoot common.Hash, bitmask [params.SyncCommitteeBitmaskSize]byte) (sig [params.BLSSignatureSize]byte) {
	for i, b := range committee[:] {
		sig[i] = b ^ signingRoot[i]
	}
	copy(sig[32:], bitmask[:])
	return
}
