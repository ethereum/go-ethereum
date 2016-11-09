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
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"golang.org/x/net/context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
)

func makeReceipt(addr common.Address) *types.Receipt {
	receipt := types.NewReceipt(nil, new(big.Int))
	receipt.Logs = vm.Logs{
		&vm.Log{Address: addr},
	}
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	return receipt
}

func BenchmarkMipmaps(b *testing.B) {
	dir, err := ioutil.TempDir("", "mipmap")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)

	var (
		db, _   = ethdb.NewLDBDatabase(dir, 0, 0)
		backend = &testBackend{mux, db}
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		addr2   = common.BytesToAddress([]byte("jeff"))
		addr3   = common.BytesToAddress([]byte("ethereum"))
		addr4   = common.BytesToAddress([]byte("random addresses please"))
	)
	defer db.Close()

	genesis := core.WriteGenesisBlockForTesting(db, core.GenesisAccount{Address: addr1, Balance: big.NewInt(1000000)})
	chain, receipts := core.GenerateChain(nil, genesis, db, 100010, func(i int, gen *core.BlockGen) {
		var receipts types.Receipts
		switch i {
		case 2403:
			receipt := makeReceipt(addr1)
			receipts = types.Receipts{receipt}
			gen.AddUncheckedReceipt(receipt)
		case 1034:
			receipt := makeReceipt(addr2)
			receipts = types.Receipts{receipt}
			gen.AddUncheckedReceipt(receipt)
		case 34:
			receipt := makeReceipt(addr3)
			receipts = types.Receipts{receipt}
			gen.AddUncheckedReceipt(receipt)
		case 99999:
			receipt := makeReceipt(addr4)
			receipts = types.Receipts{receipt}
			gen.AddUncheckedReceipt(receipt)

		}

		// store the receipts
		err := core.WriteReceipts(db, receipts)
		if err != nil {
			b.Fatal(err)
		}
		core.WriteMipmapBloom(db, uint64(i+1), receipts)
	})
	for i, block := range chain {
		core.WriteBlock(db, block)
		if err := core.WriteCanonicalHash(db, block.Hash(), block.NumberU64()); err != nil {
			b.Fatalf("failed to insert block number: %v", err)
		}
		if err := core.WriteHeadBlockHash(db, block.Hash()); err != nil {
			b.Fatalf("failed to insert block number: %v", err)
		}
		if err := core.WriteBlockReceipts(db, block.Hash(), block.NumberU64(), receipts[i]); err != nil {
			b.Fatal("error writing block receipts:", err)
		}
	}
	b.ResetTimer()

	filter := New(backend, true)
	filter.SetAddresses([]common.Address{addr1, addr2, addr3, addr4})
	filter.SetBeginBlock(0)
	filter.SetEndBlock(-1)

	for i := 0; i < b.N; i++ {
		logs, _ := filter.Find(context.Background())
		if len(logs) != 4 {
			b.Fatal("expected 4 log, got", len(logs))
		}
	}
}

