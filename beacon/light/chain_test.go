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

package light

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"math/bits"
	mrand "math/rand"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	bls "github.com/protolambda/bls12-381-util"
)

func TestStore(t *testing.T) {
	var (
		current = newSyncCommitteeSigner()
		_       = newSyncCommitteeSigner()
		base    = store{
			config:     params.SepoliaChainConfig,
			finalized:  &types.Header{},
			optimistic: &types.Header{},
			current:    current.committee(),
			next:       nil,
		}
	)
	tests := []*test{
		{ // Accept basic updates with various supermajority quorums.
			store: base.copy(),
			generators: []updateGen{
				&single{
					quorum: 1,
					signer: current,
				},
				&single{
					quorum: .9,
					signer: current,
				},
				&single{
					quorum: .8,
					signer: current,
				},
			},
		},
		{ // Reject if no sync committee members sign.
			store: base.copy(),
			generators: []updateGen{
				&single{
					quorum: 0,
					signer: current,
					err:    &errNotEnoughParticipants,
				},
			},
		},
		{ // Accept update with new finalized header and next sync committee.
			store: base.copy(),
			generators: []updateGen{
				&single{
					quorum:    .75,
					signer:    current,
					finalized: makeHeader(base.optimistic),
					next:      newSyncCommitteeSigner().serialized(),
				},
			},
		},
	}
	for i, tt := range tests {
		runTest(t, tt, i)
	}
}

func TestHashTreeRoot(t *testing.T) {
	for i, tt := range []struct {
		input []common.Hash
		root  common.Hash
	}{
		{
			input: []common.Hash{{0x42}, {0x00}},
			root:  hash(common.Hash{0x42}.Bytes(), common.Hash{0x00}.Bytes()),
		},
		{
			input: []common.Hash{{0x42}, {0x13, 0x37}, {0x11}},
			root: hash(
				hash(common.Hash{0x42}.Bytes(), common.Hash{0x13, 0x37}.Bytes()).Bytes(),
				hash(common.Hash{0x11}.Bytes(), common.Hash{0x00}.Bytes()).Bytes(),
			),
		},
	} {
		got := hashTreeRoot(tt.input)
		if got != tt.root {
			t.Fatalf("test %d: mismatched hash tree root: got %s, want %s", i, got, tt.root)
		}
	}
}

func TestFakeStateRoot(t *testing.T) {
	root, fb, nb := fakeStateRoot(common.Hash{42}, common.Hash{13, 37})
	if err := merkle.VerifyProof(root, params.StateIndexFinalBlock, fb, merkle.Value{42}); err != nil {
		t.Fatalf("failed to verify branch2: %v", err)
	}
	if err := merkle.VerifyProof(root, params.StateIndexNextSyncCommittee, nb, merkle.Value{13, 37}); err != nil {
		t.Fatalf("failed to verify branch2: %v", err)
	}
}

// syncCommitteeSigner represents a SyncCommittee and allows for signing
// functionality on behalf of the committee.
type syncCommitteeSigner struct {
	config  *params.ChainConfig
	members []*bls.SecretKey
}

func newSyncCommitteeSigner() *syncCommitteeSigner {
	c := &syncCommitteeSigner{config: params.SepoliaChainConfig}
	for i := 0; i < params.SyncCommitteeSize; i++ {
		var sk bls.SecretKey
		if err := sk.Deserialize(rand32()); err != nil {
			panic(err)
		}
		c.members = append(c.members, &sk)
	}
	return c
}

func (c *syncCommitteeSigner) shuffledCommittee() ([]int, []*bls.SecretKey) {
	var (
		shuffled []*bls.SecretKey
		indexes  []int
	)
	for i, m := range c.members {
		shuffled = append(shuffled, m)
		indexes = append(indexes, i)
	}
	mrand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		indexes[i], indexes[j] = indexes[j], indexes[i]
	})
	return indexes, shuffled
}

