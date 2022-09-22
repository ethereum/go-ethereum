//go:build integration

package bor

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/params"
)

func TestBorFilters(t *testing.T) {
	t.Parallel()

	var (
		db      = rawdb.NewMemoryDatabase()
		backend = &filters.TestBackend{DB: db}
		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		addr    = crypto.PubkeyToAddress(key1.PublicKey)

		hash1 = common.BytesToHash([]byte("topic1"))
		hash2 = common.BytesToHash([]byte("topic2"))
		hash3 = common.BytesToHash([]byte("topic3"))
		hash4 = common.BytesToHash([]byte("topic4"))
		hash5 = common.BytesToHash([]byte("topic5"))
	)

	defer db.Close()

	genesis := core.GenesisBlockForTesting(db, addr, big.NewInt(1000000))
	testBorConfig := params.TestChainConfig.Bor
	testBorConfig.Sprint = map[string]uint64{
		"0": 4,
		"8": 2,
	}

	chain, receipts := core.GenerateChain(params.TestChainConfig, genesis, ethash.NewFaker(), db, 1000, func(i int, gen *core.BlockGen) {
		switch i {
		case 7: //state-sync tx at block 8
			receipt := types.NewReceipt(nil, false, 0)
			receipt.Logs = []*types.Log{
				{
					Address: addr,
					Topics:  []common.Hash{hash1},
				},
			}
			gen.AddUncheckedReceipt(receipt)
			gen.AddUncheckedTx(types.NewTransaction(8, common.HexToAddress("0x8"), big.NewInt(8), 8, gen.BaseFee(), nil))

		case 23: //state-sync tx at block 24
			receipt := types.NewReceipt(nil, false, 0)
			receipt.Logs = []*types.Log{
				{
					Address: addr,
					Topics:  []common.Hash{hash2},
				},
			}
			gen.AddUncheckedReceipt(receipt)
			gen.AddUncheckedTx(types.NewTransaction(24, common.HexToAddress("0x24"), big.NewInt(24), 24, gen.BaseFee(), nil))

		case 991: //state-sync tx at block 992
			receipt := types.NewReceipt(nil, false, 0)
			receipt.Logs = []*types.Log{
				{
					Address: addr,
					Topics:  []common.Hash{hash3},
				},
			}
			gen.AddUncheckedReceipt(receipt)
			gen.AddUncheckedTx(types.NewTransaction(992, common.HexToAddress("0x992"), big.NewInt(992), 992, gen.BaseFee(), nil))

		case 993: //state-sync tx at block 994
			receipt := types.NewReceipt(nil, false, 0)
			receipt.Logs = []*types.Log{
				{
					Address: addr,
					Topics:  []common.Hash{hash4},
				},
			}
			gen.AddUncheckedReceipt(receipt)
			gen.AddUncheckedTx(types.NewTransaction(994, common.HexToAddress("0x994"), big.NewInt(994), 994, gen.BaseFee(), nil))

		case 999: //state-sync tx at block 1000
			receipt := types.NewReceipt(nil, false, 0)
			receipt.Logs = []*types.Log{
				{
					Address: addr,
					Topics:  []common.Hash{hash5},
				},
			}
			gen.AddUncheckedReceipt(receipt)
			gen.AddUncheckedTx(types.NewTransaction(1000, common.HexToAddress("0x1000"), big.NewInt(1000), 1000, gen.BaseFee(), nil))
		}
	})

	for i, block := range chain {
		// write the block to database
		rawdb.WriteBlock(db, block)
		rawdb.WriteCanonicalHash(db, block.Hash(), block.NumberU64())
		rawdb.WriteHeadBlockHash(db, block.Hash())

		blockBatch := db.NewBatch()

		// since all the transactions are state-sync, we will not include them as normal receipts
		rawdb.WriteReceipts(db, block.Hash(), block.NumberU64(), []*types.Receipt{})

		// check for blocks with receipts. Since the only receipt is state-sync, we can check the length of receipts
		if len(receipts[i]) > 0 {
			// write the state-sync receipts to database
			// State sync logs don't have tx index, tx hash and other necessary fields, DeriveFieldsForBorLogs will fill those fields for websocket subscriptions
			// DeriveFieldsForBorLogs argurments:
			// 1. State-sync logs
			// 2. Block Hash
			// 3. Block Number
			// 4. Transactions in the block(except state-sync) i.e. 0 in our case
			// 5. AllLogs - StateSyncLogs ; since we only have state-sync tx, it will be 0
			types.DeriveFieldsForBorLogs(receipts[i][0].Logs, block.Hash(), block.NumberU64(), uint(0), uint(0))

			rawdb.WriteBorReceipt(blockBatch, block.Hash(), block.NumberU64(), &types.ReceiptForStorage{
				Status: types.ReceiptStatusSuccessful, // make receipt status successful
				Logs:   receipts[i][0].Logs,
			})

			rawdb.WriteBorTxLookupEntry(blockBatch, block.Hash(), block.NumberU64())
		}

		if err := blockBatch.Write(); err != nil {
			fmt.Println("Failed to write block into disk", "err", err)
		}
	}

	filter := filters.NewBorBlockLogsRangeFilter(backend, testBorConfig, 0, -1, []common.Address{addr}, [][]common.Hash{{hash1, hash2, hash3, hash4, hash5}})

	logs, _ := filter.Logs(context.Background())
	if len(logs) != 5 {
		t.Error("expected 5 log, got", len(logs))
	}

	filter = filters.NewBorBlockLogsRangeFilter(backend, testBorConfig, 900, 999, []common.Address{addr}, [][]common.Hash{{hash3}})
	logs, _ = filter.Logs(context.Background())

	if len(logs) != 1 {
		t.Error("expected 1 log, got", len(logs))
	}

	if len(logs) > 0 && logs[0].Topics[0] != hash3 {
		t.Errorf("expected log[0].Topics[0] to be %x, got %x", hash3, logs[0].Topics[0])
	}

	filter = filters.NewBorBlockLogsRangeFilter(backend, testBorConfig, 992, -1, []common.Address{addr}, [][]common.Hash{{hash3}})
	logs, _ = filter.Logs(context.Background())

	if len(logs) != 1 {
		t.Error("expected 1 log, got", len(logs))
	}

	if len(logs) > 0 && logs[0].Topics[0] != hash3 {
		t.Errorf("expected log[0].Topics[0] to be %x, got %x", hash3, logs[0].Topics[0])
	}

	filter = filters.NewBorBlockLogsRangeFilter(backend, testBorConfig, 1, -1, []common.Address{addr}, [][]common.Hash{{hash1, hash2}})

	logs, _ = filter.Logs(context.Background())
	if len(logs) != 2 {
		t.Error("expected 2 log, got", len(logs))
	}

	failHash := common.BytesToHash([]byte("fail"))
	filter = filters.NewBorBlockLogsRangeFilter(backend, testBorConfig, 0, -1, nil, [][]common.Hash{{failHash}})

	logs, _ = filter.Logs(context.Background())
	if len(logs) != 0 {
		t.Error("expected 0 log, got", len(logs))
	}

	failAddr := common.BytesToAddress([]byte("failmenow"))
	filter = filters.NewBorBlockLogsRangeFilter(backend, testBorConfig, 0, -1, []common.Address{failAddr}, nil)

	logs, _ = filter.Logs(context.Background())
	if len(logs) != 0 {
		t.Error("expected 0 log, got", len(logs))
	}

	filter = filters.NewBorBlockLogsRangeFilter(backend, testBorConfig, 0, -1, nil, [][]common.Hash{{failHash}, {hash1}})

	logs, _ = filter.Logs(context.Background())
	if len(logs) != 0 {
		t.Error("expected 0 log, got", len(logs))
	}
}
