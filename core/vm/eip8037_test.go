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

// Opcode-level tests for EIP-8037 (multidimensional state-gas metering).
// They drive a single frame via evm.Call and assert the state-gas accounting
// exposed by the returned GasBudget (UsedStateGas / StateGas / Spilled).

package vm

import (
	"math"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// state-gas charges in units (CPSB applied).
var (
	stateGasNewAccount = int64(params.AccountCreationSize * params.CostPerStateByte) // 183,600
	stateGasNewSlot    = int64(params.StorageCreationSize * params.CostPerStateByte) // 97,920
)

// amsterdam8037Config clones MergedTestChainConfig with Amsterdam (EIP-8037) live.
func amsterdam8037Config() *params.ChainConfig {
	cfg := *params.MergedTestChainConfig
	cfg.AmsterdamTime = new(uint64)
	return &cfg
}

// amsterdam8037EVM builds an EVM with real value transfers and CPSB wired in.
func amsterdam8037EVM(statedb StateDB) *EVM {
	ctx := BlockContext{
		CanTransfer: func(db StateDB, addr common.Address, amount *uint256.Int) bool {
			return db.GetBalance(addr).Cmp(amount) >= 0
		},
		Transfer: func(db StateDB, sender, recipient common.Address, amount *uint256.Int, _ *params.Rules) {
			db.SubBalance(sender, amount, tracing.BalanceChangeTransfer)
			db.AddBalance(recipient, amount, tracing.BalanceChangeTransfer)
		},
		BlockNumber:      big.NewInt(0),
		Random:           &common.Hash{},
		CostPerStateByte: params.CostPerStateByte,
	}
	return NewEVM(ctx, statedb, amsterdam8037Config(), Config{})
}

// run8037 executes code at a contract address and returns the call's return
// data and the resulting budget. setup mutates the pre-state (before Finalise)
// and may fund the contract.
func run8037(t *testing.T, code []byte, gas GasBudget, value *uint256.Int, setup func(db *state.StateDB, self common.Address)) ([]byte, GasBudget, error) {
	t.Helper()
	self := common.BytesToAddress([]byte("self"))
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	statedb.CreateAccount(self)
	statedb.SetCode(self, code, tracing.CodeChangeUnspecified)
	if setup != nil {
		setup(statedb, self)
	}
	statedb.Finalise(true)
	ret, result, err := amsterdam8037EVM(statedb).Call(common.Address{}, self, nil, gas, value)
	assertBudgetSane(t, gas, result)
	return ret, result, err
}

// assertBudgetSane verifies the GasBudget conservation identities that must hold
// for any frame exit (success, revert or halt), validating the whole vector.
//
//	regular: RegularGas + UsedRegularGas + Spilled == initial.RegularGas
//	state:   StateGas + UsedStateGas               == initial.StateGas + Spilled
//	scalar:  Used(initial)                         == UsedRegularGas + UsedStateGas
func assertBudgetSane(t *testing.T, initial, got GasBudget) {
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

// hugeBudget is a budget that never runs out, with a separate state reservoir.
func hugeBudget() GasBudget { return NewGasBudget(math.MaxUint64/2, math.MaxUint64/2) }

// sstore returns "PUSH val; PUSH slot; SSTORE" bytecode.
func sstore(slot, val byte) []byte { return []byte{0x60, val, 0x60, slot, 0x55} }

// setSlot commits an original (pre-tx) value into a storage slot.
func setSlot(slot, val byte) func(*state.StateDB, common.Address) {
	return func(db *state.StateDB, self common.Address) {
		db.SetState(self, common.BytesToHash([]byte{slot}), common.BytesToHash([]byte{val}))
	}
}

// ============================ SSTORE state-gas =============================

// 0 -> 0 -> x: brand-new slot is charged one storage-creation.
func TestSStoreNewSlot(t *testing.T) {
	_, res, err := run8037(t, sstore(0, 1), hugeBudget(), new(uint256.Int), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedStateGas != stateGasNewSlot {
		t.Fatalf("state gas = %d, want %d", res.UsedStateGas, stateGasNewSlot)
	}
}

// 0 -> x -> 0: slot created then cleared in-tx, net charge refilled to zero.
func TestSStoreClearZeroAtStart(t *testing.T) {
	code := append(sstore(0, 1), sstore(0, 0)...)
	_, res, err := run8037(t, code, hugeBudget(), new(uint256.Int), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedStateGas != 0 {
		t.Fatalf("state gas = %d, want 0 (refilled)", res.UsedStateGas)
	}
}

// x -> x -> 0: clearing a slot non-zero at tx start makes no state adjustment.
func TestSStoreClearOriginalNonzero(t *testing.T) {
	_, res, err := run8037(t, sstore(0, 0), hugeBudget(), new(uint256.Int), setSlot(0, 1))
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedStateGas != 0 {
		t.Fatalf("state gas = %d, want 0", res.UsedStateGas)
	}
}

// x -> 0 -> x: clearing then restoring the original value makes no adjustment.
func TestSStoreRestoreOriginal(t *testing.T) {
	code := append(sstore(0, 0), sstore(0, 1)...)
	_, res, err := run8037(t, code, hugeBudget(), new(uint256.Int), setSlot(0, 1))
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedStateGas != 0 {
		t.Fatalf("state gas = %d, want 0", res.UsedStateGas)
	}
}

// x -> y: overwriting an existing slot with another value makes no adjustment.
func TestSStoreOtherWrite(t *testing.T) {
	_, res, err := run8037(t, sstore(0, 2), hugeBudget(), new(uint256.Int), setSlot(0, 1))
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedStateGas != 0 {
		t.Fatalf("state gas = %d, want 0", res.UsedStateGas)
	}
}

// New-slot charge is metered at the opcode: with a reservoir smaller than the
// charge it spills into regular gas exactly at the SSTORE.
func TestSStoreChargedAtOpcodeEnd(t *testing.T) {
	_, res, err := run8037(t, sstore(0, 1), NewGasBudget(1_000_000, 100), new(uint256.Int), nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := uint64(stateGasNewSlot) - 100; res.Spilled != want {
		t.Fatalf("spilled = %d, want %d", res.Spilled, want)
	}
}

// The SSTORE reentrancy sentry checks gas_left only; the reservoir is excluded.
// Uses a noop write (1->1->1) so the sentry is the sole gate.
func TestSStoreStipendExcludesReservoir(t *testing.T) {
	// regular at the sentry, huge reservoir: must still fail.
	if _, _, err := run8037(t, sstore(0, 1), NewGasBudget(2306, math.MaxUint64/2), new(uint256.Int), setSlot(0, 1)); err == nil {
		t.Fatal("expected sentry failure with regular gas at the limit")
	}
	// one more regular gas clears the sentry.
	if _, _, err := run8037(t, sstore(0, 1), NewGasBudget(2307, math.MaxUint64/2), new(uint256.Int), setSlot(0, 1)); err != nil {
		t.Fatalf("unexpected failure above sentry: %v", err)
	}
}

// ---- CALL / CREATE bytecode helpers ----

var (
	freshAddr    = common.BytesToAddress([]byte("fresh-target"))
	existAddr    = common.BytesToAddress([]byte("exist-target"))
	balanceAddr  = common.BytesToAddress([]byte("balance-only"))
	childAddr    = common.BytesToAddress([]byte("child-frame"))
	revertInit   = []byte{0x60, 0x00, 0x60, 0x00, 0xfd} // PUSH1 0; PUSH1 0; REVERT
	invalidInit  = []byte{0xfe}                         // INVALID
	deploy3Init  = []byte{0x60, 0x03, 0x60, 0x00, 0xf3} // return 3 bytes of code
	deploy0Init  = []byte{0x60, 0x00, 0x60, 0x00, 0xf3} // return 0 bytes of code
	stop         = []byte{0x00}
	revertTail   = []byte{0x60, 0x00, 0x60, 0x00, 0xfd}
	invalidTail  = []byte{0xfe}
	stateDeposit = int64(3 * params.CostPerStateByte) // 3-byte code deposit (4,590)
)

// callCode builds bytecode that CALLs `to` forwarding `value` wei and all gas,
// followed by `tail`.
func callCode(to common.Address, value byte, tail []byte) []byte {
	b := []byte{0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, value, 0x73}
	b = append(b, to.Bytes()...)
	b = append(b, 0x5a, 0xf1) // GAS; CALL
	return append(b, tail...)
}

// deployCode builds bytecode that MSTOREs init and runs CREATE/CREATE2 with value.
func deployCode(init []byte, create2 bool, value byte) []byte {
	word := make([]byte, 32)
	copy(word[32-len(init):], init)
	off, sz := byte(32-len(init)), byte(len(init))
	b := append([]byte{0x7f}, word...) // PUSH32 init-word
	b = append(b, 0x60, 0x00, 0x52)    // PUSH1 0; MSTORE
	if create2 {
		b = append(b, 0x60, 0x00, 0x60, sz, 0x60, off, 0x60, value, 0xf5) // salt,size,off,value; CREATE2
	} else {
		b = append(b, 0x60, sz, 0x60, off, 0x60, value, 0xf0) // size,off,value; CREATE
	}
	return append(b, 0x00) // STOP
}

func fund(addr common.Address, wei int64) func(*state.StateDB, common.Address) {
	return func(db *state.StateDB, _ common.Address) {
		db.AddBalance(addr, uint256.NewInt(uint64(wei)), tracing.BalanceChangeUnspecified)
	}
}

// ====================== CALL* new-account state-gas =======================

// CALL with value to a non-existent account charges one account creation.
func TestCallValueToNewAccount(t *testing.T) {
	_, res, err := run8037(t, callCode(freshAddr, 1, stop), hugeBudget(), new(uint256.Int), fund(common.BytesToAddress([]byte("self")), 10))
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedStateGas != stateGasNewAccount {
		t.Fatalf("state gas = %d, want %d", res.UsedStateGas, stateGasNewAccount)
	}
}

// CALL with value to an existing (code-bearing) account is not charged.
func TestCallValueToExistingAccount(t *testing.T) {
	setup := func(db *state.StateDB, self common.Address) {
		db.CreateAccount(existAddr)
		db.SetCode(existAddr, stop, tracing.CodeChangeUnspecified)
		db.AddBalance(self, uint256.NewInt(10), tracing.BalanceChangeUnspecified)
	}
	_, res, err := run8037(t, callCode(existAddr, 1, stop), hugeBudget(), new(uint256.Int), setup)
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedStateGas != 0 {
		t.Fatalf("state gas = %d, want 0", res.UsedStateGas)
	}
}

// CALL with zero value creates no account, so nothing is charged.
func TestCallZeroValueToNewAccount(t *testing.T) {
	_, res, err := run8037(t, callCode(freshAddr, 0, stop), hugeBudget(), new(uint256.Int), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedStateGas != 0 {
		t.Fatalf("state gas = %d, want 0", res.UsedStateGas)
	}
}

// CALL that fails before the child frame (insufficient balance) refills the charge.
func TestCallInsufficientBalanceRefill(t *testing.T) {
	// self has no balance, so the value transfer fails the CanTransfer check.
	_, res, err := run8037(t, callCode(freshAddr, 1, stop), hugeBudget(), new(uint256.Int), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedStateGas != 0 {
		t.Fatalf("state gas = %d, want 0 (refilled)", res.UsedStateGas)
	}
}

// A new-account charge is refilled when its frame reverts.
func TestCallChildRevertRefill(t *testing.T) {
	code := callCode(freshAddr, 1, revertTail)
	_, res, err := run8037(t, code, hugeBudget(), new(uint256.Int), fund(common.BytesToAddress([]byte("self")), 10))
	if err != ErrExecutionReverted {
		t.Fatalf("err = %v, want revert", err)
	}
	if res.UsedStateGas != 0 {
		t.Fatalf("state gas = %d, want 0 (refilled)", res.UsedStateGas)
	}
}

// A new-account charge is refilled when its frame halts exceptionally.
func TestCallChildExceptionalHaltRefill(t *testing.T) {
	code := callCode(freshAddr, 1, invalidTail)
	_, res, err := run8037(t, code, hugeBudget(), new(uint256.Int), fund(common.BytesToAddress([]byte("self")), 10))
	if err == nil || err == ErrExecutionReverted {
		t.Fatalf("err = %v, want exceptional halt", err)
	}
	if res.UsedStateGas != 0 {
		t.Fatalf("state gas = %d, want 0 (refilled)", res.UsedStateGas)
	}
}

// An account with balance but no code/nonce is existent: no account charge.
func TestCallBalanceOnlyAccountIsExistent(t *testing.T) {
	setup := func(db *state.StateDB, self common.Address) {
		db.AddBalance(balanceAddr, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
		db.AddBalance(self, uint256.NewInt(10), tracing.BalanceChangeUnspecified)
	}
	_, res, err := run8037(t, callCode(balanceAddr, 1, stop), hugeBudget(), new(uint256.Int), setup)
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedStateGas != 0 {
		t.Fatalf("state gas = %d, want 0", res.UsedStateGas)
	}
}

// ===================== CREATE / CREATE2 state-gas =========================

// CREATE to a fresh address charges account creation plus code deposit.
func TestCreateNewAccount(t *testing.T) {
	_, res, err := run8037(t, deployCode(deploy3Init, false, 0), hugeBudget(), new(uint256.Int), nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := stateGasNewAccount + stateDeposit; res.UsedStateGas != want {
		t.Fatalf("state gas = %d, want %d", res.UsedStateGas, want)
	}
}

// CREATE onto a pre-existing (balance-only) leaf refills the account portion;
// only the code deposit is charged.
func TestCreatePreexistingTarget(t *testing.T) {
	setup := func(db *state.StateDB, self common.Address) {
		derived := crypto.CreateAddress(self, db.GetNonce(self))
		db.AddBalance(derived, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
	}
	_, res, err := run8037(t, deployCode(deploy3Init, false, 0), hugeBudget(), new(uint256.Int), setup)
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedStateGas != stateDeposit {
		t.Fatalf("state gas = %d, want %d", res.UsedStateGas, stateDeposit)
	}
}

// CREATE whose init code reverts refills the account charge and deposits nothing.
func TestCreateInitRevertRefill(t *testing.T) {
	_, res, err := run8037(t, deployCode(revertInit, false, 0), hugeBudget(), new(uint256.Int), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedStateGas != 0 {
		t.Fatalf("state gas = %d, want 0 (refilled)", res.UsedStateGas)
	}
}

// CREATE whose init code halts exceptionally refills the account charge.
func TestCreateInitOOGRefill(t *testing.T) {
	_, res, err := run8037(t, deployCode(invalidInit, false, 0), hugeBudget(), new(uint256.Int), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedStateGas != 0 {
		t.Fatalf("state gas = %d, want 0 (refilled)", res.UsedStateGas)
	}
}

// CREATE onto an address collision (existing nonce) refills the account charge.
func TestCreateAddressCollisionRefill(t *testing.T) {
	setup := func(db *state.StateDB, self common.Address) {
		derived := crypto.CreateAddress(self, db.GetNonce(self))
		db.SetNonce(derived, 1, tracing.NonceChangeUnspecified)
	}
	_, res, err := run8037(t, deployCode(deploy3Init, false, 0), hugeBudget(), new(uint256.Int), setup)
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedStateGas != 0 {
		t.Fatalf("state gas = %d, want 0 (refilled)", res.UsedStateGas)
	}
}

// CREATE with value exceeding balance fails before the frame and is refilled.
func TestCreateInsufficientBalanceRefill(t *testing.T) {
	// self has no balance; CREATE forwards value 1.
	_, res, err := run8037(t, deployCode(deploy3Init, false, 1), hugeBudget(), new(uint256.Int), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedStateGas != 0 {
		t.Fatalf("state gas = %d, want 0 (refilled)", res.UsedStateGas)
	}
}

// CREATE2 charges account creation plus code deposit identically to CREATE.
func TestCreate2SameSemantics(t *testing.T) {
	_, res, err := run8037(t, deployCode(deploy3Init, true, 0), hugeBudget(), new(uint256.Int), nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := stateGasNewAccount + stateDeposit; res.UsedStateGas != want {
		t.Fatalf("state gas = %d, want %d", res.UsedStateGas, want)
	}
}

// The code-deposit portion is charged per byte independently of the account
// charge: the delta between a 3-byte and 0-byte deploy is exactly 3 x CPSB.
func TestCreateCodeDepositChargedSeparately(t *testing.T) {
	_, big3, err := run8037(t, deployCode(deploy3Init, false, 0), hugeBudget(), new(uint256.Int), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, big0, err := run8037(t, deployCode(deploy0Init, false, 0), hugeBudget(), new(uint256.Int), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := big3.UsedStateGas - big0.UsedStateGas; got != stateDeposit {
		t.Fatalf("deposit delta = %d, want %d", got, stateDeposit)
	}
}

// ========================= SELFDESTRUCT state-gas =========================

// selfdestruct sending balance to a non-existent beneficiary creates it.
func TestSelfdestructCreatesNewAccount(t *testing.T) {
	code := append([]byte{0x73}, freshAddr.Bytes()...) // PUSH20 beneficiary
	code = append(code, 0xff)                          // SELFDESTRUCT
	_, res, err := run8037(t, code, hugeBudget(), new(uint256.Int), fund(common.BytesToAddress([]byte("self")), 10))
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedStateGas != stateGasNewAccount {
		t.Fatalf("state gas = %d, want %d", res.UsedStateGas, stateGasNewAccount)
	}
}

// selfdestruct to an existing beneficiary creates no account.
func TestSelfdestructToExistingAccount(t *testing.T) {
	setup := func(db *state.StateDB, self common.Address) {
		db.AddBalance(existAddr, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
		db.AddBalance(self, uint256.NewInt(10), tracing.BalanceChangeUnspecified)
	}
	code := append([]byte{0x73}, existAddr.Bytes()...)
	code = append(code, 0xff)
	_, res, err := run8037(t, code, hugeBudget(), new(uint256.Int), setup)
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedStateGas != 0 {
		t.Fatalf("state gas = %d, want 0", res.UsedStateGas)
	}
}

// A contract created and self-destructed in the same tx gets no refill: the
// account-creation charge stands.
func TestSelfdestructSameTxAccountNoRefill(t *testing.T) {
	// init code selfdestructs to self (existing), so only the create charges.
	self := common.BytesToAddress([]byte("self"))
	init := append([]byte{0x73}, self.Bytes()...)
	init = append(init, 0xff)
	_, res, err := run8037(t, deployCode(init, false, 0), hugeBudget(), new(uint256.Int), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedStateGas != stateGasNewAccount {
		t.Fatalf("state gas = %d, want %d (no refill)", res.UsedStateGas, stateGasNewAccount)
	}
}

// selfdestruct of a pre-existing account refills nothing (EIP-6780: not removed).
func TestSelfdestructPreexistingNoRefill(t *testing.T) {
	setup := func(db *state.StateDB, self common.Address) {
		db.AddBalance(existAddr, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
	}
	code := append([]byte{0x73}, existAddr.Bytes()...)
	code = append(code, 0xff)
	_, res, err := run8037(t, code, hugeBudget(), new(uint256.Int), setup)
	if err != nil {
		t.Fatal(err)
	}
	if res.UsedStateGas != 0 {
		t.Fatalf("state gas = %d, want 0", res.UsedStateGas)
	}
}

// ===================== Reservoir / gas_left mechanics =====================

// State-gas is drawn from the reservoir first: a charge within reservoir size
// does not spill into regular gas.
func TestReservoirDrawnFirst(t *testing.T) {
	_, res, err := run8037(t, sstore(0, 1), NewGasBudget(1_000_000, 200_000), new(uint256.Int), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Spilled != 0 {
		t.Fatalf("spilled = %d, want 0", res.Spilled)
	}
	if want := uint64(200_000 - stateGasNewSlot); res.StateGas != want {
		t.Fatalf("reservoir left = %d, want %d", res.StateGas, want)
	}
}

// The GAS opcode returns gas_left only, excluding the reservoir.
func TestGasOpcodeExcludesReservoir(t *testing.T) {
	code := []byte{0x5a, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3} // GAS; MSTORE; RETURN(32)
	ret, _, err := run8037(t, code, NewGasBudget(1_000_000, 500_000), new(uint256.Int), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := new(uint256.Int).SetBytes(ret).Uint64(); got != 1_000_000-GasQuickStep {
		t.Fatalf("GAS = %d, want %d (reservoir excluded)", got, 1_000_000-GasQuickStep)
	}
}

// Refills are LIFO: borrowed regular gas is repaid before the reservoir. With a
// zero reservoir, a 0->x->0 SSTORE repays the spill and leaves the reservoir at 0.
func TestLIFORefillOrder(t *testing.T) {
	code := append(sstore(0, 1), sstore(0, 0)...)
	_, res, err := run8037(t, code, NewGasBudget(1_000_000, 0), new(uint256.Int), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Spilled != 0 || res.StateGas != 0 || res.UsedStateGas != 0 {
		t.Fatalf("after LIFO refill: spilled=%d reservoir=%d used=%d, want 0/0/0", res.Spilled, res.StateGas, res.UsedStateGas)
	}
}

// State-gas charged inside a child frame is refilled at the frame boundary when
// the child reverts or halts.
func TestStateGasMeteredAtFrameBoundary(t *testing.T) {
	for _, tt := range []struct {
		name string
		tail []byte
	}{
		{"revert", revertTail},
		{"halt", invalidTail},
	} {
		t.Run(tt.name, func(t *testing.T) {
			childCode := append(sstore(0, 1), tt.tail...)
			setup := func(db *state.StateDB, self common.Address) {
				db.CreateAccount(childAddr)
				db.SetCode(childAddr, childCode, tracing.CodeChangeUnspecified)
			}
			_, res, err := run8037(t, callCode(childAddr, 0, stop), hugeBudget(), new(uint256.Int), setup)
			if err != nil {
				t.Fatal(err)
			}
			if res.UsedStateGas != 0 {
				t.Fatalf("state gas = %d, want 0 (refilled at boundary)", res.UsedStateGas)
			}
		})
	}
}

// ===================== LIFO refill vector invariant =========================

// Charge A then B (both spilling into regular because the reservoir is too
// small), then refill only A. The refill must repay the borrowed regular gas
// first (Spilled -> 0) before crediting the reservoir, leaving B outstanding.
func TestLIFORefillRepaysRegularBeforeReservoir(t *testing.T) {
	initial := NewGasBudget(1000, 100) // reservoir covers only 100 of state gas
	b := initial

	b.ChargeState(150) // A: 100 from reservoir, 50 spills into regular
	b.ChargeState(30)  // B: reservoir empty, all 30 spills
	if b.Spilled != 80 || b.StateGas != 0 {
		t.Fatalf("after A+B: spilled=%d reservoir=%d, want 80/0", b.Spilled, b.StateGas)
	}

	b.RefundState(150) // refill A: repay 80 to regular first, 70 tops reservoir
	if b.Spilled != 0 {
		t.Fatalf("spilled=%d, want 0 (regular repaid before reservoir)", b.Spilled)
	}
	if b.StateGas != 70 {
		t.Fatalf("reservoir=%d, want 70 (remainder after repaying regular)", b.StateGas)
	}
	assertBudgetSane(t, initial, b)
}

// Fuzz arbitrary sequences of state/regular charges and LIFO refills around the
// reservoir/spill boundary: the GasBudget vector must stay self-consistent after
// every op and across all three frame-exit forms, and refilling every charge
// must restore the state side exactly (reservoir to initial, nothing borrowed).
func TestLIFOVectorInvariantUnderRandomOps(t *testing.T) {
	rng := rand.New(rand.NewSource(8037))
	for trial := 0; trial < 2000; trial++ {
		initial := NewGasBudget(1_000_000, uint64(rng.Intn(1000)))
		b := initial
		outstanding := int64(0) // state-gas charged but not yet refilled
		for step := 0; step < 40; step++ {
			switch rng.Intn(3) {
			case 0: // state charge (may spill into regular)
				if s := uint64(rng.Intn(400)); b.CanAfford(GasCosts{StateGas: s}) {
					b.ChargeState(s)
					outstanding += int64(s)
				}
			case 1: // regular charge
				if r := uint64(rng.Intn(400)); b.CanAfford(GasCosts{RegularGas: r}) {
					b.ChargeRegular(r)
				}
			case 2: // LIFO refill of part of the outstanding state gas
				if outstanding > 0 {
					s := uint64(rng.Int63n(outstanding) + 1)
					b.RefundState(s)
					outstanding -= int64(s)
				}
			}
			assertBudgetSane(t, initial, b)
			assertBudgetSane(t, initial, b.ExitSuccess())
			assertBudgetSane(t, initial, b.ExitRevert())
			assertBudgetSane(t, initial, b.ExitHalt())
		}
		if outstanding > 0 {
			b.RefundState(uint64(outstanding))
		}
		if b.Spilled != 0 || b.StateGas != initial.StateGas || b.UsedStateGas != 0 {
			t.Fatalf("trial %d: after full refill spilled=%d reservoir=%d used=%d, want 0/%d/0",
				trial, b.Spilled, b.StateGas, b.UsedStateGas, initial.StateGas)
		}
	}
}

// ================== Halting frame terminal state (nested) ===================

func concat(parts ...[]byte) []byte {
	var b []byte
	for _, p := range parts {
		b = append(b, p...)
	}
	return b
}

// assertHalted checks the predictable terminal budget of an exceptionally
// halted frame: regular gas fully consumed, state restored to the frame's
// initial reservoir, and no net state-gas used.
func assertHalted(t *testing.T, initial, got GasBudget) {
	t.Helper()
	if got.RegularGas != 0 {
		t.Fatalf("RegularGas = %d, want 0 (gas_left consumed on halt)", got.RegularGas)
	}
	if got.StateGas != initial.StateGas {
		t.Fatalf("StateGas = %d, want %d (reservoir restored)", got.StateGas, initial.StateGas)
	}
	if got.UsedStateGas != 0 {
		t.Fatalf("UsedStateGas = %d, want 0 (all refilled)", got.UsedStateGas)
	}
}

var (
	haltGrandchild = common.BytesToAddress([]byte("grandchild"))
	haltOKChild    = common.BytesToAddress([]byte("child-ok"))   // succeeds, calls grandchild
	haltBadChild   = common.BytesToAddress([]byte("child-halt")) // SSTOREs then INVALID
)

// haltFrameChildren is a run8037 setup that funds self and deploys a 3-level
// child set: a success child that itself calls a grandchild, and a halting child.
func haltFrameChildren(db *state.StateDB, self common.Address) {
	db.AddBalance(self, uint256.NewInt(1000), tracing.BalanceChangeUnspecified)
	db.CreateAccount(haltGrandchild)
	db.SetCode(haltGrandchild, concat(sstore(5, 5), []byte{0x00}), tracing.CodeChangeUnspecified) // new slot; STOP
	db.CreateAccount(haltOKChild)
	db.SetCode(haltOKChild, concat(sstore(1, 1), callCode(haltGrandchild, 0, nil), []byte{0x00}), tracing.CodeChangeUnspecified)
	db.CreateAccount(haltBadChild)
	db.SetCode(haltBadChild, concat(sstore(3, 3), []byte{0xfe}), tracing.CodeChangeUnspecified) // new slot; INVALID
}

// A frame that charges state, drives a successful child (with a grandchild), a
// halting child and a new-account call, then halts, returns the predictable
// terminal budget regardless of all the descendant activity.
func TestHaltFrameTerminalState(t *testing.T) {
	top := concat(
		sstore(0, 1),                   // self: new slot
		callCode(haltOKChild, 0, nil),  // child + grandchild succeed
		callCode(haltBadChild, 0, nil), // descendant halts (contained)
		callCode(freshAddr, 1, nil),    // new-account charge
		[]byte{0xfe},                   // this frame halts
	)
	initial := NewGasBudget(2_000_000, 300_000)
	_, res, err := run8037(t, top, initial, new(uint256.Int), haltFrameChildren)
	if err == nil || err == ErrExecutionReverted {
		t.Fatalf("err = %v, want exceptional halt", err)
	}
	assertHalted(t, initial, res)
}

// Fuzz: arbitrary sequences of state writes, child calls (success / halting) and
// new-account calls, always terminated by INVALID. However the descendants
// behave, a halted frame's terminal budget is always (0, initial reservoir, 0).
func TestHaltFrameTerminalStateFuzz(t *testing.T) {
	rng := rand.New(rand.NewSource(80371))
	for trial := 0; trial < 400; trial++ {
		steps := [][]byte{
			sstore(byte(1+rng.Intn(20)), 1),
			callCode(haltOKChild, 0, nil),
			callCode(haltBadChild, 0, nil),
			callCode(freshAddr, 1, nil),
		}
		var code []byte
		for n := 1 + rng.Intn(8); n > 0; n-- {
			code = append(code, steps[rng.Intn(len(steps))]...)
		}
		code = append(code, 0xfe) // halt
		initial := NewGasBudget(3_000_000, uint64(rng.Intn(400_000)))
		_, res, err := run8037(t, code, initial, new(uint256.Int), haltFrameChildren)
		if err == nil || err == ErrExecutionReverted {
			t.Fatalf("trial %d: err = %v, want halt", trial, err)
		}
		assertHalted(t, initial, res)
	}
}