func (c *syncCommitteeSigner) signHeader(header *types.Header, quorum float32) types.SyncAggregate {
	var (
		count   = int(quorum * float32(params.SyncCommitteeSize))
		indexes []int
	)
	// Random draw of committee indices.
	for i := 0; i < params.SyncCommitteeSize; i++ {
		indexes = append(indexes, i)
	}
	mrand.Shuffle(len(indexes), func(i, j int) { indexes[i], indexes[j] = indexes[j], indexes[i] })
	sort.Ints(indexes)
	indexes = indexes[:count]

	// Compute aggregate signature.
	var (
		domain  = c.config.Domain(params.SyncCommitteeDomain, header.Slot)
		root    = computeSigningRoot(header.Hash(), domain)
		sigs    = make([]*bls.Signature, count)
		keys    = make([]*bls.Pubkey, count)
		signers [params.SyncCommitteeBitmaskSize]byte
	)
	for i, j := range indexes {
		sigs[i] = bls.Sign(c.members[j], root[:])
		keys[i], _ = bls.SkToPk(c.members[j])
		setBit(signers[:], j)
	}
	var agg bls.Signature
	if len(sigs) != 0 {
		sig, err := bls.Aggregate(sigs)
		if err != nil {
			panic(err)
		}
		agg = *sig
	}
	return types.SyncAggregate{
		Signers:   signers,
		Signature: agg.Serialize(),
	}
}

func (c *syncCommitteeSigner) serialized() *types.SerializedSyncCommittee {
	var keys [params.SyncCommitteeSize]*bls.Pubkey
	for i := range keys {
		pk, err := bls.SkToPk(c.members[i])
		if err != nil {
			panic(err)
		}
		keys[i] = pk
	}
	agg, err := bls.AggregatePubkeys(keys[:])
	if err != nil {
		panic(err)
	}
	var out types.SerializedSyncCommittee
	for i, key := range keys {
		tmp := key.Serialize()
		copy(out[i*48:], tmp[:])
	}
	tmp := agg.Serialize()
	copy(out[len(out)-48:], tmp[:])
	return &out
}

func (c *syncCommitteeSigner) committee() *types.SyncCommittee {
	committee, _ := c.serialized().Deserialize()
	return committee
}

func makeHeader(parent *types.Header) *types.Header {
	h := &types.Header{
		Slot:          parent.Slot + 1,
		ProposerIndex: 42,
		ParentRoot:    parent.Hash(),
		StateRoot:     common.Hash{},
		BodyRoot:      common.Hash{},
	}
	return h
}

// updateGen is an interface representing different test generators.
type updateGen interface {
	gen(*types.Header) []*types.LightClientUpdate
	error() *error
}

// single is test generator which can generate a single custom LightClientUpdate.
type single struct {
	header    *types.Header
	quorum    float32
	signer    *syncCommitteeSigner
	finalized *types.Header
	next      *types.SerializedSyncCommittee
	err       *error
}

// gen generates a single LightClientUpdate given the generation parameters.
func (u *single) gen(parent *types.Header) []*types.LightClientUpdate {
	if u.header == nil {
		u.header = makeHeader(parent)
	}
	var (
		finalizedHeader *types.LightClientHeader
		finalizedBranch *merkle.Values
		next            *types.SerializedSyncCommittee
		nextBranch      *merkle.Values
	)
	if u.finalized != nil || u.next != nil {
		if u.finalized == nil || u.next == nil {
			panic("must provide both finalized and next")
		}
		root, fb, nb := fakeStateRoot(u.finalized.Hash(), u.next.Root())
		u.header.StateRoot = root
		finalizedHeader = &types.LightClientHeader{Header: *u.finalized}
		finalizedBranch = &fb
		next = u.next
		nextBranch = &nb
	}
	return []*types.LightClientUpdate{
		{
			AttestedHeader:          types.LightClientHeader{Header: *u.header},
			SyncAggregate:           u.signer.signHeader(u.header, u.quorum),
			SignatureSlot:           u.header.Slot,
			FinalizedHeader:         finalizedHeader,
			FinalityBranch:          finalizedBranch,
			NextSyncCommittee:       next,
			NextSyncCommitteeBranch: nextBranch,
		},
	}
}

func (u *single) error() *error {
	return u.err
}

type test struct {
	store      *store
	generators []updateGen
}

