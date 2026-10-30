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

package filters

import (
	"context"
	"fmt"
	"math/big"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

func TestFilterLogsManyAddresses(t *testing.T) {
	addresses := make([]common.Address, 100)
	for i := range addresses {
		addresses[i] = common.BigToAddress(big.NewInt(int64(i + 1)))
	}
	// The zero address must work even when it is outside the linear prefix.
	addresses[99] = common.Address{}
	// Duplicated filter entries must not duplicate the output.
	addresses = append(addresses, addresses[0], addresses[99])
	var logs, want []*types.Log
	for i := 0; i < 1000; i++ {
		entry := &types.Log{
			Address:     addresses[(i/8)%100],
			BlockNumber: uint64(i),
			Topics:      []common.Hash{{1}, {2}},
			Removed:     i%2 != 0,
		}
		switch i % 8 {
		case 0, 1, 6, 7: // repeated matched addresses, including removed logs
			if i >= 40 && i <= 860 {
				want = append(want, entry)
			}
		case 2, 3: // repeated unmatched addresses
			entry.Address[0] = 1
		case 4: // unmatched topic after a successful address lookup
			entry.Topics[1] = common.Hash{3}
		case 5: // not enough topics after a successful address lookup
			entry.Topics = entry.Topics[:1]
		}
		logs = append(logs, entry)
	}
	got := filterLogs(logs, big.NewInt(40), big.NewInt(860), addresses, [][]common.Hash{nil, {{2}}})
	if !slices.Equal(got, want) {
		t.Fatal("many-address filter changed the matching logs or their order")
	}
	// No matching logs must retain the nil result used by the other paths.
	if got := filterLogs(logs, big.NewInt(1000), nil, addresses, nil); got != nil {
		t.Fatalf("out-of-range filter returned %d logs", len(got))
	}
}

func benchmarkAddressLogs(n int) ([]common.Address, []*types.Log) {
	addresses := make([]common.Address, n)
	for i := range addresses {
		addresses[i] = common.BigToAddress(big.NewInt(int64(i + 1)))
	}
	logs := make([]*types.Log, 1000)
	for i := range logs {
		logs[i] = &types.Log{
			Address: common.BigToAddress(big.NewInt(int64(i%n + 1))),
			Topics:  []common.Hash{{1}},
		}
		if i%2 != 0 {
			logs[i].Address[0] = 1
		}
	}
	return addresses, logs
}

func BenchmarkFilterLogsAddresses(b *testing.B) {
	for _, n := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			addresses, logs := benchmarkAddressLogs(n)
			b.ReportAllocs()
			for b.Loop() {
				if got := filterLogs(logs, nil, nil, addresses, nil); len(got) != 500 {
					b.Fatalf("got %d logs, want 500", len(got))
				}
			}
		})
	}
}

// BenchmarkBlockLogsAddresses includes header lookup, bloom matching, cached
// receipt filtering, transaction hash derivation and construction of the result.
func BenchmarkBlockLogsAddresses(b *testing.B) {
	for _, n := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			addresses, logs := benchmarkAddressLogs(n)
			db := rawdb.NewMemoryDatabase()
			defer db.Close()
			_, sys := newTestFilterSystem(db, Config{})
			txs := make([]*types.Transaction, 100)
			receipts := make(types.Receipts, len(txs))
			for i := range txs {
				txs[i] = types.NewTx(&types.LegacyTx{Nonce: uint64(i), Gas: 100000, GasPrice: big.NewInt(1)})
				receipts[i] = &types.Receipt{Status: types.ReceiptStatusSuccessful, Logs: logs[i*10 : (i+1)*10]}
				receipts[i].Bloom = types.CreateBloom(receipts[i])
			}
			header := &types.Header{Number: big.NewInt(1), Bloom: types.MergeBloom(receipts)}
			block := types.NewBlockWithHeader(header).WithBody(types.Body{Transactions: txs})
			rawdb.WriteBlock(db, block)
			rawdb.WriteReceipts(db, block.Hash(), 1, receipts)
			ctx := context.Background()
			b.ReportAllocs()
			for b.Loop() {
				filter := sys.NewBlockFilter(block.Hash(), addresses, nil)
				got, err := filter.Logs(ctx)
				if err != nil || len(got) != 500 {
					b.Fatalf("got %d logs and %v, want 500 and nil", len(got), err)
				}
			}
		})
	}
}

