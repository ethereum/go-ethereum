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

package bintrie

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb/memorydb"
)

// The engine keeps stems prefix-free by binding each zone to one stem length:
// account and code stems are 33 bytes, storage stems are 65, and the zone byte
// says which. The insert and delete walks rely on that - they index into a stem
// at a position derived from the resident node's, and a stem that is a prefix of
// another would take them past its end.
//
// Records come off disk, and under the binary tree pathdb cannot verify them:
// node hashes are not keccak of the blob, so the disk layer returns the zero
// hash and the reader skips the comparison. Structural decoding is therefore the
// only thing standing between a corrupted record and the walks, which is why
// these cases check the decoder rather than the writer.

// leafBlob assembles a leaf record: tag, key, then a 32-byte value.
func leafBlob(zone byte, keyLen int) []byte {
	blob := make([]byte, 0, 1+keyLen+32)
	blob = append(blob, tagLeaf)
	key := make([]byte, keyLen)
	key[0] = zone
	blob = append(blob, key...)
	return append(blob, bytes.Repeat([]byte{0xaa}, 32)...)
}

// groupBlob assembles a two-value group record.
func groupBlob(zone byte, stemLen int) []byte {
	blob := make([]byte, 0, 4+stemLen+bitmapSize+64)
	blob = append(blob, tagGroup, 0, 0, byte(stemLen))
	stem := make([]byte, stemLen)
	stem[0] = zone
	blob = append(blob, stem...)
	var bitmap [bitmapSize]byte
	bitmap[0] = 0xc0 // subs 0 and 1, so the popcount-2 minimum is met
	blob = append(blob, bitmap[:]...)
	return append(blob, bytes.Repeat([]byte{0xbb}, 64)...)
}

// TestDecodeRejectsZoneLengthMismatch pins that a record whose zone byte and
// stem length disagree is refused.
//
// Length alone is not sufficient: account and code stems share a length, so a
// check that only enumerates legal lengths accepts a storage-length stem under
// an account zone, and with it a stem that can prefix another.
func TestDecodeRejectsZoneLengthMismatch(t *testing.T) {
	for _, tc := range []struct {
		name string
		blob []byte
		ok   bool
	}{
		{"leaf/account zone at account length", leafBlob(AccountZone, AccountKeyLength), true},
		{"leaf/code zone at code length", leafBlob(CodeZone, CodeKeyLength), true},
		{"leaf/storage zone at storage length", leafBlob(StorageZone, StorageKeyLength), true},
		{"leaf/account zone at storage length", leafBlob(AccountZone, StorageKeyLength), false},
		{"leaf/storage zone at account length", leafBlob(StorageZone, AccountKeyLength), false},
		{"leaf/unknown zone", leafBlob(0x7f, AccountKeyLength), false},

		{"group/account zone at account length", groupBlob(AccountZone, AccountKeyLength-1), true},
		{"group/storage zone at storage length", groupBlob(StorageZone, StorageKeyLength-1), true},
		{"group/account zone at storage length", groupBlob(AccountZone, StorageKeyLength-1), false},
		{"group/storage zone at account length", groupBlob(StorageZone, AccountKeyLength-1), false},
		{"group/unknown zone", groupBlob(0x7f, AccountKeyLength-1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodeNode(tc.blob)
			if tc.ok && err != nil {
				t.Fatalf("a conformant record was rejected: %v", err)
			}
			if !tc.ok && err == nil {
				t.Fatal("a record whose zone and length disagree was accepted")
			}
		})
	}
}

// TestInsertRejectsPrefixingStem pins the walk's own guard, so the decoder is
// not the only thing preventing an out-of-range read.
//
// insStem indexes a stem at the divergence bit with the resident node. If the
// incoming stem runs out at exactly that point it is a prefix of the resident
// one, and the read goes past its end. getStem has always guarded this; the
// insert path had not, so a stem that reached it reached an index-out-of-range
// panic rather than an error.
//
// It calls insStem directly on purpose. UpdateStem screens stems with
// validateStem before the walk ever sees them, so going through the public
// entry point tests that screen and never reaches this guard - which is the
// whole reason the gap survived. The reachable route in production is a
// decoded record, since PBT cannot hash-verify nodes off disk.
func TestInsertRejectsPrefixingStem(t *testing.T) {
	var (
		tr    = newTestTrie()
		valA  = bytes.Repeat([]byte{1}, 32)
		valB  = bytes.Repeat([]byte{2}, 32)
		full  = make([]byte, StorageKeyLength-1)
		short []byte
	)
	full[0] = StorageZone
	short = full[:AccountKeyLength-1] // a proper prefix of full

	resident, err := tr.insStem(empty{}, full, []byte{0}, [][]byte{valA}, 0)
	if err != nil {
		t.Fatalf("failed to seed the resident stem: %v", err)
	}
	if _, err := tr.insStem(resident, short, []byte{0}, [][]byte{valB}, 0); err == nil {
		t.Fatal("a stem that prefixes a resident stem was accepted")
	} else if !errors.Is(err, ErrNonConformantKey) {
		t.Fatalf("want ErrNonConformantKey, got %v", err)
	}
}

