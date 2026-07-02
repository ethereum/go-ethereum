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

// Transaction- and block-level tests for EIP-8037 (multidimensional state-gas
// metering). They apply whole transactions and inspect the 2D block gas pool
// (cumulativeRegular / cumulativeState) and the receipt/peak figures.

package core

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

var (
	cfg8037      = balChainConfig()
	signer8037   = types.LatestSigner(cfg8037)
	rules8037    = cfg8037.Rules(big.NewInt(0), true, 0)
	senderKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	senderAddr   = crypto.PubkeyToAddress(senderKey.PublicKey)

	// state-gas charges in units (CPSB applied).
	newAccountState = uint64(params.AccountCreationSize * params.CostPerStateByte)       // 183,600
	newSlotState    = uint64(params.StorageCreationSize * params.CostPerStateByte)       // 97,920
	authBaseState   = uint64(params.AuthorizationCreationSize * params.CostPerStateByte) // 35,190
	authWorstState  = newAccountState + authBaseState                                    // 218,790
)

// mkState builds an in-memory StateDB from a genesis allocation.
func mkState(alloc types.GenesisAlloc) *state.StateDB {
	sdb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	for addr, acc := range alloc {
		sdb.CreateAccount(addr)
		if acc.Balance != nil {
			sdb.AddBalance(addr, uint256.MustFromBig(acc.Balance), tracing.BalanceChangeUnspecified)
		}
		if acc.Nonce != 0 {
			sdb.SetNonce(addr, acc.Nonce, tracing.NonceChangeGenesis)
		}
		if len(acc.Code) != 0 {
			sdb.SetCode(addr, acc.Code, tracing.CodeChangeUnspecified)
		}
		for k, v := range acc.Storage {
			sdb.SetState(addr, k, v)
		}
	}
	sdb.Finalise(true)
	return sdb
}

// mkCommittedState is mkState with the allocation committed to disk and
// reloaded. EIP-161-empty accounts carrying only storage do not survive an
// in-memory Finalise; committing without empty-account deletion reproduces
// the synthesized prestate an EIP-7610 fixture would load from disk.
func mkCommittedState(t *testing.T, alloc types.GenesisAlloc) *state.StateDB {
	t.Helper()
	db := state.NewDatabaseForTesting()
	sdb, _ := state.New(types.EmptyRootHash, db)
	for addr, acc := range alloc {
		sdb.CreateAccount(addr)
		if acc.Balance != nil {
			sdb.AddBalance(addr, uint256.MustFromBig(acc.Balance), tracing.BalanceChangeUnspecified)
		}
		if acc.Nonce != 0 {
			sdb.SetNonce(addr, acc.Nonce, tracing.NonceChangeGenesis)
		}
		if len(acc.Code) != 0 {
			sdb.SetCode(addr, acc.Code, tracing.CodeChangeUnspecified)
		}
		for k, v := range acc.Storage {
			sdb.SetState(addr, k, v)
		}
	}
	root, err := sdb.Commit(0, false, false)
	if err != nil {
		t.Fatalf("commit prestate: %v", err)
	}
	sdb, err = state.New(root, db)
	if err != nil {
		t.Fatalf("reopen prestate: %v", err)
	}
	return sdb
}

// amsterdamCoreEVM builds an Amsterdam EVM over statedb with fees disabled.
func amsterdamCoreEVM(sdb *state.StateDB) *vm.EVM {
	ctx := vm.BlockContext{
		CanTransfer:      CanTransfer,
		Transfer:         Transfer,
		GetHash:          func(uint64) common.Hash { return common.Hash{} },
		BlockNumber:      big.NewInt(0),
		Random:           &common.Hash{},
		Difficulty:       big.NewInt(0),
		BaseFee:          big.NewInt(0),
		BlobBaseFee:      big.NewInt(0),
		GasLimit:         60_000_000,
		CostPerStateByte: params.CostPerStateByte,
	}
	return vm.NewEVM(ctx, sdb, cfg8037, vm.Config{NoBaseFee: true})
}

// applyMsg applies one transaction with a fresh block gas pool and returns the
// execution result, the gas pool (for the 2D split) and any consensus error.
func applyMsg(t *testing.T, sdb *state.StateDB, tx *types.Transaction) (*ExecutionResult, *GasPool, error) {
	t.Helper()
	evm := amsterdamCoreEVM(sdb)
	msg, err := TransactionToMessage(tx, signer8037, evm.Context.BaseFee)
	if err != nil {
		t.Fatalf("to message: %v", err)
	}
	gp := NewGasPool(evm.Context.GasLimit)

	evm.SetTxContext(NewEVMTxContext(msg))
	st := newStateTransition(evm, msg, gp)
	res, err := st.execute()
	if err == nil && res != nil {
		floor, ferr := FloorDataGas(rules8037, msg.From, msg.To, msg.Value, msg.Data, msg.AccessList)
		if ferr != nil {
			t.Fatalf("floor data gas: %v", ferr)
		}
		assertPoolSane(t, res, gp, floor)

		intrinsic, ierr := IntrinsicGas(msg.Data, msg.AccessList, msg.SetCodeAuthorizations, msg.From, msg.To, msg.Value, rules8037)
		if ierr != nil {
			t.Fatalf("intrinsic gas: %v", ierr)
		}
		executionGas := msg.GasLimit - intrinsic
		gasLeft := min(params.MaxTxGas-intrinsic, executionGas)
		assertBudgetSane(t, vm.NewGasBudget(gasLeft, executionGas-gasLeft), st.gasRemaining)
	}
	return res, gp, err
}

