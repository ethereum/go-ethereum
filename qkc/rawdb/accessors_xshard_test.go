// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The QuarkChain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The QuarkChain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the QuarkChain library. If not, see <http://www.gnu.org/licenses/>.

package rawdb

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

func TestXShardDepositsStorage(t *testing.T) {
	var (
		db    = rawdb.NewMemoryDatabase()
		empty = common.Hash{0x01} // a block sending this shard nothing
		full  = common.Hash{0x02}
	)
	deposits := []*types.DepositTx{
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
