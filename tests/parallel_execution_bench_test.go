package tests

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

func TestParallelBenchmarkOutput(t *testing.T) {
	outPath := os.Getenv("BENCHMARK_OUTPUT_FILE")
	if outPath == "" {
		t.Skip("BENCHMARK_OUTPUT_FILE not set, skipping benchmark tracking output")
	}

	txCounts := []int{50, 150, 300, 450, 600}
	txDepsTypes := []string{"Isolated", "Contended", "Mixed"}

	var results []string
	dateStr := time.Now().Format("2006-01-02")

	for _, n := range txCounts {
		for _, dep := range txDepsTypes {
			var blocks []*types.Block
			var genesis *core.Genesis
			var engine *ethash.Ethash

			switch dep {
			case "Isolated":
				blocks, genesis, engine = createIsolatedBlock(t, n)
			case "Contended":
				blocks, genesis, engine = createContendedBlock(t, n)
			case "Mixed":
				blocks, genesis, engine = createMixedBlock(t, n)
			}

			// Sequential Run
			core.ParallelTxGroupingByStorageOverlap = false
			seqTime := timeInsert(t, blocks, genesis, engine)

			// Parallel Run
			core.ParallelTxGroupingByStorageOverlap = true
			parTime := timeInsert(t, blocks, genesis, engine)

			speedup := float64(seqTime) / float64(parTime)
			seqAvgMs := (seqTime.Seconds() / float64(n)) * 1000
			parAvgMs := (parTime.Seconds() / float64(n)) * 1000
			resLine := fmt.Sprintf("[%s][%d][%s] - Sequential: %.3fs (%.2fms/tx), Parallel: %.3fs (%.2fms/tx), Speedup: %.2fx",
				dateStr, n, dep, seqTime.Seconds(), seqAvgMs, parTime.Seconds(), parAvgMs, speedup)
			t.Log(resLine)
			results = append(results, resLine)
		}
	}

	err := os.MkdirAll(filepath.Dir(outPath), 0755)
	if err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	f, err := os.OpenFile(outPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("failed to open output file: %v", err)
	}
	defer f.Close()

	for _, res := range results {
		if _, err := f.WriteString(res + "\n"); err != nil {
			t.Fatalf("failed to write result: %v", err)
		}
	}
}

func timeInsert(t *testing.T, blocks []*types.Block, genesis *core.Genesis, engine *ethash.Ethash) time.Duration {
	options := &core.BlockChainConfig{
		TrieCleanLimit: 256,
		TrieDirtyLimit: 256,
		TrieTimeLimit:  5 * time.Minute,
		SnapshotLimit:  0,
		Preimages:      true,
		ArchiveMode:    true,
	}

	chain, err := core.NewBlockChain(rawdb.NewMemoryDatabase(), genesis, engine, options)
	if err != nil {
		t.Fatalf("create chain: %v", err)
	}
	defer chain.Stop()

	// Warm up
	if n, err := chain.InsertChain(blocks[:1]); err != nil {
		t.Fatalf("warmup block %d failed: %v", n, err)
	}

	start := time.Now()
	// Insert the actual block with transactions
	if n, err := chain.InsertChain(blocks[1:]); err != nil {
		t.Fatalf("benchmark block %d failed: %v", n, err)
	}
	return time.Since(start)
}