// assertBudgetSane validates the final tx-level GasBudget vector:
//
//	regular: RegularGas + UsedRegularGas + Spilled == initial.RegularGas
//	state:   StateGas + UsedStateGas               == initial.StateGas + Spilled
//	scalar:  Used(initial)                         == UsedRegularGas + UsedStateGas
func assertBudgetSane(t *testing.T, initial, got vm.GasBudget) {
	t.Helper()
	if got.RegularGas+got.UsedRegularGas+got.Spilled != initial.RegularGas {
		t.Fatalf("regular not conserved: R=%d usedR=%d spilled=%d, want sum %d",
			got.RegularGas, got.UsedRegularGas, got.Spilled, initial.RegularGas)
	}
	if int64(got.StateGas)+got.UsedStateGas != int64(initial.StateGas)+int64(got.Spilled) {
		t.Fatalf("state not conserved: S=%d usedS=%d spilled=%d, want %d+spilled",
			got.StateGas, got.UsedStateGas, got.Spilled, initial.StateGas)
	}
	if int64(got.Used(initial)) != int64(got.UsedRegularGas)+got.UsedStateGas {
		t.Fatalf("scalar mismatch: used=%d, usedR=%d usedS=%d",
			got.Used(initial), got.UsedRegularGas, got.UsedStateGas)
	}
}

// assertPoolSane validates the whole 2D block-gas-pool vector after a single tx.
//
//	receipt:    cumulativeUsed == res.UsedGas <= res.MaxUsedGas
//	regular:    cumulativeRegular <= max(res.MaxUsedGas - cumulativeState, floor)
//	            (the calldata floor pads the regular dimension alone, so the
//	            dimension sum may exceed the pre-refund peak when it binds)
//	bottleneck: Used() == max(cumulativeRegular, cumulativeState) <= initial
func assertPoolSane(t *testing.T, res *ExecutionResult, gp *GasPool, floor uint64) {
	t.Helper()
	if gp.cumulativeUsed != res.UsedGas {
		t.Fatalf("receipt scalar = %d, want UsedGas %d", gp.cumulativeUsed, res.UsedGas)
	}
	if res.UsedGas > res.MaxUsedGas {
		t.Fatalf("post-refund gas %d exceeds peak %d", res.UsedGas, res.MaxUsedGas)
	}
	if gp.cumulativeRegular > res.MaxUsedGas {
		t.Fatalf("regular %d exceeds peak %d", gp.cumulativeRegular, res.MaxUsedGas)
	}
	if gp.cumulativeState > res.MaxUsedGas {
		t.Fatalf("state %d exceeds peak %d", gp.cumulativeState, res.MaxUsedGas)
	}
	if cap := max(res.MaxUsedGas-gp.cumulativeState, floor); gp.cumulativeRegular > cap {
		t.Fatalf("regular %d exceeds pre-refund cap %d (peak %d, state %d, floor %d)",
			gp.cumulativeRegular, cap, res.MaxUsedGas, gp.cumulativeState, floor)
	}
	if gp.Used() != max(gp.cumulativeRegular, gp.cumulativeState) {
		t.Fatalf("block used %d != max(%d,%d)", gp.Used(), gp.cumulativeRegular, gp.cumulativeState)
	}
	if gp.Used() > gp.initial {
		t.Fatalf("block used %d exceeds limit %d", gp.Used(), gp.initial)
	}
}

// senderAlloc funds the sender with the given extra accounts merged in.
func senderAlloc(extra types.GenesisAlloc) types.GenesisAlloc {
	alloc := types.GenesisAlloc{senderAddr: {Balance: big.NewInt(1e18)}}
	for a, acc := range extra {
		alloc[a] = acc
	}
	return alloc
}

// callTx builds a signed dynamic-fee call to `to` with zero fees.
func callTx(nonce uint64, to common.Address, value int64, gas uint64, data []byte) *types.Transaction {
	return types.MustSignNewTx(senderKey, signer8037, &types.DynamicFeeTx{
		ChainID: cfg8037.ChainID, Nonce: nonce, To: &to, Value: big.NewInt(value),
		Gas: gas, GasFeeCap: big.NewInt(0), GasTipCap: big.NewInt(0), Data: data,
	})
}

// createTx builds a signed contract-creation transaction.
func createTx(nonce, gas uint64, initCode []byte) *types.Transaction {
	return types.MustSignNewTx(senderKey, signer8037, &types.DynamicFeeTx{
		ChainID: cfg8037.ChainID, Nonce: nonce, To: nil, Value: big.NewInt(0),
		Gas: gas, GasFeeCap: big.NewInt(0), GasTipCap: big.NewInt(0), Data: initCode,
	})
}

var (
	deploy3 = []byte{0x60, 0x03, 0x60, 0x00, 0xf3} // init: return 3 bytes of code
	revertI = []byte{0x60, 0x00, 0x60, 0x00, 0xfd} // init: REVERT
	haltI   = []byte{0xfe, 0x00, 0x00, 0x00, 0x00} // init: INVALID, exceptional halt
)

// ===================== Top-level create transaction ======================

// A creation tx's intrinsic gas is state-independent: the new-account state
// charge depends on whether the deployment target exists and is charged at
// runtime (EIP-2780), not intrinsically.
func TestCreateTxIntrinsicNoStateGas(t *testing.T) {
	cost, err := IntrinsicGas(nil, nil, nil, common.Address{}, nil, nil, rules8037)
	if err != nil {
		t.Fatal(err)
	}
	if want := params.TxBaseCost2780 + params.CreateAccessAmsterdam; cost != want {
		t.Fatalf("intrinsic gas = %d, want %d", cost, want)
	}
}

