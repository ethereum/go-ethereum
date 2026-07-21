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

// Opcode-level tests for EIP-8038 (state-access gas cost update). They reuse the
// Amsterdam harness from eip8037_test.go and assert the re-priced regular-gas,
// state-gas and refund-counter accounting.

package vm

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// run8038 executes code at a contract address under the Amsterdam ruleset and
// returns the resulting budget together with the transaction's refund counter.
func run8038(t *testing.T, code []byte, gas GasBudget, value *uint256.Int, setup func(*state.StateDB, common.Address)) (GasBudget, uint64, error) {
	t.Helper()
	self := common.BytesToAddress([]byte("self"))
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	statedb.CreateAccount(self)
	statedb.SetCode(self, code, tracing.CodeChangeUnspecified)
	if setup != nil {
		setup(statedb, self)
	}
	statedb.Finalise(true)
	_, result, err := amsterdam8037EVM(statedb).Call(common.Address{}, self, nil, gas, value)
	return result, statedb.GetRefund(), err
}

// TestEIP8038SStore exercises SSTORE under Amsterdam (EIP-8037 + EIP-8038),
// asserting the two-dimensional charge (regular + state gas) and the net refund
// counter. It covers single stores in isolation (the EIP-8038 cases-table rows,
// cold access), the warm-access variants, the dirty-slot refund reversals and
// multi-store round trips.
//
// Each sstore() is "PUSH1 val; PUSH1 slot; SSTORE", so the non-SSTORE overhead is
// 6 gas (two PUSH1) per store. The first store to a slot is cold and the rest are
// warm, so the access component is COLD_STORAGE_ACCESS + (n-1) * WARM_ACCESS for n
// stores. STORAGE_WRITE is charged once per "first change" (current == original).
// GAS_STORAGE_SET is state gas, charged when a slot is created from zero and
// refilled to the reservoir when that creation is undone within the same tx.
func TestEIP8038SStore(t *testing.T) {
	const (
		push  = uint64(6) // two PUSH1 per SSTORE
		cold  = params.ColdStorageAccessAmsterdam
		warm  = params.WarmStorageReadCostEIP2929
		write = params.StorageWriteAmsterdam
		clear = params.StorageClearRefundAmsterdam
	)
	set := uint64(params.StorageCreationSize * params.CostPerStateByte) // GAS_STORAGE_SET

	// access(n) is the access-only regular cost for n stores: cold first, warm rest.
	access := func(n uint64) uint64 { return cold + (n-1)*warm }

	cases := []struct {
		name      string
		orig      byte   // committed (pre-tx) value; 0 means a fresh slot
		vals      []byte // values written to slot 0, in order
		wantReg   uint64
		wantState int64
		wantRfnd  uint64
	}{
		// Single store, cold access (EIP-8038 cases table, Cold rows + noop).
		{"noop (1->1)", 1, []byte{1}, push + cold, 0, 0},
		{"create (0->1)", 0, []byte{1}, push + cold + write, int64(set), 0},
		{"first change (1->2)", 1, []byte{2}, push + cold + write, 0, 0},
		{"clear (1->0)", 1, []byte{0}, push + cold + write, 0, clear},
		// Two stores, warm access on the second (Warm rows of the cases table).
		{"create warm (0->0->1)", 0, []byte{0, 1}, 2*push + access(2) + write, int64(set), 0},
		{"first change warm (1->1->2)", 1, []byte{1, 2}, 2*push + access(2) + write, 0, 0},
		{"clear warm (1->1->0)", 1, []byte{1, 0}, 2*push + access(2) + write, 0, clear},
		{"dirty modified again (1->2->3)", 1, []byte{2, 3}, 2*push + access(2) + write, 0, 0},
		// Two stores, refund reversals when a slot returns toward its original.
		{"reset to zero (0->1->0)", 0, []byte{1, 0}, 2*push + access(2) + write, 0, write},
		{"reset to original (1->2->1)", 1, []byte{2, 1}, 2*push + access(2) + write, 0, write},
		{"cleared then restored (1->0->1)", 1, []byte{0, 1}, 2*push + access(2) + write, 0, write},
		{"cleared then new value (1->0->2)", 1, []byte{0, 2}, 2*push + access(2) + write, 0, 0},
		// Three stores, round trips (note the state-gas refill on the 0-> path).
		{"0->1->0->1", 0, []byte{1, 0, 1}, 3*push + access(3) + 2*write, int64(set), write},
		{"1->0->1->0", 1, []byte{0, 1, 0}, 3*push + access(3) + 2*write, 0, clear + write},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var code []byte
			for _, v := range tc.vals {
				code = append(code, sstore(0, v)...)
			}
			var setup func(*state.StateDB, common.Address)
			if tc.orig != 0 {
				setup = setSlot(0, tc.orig)
			}
			res, refund, err := run8038(t, code, hugeBudget(), new(uint256.Int), setup)
			if err != nil {
				t.Fatal(err)
			}
			if res.UsedRegularGas != tc.wantReg {
				t.Errorf("regular gas = %d, want %d", res.UsedRegularGas, tc.wantReg)
			}
			if res.UsedStateGas != tc.wantState {
				t.Errorf("state gas = %d, want %d", res.UsedStateGas, tc.wantState)
			}
			if refund != tc.wantRfnd {
				t.Errorf("refund = %d, want %d", refund, tc.wantRfnd)
			}
		})
	}
}

