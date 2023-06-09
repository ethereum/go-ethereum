// Copyright 2015 The go-ethereum Authors
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

package filters

import (
	"context"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

func makeReceipt(addr common.Address) *types.Receipt {
	receipt := types.NewReceipt(nil, false, 0)
	receipt.Logs = []*types.Log{
		{Address: addr},
	}
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	return receipt
}

func BenchmarkFilters(b *testing.B) {
	var (
		db, _   = rawdb.NewLevelDBDatabase(b.TempDir(), 0, 0, "", false)
		_, sys  = newTestFilterSystem(b, db, Config{})
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		addr2   = common.BytesToAddress([]byte("jeff"))
		addr3   = common.BytesToAddress([]byte("ethereum"))
		addr4   = common.BytesToAddress([]byte("random addresses please"))

		gspec = &core.Genesis{
			Alloc:   core.GenesisAlloc{addr1: {Balance: big.NewInt(1000000)}},
			BaseFee: big.NewInt(params.InitialBaseFee),
			Config:  params.TestChainConfig,
		}
	)
	defer db.Close()
	_, chain, receipts := core.GenerateChainWithGenesis(gspec, ethash.NewFaker(), 100010, func(i int, gen *core.BlockGen) {
		switch i {
		case 2403:
			receipt := makeReceipt(addr1)
			gen.AddUncheckedReceipt(receipt)
			gen.AddUncheckedTx(types.NewTransaction(999, common.HexToAddress("0x999"), big.NewInt(999), 999, gen.BaseFee(), nil))
		case 1034:
			receipt := makeReceipt(addr2)
			gen.AddUncheckedReceipt(receipt)
			gen.AddUncheckedTx(types.NewTransaction(999, common.HexToAddress("0x999"), big.NewInt(999), 999, gen.BaseFee(), nil))
		case 34:
			receipt := makeReceipt(addr3)
			gen.AddUncheckedReceipt(receipt)
			gen.AddUncheckedTx(types.NewTransaction(999, common.HexToAddress("0x999"), big.NewInt(999), 999, gen.BaseFee(), nil))
		case 99999:
			receipt := makeReceipt(addr4)
			gen.AddUncheckedReceipt(receipt)
			gen.AddUncheckedTx(types.NewTransaction(999, common.HexToAddress("0x999"), big.NewInt(999), 999, gen.BaseFee(), nil))
		}
	})
	// The test txs are not properly signed, can't simply create a chain
	// and then import blocks. TODO(rjl493456442) try to get rid of the
	// manual database writes.
	gspec.MustCommit(db)

	for i, block := range chain {
		rawdb.WriteBlock(db, block)
		rawdb.WriteCanonicalHash(db, block.Hash(), block.NumberU64())
		rawdb.WriteHeadBlockHash(db, block.Hash())
		rawdb.WriteReceipts(db, block.Hash(), block.NumberU64(), receipts[i])
	}
	b.ResetTimer()

	filter := sys.NewRangeFilter(0, -1, []common.Address{addr1, addr2, addr3, addr4}, nil)

	for i := 0; i < b.N; i++ {
		logs, _ := filter.Logs(context.Background())
		if len(logs) != 4 {
			b.Fatal("expected 4 logs, got", len(logs))
		}
	}
}

func TestFilters(t *testing.T) {
	var (
		db, _   = rawdb.NewLevelDBDatabase(t.TempDir(), 0, 0, "", false)
		_, sys  = newTestFilterSystem(t, db, Config{})
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr    = crypto.PubkeyToAddress(key1.PublicKey)

		hash1 = common.BytesToHash([]byte("topic1"))
		hash2 = common.BytesToHash([]byte("topic2"))
		hash3 = common.BytesToHash([]byte("topic3"))
		hash4 = common.BytesToHash([]byte("topic4"))

		gspec = &core.Genesis{
			Config:  params.TestChainConfig,
			Alloc:   core.GenesisAlloc{addr: {Balance: big.NewInt(1000000)}},
			BaseFee: big.NewInt(params.InitialBaseFee),
		}
	)
	defer db.Close()

	_, chain, receipts := core.GenerateChainWithGenesis(gspec, ethash.NewFaker(), 1000, func(i int, gen *core.BlockGen) {
		switch i {
		case 1:
			receipt := types.NewReceipt(nil, false, 0)
			receipt.Logs = []*types.Log{
				{
					Address: addr,
					Topics:  []common.Hash{hash1},
				},
			}
			gen.AddUncheckedReceipt(receipt)
			gen.AddUncheckedTx(types.NewTransaction(1, common.HexToAddress("0x1"), big.NewInt(1), 1, gen.BaseFee(), nil))
		case 2:
			receipt := types.NewReceipt(nil, false, 0)
			receipt.Logs = []*types.Log{
				{
					Address: addr,
					Topics:  []common.Hash{hash2},
				},
			}
			gen.AddUncheckedReceipt(receipt)
			gen.AddUncheckedTx(types.NewTransaction(2, common.HexToAddress("0x2"), big.NewInt(2), 2, gen.BaseFee(), nil))

		case 998:
			receipt := types.NewReceipt(nil, false, 0)
			receipt.Logs = []*types.Log{
				{
					Address: addr,
					Topics:  []common.Hash{hash3},
				},
			}
			gen.AddUncheckedReceipt(receipt)
			gen.AddUncheckedTx(types.NewTransaction(998, common.HexToAddress("0x998"), big.NewInt(998), 998, gen.BaseFee(), nil))
		case 999:
			receipt := types.NewReceipt(nil, false, 0)
			receipt.Logs = []*types.Log{
				{
					Address: addr,
					Topics:  []common.Hash{hash4},
				},
			}
			gen.AddUncheckedReceipt(receipt)
			gen.AddUncheckedTx(types.NewTransaction(999, common.HexToAddress("0x999"), big.NewInt(999), 999, gen.BaseFee(), nil))
		}
	})
	// The test txs are not properly signed, can't simply create a chain
	// and then import blocks. TODO(rjl493456442) try to get rid of the
	// manual database writes.
	gspec.MustCommit(db)
	for i, block := range chain {
		rawdb.WriteBlock(db, block)
		rawdb.WriteCanonicalHash(db, block.Hash(), block.NumberU64())
		rawdb.WriteHeadBlockHash(db, block.Hash())
		rawdb.WriteReceipts(db, block.Hash(), block.NumberU64(), receipts[i])
	}

	// Set block 998 as Finalized (-3)
	rawdb.WriteFinalizedBlockHash(db, chain[998].Hash())

	filter := sys.NewRangeFilter(0, -1, []common.Address{addr}, [][]common.Hash{{hash1, hash2, hash3, hash4}})
	logs, _ := filter.Logs(context.Background())
	if len(logs) != 4 {
		t.Error("expected 4 log, got", len(logs))
	}

	for i, tc := range []struct {
		f          *Filter
		wantHashes []common.Hash
	}{
		{
			sys.NewRangeFilter(900, 999, []common.Address{addr}, [][]common.Hash{{hash3}}),
			[]common.Hash{hash3},
		}, {
			sys.NewRangeFilter(990, -1, []common.Address{addr}, [][]common.Hash{{hash3}}),
			[]common.Hash{hash3},
		}, {
			sys.NewRangeFilter(1, 10, nil, [][]common.Hash{{hash1, hash2}}),
			[]common.Hash{hash1, hash2},
		}, {
			sys.NewRangeFilter(0, -1, nil, [][]common.Hash{{common.BytesToHash([]byte("fail"))}}),
			nil,
		}, {
			sys.NewRangeFilter(0, -1, []common.Address{common.BytesToAddress([]byte("failmenow"))}, nil),
			nil,
		}, {
			sys.NewRangeFilter(0, -1, nil, [][]common.Hash{{common.BytesToHash([]byte("fail"))}, {hash1}}),
			nil,
		}, {
			sys.NewRangeFilter(-1, -1, nil, nil), []common.Hash{hash4},
		}, {
			sys.NewRangeFilter(-3, -1, nil, nil), []common.Hash{hash3, hash4},
		}, {
			sys.NewRangeFilter(-3, -3, nil, nil), []common.Hash{hash3},
		}, {
			sys.NewRangeFilter(-1, -3, nil, nil), nil,
		}, {
			sys.NewRangeFilter(-4, -1, nil, nil), nil,
		}, {
			sys.NewRangeFilter(-4, -4, nil, nil), nil,
		}, {
			sys.NewRangeFilter(-1, -4, nil, nil), nil,
		},
	} {
		logs, _ := tc.f.Logs(context.Background())
		var haveHashes []common.Hash
		for _, l := range logs {
			haveHashes = append(haveHashes, l.Topics[0])
		}
		if have, want := len(haveHashes), len(tc.wantHashes); have != want {
			t.Fatalf("test %d, have %d logs, want %d", i, have, want)
		}
		if len(haveHashes) == 0 {
			continue
		}
		if !reflect.DeepEqual(tc.wantHashes, haveHashes) {
			t.Fatalf("test %d, have %v want %v", i, haveHashes, tc.wantHashes)
		}
	}
}