// Creating onto a pre-existing (balance-only) address incurs no new-account
// runtime charge; only the code deposit is charged as state gas.
func TestCreateTxPreexistingDestRefill(t *testing.T) {
	derived := crypto.CreateAddress(senderAddr, 0)
	sdb := mkState(senderAlloc(types.GenesisAlloc{derived: {Balance: big.NewInt(1)}}))
	_, gp, err := applyMsg(t, sdb, createTx(0, 1_000_000, deploy3))
	if err != nil {
		t.Fatal(err)
	}
	if want := uint64(3 * params.CostPerStateByte); gp.cumulativeState != want {
		t.Fatalf("state gas = %d, want %d", gp.cumulativeState, want)
	}
}

// A creation tx that reverts refills the account-creation charge applied at
// runtime.
func TestCreateTxRevertRefill(t *testing.T) {
	sdb := mkState(senderAlloc(nil))
	res, gp, err := applyMsg(t, sdb, createTx(0, 1_000_000, revertI))
	if err != nil {
		t.Fatal(err)
	}
	if !res.Failed() {
		t.Fatal("expected failed creation")
	}
	if gp.cumulativeState != 0 {
		t.Fatalf("state gas = %d, want 0 (refilled)", gp.cumulativeState)
	}
}

// An address collision burns gas_left. The colliding target exists, so no
// new-account state gas is charged at runtime in the first place.
func TestCreateTxCollisionConsumesGasLeft(t *testing.T) {
	const gas = 1_000_000
	derived := crypto.CreateAddress(senderAddr, 0)
	sdb := mkState(senderAlloc(types.GenesisAlloc{derived: {Nonce: 1}}))
	res, gp, err := applyMsg(t, sdb, createTx(0, gas, deploy3))
	if err != nil {
		t.Fatal(err)
	}
	if !res.Failed() {
		t.Fatal("expected collision failure")
	}
	if gp.cumulativeState != 0 {
		t.Fatalf("state gas = %d, want 0 (never charged)", gp.cumulativeState)
	}
	// All forwarded gas_left is burned: the whole gas limit is consumed as
	// regular gas.
	if want := uint64(gas); gp.cumulativeRegular != want {
		t.Fatalf("regular gas = %d, want %d", gp.cumulativeRegular, want)
	}
}

// An account can exist yet be EIP-161-empty in the middle of a transaction,
// e.g. after being touched as the zero-balance beneficiary of a SELFDESTRUCT.
// Deploying onto such an account should charge account-creation cost.
func TestCreate2TransientEmptyDestNoRefill(t *testing.T) {
	var (
		orchestrator = common.HexToAddress("0xc0de000000000000000000000000000000000002")
		destructor   = common.HexToAddress("0xc0de000000000000000000000000000000000003")
		target       = crypto.CreateAddress2(orchestrator, [32]byte{}, crypto.Keccak256(deploy3))
	)
	// destructor: SELFDESTRUCT with zero balance to the future CREATE2 target,
	// leaving it existing but EIP-161-empty for the rest of the transaction.
	destructorCode := append(append([]byte{0x73}, target.Bytes()...), 0xff) // PUSH20 target, SELFDESTRUCT

	// orchestrator: CALL destructor (persist the success flag in slot 0),
	// then CREATE2 deploy3 with salt 0, targeting the touched address.
	code := []byte{
		0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00, // ret/arg sizes and offsets, value = 0
		0x73, // PUSH20 destructor
	}
	code = append(code, destructor.Bytes()...)
	code = append(code,
		0x62, 0x03, 0x0d, 0x40, // PUSH3 200,000 call gas
		0xf1,             // CALL
		0x60, 0x00, 0x55, // SSTORE the call result at slot 0
		0x64, 0x60, 0x03, 0x60, 0x00, 0xf3, // PUSH5 deploy3 init code
		0x60, 0x00, 0x52, // MSTORE at word 0 (right-aligned, code at offset 27)
		0x60, 0x00, // salt  = 0
		0x60, 0x05, // size  = 5
		0x60, 0x1b, // offset = 27
		0x60, 0x00, // endowment = 0
		0xf5, 0x50, // CREATE2, POP
		0x00, // STOP
	)
	sdb := mkState(senderAlloc(types.GenesisAlloc{
		orchestrator: {Code: code},
		destructor:   {Code: destructorCode},
	}))
	res, gp, err := applyMsg(t, sdb, callTx(0, orchestrator, 0, 2_000_000, nil))
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed() {
		t.Fatalf("execution failed: %v", res.Err)
	}
	// The inner call must have succeeded, so the target was touched into an
	// existing-but-empty account before the CREATE2 executed.
	if flag := sdb.GetState(orchestrator, common.Hash{}); flag != common.BigToHash(big.NewInt(1)) {
		t.Fatalf("destructor call flag = %v, want 1", flag)
	}
	if code := sdb.GetCode(target); len(code) != 3 {
		t.Fatalf("deployed code length = %d, want 3", len(code))
	}
	// State gas: the orchestrator's flag slot, the created contract account
	// (charged, not refilled) and the 3-byte code deposit.
	want := newSlotState + newAccountState + uint64(3*params.CostPerStateByte)
	if gp.cumulativeState != want {
		t.Fatalf("state gas = %d, want %d (account creation must not be refilled)", gp.cumulativeState, want)
	}
}

// ========== Storage-only (EIP-7610-shaped) deployment destination ===========
//
// A destination carrying storage while having zero nonce, zero balance and
// empty code is EIP-161-empty, so the account-creation state gas is
// pre-charged in the parent frame.

