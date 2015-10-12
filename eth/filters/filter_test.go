package filters

import (
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
)

func makeReceipt(addr common.Address) *types.Receipt {
	receipt := types.NewReceipt(nil, new(big.Int))
	receipt.SetLogs(vm.Logs{
		&vm.Log{Address: addr},
	})
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	return receipt
}

func BenchmarkMipmaps(b *testing.B) {
	const dbname = "/tmp/mipmap"
	var (
		db, _   = ethdb.NewLDBDatabase(dbname, 16)
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		addr2   = common.BytesToAddress([]byte("jeff"))
		addr3   = common.BytesToAddress([]byte("ethereum"))
		addr4   = common.BytesToAddress([]byte("random addresses please"))
	)
	defer func() {
		db.Close()
		os.Remove(dbname)
	}()

	genesis := core.WriteGenesisBlockForTesting(db, core.GenesisAccount{addr1, big.NewInt(1000000)})
	chain := core.GenerateChain(genesis, db, 100000, func(i int, gen *core.BlockGen) {
		var receipts types.Receipts
		switch i {
		case 2403:
			receipt := makeReceipt(addr1)
			receipts = types.Receipts{receipt}
			gen.AddReceipt(receipt)
		case 10340:
			receipt := makeReceipt(addr2)
			receipts = types.Receipts{receipt}
			gen.AddReceipt(receipt)
		case 34:
			receipt := makeReceipt(addr3)
			receipts = types.Receipts{receipt}
			gen.AddReceipt(receipt)
		case 99999:
			receipt := makeReceipt(addr4)
			receipts = types.Receipts{receipt}
			gen.AddReceipt(receipt)

		}

		// store the receipts
		err := core.PutReceipts(db, receipts)
		if err != nil {
			b.Fatal(err)
		}
	})
	for _, block := range chain {
		core.WriteBlock(db, block)
		if err := core.WriteCanonicalHash(db, block.Hash(), block.NumberU64()); err != nil {
			b.Fatalf("failed to insert block number: %v", err)
		}
		if err := core.WriteHeadBlockHash(db, block.Hash()); err != nil {
			b.Fatalf("failed to insert block number: %v", err)
		}
		if err := core.PutBlockReceipts(db, block, block.Receipts()); err != nil {
			b.Fatal("error writing block receipts:", err)
		}
	}

	b.ResetTimer()

	filter := New(db)
	filter.SetAddress([]common.Address{addr1, addr2, addr3, addr4})
	filter.SetEarliestBlock(0)
	filter.SetLatestBlock(-1)

	for i := 0; i < b.N; i++ {
		logs := filter.Find()
		if len(logs) != 4 {
			b.Fatal("expected 4 log, got", len(logs))
		}
	}
}
