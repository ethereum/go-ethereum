package tests

import (
	"crypto/ecdsa"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// Gas budget for TestParallelVMBlockAccessListIsolated: block limit must exceed the
// sum of per-tx caps when multiple heavy txs share a block. Raising `rounds` in the
// isolatedJob calls may require increasing both constants.
//
// isolatedTestNumIndependentTxs is how many contracts are deployed (deployer nonces
// 0..N-1) and how many parallel-isolated txs run in block 2 (senders keys[1..N],
// contract lanes 1..N). Needs newTestAccounts(t, N+1) for one deployer + N senders.
const (
	isolatedJobTxGasLimit         = uint64(550_000_000)
	isolatedTestNumIndependentTxs = 3
	isolatedTestBlockGasLimit     = (isolatedTestNumIndependentTxs + 1) * isolatedJobTxGasLimit
)

func TestParallelVMBlockAccessListIsolated(t *testing.T) {
	n := isolatedTestNumIndependentTxs
	if n < 1 {
		t.Fatal("isolatedTestNumIndependentTxs must be at least 1")
	}
	keys, addrs := newTestAccounts(t, n+1)
	deployKey := keys[0]
	deployFrom := addrs[0]

	contractABI := mustReadContractABI(t)
	contractBin := mustReadContractBinBlockTest(t)

	// N independent deployments; block-2 txs use distinct recipients and senders
	// (pairwise disjoint declared addresses → one wave when grouping is enabled).
	contractAddrs := make([]common.Address, n)
	for i := 0; i < n; i++ {
		contractAddrs[i] = crypto.CreateAddress(deployFrom, uint64(i))
	}

	genesis := &core.Genesis{
		Config:   params.AllEthashProtocolChanges,
		Alloc:    genesisAllocEther(addrs[:]...),
		GasLimit: isolatedTestBlockGasLimit,
		BaseFee:  big.NewInt(params.InitialBaseFee),
	}

	engine := ethash.NewFaker()

	// Block 1: deploy N contracts (deployer nonces 0..N-1).
	// Block 2: each of N senders calls only its own contract (disjoint address sets).
	_, blocks, _ := core.GenerateChainWithGenesis(genesis, engine, 2, func(i int, b *core.BlockGen) {
		switch i {
		case 0:
			for nonce := uint64(0); nonce < uint64(n); nonce++ {
				deployTx := mustSignAccessListTx(t, params.AllEthashProtocolChanges, deployKey, &types.AccessListTx{
					ChainID:    params.AllEthashProtocolChanges.ChainID,
					Nonce:      nonce,
					To:         nil,
					Gas:        12_000_000,
					GasPrice:   b.BaseFee(),
					Value:      big.NewInt(0),
					Data:       contractBin,
					AccessList: nil,
				})
				b.AddTx(deployTx)
			}

		case 1:
			// Increase isolatedTestBlockGasLimit / isolatedJobTxGasLimit if you raise this.
			const rounds int64 = 120000

			for i := 0; i < n; i++ {
				lane := int64(i + 1)
				senderKey := keys[i+1]
				contract := contractAddrs[i]
				data, err := contractABI.Pack("isolatedJob", big.NewInt(lane), big.NewInt(1), big.NewInt(rounds))
				if err != nil {
					t.Fatalf("pack isolatedJob: %v", err)
				}
				tx := mustSignAccessListTx(t, params.AllEthashProtocolChanges, senderKey, &types.AccessListTx{
					ChainID:    params.AllEthashProtocolChanges.ChainID,
					Nonce:      0,
					To:         &contract,
					Gas:        isolatedJobTxGasLimit,
					GasPrice:   b.BaseFee(),
					Value:      big.NewInt(0),
					Data:       data,
					AccessList: isolatedAccessList(contract, uint64(lane)),
				})
				b.AddTx(tx)
			}
		}
	})

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
		t.Fatalf("failed to create blockchain: %v", err)
	}
	defer chain.Stop()

	if insertN, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("block %d failed to insert: %s", insertN, formatInsertChainErrorForDebug(err))
	}

	// Debug: receipt status / gas (e.g. OOG with high `rounds` shows failed receipts).
	head := chain.CurrentBlock()
	if head != nil {
		if rs := chain.GetReceiptsByHash(head.Hash()); len(rs) > 0 {
			// for i, r := range rs {
			// 	status := "failed"
			// 	if r.Status == types.ReceiptStatusSuccessful {
			// 		status = "successful"
			// 	}
			// t.Logf("head block #%d tx %d: receipt status=%s (raw=%d) gasUsed=%d cumulativeGas=%d txHash=%s",
			// head.Number.Uint64(), i, status, r.Status, r.GasUsed, r.CumulativeGasUsed, r.TxHash.Hex())
			// }
		} else {
			t.Log("head block: no receipts returned (unexpected)")
		}
	}

	// Read final state after block import.
	statedb := mustStateAtCurrentHead(t, chain)

	for i := 0; i < n; i++ {
		lane := uint64(i + 1)
		addr := contractAddrs[i]
		assertStorageUintBlock(t, statedb, addr, mappingSlotBlock(lane, 0), 1)
		assertStorageUintBlock(t, statedb, addr, common.BigToHash(big.NewInt(2)), 0)
	}
}

