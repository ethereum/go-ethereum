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

package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// newAuthTestTransition builds a minimal stateTransition with a runtime gas
// budget, suitable for calling applyAuthorization directly.
func newAuthTestTransition(sdb *state.StateDB) *stateTransition {
	st := newStateTransition(amsterdamCoreEVM(sdb), &Message{}, NewGasPool(30_000_000))
	st.gasRemaining = vm.NewGasBudget(1_000_000, 1_000_000)
	return st
}

// A net-new delegation on a fresh, cold authority is charged ACCOUNT_WRITE in
// regular gas (the authority's cold access is paid unconditionally at the
// intrinsic phase, not here), plus the account leaf and the indicator bytes in
// state gas.
func TestAuthRuntimeChargeNetNew(t *testing.T) {
	auth, _ := signAuth(t, authKeyA, delegate8037, 0)
	st := newAuthTestTransition(mkState(senderAlloc(nil)))
	if err := st.applyAuthorization(rules8037, &auth, map[common.Address]*authDelegationState{}); err != nil {
		t.Fatal(err)
	}
	if want := params.AccountWriteAmsterdam; st.gasRemaining.UsedRegularGas != want {
		t.Fatalf("regular charged = %d, want %d", st.gasRemaining.UsedRegularGas, want)
	}
	if want := int64(authWorstState); st.gasRemaining.UsedStateGas != want {
		t.Fatalf("state charged = %d, want %d", st.gasRemaining.UsedStateGas, want)
	}
}

// A pre-existing authority writes no new account leaf, but its first write in
// the transaction still carries ACCOUNT_WRITE; the authority's cold access is
// paid at the intrinsic phase, so only the net-new indicator bytes are charged
// as state gas here.
func TestAuthRuntimeChargeExistingAccount(t *testing.T) {
	auth, authority := signAuth(t, authKeyA, delegate8037, 0)
	st := newAuthTestTransition(mkState(senderAlloc(types.GenesisAlloc{authority: {Balance: big.NewInt(1)}})))
	if err := st.applyAuthorization(rules8037, &auth, map[common.Address]*authDelegationState{}); err != nil {
		t.Fatal(err)
	}
	if want := params.AccountWriteAmsterdam; st.gasRemaining.UsedRegularGas != want {
		t.Fatalf("regular charged = %d, want %d", st.gasRemaining.UsedRegularGas, want)
	}
	if want := int64(authBaseState); st.gasRemaining.UsedStateGas != want {
		t.Fatalf("state charged = %d, want %d", st.gasRemaining.UsedStateGas, want)
	}
}

// No cold surcharge is ever charged at runtime — the authority access is priced
// at the intrinsic phase — so an authority already warmed by the access list or
// an earlier authorization pays only the first-write surcharge, as it would
// whether warm or cold.
func TestAuthRuntimeChargeWarmAuthority(t *testing.T) {
	auth, authority := signAuth(t, authKeyA, delegate8037, 0)
	st := newAuthTestTransition(mkState(senderAlloc(types.GenesisAlloc{authority: {Balance: big.NewInt(1)}})))
	st.state.AddAddressToAccessList(authority)
	if err := st.applyAuthorization(rules8037, &auth, map[common.Address]*authDelegationState{}); err != nil {
		t.Fatal(err)
	}
	if want := params.AccountWriteAmsterdam; st.gasRemaining.UsedRegularGas != want {
		t.Fatalf("regular charged = %d, want %d (warm authority)", st.gasRemaining.UsedRegularGas, want)
	}
	if want := int64(authBaseState); st.gasRemaining.UsedStateGas != want {
		t.Fatalf("state charged = %d, want %d", st.gasRemaining.UsedStateGas, want)
	}
}

// An invalid authorization is skipped without any runtime charge.
func TestAuthRuntimeInvalidNoCharge(t *testing.T) {
	k, _ := crypto.HexToECDSA(authKeyA)
	bad, _ := types.SignSetCode(k, types.SetCodeAuthorization{
		ChainID: *uint256.NewInt(999), Address: delegate8037, Nonce: 0, // wrong chain id
	})
	st := newAuthTestTransition(mkState(senderAlloc(nil)))
	if err := st.applyAuthorization(rules8037, &bad, map[common.Address]*authDelegationState{}); err == nil {
		t.Fatal("expected invalid-authorization error")
	}
	if st.gasRemaining.UsedRegularGas != 0 || st.gasRemaining.UsedStateGas != 0 {
		t.Fatalf("charged = <%d,%d>, want <0,0> (invalid authorization)",
			st.gasRemaining.UsedRegularGas, st.gasRemaining.UsedStateGas)
	}
}

// The same authority across two authorizations is charged once: the first auth
// warms the authority, materializes the account and installs the indicator, so
// the second incurs no further charge.
func TestAuthRuntimeDuplicateAuthorityOnce(t *testing.T) {
	a0, _ := signAuth(t, authKeyA, delegate8037, 0)
	a1, _ := signAuth(t, authKeyA, delegate8037, 1)
	st := newAuthTestTransition(mkState(senderAlloc(nil)))
	delegates := map[common.Address]*authDelegationState{}
	if err := st.applyAuthorization(rules8037, &a0, delegates); err != nil {
		t.Fatal(err)
	}
	if err := st.applyAuthorization(rules8037, &a1, delegates); err != nil {
		t.Fatal(err)
	}
	if want := params.AccountWriteAmsterdam; st.gasRemaining.UsedRegularGas != want {
		t.Fatalf("regular charged = %d, want %d (once)", st.gasRemaining.UsedRegularGas, want)
	}
	if want := int64(authWorstState); st.gasRemaining.UsedStateGas != want {
		t.Fatalf("state charged = %d, want %d (once)", st.gasRemaining.UsedStateGas, want)
	}
}

// A budget that cannot cover the runtime charge aborts authorization
// processing with ErrOutOfGasRuntime, without mutating the authority.
func TestAuthRuntimeOutOfGas(t *testing.T) {
	auth, authority := signAuth(t, authKeyA, delegate8037, 0)
	st := newAuthTestTransition(mkState(senderAlloc(nil)))
	st.gasRemaining = vm.NewGasBudget(10_000, 0) // covers neither leaf nor indicator
	if err := st.applyAuthorization(rules8037, &auth, map[common.Address]*authDelegationState{}); err != ErrOutOfGasRuntime {
		t.Fatalf("err = %v, want ErrOutOfGasRuntime", err)
	}
	if st.state.GetNonce(authority) != 0 || len(st.state.GetCode(authority)) != 0 {
		t.Fatal("authority mutated despite out-of-gas runtime charge")
	}
}
