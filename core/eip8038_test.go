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

// EIP-8038 authorization accounting tests. The per-authorization intrinsic gas
// pre-charges ACCOUNT_WRITE (regular) on top of REGULAR_PER_AUTH_BASE_COST.
// applyAuthorization refunds that ACCOUNT_WRITE to the refund counter in exactly
// the cases where no new account leaf is written: an invalid authorization, or
// an authority whose account already exists. These white-box tests invoke
// applyAuthorization directly and read the raw refund counter, so they observe
// the refund before the EIP-3529 cap is applied.

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

// newAuthTestTransition builds a minimal stateTransition with a state reservoir,
// suitable for calling applyAuthorization directly.
func newAuthTestTransition(sdb *state.StateDB) *stateTransition {
	st := newStateTransition(amsterdamCoreEVM(sdb), &Message{}, NewGasPool(30_000_000))
	st.gasRemaining = vm.NewGasBudget(0, 1_000_000) // reservoir for state-gas refills
	return st
}

// A net-new delegation on a fresh authority writes a new account leaf, so the
// intrinsic ACCOUNT_WRITE stands (no refund).
func TestAuthAccountWriteNetNewNoRefund(t *testing.T) {
	auth, _ := signAuth(t, authKeyA, delegate8037, 0)
	st := newAuthTestTransition(mkState(senderAlloc(nil)))
	if err := st.applyAuthorization(rules8037, &auth, map[common.Address]bool{}); err != nil {
		t.Fatal(err)
	}
	if got := st.state.GetRefund(); got != 0 {
		t.Fatalf("refund = %d, want 0 (net-new account write)", got)
	}
}

// A pre-existing authority writes no new account leaf, so the intrinsic
// ACCOUNT_WRITE is refunded.
func TestAuthAccountWriteExistsRefund(t *testing.T) {
	auth, authority := signAuth(t, authKeyA, delegate8037, 0)
	st := newAuthTestTransition(mkState(senderAlloc(types.GenesisAlloc{authority: {Balance: big.NewInt(1)}})))
	if err := st.applyAuthorization(rules8037, &auth, map[common.Address]bool{}); err != nil {
		t.Fatal(err)
	}
	if got := st.state.GetRefund(); got != params.AccountWriteAmsterdam {
		t.Fatalf("refund = %d, want %d (account already exists)", got, params.AccountWriteAmsterdam)
	}
}

// An invalid authorization is skipped without writing any account leaf, so its
// intrinsic ACCOUNT_WRITE is refunded.
func TestAuthAccountWriteInvalidRefund(t *testing.T) {
	k, _ := crypto.HexToECDSA(authKeyA)
	bad, _ := types.SignSetCode(k, types.SetCodeAuthorization{
		ChainID: *uint256.NewInt(999), Address: delegate8037, Nonce: 0, // wrong chain id
	})
	st := newAuthTestTransition(mkState(senderAlloc(nil)))
	if err := st.applyAuthorization(rules8037, &bad, map[common.Address]bool{}); err == nil {
		t.Fatal("expected invalid-authorization error")
	}
	if got := st.state.GetRefund(); got != params.AccountWriteAmsterdam {
		t.Fatalf("refund = %d, want %d (invalid authorization)", got, params.AccountWriteAmsterdam)
	}
}

// The same authority across two authorizations writes its account leaf only
// once: the first auth pays ACCOUNT_WRITE, the second (which now sees the
// account as existing) is refunded.
func TestAuthAccountWriteDuplicateOnce(t *testing.T) {
	a0, _ := signAuth(t, authKeyA, delegate8037, 0)
	a1, _ := signAuth(t, authKeyA, delegate8037, 1)
	st := newAuthTestTransition(mkState(senderAlloc(nil)))
	delegates := map[common.Address]bool{}
	if err := st.applyAuthorization(rules8037, &a0, delegates); err != nil {
		t.Fatal(err)
	}
	if got := st.state.GetRefund(); got != 0 {
		t.Fatalf("refund after first auth = %d, want 0", got)
	}
	if err := st.applyAuthorization(rules8037, &a1, delegates); err != nil {
		t.Fatal(err)
	}
	if got := st.state.GetRefund(); got != params.AccountWriteAmsterdam {
		t.Fatalf("refund after duplicate auth = %d, want %d", got, params.AccountWriteAmsterdam)
	}
}