// create2Orchestrator returns runtime code that CREATE2-deploys the given
// 5-byte init code with salt 0 and stores the result address at slot 0.
func create2Orchestrator(initCode []byte) []byte {
	code := append([]byte{0x64}, initCode...) // PUSH5 init code
	return append(code,
		0x60, 0x00, 0x52, // MSTORE at word 0 (right-aligned, code at offset 27)
		0x60, 0x00, // salt = 0
		0x60, 0x05, // size = 5
		0x60, 0x1b, // offset = 27
		0x60, 0x00, // endowment = 0
		0xf5,             // CREATE2
		0x60, 0x00, 0x55, // SSTORE the result address at slot 0
		0x00, // STOP
	)
}

// storageOnlyAlloc allocates the orchestrator and its CREATE2 target, the
// latter carrying a single storage slot while remaining EIP-161-empty.
func storageOnlyAlloc(orchestrator common.Address, initCode []byte) (types.GenesisAlloc, common.Address) {
	target := crypto.CreateAddress2(orchestrator, [32]byte{}, crypto.Keccak256(initCode))
	return types.GenesisAlloc{
		orchestrator: {Code: create2Orchestrator(initCode)},
		target:       {Storage: map[common.Hash]common.Hash{{}: common.BigToHash(big.NewInt(1))}},
	}, target
}

// Deploying onto a storage-only destination pre-charges the account creation.
// Under the registry-based EIP-7610 check the creation proceeds, so the
// charge is consumed like any other creation.
func TestCreate2StorageOnlyDestCharged(t *testing.T) {
	orchestrator := common.HexToAddress("0xc0de000000000000000000000000000000000004")
	alloc, target := storageOnlyAlloc(orchestrator, deploy3)
	sdb := mkCommittedState(t, senderAlloc(alloc))
	res, gp, err := applyMsg(t, sdb, callTx(0, orchestrator, 0, 1_000_000, nil))
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed() {
		t.Fatalf("execution failed: %v", res.Err)
	}
	if code := sdb.GetCode(target); len(code) != 3 {
		t.Fatalf("deployed code length = %d, want 3", len(code))
	}
	// The created account (charged, consumed), the orchestrator's result slot
	// and the 3-byte code deposit.
	want := newAccountState + newSlotState + uint64(3*params.CostPerStateByte)
	if gp.cumulativeState != want {
		t.Fatalf("state gas = %d, want %d", gp.cumulativeState, want)
	}
}

// If the pre-charge succeeds and the create frame then fails, only the create
// frame halts: the forwarded regular gas is burnt, the account-creation
// charge is refilled, and the parent frame continues.
func TestCreate2StorageOnlyDestRefillOnFrameHalt(t *testing.T) {
	const gas = 1_000_000
	orchestrator := common.HexToAddress("0xc0de000000000000000000000000000000000005")
	alloc, target := storageOnlyAlloc(orchestrator, haltI)
	sdb := mkCommittedState(t, senderAlloc(alloc))
	res, gp, err := applyMsg(t, sdb, callTx(0, orchestrator, 0, gas, nil))
	if err != nil {
		t.Fatal(err)
	}
	if res.Failed() {
		t.Fatalf("parent frame must survive the create-frame halt: %v", res.Err)
	}
	// The CREATE2 pushed zero and nothing was deployed.
	if flag := sdb.GetState(orchestrator, common.Hash{}); flag != (common.Hash{}) {
		t.Fatalf("create result = %v, want 0", flag)
	}
	if code := sdb.GetCode(target); len(code) != 0 {
		t.Fatalf("deployed code length = %d, want 0", len(code))
	}
	// The account-creation charge was refilled in full.
	if gp.cumulativeState != 0 {
		t.Fatalf("state gas = %d, want 0 (refilled)", gp.cumulativeState)
	}
	if res.UsedGas > gas-newAccountState {
		t.Fatalf("used gas = %d, want at most %d (charge not refilled?)", res.UsedGas, gas-newAccountState)
	}
}

// If the remaining gas cannot cover the account-creation pre-charge, the
// parent frame itself halts with out-of-gas instead of the create frame.
func TestCreate2StorageOnlyDestPrechargeOOG(t *testing.T) {
	// Enough for the CREATE2 constant cost, short of the 183,600 pre-charge.
	const gas = 150_000
	orchestrator := common.HexToAddress("0xc0de000000000000000000000000000000000006")
	alloc, _ := storageOnlyAlloc(orchestrator, deploy3)
	sdb := mkCommittedState(t, senderAlloc(alloc))
	res, gp, err := applyMsg(t, sdb, callTx(0, orchestrator, 0, gas, nil))
	if err != nil {
		t.Fatal(err)
	}
	if !res.Failed() || !errors.Is(res.Err, vm.ErrOutOfGas) {
		t.Fatalf("err = %v, want out of gas in the parent frame", res.Err)
	}
	if gp.cumulativeState != 0 {
		t.Fatalf("state gas = %d, want 0 (charge never applied)", gp.cumulativeState)
	}
	// The parent is the topmost frame, so its halt burns the whole gas limit.
	if gp.cumulativeRegular != gas {
		t.Fatalf("regular gas = %d, want %d", gp.cumulativeRegular, gas)
	}
}

// ======================== Transaction validation =========================

// The regular dimension must have room for min(tx.gas, MaxTxGas).
func TestValidationRegularGasAvailable(t *testing.T) {
	gp := NewGasPool(30_000_000)
	gp.cumulativeRegular = 29_000_000
	if gp.CheckGasAmsterdam(2_000_000, 0) == nil {
		t.Fatal("expected regular dimension full")
	}
	if err := gp.CheckGasAmsterdam(1_000_000, 0); err != nil {
		t.Fatalf("regular fits but rejected: %v", err)
	}
}