// runTest executes a test case against the light client store and verifies the
// resulting state of the store.
func runTest(t *testing.T, tt *test, i int) {
	parent := tt.store.optimistic
	for _, g := range tt.generators {
		updates := g.gen(parent)

		// Send all updates to store.
		for _, update := range updates {
			err := tt.store.Insert(update)
			if g.error() != nil && !errors.Is(err, *g.error()) {
				t.Fatalf("test %d: mismatch errors: got %v, want %v", i, err, *g.error())
			}
			if g.error() == nil && err != nil {
				t.Fatalf("test %d: unexpected error: %v", i, err)
			}

			// No error, verify update was successfully accepted.
			if err == nil {
				header := &update.AttestedHeader.Header
				if tt.store.optimistic.Hash() != header.Hash() {
					t.Fatalf("test %d: expected optimistic head to be updated: have %d, want %d", i, tt.store.optimistic.Slot, header.Slot)
				}
				if update.FinalizedHeader != nil && tt.store.finalized.Hash() != update.FinalizedHeader.Hash() {
					t.Fatalf("test %d: expected finalized to be updated: have %d, want %d", i, tt.store.finalized.Slot, update.FinalizedHeader.Slot)
				}
				if update.NextSyncCommittee != nil {
					if tt.store.next == nil {
						t.Fatalf("test %d: expected next sync committe set, but found nil", i)
					}
					var (
						have = tt.store.next.Aggregate.Serialize()
						want = update.NextSyncCommittee[types.SerializedSyncCommitteeSize-48:]
					)
					if !bytes.Equal(have[:], want) {
						t.Fatalf("test %d: mismatched next sync committee: have %s, want %s", i, common.Bytes2Hex(have[:]), common.Bytes2Hex(want))
					}
				}
				parent = header
			}
		}
	}
}

func setBit(bits []byte, pos int) {
	bits[pos/8] |= 1 << (pos % 8)
}

func rand32() *[32]byte {
	var b [32]byte
	_, err := rand.Reader.Read(b[:])
	if err != nil {
		panic(err)
	}
	return &b
}

func hashTreeRoot(leaves []common.Hash) common.Hash {
	var (
		size   = nextPowerOfTwo(len(leaves))
		hasher = sha256.New()
	)
	leaves = append(leaves, make([]common.Hash, size-len(leaves))...)
	for i := 0; i < bits.Len(uint(size)); i++ {
		size = size >> 1
		for j := 0; j < size; j++ {
			hasher.Write(leaves[2*j][:])
			hasher.Write(leaves[2*j+1][:])
			hasher.Sum(leaves[j][:0])
			hasher.Reset()
		}
	}
	return leaves[0]
}

func nextPowerOfTwo(x int) int {
	if x&(x-1) == 0 {
		return x
	}
	return 1 << (bits.Len(uint(x)))
}

func fakeStateRoot(finalized, next common.Hash) (common.Hash, merkle.Values, merkle.Values) {
	var (
		hasher          = sha256.New()
		buf             = make([]common.Hash, 32)
		finalizedIdx    = params.StateIndexFinalBlock
		finalizedBranch merkle.Values
		nextIdx         = params.StateIndexNextSyncCommittee
		nextBranch      merkle.Values
	)
	// Compute root for Checkpoint structure and begin branch.
	hasher.Write(common.Hash{}.Bytes())
	hasher.Write(finalized.Bytes())
	hasher.Sum(buf[20][:0])
	finalizedBranch = append(finalizedBranch, merkle.Value{})
	finalizedIdx >>= 1

	// Write sync committee root in to correct location.
	buf[23] = next

	// Compute the root for the state object.
	for i := 32; i > 1; i /= 2 {
		for j := 0; j < i; j += 2 {
			hasher.Reset()
			if i+j == finalizedIdx || i+j+1 == finalizedIdx {
				if finalizedIdx&1 == 0 {
					finalizedBranch = append(finalizedBranch, merkle.Value(buf[j+1]))
				} else {
					finalizedBranch = append(finalizedBranch, merkle.Value(buf[j]))
				}
				finalizedIdx >>= 1
			}
			if i+j == nextIdx || i+j+1 == nextIdx {
				if nextIdx&1 == 0 {
					nextBranch = append(nextBranch, merkle.Value(buf[j+1]))
				} else {
					nextBranch = append(nextBranch, merkle.Value(buf[j]))
				}
				nextIdx >>= 1
			}
			hasher.Write(buf[j].Bytes())
			hasher.Write(buf[j+1].Bytes())
			hasher.Sum(buf[j/2][:0])
		}
	}
	return buf[0], finalizedBranch, nextBranch
}
