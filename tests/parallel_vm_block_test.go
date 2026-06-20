package tests

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
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
	isolatedTestBlockGasLimit     = uint64((isolatedTestNumIndependentTxs + 1) * isolatedJobTxGasLimit)
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

func runAndCheckChain(t *testing.T, genesis *core.Genesis, engine *ethash.Ethash, blocks []*types.Block, check func(*state.StateDB)) {
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

	if n, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("block %d failed: %s", n, formatInsertChainErrorForDebug(err))
	}

	statedb := mustStateAtCurrentHead(t, chain)
	check(statedb)
}

// formatInsertChainErrorForDebug prints the full errors.Unwrap chain (outer → inner)
// and adds a file hint for known core errors so test failures are easier to trace.
func formatInsertChainErrorForDebug(err error) string {
	var b strings.Builder
	b.WriteString(err.Error())
	for u := errors.Unwrap(err); u != nil; u = errors.Unwrap(u) {
		b.WriteString("\n  ← unwrap: ")
		b.WriteString(u.Error())
	}
	switch {
	case errors.Is(err, core.ErrNonceTooHigh):
		b.WriteString("\n  [origin] core/state_transition.go — (*stateTransition).preCheck when !msg.SkipNonceChecks (tx nonce > state nonce)")
	case errors.Is(err, core.ErrNonceTooLow):
		b.WriteString("\n  [origin] core/state_transition.go — (*stateTransition).preCheck when !msg.SkipNonceChecks")
	case errors.Is(err, core.ErrInsufficientFunds):
		b.WriteString("\n  [origin] core/state_transition.go — (*stateTransition).buyGas or balance checks in preCheck")
	}
	return b.String()
}

func newTestAccount(t *testing.T) (*ecdsa.PrivateKey, common.Address) {
	t.Helper()

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return key, crypto.PubkeyToAddress(key.PublicKey)
}

func newTestAccounts(t *testing.T, n int) ([]*ecdsa.PrivateKey, []common.Address) {
	t.Helper()
	keys := make([]*ecdsa.PrivateKey, n)
	addrs := make([]common.Address, n)
	for i := 0; i < n; i++ {
		keys[i], addrs[i] = newTestAccount(t)
	}
	return keys, addrs
}

func genesisAllocEther(addrs ...common.Address) types.GenesisAlloc {
	bal := new(big.Int).Mul(big.NewInt(1_000_000), big.NewInt(params.Ether))
	alloc := make(types.GenesisAlloc, len(addrs))
	for _, a := range addrs {
		alloc[a] = types.Account{Balance: bal}
	}
	return alloc
}

func mustReadContractABI(t *testing.T) abi.ABI {
	t.Helper()

	abiPath := filepath.Join("contracts", "ParallelVMTest.abi")
	abiBytes, err := os.ReadFile(abiPath)
	if err != nil {
		t.Fatalf("read ABI: %v", err)
	}
	parsedABI, err := abi.JSON(bytes.NewReader(abiBytes))
	if err != nil {
		t.Fatalf("parse ABI: %v", err)
	}
	return parsedABI
}

func mustReadContractBinBlockTest(t *testing.T) []byte {
	t.Helper()

	binPath := filepath.Join("contracts", "ParallelVMTest.bin")
	binBytes, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("read bytecode: %v", err)
	}
	hexStr := strings.TrimSpace(string(binBytes))
	if strings.HasPrefix(hexStr, "0x") {
		hexStr = hexStr[2:]
	}
	return common.FromHex("0x" + hexStr)
}

func mustSignAccessListTx(t *testing.T, cfg *params.ChainConfig, key *ecdsa.PrivateKey, txdata *types.AccessListTx) *types.Transaction {
	t.Helper()

	signer := types.LatestSigner(cfg)
	tx := types.NewTx(txdata)
	signed, err := types.SignTx(tx, signer, key)
	if err != nil {
		t.Fatalf("sign tx: %v", err)
	}
	return signed
}

func isolatedAccessList(contract common.Address, lane uint64) types.AccessList {
	return types.AccessList{
		{
			Address: contract,
			StorageKeys: []common.Hash{
				mappingSlotBlock(lane, 0), // laneValue[lane]
				mappingSlotBlock(lane, 1), // laneDigest[lane]
			},
		},
	}
}

func mappingSlotBlock(key uint64, baseSlot uint64) common.Hash {
	return crypto.Keccak256Hash(
		common.LeftPadBytes(new(big.Int).SetUint64(key).Bytes(), 32),
		common.LeftPadBytes(new(big.Int).SetUint64(baseSlot).Bytes(), 32),
	)
}

func assertStorageUintBlock(t *testing.T, statedb *state.StateDB, addr common.Address, slot common.Hash, want uint64) {
	t.Helper()

	got := statedb.GetState(addr, slot).Big().Uint64()
	if got != want {
		t.Fatalf("slot %s got %d want %d", slot.Hex(), got, want)
	}
}

func mustStateAtCurrentHead(t *testing.T, chain *core.BlockChain) *state.StateDB {
	t.Helper()

	root := chain.CurrentBlock().Root

	// Most branches expose one of these two shapes:
	// 1) chain.StateAt(root)
	// 2) rebuild using state.New(root, state.NewDatabase(...))
	//
	// Try the direct API first if your branch has it.
	statedb, err := chain.StateAt(root)
	if err == nil {
		return statedb
	}

	// Fallback path for branches without StateAt.
	db := rawdb.NewMemoryDatabase()
	tdb := triedb.NewDatabase(db, &triedb.Config{Preimages: true})
	sdb := state.NewDatabase(tdb, nil)
	st, err2 := state.New(root, sdb)
	if err2 != nil {
		t.Fatalf("state lookup failed: direct=%v fallback=%v", err, err2)
	}
	return st
}

// This exists only to keep imports aligned if your branch keeps tracing/uint256
// in nearby tests and gofmt otherwise complains when you copy patterns around.
var (
	_ = tracing.BalanceChangeUnspecified
	_ = uint256.NewInt
)