// The state dimension must have room for the whole tx.gas.
func TestValidationStateGasAvailable(t *testing.T) {
	gp := NewGasPool(30_000_000)
	gp.cumulativeState = 29_000_000
	if gp.CheckGasAmsterdam(0, 2_000_000) == nil {
		t.Fatal("expected state dimension full")
	}
	if err := gp.CheckGasAmsterdam(0, 1_000_000); err != nil {
		t.Fatalf("state fits but rejected: %v", err)
	}
}

// tx.gas may exceed MaxTxGas: regular is capped at MaxTxGas while the state
// dimension reserves the full tx.gas (the excess lands in the reservoir).
func TestValidationStateGasOverflowAllowed(t *testing.T) {
	gas := params.MaxTxGas + 5_000_000
	gp := NewGasPool(40_000_000)
	if err := gp.CheckGasAmsterdam(min(gas, params.MaxTxGas), gas); err != nil {
		t.Fatalf("overflow tx rejected at pool: %v", err)
	}
	// A real transfer with gas above MaxTxGas is accepted under Amsterdam.
	sdb := mkState(senderAlloc(nil))
	to := common.HexToAddress("0xc0ffee")
	if _, _, err := applyMsg(t, sdb, callTx(0, to, 1, gas, nil)); err != nil {
		t.Fatalf("tx with gas > MaxTxGas rejected: %v", err)
	}
}

// Intrinsic regular gas above MaxTxGas (EIP-7825 cap) is rejected.
func TestValidationIntrinsicRegularCap(t *testing.T) {
	al := make(types.AccessList, 8000) // ~19.2M regular, over the 16.77M cap
	for i := range al {
		al[i].Address = common.BigToAddress(big.NewInt(int64(i + 1)))
	}
	tx := types.MustSignNewTx(senderKey, signer8037,
		&types.DynamicFeeTx{
			ChainID:    cfg8037.ChainID,
			Nonce:      0,
			To:         &senderAddr,
			Value:      big.NewInt(0),
			Gas:        25_000_000,
			GasFeeCap:  big.NewInt(0),
			GasTipCap:  big.NewInt(0),
			AccessList: al,
		})
	if _, _, err := applyMsg(t, mkState(senderAlloc(nil)), tx); err == nil {
		t.Fatal("expected rejection for intrinsic regular over MaxTxGas")
	}
}

// The EIP-7623/7976 calldata floor is capped by MaxTxGas even when the gas
// limit covers it: a transaction whose floor cost exceeds the cap is rejected
// regardless of its (much smaller) intrinsic gas.
func TestValidationFloorCostCap(t *testing.T) {
	// All-zero calldata: the floor charges 64/byte while the intrinsic
	// charges only 4/byte, so the floor crosses the cap long before the
	// intrinsic does.
	data := make([]byte, 300_000) // floor ~19.2M > 16.77M cap, intrinsic ~1.2M
	floor, err := FloorDataGas(rules8037, senderAddr, &senderAddr, new(uint256.Int), data, nil)
	if err != nil {
		t.Fatal(err)
	}
	intrinsic, err := IntrinsicGas(data, nil, nil, senderAddr, &senderAddr, new(uint256.Int), rules8037)
	if err != nil {
		t.Fatal(err)
	}
	if floor <= params.MaxTxGas || intrinsic > params.MaxTxGas {
		t.Fatalf("setup: floor %d must exceed cap %d while intrinsic %d stays below",
			floor, params.MaxTxGas, intrinsic)
	}
	// The gas limit covers the floor, so the rejection can only come from
	// the MaxTxGas cap on the floor cost.
	tx := callTx(0, senderAddr, 0, floor+1_000_000, data)
	if _, _, err := applyMsg(t, mkState(senderAlloc(nil)), tx); !errors.Is(err, ErrFloorDataGas) {
		t.Fatalf("expected ErrFloorDataGas, got %v", err)
	}
}

// ========================= Refund and gas used ===========================

// clearSlots deploys a contract that zeroes slots 1..n, each preset to 1.
func clearSlots(addr common.Address, n int) (types.GenesisAlloc, []byte) {
	var code []byte
	storage := make(map[common.Hash]common.Hash, n)
	for s := 1; s <= n; s++ {
		code = append(code, 0x60, 0x00, 0x60, byte(s), 0x55) // PUSH1 0; PUSH1 s; SSTORE
		storage[common.BytesToHash([]byte{byte(s)})] = common.BytesToHash([]byte{1})
	}
	return types.GenesisAlloc{addr: {Code: append(code, 0x00), Storage: storage}}, nil
}

// tx_gas_used_before_refund (peak) exceeds the post-refund gas used.
func TestGasUsedBeforeRefund(t *testing.T) {
	c := common.HexToAddress("0xc1ea0")
	alloc, _ := clearSlots(c, 4)
	res, _, err := applyMsg(t, mkState(senderAlloc(alloc)), callTx(0, c, 0, 1_000_000, nil))
	if err != nil {
		t.Fatal(err)
	}
	if res.MaxUsedGas <= res.UsedGas {
		t.Fatalf("peak %d must exceed post-refund %d", res.MaxUsedGas, res.UsedGas)
	}
}