func createIsolatedBlock(t *testing.T, n int) ([]*types.Block, *core.Genesis, *ethash.Ethash) {
	keys, addrs := newTestAccounts(t, n+1)
	deployKey := keys[0]
	deployFrom := addrs[0]

	contractABI := mustReadContractABI(t)
	contractBin := mustReadContractBinBlockTest(t)

	contractAddrs := make([]common.Address, n)
	for i := 0; i < n; i++ {
		contractAddrs[i] = crypto.CreateAddress(deployFrom, uint64(i))
	}

	genesis := &core.Genesis{
		Config:   params.AllEthashProtocolChanges,
		Alloc:    genesisAllocEther(addrs[:]...),
		GasLimit: uint64(n+1) * 20_000_000,
		BaseFee:  big.NewInt(params.InitialBaseFee),
	}

	engine := ethash.NewFaker()

	// Create chain sequentially first to avoid race conditions during test setup
	core.ParallelTxGroupingByStorageOverlap = false

	_, blocks, _ := core.GenerateChainWithGenesis(genesis, engine, 2, func(i int, b *core.BlockGen) {
		switch i {
		case 0:
			for nonce := uint64(0); nonce < uint64(n); nonce++ {
				deployTx := mustSignAccessListTx(t, params.AllEthashProtocolChanges, deployKey, &types.AccessListTx{
					ChainID:  params.AllEthashProtocolChanges.ChainID,
					Nonce:    nonce,
					Gas:      12_000_000,
					GasPrice: b.BaseFee(),
					Data:     contractBin,
				})
				b.AddTx(deployTx)
			}
		case 1:
			const rounds int64 = 1000
			for i := 0; i < n; i++ {
				lane := int64(i + 1)
				senderKey := keys[i+1]
				contract := contractAddrs[i]
				data, _ := contractABI.Pack("isolatedJob", big.NewInt(lane), big.NewInt(1), big.NewInt(rounds))
				tx := mustSignAccessListTx(t, params.AllEthashProtocolChanges, senderKey, &types.AccessListTx{
					ChainID:    params.AllEthashProtocolChanges.ChainID,
					Nonce:      0,
					To:         &contract,
					Gas:        18_000_000,
					GasPrice:   b.BaseFee(),
					Data:       data,
					AccessList: isolatedAccessList(contract, uint64(lane)),
				})
				b.AddTx(tx)
			}
		}
	})
	return blocks, genesis, engine
}

func createContendedBlock(t *testing.T, n int) ([]*types.Block, *core.Genesis, *ethash.Ethash) {
	keys, addrs := newTestAccounts(t, n+1)
	deployKey := keys[0]
	deployFrom := addrs[0]

	contractABI := mustReadContractABI(t)
	contractBin := mustReadContractBinBlockTest(t)

	contractAddr := crypto.CreateAddress(deployFrom, 0)

	genesis := &core.Genesis{
		Config:   params.AllEthashProtocolChanges,
		Alloc:    genesisAllocEther(addrs[:]...),
		GasLimit: uint64(n+1) * 20_000_000,
		BaseFee:  big.NewInt(params.InitialBaseFee),
	}

	engine := ethash.NewFaker()

	core.ParallelTxGroupingByStorageOverlap = false

	_, blocks, _ := core.GenerateChainWithGenesis(genesis, engine, 2, func(i int, b *core.BlockGen) {
		switch i {
		case 0:
			deployTx := mustSignAccessListTx(t, params.AllEthashProtocolChanges, deployKey, &types.AccessListTx{
				ChainID:  params.AllEthashProtocolChanges.ChainID,
				Nonce:    0,
				Gas:      12_000_000,
				GasPrice: b.BaseFee(),
				Data:     contractBin,
			})
			b.AddTx(deployTx)
		case 1:
			const rounds int64 = 1000
			contendedAccess := types.AccessList{{
				Address: contractAddr,
				StorageKeys: []common.Hash{
					common.BigToHash(big.NewInt(2)),
					common.BigToHash(big.NewInt(3)),
				},
			}}
			for i := 0; i < n; i++ {
				senderKey := keys[i+1]
				data, _ := contractABI.Pack("contendedJob", big.NewInt(1), big.NewInt(rounds))
				tx := mustSignAccessListTx(t, params.AllEthashProtocolChanges, senderKey, &types.AccessListTx{
					ChainID:    params.AllEthashProtocolChanges.ChainID,
					Nonce:      0,
					To:         &contractAddr,
					Gas:        18_000_000,
					GasPrice:   b.BaseFee(),
					Data:       data,
					AccessList: contendedAccess,
				})
				b.AddTx(tx)
			}
		}
	})
	return blocks, genesis, engine
}