func BenchmarkAddressDistribution(b *testing.B) {
	for _, cfg := range []struct{ addresses, logs int }{{1, 1000}, {10, 11}, {10, 1000}, {64, 65}, {65, 260}, {65, 1000}, {100, 101}, {100, 1000}} {
		for _, match := range []string{"first", "boundary", "alternating", "last", "none"} {
			if cfg.addresses < 34 && (match == "boundary" || match == "alternating") {
				continue
			}
			b.Run(fmt.Sprintf("N=%d/Logs=%d/%s", cfg.addresses, cfg.logs, match), func(b *testing.B) {
				addresses := make([]common.Address, cfg.addresses)
				for i := range addresses {
					addresses[i] = common.BigToAddress(big.NewInt(int64(i + 1)))
				}
				logs := make([]*types.Log, cfg.logs)
				address := addresses[0]
				if match == "boundary" || match == "alternating" {
					address = addresses[32]
				}
				if match == "last" {
					address = addresses[len(addresses)-1]
				}
				want := len(logs)
				if match == "none" {
					address[0] = 1
					want = 0
				}
				for i := range logs {
					logs[i] = &types.Log{Address: address}
					if match == "alternating" && i%2 != 0 {
						logs[i].Address = addresses[33]
					}
				}
				b.ReportAllocs()
				for b.Loop() {
					if len(filterLogs(logs, nil, nil, addresses, nil)) != want {
						b.Fatal("wrong matches")
					}
				}
			})
		}
	}
}

// BenchmarkGetLogsRPCAddresses includes JSON request/response encoding over an
// in-process RPC connection. The block and receipts are cached between calls.
func BenchmarkGetLogsRPCAddresses(b *testing.B) {
	for _, matches := range []int{5, 500} {
		for _, n := range []int{1, 10, 100} {
			b.Run(fmt.Sprintf("Matches=%d/N=%d", matches, n), func(b *testing.B) {
				addresses, logs := benchmarkAddressLogs(n)
				if matches == 5 {
					for i, log := range logs {
						if i%200 == 0 {
							log.Address = addresses[(i/200)%n]
						} else {
							log.Address[0] = 1
						}
					}
				}
				db := rawdb.NewMemoryDatabase()
				defer db.Close()
				_, sys := newTestFilterSystem(db, Config{})
				txs := make([]*types.Transaction, 100)
				receipts := make(types.Receipts, len(txs))
				for i := range txs {
					txs[i] = types.NewTx(&types.LegacyTx{Nonce: uint64(i), Gas: 100000, GasPrice: big.NewInt(1)})
					receipts[i] = &types.Receipt{Status: types.ReceiptStatusSuccessful, Logs: logs[i*10 : (i+1)*10]}
					receipts[i].Bloom = types.CreateBloom(receipts[i])
				}
				header := &types.Header{Number: big.NewInt(1), Bloom: types.MergeBloom(receipts)}
				block := types.NewBlockWithHeader(header).WithBody(types.Body{Transactions: txs})
				rawdb.WriteBlock(db, block)
				rawdb.WriteReceipts(db, block.Hash(), 1, receipts)
				api := NewFilterAPI(sys)
				defer api.events.chainSub.Unsubscribe()
				server := rpc.NewServer()
				defer server.Stop()
				if err := server.RegisterName("eth", api); err != nil {
					b.Fatal(err)
				}
				client := rpc.DialInProc(server)
				defer client.Close()
				ctx := context.Background()
				criteria := map[string]any{"blockHash": block.Hash(), "address": addresses}
				b.ReportAllocs()
				for b.Loop() {
					var got []*types.Log
					if err := client.CallContext(ctx, &got, "eth_getLogs", criteria); err != nil {
						b.Fatal(err)
					}
					if len(got) != matches {
						b.Fatalf("got %d logs, want %d", len(got), matches)
					}
				}
			})
		}
	}
}
