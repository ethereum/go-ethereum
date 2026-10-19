// Copyright 2026 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package rawdb

import (
	"bytes"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	coretypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/qkc/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

func TestRootBlockStorage(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	header := &types.RootBlockHeader{
		Number:          7,
		ParentHash:      common.Hash{0x01},
		Coinbase:        common.Address{0x02},
		CoinbaseAmount:  *uint256.NewInt(3),
		Time:            1000,
		Difficulty:      *uint256.NewInt(4),
		TotalDifficulty: *uint256.NewInt(5),
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

func TestXShardDepositsStorage(t *testing.T) {
	var (
		db    = rawdb.NewMemoryDatabase()
		empty = common.Hash{0x01} // a block sending this shard nothing
		full  = common.Hash{0x02}
	)
	deposits := []*coretypes.DepositTx{
		{
			SourceBlockHash: full,
			SourceLogIndex:  0,
			From:            common.Address{0x0a},
			To:              &common.Address{0x0b},
			Value:           uint256.NewInt(1),
			GasPrice:        uint256.NewInt(2),
			Gas:             30000,
			Data:            []byte{0xca, 0xfe},
		},
		{
			SourceBlockHash: full,
			SourceLogIndex:  3,
			From:            common.Address{0x0c},
			To:              &common.Address{0x0d},
			Value:           uint256.NewInt(4),
			GasPrice:        uint256.NewInt(5),
			Gas:             40000,
			Data:            []byte{0xbe, 0xef},
		},
	}
	for _, hash := range []common.Hash{empty, full} {
		if HasXShardDeposits(db, hash) || ReadXShardDeposits(db, hash) != nil {
			t.Fatalf("block %x: deposits stored before writing", hash)
		}
	}
	WriteXShardDeposits(db, empty, nil)
	WriteXShardDeposits(db, full, deposits)

	// An empty list is stored, and read back as an empty list, not as missing.
	if !HasXShardDeposits(db, empty) {
		t.Error("empty deposit list not stored")
	}
	if have := ReadXShardDeposits(db, empty); have == nil || len(have) != 0 {
		t.Errorf("empty deposit list read as %v", have)
	}
	if !HasXShardDeposits(db, full) {
		t.Error("deposit list not stored")
	}
	if have := ReadXShardDeposits(db, full); !reflect.DeepEqual(have, deposits) {
		t.Errorf("read deposits %+v, want %+v", have, deposits)
	}
	// The value is the RLP encoding of the list, keyed by the source block hash.
	want, _ := rlp.EncodeToBytes(deposits)
	if have, _ := db.Get(append([]byte("qkc_xShard"), full.Bytes()...)); !bytes.Equal(have, want) {
		t.Errorf("stored %x, want %x", have, want)
	}
}