// The refund is capped at 20% of gas used before refund.
func TestRefundCappedAt20Percent(t *testing.T) {
	c := common.HexToAddress("0xc1ea3")
	alloc, _ := clearSlots(c, 3) // refund (3x4800) exceeds the 20% cap
	res, _, err := applyMsg(t, mkState(senderAlloc(alloc)), callTx(0, c, 0, 1_000_000, nil))
	if err != nil {
		t.Fatal(err)
	}
	if want := res.MaxUsedGas - res.MaxUsedGas/5; res.UsedGas != want {
		t.Fatalf("gas used = %d, want capped %d", res.UsedGas, want)
	}
}

// The EIP-7623 calldata floor is applied after the refund.
func TestRefundCalldataFloorAfterRefund(t *testing.T) {
	data := make([]byte, 1000) // all-zero calldata: floor dominates a bare call
	to := common.HexToAddress("0xeeee")
	floor, _ := FloorDataGas(rules8037, senderAddr, &to, new(uint256.Int), data, nil)
	res, _, err := applyMsg(t, mkState(senderAlloc(nil)), callTx(0, to, 0, 1_000_000, data))
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedGas != floor {
		t.Fatalf("gas used = %d, want floor %d", res.UsedGas, floor)
	}
}

// When the floor exceeds the post-refund gas, it negates part of the refund.
func TestRefundFloorNegatesRefund(t *testing.T) {
	c := common.HexToAddress("0xc1ea1")
	alloc, _ := clearSlots(c, 1)
	data := make([]byte, 1000)
	floor, _ := FloorDataGas(rules8037, senderAddr, &c, new(uint256.Int), data, nil)
	res, _, err := applyMsg(t, mkState(senderAlloc(alloc)), callTx(0, c, 0, 1_000_000, data))
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedGas != floor {
		t.Fatalf("gas used = %d, want floor %d (refund negated)", res.UsedGas, floor)
	}
}

// ========================= Block-level accounting ========================

// The pool tracks regular and state cumulatively in separate counters.
func TestBlockTracksTwoCounters(t *testing.T) {
	gp := NewGasPool(60_000_000)
	if err := gp.ChargeGasAmsterdam(100, 200, 300); err != nil {
		t.Fatal(err)
	}
	if gp.cumulativeRegular != 100 || gp.cumulativeState != 200 {
		t.Fatalf("counters = (%d,%d), want (100,200)", gp.cumulativeRegular, gp.cumulativeState)
	}
}

// Block gas used is the max of the two dimensions.
func TestBlockGasUsedIsMax(t *testing.T) {
	gp := NewGasPool(60_000_000)
	gp.ChargeGasAmsterdam(100, 200, 300)
	if gp.Used() != 200 {
		t.Fatalf("block used = %d, want 200", gp.Used())
	}
}

// Block validity is checked against the max dimension, not the sum.
func TestBlockValidityAgainstMax(t *testing.T) {
	gp := NewGasPool(150)
	// regular 100 + state 120: sum 220 > 150 but max 120 <= 150 is valid.
	if err := gp.ChargeGasAmsterdam(100, 120, 0); err != nil {
		t.Fatalf("max within limit but rejected: %v", err)
	}
	// state 200 alone exceeds the limit.
	if err := gp.ChargeGasAmsterdam(0, 200, 0); err == nil {
		t.Fatal("expected block overflow on state dimension")
	}
}

// The block header gas_used reflects the bottleneck dimension (here, state),
// which the base-fee update then equilibrates against.
func TestBlockBaseFeeUsesMax(t *testing.T) {
	c := common.HexToAddress("0x5707e5")
	var code []byte
	for s := 1; s <= 5; s++ {
		code = append(code, 0x60, byte(s), 0x60, byte(s), 0x55) // SSTORE new slot s
	}
	env := newBALTestEnv(types.GenesisAlloc{c: {Code: append(code, 0x00)}})
	engine := beacon.New(ethash.NewFaker())
	_, blocks, _ := GenerateChainWithGenesis(env.gspec, engine, 1, func(_ int, b *BlockGen) {
		b.AddTx(env.tx(0, &c, big.NewInt(0), 1_000_000, 0, nil))
	})
	if want := 5 * newSlotState; blocks[0].GasUsed() != want {
		t.Fatalf("block gas used = %d, want %d (state bottleneck)", blocks[0].GasUsed(), want)
	}
}

// Receipt cumulative_gas_used is the running sum of per-tx gas (post-refund,
// post-floor), so consecutive receipts differ by exactly that tx's gas.
func TestReceiptCumulativeGasUsed(t *testing.T) {
	env := newBALTestEnv(nil)
	a, b := common.HexToAddress("0xaaaa"), common.HexToAddress("0xbbbb")
	engine := beacon.New(ethash.NewFaker())
	_, _, receipts := GenerateChainWithGenesis(env.gspec, engine, 1, func(_ int, g *BlockGen) {
		g.AddTx(env.tx(0, &a, big.NewInt(1), txGasNewAccount, 0, nil))
		g.AddTx(env.tx(1, &b, big.NewInt(1), txGasNewAccount, 0, nil))
	})
	r := receipts[0]
	if got := r[1].CumulativeGasUsed - r[0].CumulativeGasUsed; got != r[1].GasUsed {
		t.Fatalf("cumulative delta = %d, want tx gas %d", got, r[1].GasUsed)
	}
}

// ======================= EIP-7702 authorizations =========================

// signAuth signs an authorization from authKey for the given delegate and nonce.
func signAuth(t *testing.T, authKey string, delegate common.Address, nonce uint64) (types.SetCodeAuthorization, common.Address) {
	t.Helper()
	k, _ := crypto.HexToECDSA(authKey)
	auth, err := types.SignSetCode(k, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(cfg8037.ChainID), Address: delegate, Nonce: nonce,
	})
	if err != nil {
		t.Fatalf("sign auth: %v", err)
	}
	return auth, crypto.PubkeyToAddress(k.PublicKey)
}

