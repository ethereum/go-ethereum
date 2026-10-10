// Copyright 2026 The go-ethereum Authors
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
	"crypto/sha256"
	"encoding/binary"
	"math/bits"
	"testing"

	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/common"
)

// testTree is a binary Merkle tree whose leaves are deterministic filler, except
// for the nodes set in values; it gives proofs of several nodes against one root.
type testTree struct {
	depth  int
	values map[uint64]merkle.Value
	cache  map[uint64]merkle.Value
}

func newTestTree(values map[uint64]merkle.Value) *testTree {
	t := &testTree{values: values, cache: make(map[uint64]merkle.Value)}
	for index := range values {
		t.depth = max(t.depth, bits.Len64(index)-1)
	}
	return t
}

func (t *testTree) node(index uint64) merkle.Value {
	if v, ok := t.values[index]; ok {
		return v
	}
	if v, ok := t.cache[index]; ok {
		return v
	}
	var v merkle.Value
	if bits.Len64(index)-1 == t.depth {
		binary.LittleEndian.PutUint64(v[:], index)
		v[31] = 0xff // filler leaf
	} else {
		left, right := t.node(2*index), t.node(2*index+1)
		v = sha256.Sum256(append(left[:], right[:]...))
	}
	t.cache[index] = v
	return v
}

func (t *testTree) root() common.Hash { return common.Hash(t.node(1)) }

// branch returns the proof of a node, normalized to depth (zero items prepended).
func (t *testTree) branch(index uint64, depth int) merkle.Values {
	var branch merkle.Values
	for i := index; i > 1; i >>= 1 {
		branch = append(branch, t.node(i^1))
	}
	return append(make(merkle.Values, depth-len(branch)), branch...)
}

// Fork epochs of the test chain, and a slot in each.
const (
	testBellatrixSlot = 15*params.EpochLength + 1
	testCapellaSlot   = 25*params.EpochLength + 2
	testFuluSlot      = 55*params.EpochLength + 3
	testGloasSlot     = 105*params.EpochLength + 4
)

func testChainConfig() *params.ChainConfig {
	c := new(params.ChainConfig)
	for i, fork := range []struct {
		name  string
		epoch uint64
	}{{"GENESIS", 0}, {"ALTAIR", 0}, {"BELLATRIX", 10}, {"CAPELLA", 20}, {"DENEB", 30}, {"ELECTRA", 40}, {"FULU", 50}, {"GLOAS", 100}} {
		c.AddFork(fork.name, fork.epoch, []byte{byte(i), 0, 0, 0})
	}
	return c
}

// gloasDepth returns the depth of a Gloas branch (the light client types' vector length).
func gloasDepth(index uint64) int { return bits.Len64(index) - 1 }

// testGloasHeader returns a header at slot in the Gloas light client format: an
// execution block hash proven at the index of the slot's fork, with the branch
// normalized to the Gloas depth.
func testGloasHeader(config *params.ChainConfig, slot uint64, stateRoot common.Hash) HeaderWithExecProof {
	hash := common.Hash{byte(slot), byte(slot >> 8), 0xee}
	index := params.BodyIndexExecBlockHash(config.ForkNameAtSlot(slot))
	tree := newTestTree(map[uint64]merkle.Value{index: merkle.Value(hash)})
	return HeaderWithExecProof{
		Header: Header{Slot: slot, StateRoot: stateRoot, BodyRoot: tree.root()},
		Proof:  &GloasExecutionProof{ExecutionBlockHash: hash, Branch: tree.branch(index, gloasDepth(params.BodyIndexExecBlockHashGloas))},
	}
}

func TestGloasExecutionProofs(t *testing.T) {
	config := testChainConfig()
	for _, slot := range []uint64{testCapellaSlot, testFuluSlot, testGloasSlot} {
		h := testGloasHeader(config, slot, common.Hash{})
		if err := h.Validate(config); err != nil {
			t.Errorf("header at slot %d (%s): %v", slot, config.ForkNameAtSlot(slot), err)
		}
	}
	// A Fulu header's proof doesn't hold for a Gloas block, nor with a non-zero padding item.
	h := testGloasHeader(config, testFuluSlot, common.Hash{})
	h.Slot = testGloasSlot
	if err := h.Validate(config); err == nil {
		t.Error("Fulu block hash proof accepted for a Gloas block")
	}
	h = testGloasHeader(config, testFuluSlot, common.Hash{})
	h.Proof.(*GloasExecutionProof).Branch[0][0] = 1
	if err := h.Validate(config); err == nil {
		t.Error("non-zero normalization item accepted")
	}
	// Before Capella: no block hash.
	h = HeaderWithExecProof{Header: Header{Slot: testBellatrixSlot}, Proof: &GloasExecutionProof{Branch: make(merkle.Values, 11)}}
	if err := h.Validate(config); err != nil {
		t.Errorf("empty Bellatrix header: %v", err)
	}
	h.Proof.(*GloasExecutionProof).ExecutionBlockHash[0] = 1
	if err := h.Validate(config); err == nil {
		t.Error("Bellatrix header with a block hash accepted")
	}
}

