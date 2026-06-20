package tests

import (
	"bytes"
	"crypto/ecdsa"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

type parallelVMHarness struct {
	statedb      *state.StateDB
	chainConfig  *params.ChainConfig
	privateKey   *ecdsa.PrivateKey
	from         common.Address
	contractAddr common.Address
	nonce        uint64
	testABI      abi.ABI
}

func TestParallelVMIsolatedBatch(t *testing.T) {
	h := newParallelVMHarness(t)
	h.deployContract(t)

	const rounds int64 = 50

	h.callContract(t, "isolatedJob", big.NewInt(1), big.NewInt(1), big.NewInt(rounds))
	h.callContract(t, "isolatedJob", big.NewInt(2), big.NewInt(1), big.NewInt(rounds))
	h.callContract(t, "isolatedJob", big.NewInt(3), big.NewInt(1), big.NewInt(rounds))

	assertStorageUint(t, h.statedb, h.contractAddr, mappingSlot(1, 0), 1)
	assertStorageUint(t, h.statedb, h.contractAddr, mappingSlot(2, 0), 1)
	assertStorageUint(t, h.statedb, h.contractAddr, mappingSlot(3, 0), 1)
	assertStorageUint(t, h.statedb, h.contractAddr, common.BigToHash(big.NewInt(2)), 0)
}

func TestParallelVMContendedBatch(t *testing.T) {
	h := newParallelVMHarness(t)
	h.deployContract(t)

	const rounds int64 = 50

	h.callContract(t, "contendedJob", big.NewInt(1), big.NewInt(rounds))
	h.callContract(t, "contendedJob", big.NewInt(1), big.NewInt(rounds))
	h.callContract(t, "contendedJob", big.NewInt(1), big.NewInt(rounds))

	assertStorageUint(t, h.statedb, h.contractAddr, common.BigToHash(big.NewInt(2)), 3)
}

func TestParallelVMMixedBatch(t *testing.T) {
	h := newParallelVMHarness(t)
	h.deployContract(t)

	const rounds int64 = 50

	h.callContract(t, "mixedJob", big.NewInt(1), big.NewInt(2), big.NewInt(rounds))
	h.callContract(t, "mixedJob", big.NewInt(2), big.NewInt(3), big.NewInt(rounds))

	assertStorageUint(t, h.statedb, h.contractAddr, mappingSlot(1, 0), 2)
	assertStorageUint(t, h.statedb, h.contractAddr, mappingSlot(2, 0), 3)
	assertStorageUint(t, h.statedb, h.contractAddr, common.BigToHash(big.NewInt(2)), 5)
}

func newParallelVMHarness(t *testing.T) *parallelVMHarness {
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

	memdb := rawdb.NewMemoryDatabase()
	trieDB := triedb.NewDatabase(memdb, &triedb.Config{Preimages: true})
	cacheDB := state.NewDatabase(trieDB, nil)

	statedb, err := state.New(types.EmptyRootHash, cacheDB)
	if err != nil {
		t.Fatalf("create state db: %v", err)
	}

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	from := crypto.PubkeyToAddress(key.PublicKey)

	statedb.AddBalance(
		from,
		uint256.MustFromBig(new(big.Int).Mul(big.NewInt(1_000_000), big.NewInt(params.Ether))),
		tracing.BalanceChangeUnspecified,
	)
	return &parallelVMHarness{
		statedb:     statedb,
		chainConfig: params.AllEthashProtocolChanges,
		privateKey:  key,
		from:        from,
		testABI:     parsedABI,
	}
}

func (h *parallelVMHarness) deployContract(t *testing.T) {
	t.Helper()

	bytecode := mustReadContractBin(t)

	msg := &core.Message{
		From:             h.from,
		To:               nil,
		Nonce:            h.nonce,
		Value:            big.NewInt(0),
		GasLimit:         12_000_000,
		GasPrice:         big.NewInt(1),
		GasFeeCap:        big.NewInt(1),
		GasTipCap:        big.NewInt(1),
		Data:             bytecode,
		SkipNonceChecks:  true,
		SkipFromEOACheck: true,
	}

	h.nonce++

	blockCtx := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		GetHash:     func(uint64) common.Hash { return common.Hash{} },
		Coinbase:    common.Address{},
		BlockNumber: big.NewInt(1),
		Time:        1,
		Difficulty:  big.NewInt(1),
		GasLimit:    30_000_000,
		BaseFee:     big.NewInt(0),
	}

	txCtx := core.NewEVMTxContext(msg)
	evm := vm.NewEVM(blockCtx, h.statedb, h.chainConfig, vm.Config{})
	evm.SetTxContext(txCtx)

	result, err := core.ApplyMessage(evm, msg, new(core.GasPool).AddGas(msg.GasLimit))
	if err != nil {
		t.Fatalf("deploy ApplyMessage: %v", err)
	}
	if result.Failed() {
		t.Fatalf("deploy reverted: %x | err: %v", result.Return(), err)
	}

	h.contractAddr = crypto.CreateAddress(h.from, 0)
}

func (h *parallelVMHarness) callContract(t *testing.T, method string, args ...interface{}) {
	t.Helper()

	input, err := h.testABI.Pack(method, args...)
	if err != nil {
		t.Fatalf("ABI pack %s: %v", method, err)
	}

	msg := &core.Message{
		From:             h.from,
		To:               &h.contractAddr,
		Nonce:            h.nonce,
		Value:            big.NewInt(0),
		GasLimit:         12_000_000,
		GasPrice:         big.NewInt(1),
		GasFeeCap:        big.NewInt(1),
		GasTipCap:        big.NewInt(1),
		Data:             input,
		SkipNonceChecks:  true,
		SkipFromEOACheck: true,
	}

	h.nonce++

	blockCtx := vm.BlockContext{
		CanTransfer: core.CanTransfer,
		Transfer:    core.Transfer,
		GetHash:     func(uint64) common.Hash { return common.Hash{} },
		Coinbase:    common.Address{},
		BlockNumber: big.NewInt(1),
		Time:        1,
		Difficulty:  big.NewInt(1),
		GasLimit:    30_000_000,
		BaseFee:     big.NewInt(0),
	}

	txCtx := core.NewEVMTxContext(msg)
	evm := vm.NewEVM(blockCtx, h.statedb, h.chainConfig, vm.Config{})
	evm.SetTxContext(txCtx)

	result, err := core.ApplyMessage(evm, msg, new(core.GasPool).AddGas(msg.GasLimit))
	if err != nil {
		t.Fatalf("call %s ApplyMessage: %v", method, err)
	}
	if result.Failed() {
		t.Fatalf("call %s reverted: %x", method, result.Return())
	}
}

func mustReadContractBin(t *testing.T) []byte {
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

func mappingSlot(key uint64, baseSlot uint64) common.Hash {
	return crypto.Keccak256Hash(
		common.LeftPadBytes(new(big.Int).SetUint64(key).Bytes(), 32),
		common.LeftPadBytes(new(big.Int).SetUint64(baseSlot).Bytes(), 32),
	)
}

func assertStorageUint(t *testing.T, statedb *state.StateDB, addr common.Address, slot common.Hash, want uint64) {
	t.Helper()

	got := statedb.GetState(addr, slot).Big().Uint64()
	if got != want {
		t.Fatalf("slot %s got %d want %d", slot.Hex(), got, want)
	}
}