// TestParallelVMBlockAccessListContended exercises true storage contention: three
// senders each call the same contract, which updates the same global slots (2, 3).
//
// Declared address sets all include that contract → address-parallel grouping must
// not put these txs in one wave ([0]|[1]|[2] with current greedy builder).
//
// Sequential / correctly ordered merge must leave globalValue == 3. Running these
// txs concurrently on one shared StateDB without proper isolation typically loses
// updates or corrupts state, so this check fails under broken parallelization.
func TestParallelVMBlockAccessListContended(t *testing.T) {
	keys, addrs := newTestAccounts(t, 4)
	deployKey, key1, key2, key3 := keys[0], keys[1], keys[2], keys[3]
	deployFrom := addrs[0]

	contractABI := mustReadContractABI(t)
	contractBin := mustReadContractBinBlockTest(t)

	contractAddr := crypto.CreateAddress(deployFrom, 0)

	genesis := &core.Genesis{
		Config:   params.AllEthashProtocolChanges,
		Alloc:    genesisAllocEther(addrs[:]...),
		GasLimit: 30_000_000,
		BaseFee:  big.NewInt(params.InitialBaseFee),
	}

	engine := ethash.NewFaker()

	_, blocks, _ := core.GenerateChainWithGenesis(genesis, engine, 2, func(i int, b *core.BlockGen) {
		switch i {
		case 0:
			deployTx := mustSignAccessListTx(t, params.AllEthashProtocolChanges, deployKey, &types.AccessListTx{
				ChainID:  params.AllEthashProtocolChanges.ChainID,
				Nonce:    0,
				To:       nil,
				Gas:      12_000_000,
				GasPrice: b.BaseFee(),
				Data:     contractBin,
			})
			b.AddTx(deployTx)

		case 1:
			const rounds int64 = 50

			contendedAccess := types.AccessList{
				{
					Address: contractAddr,
					StorageKeys: []common.Hash{
						common.BigToHash(big.NewInt(2)),
						common.BigToHash(big.NewInt(3)),
					},
				},
			}

			callers := []*ecdsa.PrivateKey{key1, key2, key3}
			for _, ck := range callers {
				data, _ := contractABI.Pack("contendedJob", big.NewInt(1), big.NewInt(rounds))
				tx := mustSignAccessListTx(t, params.AllEthashProtocolChanges, ck, &types.AccessListTx{
					ChainID:    params.AllEthashProtocolChanges.ChainID,
					Nonce:      0,
					To:         &contractAddr,
					Gas:        8_000_000,
					GasPrice:   b.BaseFee(),
					Data:       data,
					AccessList: contendedAccess,
				})
				b.AddTx(tx)
			}
		}
	})

	runAndCheckChain(t, genesis, engine, blocks, func(statedb *state.StateDB) {
		assertStorageUintBlock(t, statedb, contractAddr, common.BigToHash(big.NewInt(2)), 3)
	})
}

