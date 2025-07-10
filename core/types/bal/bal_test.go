// Copyright 2025 The go-ethereum Authors
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

package bal

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

func equalBALs(a *BlockAccessList, b *BlockAccessList) bool {
	if len(a.Accounts) != len(b.Accounts) {
		return false
	}
	for addr, aaA := range a.Accounts {
		aaB, ok := b.Accounts[addr]
		if !ok {
			return false
		}
		if !reflect.DeepEqual(aaA.StorageWrites, aaB.StorageWrites) {
			return false
		}
		if !reflect.DeepEqual(aaA.StorageReads, aaB.StorageReads) {
			return false
		}
		if !reflect.DeepEqual(aaA.BalanceChanges, aaB.BalanceChanges) {
			return false
		}
		if !reflect.DeepEqual(aaA.NonceChanges, aaB.NonceChanges) {
			return false
		}
		if !reflect.DeepEqual(aaA.CodeChange, aaB.CodeChange) {
			return false
		}
	}
	return true
}

func makeTestBAL() *BlockAccessList {
	return &BlockAccessList{
		map[common.Address]*AccountAccess{
			common.BytesToAddress([]byte{0xff, 0xff}): {
				StorageWrites: map[common.Hash]map[uint16]common.Hash{
					common.BytesToHash([]byte{0x01}): {
						1: common.BytesToHash([]byte{1, 2, 3, 4}),
						2: common.BytesToHash([]byte{1, 2, 3, 4, 5, 6}),
					},
					common.BytesToHash([]byte{0x10}): {
						20: common.BytesToHash([]byte{1, 2, 3, 4}),
					},
				},
				StorageReads: map[common.Hash]struct{}{
					common.BytesToHash([]byte{1, 2, 3, 4, 5, 6, 7}): {},
				},
				BalanceChanges: map[uint16]*uint256.Int{
					1: uint256.NewInt(100),
					2: uint256.NewInt(500),
				},
				NonceChanges: map[uint16]uint64{
					1: 2,
					2: 6,
				},
				CodeChange: &CodeChange{
					TxIndex: 0,
					Code:    common.Hex2Bytes("deadbeef"),
				},
			},
			common.BytesToAddress([]byte{0xff, 0xff, 0xff}): {
				StorageWrites: map[common.Hash]map[uint16]common.Hash{
					common.BytesToHash([]byte{0x01}): {
						2: common.BytesToHash([]byte{1, 2, 3, 4, 5, 6}),
						3: common.BytesToHash([]byte{1, 2, 3, 4, 5, 6, 7, 8}),
					},
					common.BytesToHash([]byte{0x10}): {
						21: common.BytesToHash([]byte{1, 2, 3, 4, 5}),
					},
				},
				StorageReads: map[common.Hash]struct{}{
					common.BytesToHash([]byte{1, 2, 3, 4, 5, 6, 7, 8}): {},
				},
				BalanceChanges: map[uint16]*uint256.Int{
					2: uint256.NewInt(100),
					3: uint256.NewInt(500),
				},
				NonceChanges: map[uint16]uint64{
					1: 2,
				},
			},
		},
	}
}

// TestBALEncoding tests that a populated access list can be encoded/decoded correctly.
func TestBALEncoding(t *testing.T) {
	var buf bytes.Buffer
	bal := makeTestBAL()
	err := bal.EncodeRLP(&buf)
	if err != nil {
		t.Fatalf("encoding failed: %v\n", err)
	}
	var dec BlockAccessList
	if err := dec.DecodeRLP(rlp.NewStream(bytes.NewReader(buf.Bytes()), 10000000)); err != nil {
		t.Fatalf("decoding failed: %v\n", err)
	}
	if dec.Hash() != bal.Hash() {
		t.Fatalf("encoded block hash doesn't match decoded")
	}
	if !equalBALs(bal, &dec) {
		t.Fatal("decoded BAL doesn't match")
	}
}

// TestBALEncoding tests that a populated access list can be encoded/decoded correctly.
func TestBALFullRLPEncoding(t *testing.T) {
	var buf bytes.Buffer
	bal := makeTestBAL()
	err := bal.EncodeFullRLP(&buf)
	if err != nil {
		t.Fatalf("encoding failed: %v\n", err)
	}
	var dec BlockAccessList
	if err := dec.DecodeFullRLP(rlp.NewStream(bytes.NewReader(buf.Bytes()), 10000000)); err != nil {
		t.Fatalf("decoding failed: %v\n", err)
	}
	if dec.Hash() != bal.Hash() {
		t.Fatalf("encoded block hash doesn't match decoded")
	}
	if !equalBALs(bal, &dec) {
		t.Fatal("decoded BAL doesn't match")
	}
}

// TestBALDecoding tests that a mainnet BAL produced by https://github.com/nerolation/eth-bal-analysis
// can be decoded.
func TestBALDecoding(t *testing.T) {
	filepath.WalkDir("testdata/ssz", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var b BlockAccessList
		if err := b.decodeSSZ(data); err != nil {
			t.Fatal(err)
		}
		return nil
	})
}

func TestBALEncodeSizeDifference(t *testing.T) {
	filepath.WalkDir("testdata/ssz", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var b BlockAccessList
		if err := b.decodeSSZ(data); err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if err := b.EncodeFullRLP(&buf); err != nil {
			t.Fatal(err)
		}
		t.Logf("SSZ: %v, RLP: %v\n", common.StorageSize(len(data)), common.StorageSize(buf.Len()))
		return nil
	})
}
