package eth

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
)

func TestMipmapUpgrade(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	addr := common.BytesToAddress([]byte("jeff"))
	genesis := core.WriteGenesisBlockForTesting(db)

	chain, receipts := core.GenerateChain(genesis, db, 10, func(i int, gen *core.BlockGen) {
		var receipts types.Receipts
		switch i {
		case 1:
			receipt := types.NewReceipt(nil, new(big.Int))
			receipt.Logs = vm.Logs{&vm.Log{Address: addr}}
			gen.AddUncheckedReceipt(receipt)
			receipts = types.Receipts{receipt}
		case 2:
			receipt := types.NewReceipt(nil, new(big.Int))
			receipt.Logs = vm.Logs{&vm.Log{Address: addr}}
			gen.AddUncheckedReceipt(receipt)
			receipts = types.Receipts{receipt}
		}

		// store the receipts
		err := core.PutReceipts(db, receipts)
		if err != nil {
			t.Fatal(err)
		}
	})
	for i, block := range chain {
		core.WriteBlock(db, block)
		if err := core.WriteCanonicalHash(db, block.Hash(), block.NumberU64()); err != nil {
			t.Fatalf("failed to insert block number: %v", err)
		}
		if err := core.WriteHeadBlockHash(db, block.Hash()); err != nil {
			t.Fatalf("failed to insert block number: %v", err)
		}
		if err := core.PutBlockReceipts(db, block.Hash(), receipts[i]); err != nil {
			t.Fatal("error writing block receipts:", err)
		}
	}

	err := addMipmapBloomBins(db)
	if err != nil {
		t.Fatal(err)
	}

	bloom := core.GetMipmapBloom(db, 1, core.MIPMapLevels[0])
	if (bloom == types.Bloom{}) {
		t.Error("got empty bloom filter")
	}

	data, _ := db.Get([]byte("setting-mipmap-version"))
	if len(data) == 0 {
		t.Error("setting-mipmap-version not written to database")
	}
}