// TestGloasFinalityTransition checks finality updates in the Gloas format around the
// fork: a pre-Gloas finalized header, and a pre-Gloas attested header (the update
// signed in the fork's first slot).
func TestGloasFinalityTransition(t *testing.T) {
	config := testChainConfig()
	gloasFinalityDepth := gloasDepth(params.StateIndexFinalBlockGloas)
	for _, attestedSlot := range []uint64{testGloasSlot, testFuluSlot + 40} {
		finalized := testGloasHeader(config, testFuluSlot, common.Hash{1})
		index := params.StateIndexFinalBlock(config.ForkNameAtSlot(attestedSlot))
		state := newTestTree(map[uint64]merkle.Value{index: merkle.Value(finalized.Hash())})
		update := FinalityUpdate{
			Version:        "gloas",
			Attested:       testGloasHeader(config, attestedSlot, state.root()),
			Finalized:      finalized,
			FinalityBranch: state.branch(index, gloasFinalityDepth),
			SignatureSlot:  attestedSlot + 1,
		}
		if err := update.Validate(config); err != nil {
			t.Errorf("attested header at slot %d (%s): %v", attestedSlot, config.ForkNameAtSlot(attestedSlot), err)
		}
		update.Finalized.Slot++ // a different finalized header
		if err := update.Validate(config); err == nil {
			t.Errorf("attested header at slot %d: forged finalized header accepted", attestedSlot)
		}
	}
}

// TestGloasUpdateTransition checks a committee update and a bootstrap in the Gloas
// format whose (attested) header is from before the fork.
func TestGloasUpdateTransition(t *testing.T) {
	config := testChainConfig()
	attestedSlot := uint64(testFuluSlot + 40)
	finalized := Header{Slot: testFuluSlot, StateRoot: common.Hash{2}}
	committee := merkle.Value{3}
	fork := config.ForkNameAtSlot(attestedSlot)
	finalIndex, nextIndex := params.StateIndexFinalBlock(fork), params.StateIndexNextSyncCommittee(fork)
	state := newTestTree(map[uint64]merkle.Value{finalIndex: merkle.Value(finalized.Hash()), nextIndex: committee})
	update := LightClientUpdate{
		Version: "gloas",
		AttestedHeader: SignedHeader{
			Header:        Header{Slot: attestedSlot, StateRoot: state.root()},
			SignatureSlot: attestedSlot + 1,
		},
		NextSyncCommitteeRoot:   common.Hash(committee),
		NextSyncCommitteeBranch: state.branch(nextIndex, gloasDepth(params.StateIndexNextSyncCommitteeGloas)),
		FinalizedHeader:         &finalized,
		FinalityBranch:          state.branch(finalIndex, gloasDepth(params.StateIndexFinalBlockGloas)),
	}
	if err := update.Validate(config); err != nil {
		t.Fatalf("update with a Fulu attested header: %v", err)
	}
	update.NextSyncCommitteeRoot[0]++
	if err := update.Validate(config); err == nil {
		t.Fatal("forged next committee accepted")
	}

	committees := new(SerializedSyncCommittee)
	committees[0] = 4
	index := params.StateIndexSyncCommittee(config.ForkNameAtSlot(testFuluSlot))
	state = newTestTree(map[uint64]merkle.Value{index: merkle.Value(committees.Root())})
	bootstrap := BootstrapData{
		Version:         "gloas",
		Header:          Header{Slot: testFuluSlot, StateRoot: state.root()},
		CommitteeRoot:   committees.Root(),
		Committee:       committees,
		CommitteeBranch: state.branch(index, gloasDepth(params.StateIndexSyncCommitteeGloas)),
	}
	if err := bootstrap.Validate(config); err != nil {
		t.Fatalf("bootstrap with a Fulu header: %v", err)
	}
	bootstrap.Header.Slot = testGloasSlot
	if err := bootstrap.Validate(config); err == nil {
		t.Fatal("Fulu committee proof accepted for a Gloas header")
	}
}
