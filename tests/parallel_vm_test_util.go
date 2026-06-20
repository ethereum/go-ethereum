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