func TestFilters(t *testing.T) {
	dir, err := ioutil.TempDir("", "mipmap")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	var (
		db, _   = ethdb.NewLDBDatabase(dir, 0, 0)
		backend = &testBackend{mux, db}
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr    = crypto.PubkeyToAddress(key1.PublicKey)

		hash1 = common.BytesToHash([]byte("topic1"))
		hash2 = common.BytesToHash([]byte("topic2"))
		hash3 = common.BytesToHash([]byte("topic3"))
		hash4 = common.BytesToHash([]byte("topic4"))
	)
	defer db.Close()

	genesis := core.WriteGenesisBlockForTesting(db, core.GenesisAccount{Address: addr, Balance: big.NewInt(1000000)})
	chain, receipts := core.GenerateChain(nil, genesis, db, 1000, func(i int, gen *core.BlockGen) {
		var receipts types.Receipts
		switch i {
		case 1:
			receipt := types.NewReceipt(nil, new(big.Int))
			receipt.Logs = vm.Logs{
				&vm.Log{
					Address: addr,
					Topics:  []common.Hash{hash1},
				},
			}
			gen.AddUncheckedReceipt(receipt)
			receipts = types.Receipts{receipt}
		case 2:
			receipt := types.NewReceipt(nil, new(big.Int))
			receipt.Logs = vm.Logs{
				&vm.Log{
					Address: addr,
					Topics:  []common.Hash{hash2},
				},
			}
			gen.AddUncheckedReceipt(receipt)
			receipts = types.Receipts{receipt}
		case 998:
			receipt := types.NewReceipt(nil, new(big.Int))
			receipt.Logs = vm.Logs{
				&vm.Log{
					Address: addr,
					Topics:  []common.Hash{hash3},
				},
			}
			gen.AddUncheckedReceipt(receipt)
			receipts = types.Receipts{receipt}
		case 999:
			receipt := types.NewReceipt(nil, new(big.Int))
			receipt.Logs = vm.Logs{
				&vm.Log{
					Address: addr,
					Topics:  []common.Hash{hash4},
				},
			}
			gen.AddUncheckedReceipt(receipt)
			receipts = types.Receipts{receipt}
		}

		// store the receipts
		err := core.WriteReceipts(db, receipts)
		if err != nil {
			t.Fatal(err)
		}
		// i is used as block number for the writes but since the i
		// starts at 0 and block 0 (genesis) is already present increment
		// by one
		core.WriteMipmapBloom(db, uint64(i+1), receipts)
	})
	for i, block := range chain {
		core.WriteBlock(db, block)
		if err := core.WriteCanonicalHash(db, block.Hash(), block.NumberU64()); err != nil {
			t.Fatalf("failed to insert block number: %v", err)
		}
		if err := core.WriteHeadBlockHash(db, block.Hash()); err != nil {
			t.Fatalf("failed to insert block number: %v", err)
		}
		if err := core.WriteBlockReceipts(db, block.Hash(), block.NumberU64(), receipts[i]); err != nil {
			t.Fatal("error writing block receipts:", err)
		}
	}

	filter := New(backend, true)
	filter.SetAddresses([]common.Address{addr})
	filter.SetTopics([][]common.Hash{[]common.Hash{hash1, hash2, hash3, hash4}})
	filter.SetBeginBlock(0)
	filter.SetEndBlock(-1)

	logs, _ := filter.Find(context.Background())
	if len(logs) != 4 {
		t.Error("expected 4 log, got", len(logs))
	}

	filter = New(backend, true)
	filter.SetAddresses([]common.Address{addr})
	filter.SetTopics([][]common.Hash{[]common.Hash{hash3}})
	filter.SetBeginBlock(900)
	filter.SetEndBlock(999)
	logs, _ = filter.Find(context.Background())
	if len(logs) != 1 {
		t.Error("expected 1 log, got", len(logs))
	}
	if len(logs) > 0 && logs[0].Topics[0] != hash3 {
		t.Errorf("expected log[0].Topics[0] to be %x, got %x", hash3, logs[0].Topics[0])
	}

	filter = New(backend, true)
	filter.SetAddresses([]common.Address{addr})
	filter.SetTopics([][]common.Hash{[]common.Hash{hash3}})
	filter.SetBeginBlock(990)
	filter.SetEndBlock(-1)
	logs, _ = filter.Find(context.Background())
	if len(logs) != 1 {
		t.Error("expected 1 log, got", len(logs))
	}
	if len(logs) > 0 && logs[0].Topics[0] != hash3 {
		t.Errorf("expected log[0].Topics[0] to be %x, got %x", hash3, logs[0].Topics[0])
	}

	filter = New(backend, true)
	filter.SetTopics([][]common.Hash{[]common.Hash{hash1, hash2}})
	filter.SetBeginBlock(1)
	filter.SetEndBlock(10)

	logs, _ = filter.Find(context.Background())
	if len(logs) != 2 {
		t.Error("expected 2 log, got", len(logs))
	}

	failHash := common.BytesToHash([]byte("fail"))
	filter = New(backend, true)
	filter.SetTopics([][]common.Hash{[]common.Hash{failHash}})
	filter.SetBeginBlock(0)
	filter.SetEndBlock(-1)

	logs, _ = filter.Find(context.Background())
	if len(logs) != 0 {
		t.Error("expected 0 log, got", len(logs))
	}

	failAddr := common.BytesToAddress([]byte("failmenow"))
	filter = New(backend, true)
	filter.SetAddresses([]common.Address{failAddr})
	filter.SetBeginBlock(0)
	filter.SetEndBlock(-1)

	logs, _ = filter.Find(context.Background())
	if len(logs) != 0 {
		t.Error("expected 0 log, got", len(logs))
	}

	filter = New(backend, true)
	filter.SetTopics([][]common.Hash{[]common.Hash{failHash}, []common.Hash{hash1}})
	filter.SetBeginBlock(0)
	filter.SetEndBlock(-1)

	logs, _ = filter.Find(context.Background())
	if len(logs) != 0 {
		t.Error("expected 0 log, got", len(logs))
	}
}