// TestDecodeRejectsOverlongPrefix pins that a branch cannot declare a prefix
// deeper than any legal path.
//
// The bit count is a two-byte field, so a record can claim up to 65535 bits.
// Only the allocation was bounded, against the blob length; the count itself
// then flowed into the position arithmetic the insert and delete walks do.
func TestDecodeRejectsOverlongPrefix(t *testing.T) {
	// A prefix one bit deeper than the longest zone key can reach.
	overlong := make([]byte, 2+((maxPathBits+8)+7)/8)
	binary.BigEndian.PutUint16(overlong, uint16(maxPathBits+1))
	blob := append([]byte{tagBranch}, overlong...)
	blob = append(blob, bytes.Repeat([]byte{1}, 64)...)

	if _, err := decodeNode(blob); err == nil {
		t.Fatal("a branch declaring a prefix deeper than any legal path was accepted")
	}

	// The control: a prefix at exactly the deepest legal path must still decode,
	// or the bound is off by one and rejects valid records.
	atLimit := make([]byte, 2+(maxPathBits+7)/8)
	binary.BigEndian.PutUint16(atLimit, uint16(maxPathBits))
	ok := append([]byte{tagBranch}, atLimit...)
	ok = append(ok, bytes.Repeat([]byte{1}, 64)...)

	if _, err := decodeNode(ok); err != nil {
		t.Fatalf("a prefix at the deepest legal path was rejected: %v", err)
	}
}

// TestVerifyProofRejectsMalformedLeaf pins that a proof node which hashes
// correctly but is not shaped like a leaf is refused rather than sliced.
//
// VerifyProof took the leaf key as preimage[1:len-32]. The hash check before it
// proves only that the bytes are the ones committed to, not that they are long
// enough to hold a key and a value, so a short node produced a negative bound
// and panicked. The caller supplies the root here, which is exactly the position
// an untrusted proof puts them in.
func TestVerifyProofRejectsMalformedLeaf(t *testing.T) {
	for _, tc := range []struct {
		name     string
		preimage []byte
	}{
		{"leaf tag alone", []byte{tagLeaf}},
		{"shorter than a value", append([]byte{tagLeaf}, bytes.Repeat([]byte{0}, 8)...)},
		{"key length not a legal one", append([]byte{tagLeaf}, bytes.Repeat([]byte{0}, 40+32)...)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := memorydb.New()
			root := hashOf(tc.preimage)
			if err := db.Put(root.Bytes(), tc.preimage); err != nil {
				t.Fatal(err)
			}
			key := make([]byte, AccountKeyLength)
			key[0] = AccountZone

			// The contract is an error; a panic here fails the test by crashing.
			if _, err := VerifyProof(root, key, db); err == nil {
				t.Fatal("a malformed proof leaf was accepted")
			}
		})
	}
}

// TestProveEmptyTreeRefused pins that a tree with nothing in it cannot be
// proved: success would hand the caller bytes that drop silently, so an empty
// pre-state has to be recognised by the caller instead.
func TestProveEmptyTreeRefused(t *testing.T) {
	tr := partialTrie(t, empty{})
	if _, err := tr.ProveRequests(ProofRequests{Paths: []ProofPath{{Bits: 0}}}); !errors.Is(err, ErrProofMalformed) {
		t.Fatalf("proving an empty tree: got %v, want ErrProofMalformed", err)
	}
}

// TestRemoveStemValidatesStem pins that a malformed stem is refused before it
// is recorded: ProveRequests fails the whole set on the first bad stem, so an
// unvalidated one would cost the block its proof.
func TestRemoveStemValidatesStem(t *testing.T) {
	_, whole, _ := partialFixture(t, []byte{0x05, 0x80})
	tr := partialTrie(t, whole)
	tr.SetProofRecorder(NewProofRecorder())

	if err := tr.removeStem([]byte{AccountZone, 0x01}); err == nil {
		t.Fatal("removing a stem of the wrong length succeeded")
	}
	// The block's proof must be unaffected: only the seeded root request remains.
	if got := tr.recorder.Len(); got != 1 {
		t.Fatalf("the recorder holds %d requests, want only the seeded root", got)
	}
}