func setCodeTx(nonce uint64, to common.Address, auths []types.SetCodeAuthorization) *types.Transaction {
	return types.MustSignNewTx(senderKey, signer8037, &types.SetCodeTx{
		ChainID: uint256.MustFromBig(cfg8037.ChainID), Nonce: nonce, To: to, Value: new(uint256.Int),
		Gas: 1_000_000, GasFeeCap: new(uint256.Int), GasTipCap: new(uint256.Int), AuthList: auths,
	})
}

const authKeyA = "0202020202020202020202020202020202020202020202020202002020202020"

var delegate8037 = common.HexToAddress("0xde1e8a7e")

// Intrinsic gas charges only the state-independent per-authorization base;
// the state-dependent charges are applied at runtime (EIP-2780).
func TestAuthIntrinsicBaseOnly(t *testing.T) {
	cost, err := IntrinsicGas(nil, nil, []types.SetCodeAuthorization{{}}, common.Address{}, &delegate8037, nil, rules8037)
	if err != nil {
		t.Fatal(err)
	}
	// The recipient touch and the per-authorization authority access (priced
	// into RegularPerAuthBaseCost) are both charged at the cold rate
	// unconditionally at the intrinsic phase (EIP-2780).
	want := params.TxBaseCost2780 + params.ColdAccountAccessAmsterdam + params.RegularPerAuthBaseCost
	if cost != want {
		t.Fatalf("intrinsic gas = %d, want %d", cost, want)
	}
}

// An invalid authorization incurs no runtime state-gas charge.
func TestAuthInvalidRefillFull(t *testing.T) {
	k, _ := crypto.HexToECDSA(authKeyA)
	bad, _ := types.SignSetCode(k, types.SetCodeAuthorization{
		ChainID: *uint256.NewInt(999), Address: delegate8037, Nonce: 0, // wrong chain id
	})
	sdb := mkState(senderAlloc(nil))
	_, gp, err := applyMsg(t, sdb, setCodeTx(0, senderAddr, []types.SetCodeAuthorization{bad}))
	if err != nil {
		t.Fatal(err)
	}
	if gp.cumulativeState != 0 {
		t.Fatalf("state gas = %d, want 0 (fully refilled)", gp.cumulativeState)
	}
}

// A pre-existing authority is not charged for an account leaf; only the
// net-new indicator bytes are charged at runtime.
func TestAuthAccountExistsRefill(t *testing.T) {
	auth, authority := signAuth(t, authKeyA, delegate8037, 0)
	sdb := mkState(senderAlloc(types.GenesisAlloc{authority: {Balance: big.NewInt(1)}}))
	_, gp, err := applyMsg(t, sdb, setCodeTx(0, senderAddr, []types.SetCodeAuthorization{auth}))
	if err != nil {
		t.Fatal(err)
	}
	if gp.cumulativeState != authBaseState {
		t.Fatalf("state gas = %d, want %d (indicator only)", gp.cumulativeState, authBaseState)
	}
}

// Setting a delegation on an already-delegated authority writes no net-new
// bytes (and no account leaf, since the authority exists): no state charge.
func TestAuthSetOnDelegatedRefillBase(t *testing.T) {
	auth, authority := signAuth(t, authKeyA, delegate8037, 0)
	pre := types.AddressToDelegation(common.HexToAddress("0xabcd"))
	sdb := mkState(senderAlloc(types.GenesisAlloc{authority: {Code: pre}}))
	_, gp, err := applyMsg(t, sdb, setCodeTx(0, senderAddr, []types.SetCodeAuthorization{auth}))
	if err != nil {
		t.Fatal(err)
	}
	if gp.cumulativeState != 0 {
		t.Fatalf("state gas = %d, want 0 (nothing net-new)", gp.cumulativeState)
	}
}

// A net-new delegation on a fresh authority is charged the account leaf plus
// the indicator bytes at runtime.
func TestAuthSetNetNewNoRefill(t *testing.T) {
	auth, _ := signAuth(t, authKeyA, delegate8037, 0)
	sdb := mkState(senderAlloc(nil))
	_, gp, err := applyMsg(t, sdb, setCodeTx(0, senderAddr, []types.SetCodeAuthorization{auth}))
	if err != nil {
		t.Fatal(err)
	}
	if gp.cumulativeState != authWorstState {
		t.Fatalf("state gas = %d, want %d (leaf + indicator)", gp.cumulativeState, authWorstState)
	}
}

// Clearing a delegation writes no indicator, so only the (new) account leaf is
// charged at runtime.
func TestAuthClearRefillBase(t *testing.T) {
	auth, _ := signAuth(t, authKeyA, common.Address{}, 0) // clear (address ZERO)
	sdb := mkState(senderAlloc(nil))
	_, gp, err := applyMsg(t, sdb, setCodeTx(0, senderAddr, []types.SetCodeAuthorization{auth}))
	if err != nil {
		t.Fatal(err)
	}
	if want := newAccountState; gp.cumulativeState != want {
		t.Fatalf("state gas = %d, want %d (account leaf only)", gp.cumulativeState, want)
	}
}