// TestEIP8038SLoad checks the re-priced SLOAD access costs (cold 3000, warm 100).
func TestEIP8038SLoad(t *testing.T) {
	push := uint64(3) // PUSH1 slot
	// PUSH1 0x00; SLOAD
	cold := []byte{0x60, 0x00, 0x54}
	res, _, err := run8038(t, cold, hugeBudget(), new(uint256.Int), nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := push + params.ColdStorageAccessAmsterdam; res.UsedRegularGas != want {
		t.Fatalf("cold SLOAD = %d, want %d", res.UsedRegularGas, want)
	}
	// PUSH1 0x00; SLOAD; PUSH1 0x00; SLOAD  -> second access is warm.
	warm := []byte{0x60, 0x00, 0x54, 0x60, 0x00, 0x54}
	res, _, err = run8038(t, warm, hugeBudget(), new(uint256.Int), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := 2*push + params.ColdStorageAccessAmsterdam + params.WarmStorageReadCostEIP2929
	if res.UsedRegularGas != want {
		t.Fatalf("cold+warm SLOAD = %d, want %d", res.UsedRegularGas, want)
	}
}

// TestEIP8038AccountAccess checks the re-priced cold-account access for the
// account-reading opcodes and the extra WARM_ACCESS surcharge for EXTCODESIZE
// and EXTCODECOPY (their second database read).
func TestEIP8038AccountAccess(t *testing.T) {
	push20 := uint64(3)
	addr := common.BytesToAddress([]byte("some-cold-account"))

	// pushAddr emits PUSH20 <addr>.
	pushAddr := func() []byte { return append([]byte{0x73}, addr.Bytes()...) }

	cold := params.ColdAccountAccessAmsterdam
	warm := params.WarmStorageReadCostEIP2929

	t.Run("BALANCE", func(t *testing.T) {
		code := append(pushAddr(), 0x31) // BALANCE
		res, _, err := run8038(t, code, hugeBudget(), new(uint256.Int), nil)
		if err != nil {
			t.Fatal(err)
		}
		if want := push20 + cold; res.UsedRegularGas != want {
			t.Fatalf("cold BALANCE = %d, want %d", res.UsedRegularGas, want)
		}
	})
	t.Run("EXTCODEHASH", func(t *testing.T) {
		code := append(pushAddr(), 0x3f) // EXTCODEHASH
		res, _, err := run8038(t, code, hugeBudget(), new(uint256.Int), nil)
		if err != nil {
			t.Fatal(err)
		}
		if want := push20 + cold; res.UsedRegularGas != want {
			t.Fatalf("cold EXTCODEHASH = %d, want %d", res.UsedRegularGas, want)
		}
	})
	t.Run("EXTCODESIZE adds WARM_ACCESS", func(t *testing.T) {
		code := append(pushAddr(), 0x3b) // EXTCODESIZE
		res, _, err := run8038(t, code, hugeBudget(), new(uint256.Int), nil)
		if err != nil {
			t.Fatal(err)
		}
		if want := push20 + cold + warm; res.UsedRegularGas != want {
			t.Fatalf("cold EXTCODESIZE = %d, want %d", res.UsedRegularGas, want)
		}
	})
	t.Run("EXTCODECOPY adds WARM_ACCESS", func(t *testing.T) {
		// PUSH1 0 (length); PUSH1 0 (codeOffset); PUSH1 0 (destOffset); PUSH20 addr; EXTCODECOPY
		code := []byte{0x60, 0x00, 0x60, 0x00, 0x60, 0x00}
		code = append(code, pushAddr()...)
		code = append(code, 0x3c) // EXTCODECOPY
		res, _, err := run8038(t, code, hugeBudget(), new(uint256.Int), nil)
		if err != nil {
			t.Fatal(err)
		}
		// three PUSH1 + one PUSH20 = 12 gas, zero-length copy => no memory/copy gas.
		if want := uint64(12) + cold + warm; res.UsedRegularGas != want {
			t.Fatalf("cold EXTCODECOPY = %d, want %d", res.UsedRegularGas, want)
		}
	})
}

// callFamily8038 builds a zero-input/output call-family operation that forwards
// all remaining regular gas and discards its success flag. CALL and CALLCODE
// take a value argument; DELEGATECALL and STATICCALL do not.
func callFamily8038(to common.Address, op OpCode, value byte) []byte {
	code := []byte{0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00}
	if op == CALL || op == CALLCODE {
		code = append(code, 0x60, value)
	}
	code = append(code, 0x73)
	code = append(code, to.Bytes()...)
	return append(code, 0x5a, byte(op), 0x50, 0x00) // GAS; <op>; POP; STOP
}

// TestEIP8038Calls pins the re-priced account access and value-transfer costs
// for every member of the CALL family. The opcode's constant cost is the warm
// access component, so a cold target adds only COLD_ACCOUNT_ACCESS-WARM.
func TestEIP8038Calls(t *testing.T) {
	const (
		push1  = uint64(3)
		push20 = uint64(3)
		gasOp  = uint64(2)
		pop    = uint64(2)
	)
	cold := params.ColdAccountAccessAmsterdam - params.WarmAccountAccessAmsterdam
	callBase := 5*push1 + push20 + gasOp + pop + params.WarmAccountAccessAmsterdam
	plainBase := 4*push1 + push20 + gasOp + pop + params.WarmAccountAccessAmsterdam
	target := common.BytesToAddress([]byte("call-target"))

	cases := []struct {
		name      string
		op        OpCode
		value     byte
		fundSelf  bool
		wantReg   uint64
		wantState int64
	}{
		{"call/cold", CALL, 0, false, callBase + cold, 0},
		// A callee that immediately returns gives the 2,300 stipend back, so
		// the net regular cost is ACCOUNT_WRITE.
		{"call/value", CALL, 1, true, callBase + cold + params.AccountWriteAmsterdam, stateGasNewAccount},
		{"callcode/value", CALLCODE, 1, true, callBase + cold + params.AccountWriteAmsterdam, 0},
		{"delegatecall/cold", DELEGATECALL, 0, false, plainBase + cold, 0},
		{"staticcall/cold", STATICCALL, 0, false, plainBase + cold, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var setup func(*state.StateDB, common.Address)
			if tc.fundSelf {
				setup = fund(common.BytesToAddress([]byte("self")), 1)
			}
			res, _, err := run8038(t, callFamily8038(target, tc.op, tc.value), hugeBudget(), new(uint256.Int), setup)
			if err != nil {
				t.Fatal(err)
			}
			if res.UsedRegularGas != tc.wantReg {
				t.Fatalf("regular gas = %d, want %d", res.UsedRegularGas, tc.wantReg)
			}
			if res.UsedStateGas != tc.wantState {
				t.Fatalf("state gas = %d, want %d", res.UsedStateGas, tc.wantState)
			}
		})
	}

	// The first CALL makes its target warm. A second CALL pays no dynamic
	// access surcharge, leaving only the CALL base cost (which includes warm).
	first := callFamily8038(target, CALL, 0)
	code := append(first[:len(first)-1], callFamily8038(target, CALL, 0)...)
	res, _, err := run8038(t, code, hugeBudget(), new(uint256.Int), nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := 2*callBase + cold; res.UsedRegularGas != want {
		t.Fatalf("cold+warm CALL = %d, want %d", res.UsedRegularGas, want)
	}

	// Calling an EIP-7702 authority accesses both the authority and its
	// delegation target. The authority uses CALL's warm-included pricing; the
	// separately resolved cold target pays a full COLD_ACCOUNT_ACCESS charge.
	authority := common.BytesToAddress([]byte("delegated-authority"))
	implementation := common.BytesToAddress([]byte("delegated-target"))
	setup := func(db *state.StateDB, _ common.Address) {
		db.CreateAccount(authority)
		db.SetCode(authority, types.AddressToDelegation(implementation), tracing.CodeChangeUnspecified)
		db.CreateAccount(implementation)
		db.SetCode(implementation, []byte{0x00}, tracing.CodeChangeUnspecified)
	}
	res, _, err = run8038(t, callFamily8038(authority, CALL, 0), hugeBudget(), new(uint256.Int), setup)
	if err != nil {
		t.Fatal(err)
	}
	if want := callBase + cold + params.ColdAccountAccessAmsterdam; res.UsedRegularGas != want {
		t.Fatalf("delegated CALL = %d, want %d (authority + target)", res.UsedRegularGas, want)
	}

	// A value CALL receives the 2,300 stipend even when it asks to forward no
	// regular gas. If the child burns that stipend, the full CALL_VALUE
	// (ACCOUNT_WRITE + stipend) remains charged to the caller.
	stipendTarget := common.BytesToAddress([]byte("stipend-target"))
	base := callFamily8038(stipendTarget, CALL, 1)
	code = append(base[:len(base)-4], 0x60, 0x00, byte(CALL), 0x50, 0x00) // PUSH1 0; CALL; POP; STOP
	setup = func(db *state.StateDB, self common.Address) {
		db.AddBalance(self, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
		db.CreateAccount(stipendTarget)
		db.SetCode(stipendTarget, []byte{0xfe}, tracing.CodeChangeUnspecified)
	}
	res, _, err = run8038(t, code, hugeBudget(), new(uint256.Int), setup)
	if err != nil {
		t.Fatal(err)
	}
	if want := 5*push1 + push20 + push1 + pop + params.WarmAccountAccessAmsterdam + cold + params.CallValueTransferAmsterdam; res.UsedRegularGas != want {
		t.Fatalf("value CALL with burnt stipend = %d, want %d", res.UsedRegularGas, want)
	}
}

// TestEIP8038Create checks that CREATE and CREATE2 always pay CREATE_ACCESS
// in regular gas. With otherwise identical initcode, CREATE2 additionally has
// one salt push and the address-hash word charge.
func TestEIP8038Create(t *testing.T) {
	create, _, err := run8038(t, deployCode(deploy0Init, false, 0), hugeBudget(), new(uint256.Int), nil)
	if err != nil {
		t.Fatal(err)
	}
	create2, _, err := run8038(t, deployCode(deploy0Init, true, 0), hugeBudget(), new(uint256.Int), nil)
	if err != nil {
		t.Fatal(err)
	}
	// Outer setup is PUSH32 + PUSH1 + MSTORE (including one-word expansion),
	// then three CREATE operands. The child initcode is two PUSH1s and RETURN.
	const outer = uint64(3 + 3 + 3 + 3 + 3*3)
	const init = uint64(2 * 3)
	want := outer + params.CreateAccessAmsterdam + params.InitCodeWordGas + init
	if create.UsedRegularGas != want {
		t.Fatalf("CREATE regular gas = %d, want %d", create.UsedRegularGas, want)
	}
	if want := create.UsedRegularGas + 3 + params.Keccak256WordGas; create2.UsedRegularGas != want {
		t.Fatalf("CREATE2 regular gas = %d, want %d", create2.UsedRegularGas, want)
	}
}

// TestEIP8038SelfdestructAccountWrite checks that SELFDESTRUCT sending a positive
// balance to an empty account is charged the cold access, an additional
// ACCOUNT_WRITE (regular) and GAS_NEW_ACCOUNT (state).
func TestEIP8038SelfdestructAccountWrite(t *testing.T) {
	beneficiary := common.BytesToAddress([]byte("fresh-beneficiary"))
	// PUSH20 beneficiary; SELFDESTRUCT
	code := append([]byte{0x73}, beneficiary.Bytes()...)
	code = append(code, 0xff)

	// Fund the contract so it sends a positive balance on self-destruct.
	fundSelf := func(db *state.StateDB, self common.Address) {
		db.AddBalance(self, uint256.NewInt(1), tracing.BalanceChangeUnspecified)
	}
	res, _, err := run8038(t, code, hugeBudget(), new(uint256.Int), fundSelf)
	if err != nil {
		t.Fatal(err)
	}
	const push20 = uint64(3)
	wantReg := push20 + params.SelfdestructGasEIP150 + params.ColdAccountAccessAmsterdam + params.AccountWriteAmsterdam
	if res.UsedRegularGas != wantReg {
		t.Fatalf("regular gas = %d, want %d", res.UsedRegularGas, wantReg)
	}
	if want := int64(params.AccountCreationSize * params.CostPerStateByte); res.UsedStateGas != want {
		t.Fatalf("state gas = %d, want %d", res.UsedStateGas, want)
	}
}

// TestEIP8038SStoreAccessGuard covers the affordability check that bails out
// before the slot is read once the gas left cannot cover the slot's access cost.
// The two PUSH1s cost 6, so a 2506 budget leaves 2500 at the SSTORE: above the
// reentrancy sentry (2300) yet below COLD_STORAGE_ACCESS (3000). The guard must
// fire, distinguishable from the sentry/charge OOG by its "slot access" message.
func TestEIP8038SStoreAccessGuard(t *testing.T) {
	budget := NewGasBudget(6+params.SstoreSentryGasEIP2200+200, 0)
	_, _, err := run8038(t, sstore(0, 1), budget, new(uint256.Int), nil)
	if err == nil {
		t.Fatal("expected failure: gas left cannot cover cold-slot access")
	}
	if !strings.Contains(err.Error(), "not enough gas for slot access") {
		t.Fatalf("got %q, want the slot-access guard error", err)
	}
}
