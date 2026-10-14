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

package rawdb

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/qkc/types"
)

func TestRootBlockStorage(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	header := &types.RootBlockHeader{
		Number:          7,
		ParentHash:      common.Hash{0x01},
		Coinbase:        common.Address{0x02},
		CoinbaseAmount:  big.NewInt(3),
		Time:            1000,
		Difficulty:      big.NewInt(4),
		TotalDifficulty: big.NewInt(5),
		Nonce:           6,
		Extra:           []byte("root"),
	}
	minorHeaders := types.MinorBlockHeaders{
		{Number: big.NewInt(1), Difficulty: common.Big0, Extra: []byte("minor")},
	}
	block := types.NewRootBlock(header, minorHeaders, []byte("tracking"))
	hash := block.Hash()

	if HasRootBlock(db, hash) {
		t.Fatal("non existent root block returned")
	}
	if entry := ReadRootBlock(db, hash); entry != nil {
		t.Fatalf("non existent root block returned: %v", entry)
	}
	if entry := ReadRootBlockHeader(db, hash); entry != nil {
		t.Fatalf("non existent root block header returned: %v", entry)
	}
	WriteRootBlock(db, block)

	if !HasRootBlock(db, hash) {
		t.Fatal("stored root block not found")
	}
	entry := ReadRootBlock(db, hash)
	if entry == nil {
		t.Fatal("stored root block not found")
	}
	if entry.Hash() != hash {
		t.Fatalf("retrieved root block mismatch: have %x, want %x", entry.Hash(), hash)
	}
	if len(entry.MinorBlockHeaders()) != 1 || entry.MinorBlockHeaders()[0].Hash() != minorHeaders[0].Hash() {
		t.Fatal("retrieved shard block headers mismatch")
	}
	if string(entry.TrackingData()) != "tracking" {
		t.Fatalf("retrieved tracking data mismatch: have %q", entry.TrackingData())
	}
	headerEntry := ReadRootBlockHeader(db, hash)
	if headerEntry == nil {
		t.Fatal("stored root block header not found")
	}
	if headerEntry.Hash() != hash {
		t.Fatalf("retrieved root block header mismatch: have %x, want %x", headerEntry.Hash(), hash)
	}
}

func TestRootHeadStorage(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	if hash := ReadRootHeadHash(db); hash != (common.Hash{}) {
		t.Fatalf("non existent root head returned: %x", hash)
	}
	want := common.Hash{0x01}
	WriteRootHeadHash(db, want)
	if hash := ReadRootHeadHash(db); hash != want {
		t.Fatalf("root head mismatch: have %x, want %x", hash, want)
	}
}

func TestLastConfirmedStorage(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	root := common.Hash{0x01}

	if hash := ReadLastConfirmedBlockHash(db, root); hash != (common.Hash{}) {
		t.Fatalf("non existent last confirmed block returned: %x", hash)
	}
	want := common.Hash{0x02}
	WriteLastConfirmedBlockHash(db, root, want)
	if hash := ReadLastConfirmedBlockHash(db, root); hash != want {
		t.Fatalf("last confirmed block mismatch: have %x, want %x", hash, want)
	}
	if hash := ReadLastConfirmedBlockHash(db, common.Hash{0x03}); hash != (common.Hash{}) {
		t.Fatalf("last confirmed block of another root returned: %x", hash)
	}
}