// 0->a->0 in one tx: the indicator charge applies when the delegation is set
// and is never credited back when a later auth clears it in the same
// transaction.
func TestAuthClearSameTxNoRefill(t *testing.T) {
	set, authority := signAuth(t, authKeyA, delegate8037, 0)
	clr, _ := signAuth(t, authKeyA, common.Address{}, 1)
	sdb := mkState(senderAlloc(nil))
	_, gp, err := applyMsg(t, sdb, setCodeTx(0, senderAddr, []types.SetCodeAuthorization{set, clr}))
	if err != nil {
		t.Fatal(err)
	}
	_ = authority
	if want := authWorstState; gp.cumulativeState != want {
		t.Fatalf("state gas = %d, want %d (indicator charge kept on clear)", gp.cumulativeState, want)
	}
}

// 0->a->0->b in one tx: the indicator charge applies at most once per
// authority — re-installing a delegation after an intra-tx clear is free.
func TestAuthSetClearSetChargedOnce(t *testing.T) {
	set, _ := signAuth(t, authKeyA, delegate8037, 0)
	clr, _ := signAuth(t, authKeyA, common.Address{}, 1)
	set2, authority := signAuth(t, authKeyA, common.HexToAddress("0xde1e8a7f"), 2)
	sdb := mkState(senderAlloc(nil))
	_, gp, err := applyMsg(t, sdb, setCodeTx(0, senderAddr, []types.SetCodeAuthorization{set, clr, set2}))
	if err != nil {
		t.Fatal(err)
	}
	// The final delegation is installed and the indicator was paid exactly once.
	if _, delegated := types.ParseDelegation(sdb.GetCode(authority)); !delegated {
		t.Fatal("final delegation not installed")
	}
	if want := authWorstState; gp.cumulativeState != want {
		t.Fatalf("state gas = %d, want %d (leaf + indicator exactly once)", gp.cumulativeState, want)
	}
}

// The same authority across two auths is charged for its account only once.
func TestAuthDuplicateAuthorityOnce(t *testing.T) {
	a0, _ := signAuth(t, authKeyA, delegate8037, 0)
	a1, _ := signAuth(t, authKeyA, delegate8037, 1)
	sdb := mkState(senderAlloc(nil))
	_, gp, err := applyMsg(t, sdb, setCodeTx(0, senderAddr, []types.SetCodeAuthorization{a0, a1}))
	if err != nil {
		t.Fatal(err)
	}
	if gp.cumulativeState != authWorstState {
		t.Fatalf("state gas = %d, want %d (leaf+indicator once)", gp.cumulativeState, authWorstState)
	}
}

// ===================== System contracts / system calls ===================

// System call gas limit keeps 30M regular plus a state reservoir for new slots.
func TestSystemCallGasLimit(t *testing.T) {
	limit, budget := systemCallGasBudget(amsterdamCoreEVM(mkState(nil)))
	if limit != 30_000_000 || budget.RegularGas != 30_000_000 {
		t.Fatalf("limit/regular = %d/%d, want 30M/30M", limit, budget.RegularGas)
	}
}

// The extra system budget is placed in the state reservoir (16 new slots).
func TestSystemCallExtraInReservoir(t *testing.T) {
	_, budget := systemCallGasBudget(amsterdamCoreEVM(mkState(nil)))
	want := uint64(params.SystemMaxSStoresPerCall * params.CostPerStateByte * params.StorageCreationSize)
	if budget.StateGas != want {
		t.Fatalf("reservoir = %d, want %d", budget.StateGas, want)
	}
}

// System calls do not contribute to either block dimension: an empty block
// (whose system calls still write state) reports zero gas used.
func TestSystemCallNotCountedInBlock(t *testing.T) {
	env := newBALTestEnv(nil)
	engine := beacon.New(ethash.NewFaker())
	_, blocks, _ := GenerateChainWithGenesis(env.gspec, engine, 1, func(_ int, b *BlockGen) {})
	if blocks[0].GasUsed() != 0 {
		t.Fatalf("block gas used = %d, want 0 (system calls excluded)", blocks[0].GasUsed())
	}
}

func TestParallelReservationOverflowRejected(t *testing.T) {
	env := newBALTestEnv(nil)
	env.gspec.GasLimit = 30_000_000
	engine := beacon.New(ethash.NewFaker())

	// A single self-transfer with a 5,000,000 gas limit but only ~21,000 of
	// actual usage (recipient exists, no new state).
	to := env.from
	_, blocks, _ := GenerateChainWithGenesis(env.gspec, engine, 1, func(_ int, b *BlockGen) {
		b.AddTx(env.tx(0, &to, big.NewInt(1), 5_000_000, 0, nil))
	})
	valid := blocks[0]

	bc, err := NewBlockChain(rawdb.NewMemoryDatabase(), env.gspec, engine, nil)
	if err != nil {
		t.Fatalf("new blockchain: %v", err)
	}
	defer bc.Stop()

	// The block as built (30M limit, well above the 5M reservation) is accepted:
	// the reservation check must not over-reject valid blocks.
	statedb, err := bc.State()
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	if _, err := NewStateProcessor(bc).Process(context.Background(), valid, statedb, nil, vm.Config{}); err != nil {
		t.Fatalf("valid block rejected by parallel processor: %v", err)
	}

	// Lower the block gas limit below the transaction's worst-case reservation
	// (5,000,000) while keeping it above the actual usage (~21,000). The
	// transaction can no longer be admitted, so the block is invalid.
	hdr := valid.Header()
	hdr.GasLimit = 100_000
	invalid := valid.WithSeal(hdr)

	statedb, err = bc.State()
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	_, err = NewStateProcessor(bc).Process(context.Background(), invalid, statedb, nil, vm.Config{})
	if !errors.Is(err, ErrGasLimitReached) {
		t.Fatalf("parallel processor accepted a reservation-overflow block (err = %v), want ErrGasLimitReached", err)
	}
}