func createMixedBlock(t *testing.T, n int) ([]*types.Block, *core.Genesis, *ethash.Ethash) {
	keys, addrs := newTestAccounts(t, n+1)
	deployKey := keys[0]
	deployFrom := addrs[0]

	contractABI := mustReadContractABI(t)
	contractBin := mustReadContractBinBlockTest(t)

	cShared := crypto.CreateAddress(deployFrom, 0)

	// Half shared, half isolated. The shared contract is deployed at nonce 0,
	// the rest of the isolated contracts at nonces 1.. n/2.
	numIsolated := n / 2
	numContended := n - numIsolated

	cIsolatedAddrs := make([]common.Address, numIsolated)
	for i := 0; i < numIsolated; i++ {
		cIsolatedAddrs[i] = crypto.CreateAddress(deployFrom, uint64(i+1))
	}

	genesis := &core.Genesis{
		Config:   params.AllEthashProtocolChanges,
		Alloc:    genesisAllocEther(addrs[:]...),
		GasLimit: uint64(n+1) * 20_000_000,
		BaseFee:  big.NewInt(params.InitialBaseFee),
	}

	engine := ethash.NewFaker()

	core.ParallelTxGroupingByStorageOverlap = false

	_, blocks, _ := core.GenerateChainWithGenesis(genesis, engine, 2, func(i int, b *core.BlockGen) {
		switch i {
		case 0:
			// Deploy shared contract
			deployTx0 := mustSignAccessListTx(t, params.AllEthashProtocolChanges, deployKey, &types.AccessListTx{
				ChainID:  params.AllEthashProtocolChanges.ChainID,
				Nonce:    0,
				Gas:      12_000_000,
				GasPrice: b.BaseFee(),
				Data:     contractBin,
			})
			b.AddTx(deployTx0)

			// Deploy isolated contracts
			for i := 0; i < numIsolated; i++ {
				deployTx := mustSignAccessListTx(t, params.AllEthashProtocolChanges, deployKey, &types.AccessListTx{
					ChainID:  params.AllEthashProtocolChanges.ChainID,
					Nonce:    uint64(i + 1),
					Gas:      12_000_000,
					GasPrice: b.BaseFee(),
					Data:     contractBin,
				})
				b.AddTx(deployTx)
			}
		case 1:
			const rounds int64 = 1000

			mixedAccess := func(caddr common.Address, lane uint64) types.AccessList {
				return append(
					isolatedAccessList(caddr, lane),
					types.AccessTuple{
						Address: caddr,
						StorageKeys: []common.Hash{
							common.BigToHash(big.NewInt(2)),
							common.BigToHash(big.NewInt(3)),
						},
					},
				)
			}

			// Add contended transactions
			for i := 0; i < numContended; i++ {
				senderKey := keys[i+1]
				lane := int64(i + 1)
				data, _ := contractABI.Pack("mixedJob", big.NewInt(lane), big.NewInt(lane+1), big.NewInt(rounds))
				tx := mustSignAccessListTx(t, params.AllEthashProtocolChanges, senderKey, &types.AccessListTx{
					ChainID:    params.AllEthashProtocolChanges.ChainID,
					Nonce:      0,
					To:         &cShared,
					Gas:        18_000_000,
					GasPrice:   b.BaseFee(),
					Data:       data,
					AccessList: mixedAccess(cShared, uint64(lane)),
				})
				b.AddTx(tx)
			}

			// Add isolated transactions
			for i := 0; i < numIsolated; i++ {
				senderKey := keys[numContended+1+i]
				lane := int64(i + 1)
				contract := cIsolatedAddrs[i]
				data, _ := contractABI.Pack("isolatedJob", big.NewInt(lane), big.NewInt(1), big.NewInt(rounds))
				tx := mustSignAccessListTx(t, params.AllEthashProtocolChanges, senderKey, &types.AccessListTx{
					ChainID:    params.AllEthashProtocolChanges.ChainID,
					Nonce:      0,
					To:         &contract,
					Gas:        18_000_000,
					GasPrice:   b.BaseFee(),
					Data:       data,
					AccessList: isolatedAccessList(contract, uint64(lane)),
				})
				b.AddTx(tx)
			}
		}
	})
	return blocks, genesis, engine
}