// TestParallelVMBlockAccessListMixed: tx1 and tx2 both call mixedJob on the same
// contract (shared globals + different lanes → declared address set overlaps).
// tx3 calls isolatedJob on a second deployment only; its declared addresses are
// disjoint from tx1 and tx2, so grouping may place tx3 in a different wave than
// the contended pair (e.g. [0,2] then [1] depending on greedy order).
func TestParallelVMBlockAccessListMixed(t *testing.T) {
	keys, addrs := newTestAccounts(t, 4)
	deployKey, key1, key2, key3 := keys[0], keys[1], keys[2], keys[3]
	deployFrom := addrs[0]

	contractABI := mustReadContractABI(t)
	contractBin := mustReadContractBinBlockTest(t)

	cShared := crypto.CreateAddress(deployFrom, 0)
	cIsolated := crypto.CreateAddress(deployFrom, 1)

	genesis := &core.Genesis{
		Config:   params.AllEthashProtocolChanges,
		Alloc:    genesisAllocEther(addrs[:]...),
		GasLimit: 30_000_000,
		BaseFee:  big.NewInt(params.InitialBaseFee),
	}

	engine := ethash.NewFaker()

	_, blocks, _ := core.GenerateChainWithGenesis(genesis, engine, 2, func(i int, b *core.BlockGen) {
		switch i {
		case 0:
			for n := uint64(0); n < 2; n++ {
				deployTx := mustSignAccessListTx(t, params.AllEthashProtocolChanges, deployKey, &types.AccessListTx{
					ChainID:  params.AllEthashProtocolChanges.ChainID,
					Nonce:    n,
					To:       nil,
					Gas:      12_000_000,
					GasPrice: b.BaseFee(),
					Data:     contractBin,
				})
				b.AddTx(deployTx)
			}

		case 1:
			const rounds int64 = 50

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

			// Tx1 & tx2: same cShared → contention on contract + global slots.
			data1, _ := contractABI.Pack("mixedJob", big.NewInt(1), big.NewInt(2), big.NewInt(rounds))
			tx1 := mustSignAccessListTx(t, params.AllEthashProtocolChanges, key1, &types.AccessListTx{
				ChainID:    params.AllEthashProtocolChanges.ChainID,
				Nonce:      0,
				To:         &cShared,
				Gas:        8_000_000,
				GasPrice:   b.BaseFee(),
				Data:       data1,
				AccessList: mixedAccess(cShared, 1),
			})
			b.AddTx(tx1)

			data2, _ := contractABI.Pack("mixedJob", big.NewInt(2), big.NewInt(3), big.NewInt(rounds))
			tx2 := mustSignAccessListTx(t, params.AllEthashProtocolChanges, key2, &types.AccessListTx{
				ChainID:    params.AllEthashProtocolChanges.ChainID,
				Nonce:      0,
				To:         &cShared,
				Gas:        8_000_000,
				GasPrice:   b.BaseFee(),
				Data:       data2,
				AccessList: mixedAccess(cShared, 2),
			})
			b.AddTx(tx2)

			// Tx3: different contract only — disjoint from tx1/tx2 declared sets (aside from unique senders).
			data3, err := contractABI.Pack("isolatedJob", big.NewInt(3), big.NewInt(1), big.NewInt(rounds))
			if err != nil {
				t.Fatalf("pack isolatedJob lane3: %v", err)
			}
			tx3 := mustSignAccessListTx(t, params.AllEthashProtocolChanges, key3, &types.AccessListTx{
				ChainID:    params.AllEthashProtocolChanges.ChainID,
				Nonce:      0,
				To:         &cIsolated,
				Gas:        8_000_000,
				GasPrice:   b.BaseFee(),
				Value:      big.NewInt(0),
				Data:       data3,
				AccessList: isolatedAccessList(cIsolated, 3),
			})
			b.AddTx(tx3)
		}
	})

	runAndCheckChain(t, genesis, engine, blocks, func(statedb *state.StateDB) {
		// Sequential order 1 then 2 on cShared matches original single-contract mixed expectations.
		assertStorageUintBlock(t, statedb, cShared, mappingSlotBlock(1, 0), 2)
		assertStorageUintBlock(t, statedb, cShared, mappingSlotBlock(2, 0), 3)
		assertStorageUintBlock(t, statedb, cShared, common.BigToHash(big.NewInt(2)), 5)
		assertStorageUintBlock(t, statedb, cIsolated, mappingSlotBlock(3, 0), 1)
		assertStorageUintBlock(t, statedb, cIsolated, common.BigToHash(big.NewInt(2)), 0)
	})
}
