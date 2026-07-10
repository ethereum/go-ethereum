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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// TestEIP2780Intrinsic checks the intrinsic-gas decomposition.
func TestEIP2780Intrinsic(t *testing.T) {
	var (
		from = common.HexToAddress("0x1111111111111111111111111111111111111111")
		to   = common.HexToAddress("0x2222222222222222222222222222222222222222")
	)
	cases := []struct {
		name  string
		to    *common.Address
		value *uint256.Int
		auths []types.SetCodeAuthorization
		want  uint64
	}{
		{
			name:  "self-transfer",
			to:    &from,
			value: uint256.NewInt(1),
			want:  params.TxBaseCost2780, // 12,000
		},
		{
			name:  "self-transfer/zero-value",
			to:    &from,
			value: uint256.NewInt(0),
			want:  params.TxBaseCost2780, // 12,000
		},
		{
			name:  "zero-value call",
			to:    &to,
			value: uint256.NewInt(0),
			// TxBaseCost + ColdAccountAccess = 15,000; the recipient touch is
			// charged at the cold rate unconditionally at the intrinsic phase.
			want: params.TxBaseCost2780 + params.ColdAccountAccessAmsterdam,
		},
		{
			name:  "value transfer to existing EOA",
			to:    &to,
			value: uint256.NewInt(1),
			// TxBaseCost + ColdAccountAccess + TxValueCost + TransferLogCost = 21,000
			want: params.TxBaseCost2780 + params.ColdAccountAccessAmsterdam +
				params.TxValueCost2780 + params.TransferLogCost2780,
		},
		{
			name:  "contract creation, value = 0",
			to:    nil,
			value: uint256.NewInt(0),
			// TxBaseCost + CreateAccess = 23,000 regular. The new-account state
			// charge depends on whether the deployment target exists and is
			// charged at runtime, not intrinsically.
			want: params.TxBaseCost2780 + params.CreateAccessAmsterdam,
		},
		{
			name:  "contract creation, value > 0",
			to:    nil,
			value: uint256.NewInt(1),
			// TxBaseCost + CreateAccess + TransferLogCost = 24,756 regular.
			want: params.TxBaseCost2780 + params.CreateAccessAmsterdam + params.TransferLogCost2780,
		},
		{
			name:  "value transfer with authorizations",
			to:    &to,
			value: uint256.NewInt(1),
			auths: make([]types.SetCodeAuthorization, 3),
			// Each authorization adds the state-independent per-auth base
			// (cold authority access included).
			want: params.TxBaseCost2780 + params.ColdAccountAccessAmsterdam +
				params.TxValueCost2780 + params.TransferLogCost2780 + 3*params.RegularPerAuthBaseCost,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := IntrinsicGas(nil, nil, tc.auths, from, tc.to, tc.value, rules8037)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("gas mismatch: got %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestEIP2780Gas checks every "Transaction reference case" in
// the EIP-2780 specification end-to-end, asserting the two-dimensional charge
// (intrinsic + top-level + execution) recorded in the block gas pool.
func TestEIP2780Gas(t *testing.T) {
	const (
		cold     = params.ColdAccountAccessAmsterdam
		base     = params.TxBaseCost2780
		valueCst = params.TxValueCost2780 + params.TransferLogCost2780
	)
	var (
		existingEOA  = common.HexToAddress("0xe0a0000000000000000000000000000000000001")
		stopContract = common.HexToAddress("0xc0de000000000000000000000000000000000001")
		delegated    = common.HexToAddress("0xde1e000000000000000000000000000000000001")
		emptyTarget  = common.HexToAddress("0x7a76000000000000000000000000000000000001") // never allocated
		freshEOA     = common.HexToAddress("0xbeef000000000000000000000000000000000001") // never allocated
	)
	// Shared world: a funded EOA, a STOP contract and an account delegated to a
	// non-existent (codeless) target. The delegation target is intentionally
	// absent so resolving it executes no code.
	base7702 := types.GenesisAlloc{
		existingEOA:  {Balance: big.NewInt(1)},
		stopContract: {Code: []byte{0x00}}, // STOP
		delegated:    {Code: types.AddressToDelegation(emptyTarget)},
	}
	// valueCreateTx builds a contract-creation transaction carrying value.
	valueCreateTx := func(value int64) *types.Transaction {
		return types.MustSignNewTx(senderKey, signer8037, &types.DynamicFeeTx{
			ChainID: cfg8037.ChainID, Nonce: 0, To: nil, Value: big.NewInt(value),
			Gas: 300_000, GasFeeCap: big.NewInt(0), GasTipCap: big.NewInt(0),
		})
	}

	cases := []struct {
		name                   string
		tx                     *types.Transaction
		wantRegular, wantState uint64
	}{
		// case 1: ETH transfer to self.
		{"self-transfer", callTx(0, senderAddr, 1, 100_000, nil), base, 0},
		// case 2: no-transfer to an existing EOA.
		{"zero-value/eoa", callTx(0, existingEOA, 0, 100_000, nil), base + cold, 0},
		// case 3: no-transfer to a contract.
		{"zero-value/contract", callTx(0, stopContract, 0, 100_000, nil), base + cold, 0},
		// case 4: ETH transfer to an existing EOA.
		{"value/eoa", callTx(0, existingEOA, 1, 100_000, nil), base + cold + valueCst, 0},
		// case 5: ETH transfer to a contract.
		{"value/contract", callTx(0, stopContract, 1, 100_000, nil), base + cold + valueCst, 0},
		// case 6: no-transfer to a 7702-delegated account.
		{"zero-value/delegated", callTx(0, delegated, 0, 100_000, nil), base + 2*cold, 0},
		// case 7: ETH transfer to a 7702-delegated account (no new-account charge).
		{"value/delegated", callTx(0, delegated, 1, 100_000, nil), base + 2*cold + valueCst, 0},
		// case 8: ETH transfer creating a new account.
		{"value/new-account", callTx(0, freshEOA, 1, 300_000, nil), base + cold + valueCst, newAccountState},
		// case 9: contract-creation transaction, value = 0.
		{"create/zero-value", createTx(0, 300_000, nil), base + params.CreateAccessAmsterdam, newAccountState},
		// case 10: contract-creation transaction, value > 0.
		{"create/value", valueCreateTx(1), base + params.CreateAccessAmsterdam + params.TransferLogCost2780, newAccountState},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, gp, err := applyMsg(t, mkState(senderAlloc(base7702)), tc.tx)
			if err != nil {
				t.Fatalf("consensus error: %v", err)
			}
			if res.Err != nil {
				t.Fatalf("execution failed: %v", res.Err)
			}
			if gp.cumulativeRegular != tc.wantRegular {
				t.Errorf("regular gas = %d, want %d", gp.cumulativeRegular, tc.wantRegular)
			}
			if gp.cumulativeState != tc.wantState {
				t.Errorf("state gas = %d, want %d", gp.cumulativeState, tc.wantState)
			}
		})
	}
}

// callTxAL builds a signed dynamic-fee call carrying an access list.
func callTxAL(nonce uint64, to common.Address, value int64, gas uint64, al types.AccessList) *types.Transaction {
	return types.MustSignNewTx(senderKey, signer8037, &types.DynamicFeeTx{
		ChainID: cfg8037.ChainID, Nonce: nonce, To: &to, Value: big.NewInt(value),
		Gas: gas, GasFeeCap: big.NewInt(0), GasTipCap: big.NewInt(0), AccessList: al,
	})
}

// accessListEntryCost is the total intrinsic cost of one address-only access
// list entry: the EIP-8038 per-address charge plus the EIP-7981 data charge.
const accessListEntryCost = params.TxAccessListAddressGasAmsterdam +
	common.AddressLength*params.TxCostFloorPerToken7976*params.TxTokenPerNonZeroByte

// TestEIP2780WarmRecipientStillChargedCold verifies that a recipient warmed by
// the transaction's access list is still charged the recipient at the cold rate.
func TestEIP2780WarmRecipientStillChargedCold(t *testing.T) {
	to := common.HexToAddress("0xe0a0000000000000000000000000000000000009")
	sdb := mkState(senderAlloc(types.GenesisAlloc{to: {Balance: big.NewInt(1)}}))
	al := types.AccessList{{Address: to}}
	res, gp, err := applyMsg(t, sdb, callTxAL(0, to, 0, 100_000, al))
	if err != nil {
		t.Fatal(err)
	}
	if res.Err != nil {
		t.Fatalf("execution failed: %v", res.Err)
	}
	want := params.TxBaseCost2780 + params.ColdAccountAccessAmsterdam + accessListEntryCost
	if gp.cumulativeRegular != want {
		t.Errorf("regular gas = %d, want %d (cold recipient, no access-list discount)", gp.cumulativeRegular, want)
	}
}

// TestEIP2780DelegatedWarmTarget verifies that resolving the recipient's
// delegation is charged at the warm rate when the target was warmed by the
// access list, rather than the flat cold rate.
func TestEIP2780DelegatedWarmTarget(t *testing.T) {
	var (
		target    = common.HexToAddress("0x7a76000000000000000000000000000000000002") // codeless
		delegated = common.HexToAddress("0xde1e000000000000000000000000000000000002")
	)
	sdb := mkState(senderAlloc(types.GenesisAlloc{
		delegated: {Code: types.AddressToDelegation(target)},
	}))
	al := types.AccessList{{Address: target}}
	res, gp, err := applyMsg(t, sdb, callTxAL(0, delegated, 0, 100_000, al))
	if err != nil {
		t.Fatal(err)
	}
	if res.Err != nil {
		t.Fatalf("execution failed: %v", res.Err)
	}
	want := params.TxBaseCost2780 + params.ColdAccountAccessAmsterdam + accessListEntryCost + // recipient cold access (intrinsic)
		params.WarmAccountAccessAmsterdam // warm delegation-target access (runtime)
	if gp.cumulativeRegular != want {
		t.Errorf("regular gas = %d, want %d (warm delegation target)", gp.cumulativeRegular, want)
	}
}

// TestEIP2780RuntimeOOGRevertsDelegations verifies that running out of gas on
// a runtime authorization charge halts the transaction and reverts all state
// changes, including the already applied EIP-7702 delegations — while the
// sender's nonce increment persists.
//
// The halt burns the regular dimension in full; the state dimension is
// refilled by the revert and the reservoir — if any — is preserved and
// returned to the sender rather than burnt.
func TestEIP2780RuntimeOOGRevertsDelegations(t *testing.T) {
	cases := []struct {
		name     string
		gas      uint64
		numAuths int
		wantUsed uint64 // = gas − reservoir: all regular burnt, reservoir returned
	}{
		// No state reservoir (gas below MaxTxGas). Gas covers the intrinsic
		// cost (TX_BASE_COST + the cold-inclusive per-authorization base for
		// a self-call) but not the runtime authorization charges
		// (ACCOUNT_WRITE + account + indicator bytes): everything is burnt.
		{"no-reservoir", 30_000, 1, 30_000},

		// A 100,000 state reservoir (gas above MaxTxGas). The 100
		// authorizations' state charges (~21.9M) overwhelm the reservoir and
		// the regular budget they spill into. The reservoir is made whole by
		// the halt-refill and returned to the sender.
		{"with-reservoir", params.MaxTxGas + 100_000, 100, params.MaxTxGas},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				auths       = make([]types.SetCodeAuthorization, tc.numAuths)
				authorities = make([]common.Address, tc.numAuths)
			)
			for i := range auths {
				key, _ := crypto.GenerateKey()
				auth, err := types.SignSetCode(key, types.SetCodeAuthorization{
					ChainID: *uint256.MustFromBig(cfg8037.ChainID), Address: delegate8037, Nonce: 0,
				})
				if err != nil {
					t.Fatalf("sign auth: %v", err)
				}
				auths[i], authorities[i] = auth, crypto.PubkeyToAddress(key.PublicKey)
			}
			sdb := mkState(senderAlloc(nil))
			tx := types.MustSignNewTx(senderKey, signer8037,
				&types.SetCodeTx{
					ChainID:   uint256.MustFromBig(cfg8037.ChainID),
					Nonce:     0,
					To:        senderAddr,
					Value:     new(uint256.Int),
					Gas:       tc.gas,
					GasFeeCap: new(uint256.Int),
					GasTipCap: new(uint256.Int),
					AuthList:  auths,
				})
			res, gp, err := applyMsg(t, sdb, tx)
			if err != nil {
				t.Fatalf("transaction should remain valid: %v", err)
			}
			if res.Err != vm.ErrOutOfGas {
				t.Fatalf("expected out of gas, got %v", res.Err)
			}
			if res.UsedGas != tc.wantUsed {
				t.Fatalf("used gas = %d, want %d", res.UsedGas, tc.wantUsed)
			}
			// The charged state gas was refilled on the halt: the receipt is
			// all regular, burnt in full, and only the reservoir survives.
			if gp.cumulativeState != 0 {
				t.Fatalf("state gas = %d, want 0 (refilled on halt)", gp.cumulativeState)
			}
			if gp.cumulativeRegular != tc.wantUsed {
				t.Fatalf("regular gas = %d, want %d (burnt in full)", gp.cumulativeRegular, tc.wantUsed)
			}
			for i, authority := range authorities {
				if code := sdb.GetCode(authority); len(code) != 0 {
					t.Fatalf("delegation %d persisted despite runtime OOG: %x", i, code)
				}
				if sdb.GetNonce(authority) != 0 {
					t.Fatalf("authority %d nonce persisted despite runtime OOG", i)
				}
			}
			if sdb.GetNonce(senderAddr) != 1 {
				t.Fatal("sender nonce not consumed")
			}
		})
	}
}

// TestEIP2780SelfTransferDelegated verifies that a self-transfer incurs no
// recipient touch or value charges, while resolving the sender's own
// delegation is still paid for.
func TestEIP2780SelfTransferDelegated(t *testing.T) {
	target := common.HexToAddress("0x7a76000000000000000000000000000000000003") // codeless
	sdb := mkState(types.GenesisAlloc{
		senderAddr: {Balance: big.NewInt(1e18), Code: types.AddressToDelegation(target)},
	})
	res, gp, err := applyMsg(t, sdb, callTx(0, senderAddr, 1, 100_000, nil))
	if err != nil {
		t.Fatal(err)
	}
	if res.Err != nil {
		t.Fatalf("execution failed: %v", res.Err)
	}
	want := params.TxBaseCost2780 + params.ColdAccountAccessAmsterdam // base + cold delegation target
	if gp.cumulativeRegular != want {
		t.Errorf("regular gas = %d, want %d (base + delegation resolution)", gp.cumulativeRegular, want)
	}
}

// TestEIP2780CreateInsufficientStateGas verifies that a contract-creation
// transaction funded for its intrinsic gas but not the runtime new-account
// state charge is included, halts out of gas and consumes the nonce.
func TestEIP2780CreateInsufficientStateGas(t *testing.T) {
	sdb := mkState(senderAlloc(nil))
	intrinsic := params.TxBaseCost2780 + params.CreateAccessAmsterdam // 23,000
	res, _, err := applyMsg(t, sdb, createTx(0, intrinsic, nil))
	if err != nil {
		t.Fatalf("transaction should remain valid: %v", err)
	}
	if res.Err != vm.ErrOutOfGas {
		t.Fatalf("expected out of gas, got %v", res.Err)
	}
	if res.UsedGas != intrinsic {
		t.Fatalf("used gas = %d, want %d", res.UsedGas, intrinsic)
	}
	if sdb.GetNonce(senderAddr) != 1 {
		t.Fatal("sender nonce not consumed")
	}
}

// TestEIP2780InsufficientGasForCallCharge verifies that a value transfer
// creating a new account, whose gas limit only covers the 21,000 intrinsic base
// and not the additional new-account state gas charged before the call executes,
// halts out of gas. The transaction stays valid (no consensus error) but
// execution fails and the recipient is not created.
func TestEIP2780InsufficientGasForCallCharge(t *testing.T) {
	fresh := common.HexToAddress("0xbeef000000000000000000000000000000000003")
	sdb := mkState(senderAlloc(nil))
	res, _, err := applyMsg(t, sdb, callTx(0, fresh, 1, 21_000, nil))
	if err != nil {
		t.Fatalf("transaction should remain valid: %v", err)
	}
	if res.Err != vm.ErrOutOfGas {
		t.Fatalf("expected out of gas, got %v", res.Err)
	}
	if res.UsedGas != 21_000 {
		t.Fatalf("expected used gas, got %v", res.UsedGas)
	}
	if sdb.Exist(fresh) {
		t.Fatal("recipient should not be created when the call charge cannot be paid")
	}
	if sdb.GetNonce(senderAddr) != 1 {
		t.Fatal("sender nonce not consumed")
	}
}

// TestEIP2780FirstFrameHaltPreservesPreExecution verifies the gas and state
// semantics when the top-most frame — message call or creation — halts
// exceptionally after the pre-execution phase completed:
//
//   - state changes applied before the frame was entered persist together
//     with their state-gas charge (the EIP-7702 delegations of a call tx);
//   - state gas pre-charged for the frame itself is refilled when the halt
//     voids it (the account-creation charge of a creation tx);
//   - after the refill the regular dimension is burnt in full, while any
//     remaining state reservoir is preserved and returned to the sender.
func TestEIP2780FirstFrameHaltPreservesPreExecution(t *testing.T) {
	halting := common.HexToAddress("0xbad0000000000000000000000000000000000002")
	cases := []struct {
		name        string
		create      bool
		gas         uint64
		wantUsed    uint64 // = gas − preserved reservoir
		wantRegular uint64
		wantState   uint64
	}{
		// Message call carrying one authorization: the delegation and its
		// state charge (account + indicator) survive the halt.
		//
		// Without a reservoir the charge spills from regular gas and everything is
		// burnt;
		//
		// With a reservoir, the reservoir remainder is preserved.
		{"call/no-reservoir", false, 1_000_000, 1_000_000, 1_000_000 - authWorstState, authWorstState},
		{"call/with-reservoir", false, params.MaxTxGas + 300_000, params.MaxTxGas + authWorstState, params.MaxTxGas, authWorstState},

		// Creation whose init code halts: no durable account is created, so
		// the pre-charged account creation is refilled and no state gas
		// remains.
		//
		// Without a reservoir the refill repays spilled regular gas, which the
		// halt then burns along with the rest;
		//
		// With a reservoir, the refill makes the reservoir whole again and it
		// is preserved.
		{"create/no-reservoir", true, 1_000_000, 1_000_000, 1_000_000, 0},
		{"create/with-reservoir", true, params.MaxTxGas + 100_000, params.MaxTxGas, params.MaxTxGas, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sdb := mkState(senderAlloc(types.GenesisAlloc{
				halting: {Code: []byte{0xfe}}, // INVALID
			}))
			var (
				tx        *types.Transaction
				authority common.Address
			)
			if tc.create {
				tx = createTx(0, tc.gas, []byte{0xfe}) // init code: INVALID
			} else {
				var auth types.SetCodeAuthorization
				auth, authority = signAuth(t, authKeyA, delegate8037, 0)
				tx = types.MustSignNewTx(senderKey, signer8037,
					&types.SetCodeTx{
						ChainID:   uint256.MustFromBig(cfg8037.ChainID),
						Nonce:     0,
						To:        halting,
						Value:     new(uint256.Int),
						Gas:       tc.gas,
						GasFeeCap: new(uint256.Int),
						GasTipCap: new(uint256.Int),
						AuthList:  []types.SetCodeAuthorization{auth},
					})
			}
			res, gp, err := applyMsg(t, sdb, tx)
			if err != nil {
				t.Fatalf("transaction should remain valid: %v", err)
			}
			if res.Err == nil {
				t.Fatal("expected the frame to halt")
			}
			if res.UsedGas != tc.wantUsed {
				t.Fatalf("used gas = %d, want %d", res.UsedGas, tc.wantUsed)
			}
			if gp.cumulativeRegular != tc.wantRegular {
				t.Fatalf("regular gas = %d, want %d (burnt in full)", gp.cumulativeRegular, tc.wantRegular)
			}
			if gp.cumulativeState != tc.wantState {
				t.Fatalf("state gas = %d, want %d", gp.cumulativeState, tc.wantState)
			}
			if tc.create {
				// The halted creation is fully reverted: no durable account.
				derived := crypto.CreateAddress(senderAddr, 0)
				if code := sdb.GetCode(derived); len(code) != 0 {
					t.Fatalf("created code persisted despite halt: %x", code)
				}
				if sdb.GetNonce(derived) != 0 {
					t.Fatal("created account nonce persisted despite halt")
				}
			} else {
				// The delegation applied before the frame was entered persists.
				if code := sdb.GetCode(authority); len(code) == 0 {
					t.Fatal("delegation should persist through an in-frame halt")
				}
			}
			if sdb.GetNonce(senderAddr) != 1 {
				t.Fatal("sender nonce not consumed")
			}
		})
	}
}

// TestEIP2780CreatePreExecutionOOGPreservesReservoir verifies that when a
// creation transaction cannot afford the pre-execution account-creation state
// charge (before the init-code frame is entered), the transaction halts with
// all regular gas burnt while the state reservoir — never touched, since the
// charge is atomic and was not applied — is preserved and returned to the
// sender.
func TestEIP2780CreatePreExecutionOOGPreservesReservoir(t *testing.T) {
	// Regular gas left for the pre-execution charge; together with the
	// reservoir it must not cover the account-creation cost.
	const (
		regularLeft = 100_000
		reservoir   = 50_000
	)
	// Plain creation intrinsic: TX_BASE_COST + CREATE_ACCESS.
	plainIntrinsic, err := IntrinsicGas(nil, nil, nil, senderAddr, nil, new(uint256.Int), rules8037)
	if err != nil {
		t.Fatal(err)
	}
	// For the reservoir case the gas limit must exceed MaxTxGas, which leaves
	// a huge regular budget by default. A big access list drives the intrinsic
	// cost close to MaxTxGas, shrinking the regular budget back down to
	// roughly regularLeft. Storage keys work because their intrinsic charge
	// exceeds their EIP-7623/7976 floor contribution.
	al := types.AccessList{{Address: common.HexToAddress("0xa1")}}
	baseIntrinsic, err := IntrinsicGas(nil, al, nil, senderAddr, nil, new(uint256.Int), rules8037)
	if err != nil {
		t.Fatal(err)
	}
	perKey := params.TxAccessListStorageKeyGasAmsterdam + uint64(common.HashLength)*params.TxCostFloorPerToken7976*params.TxTokenPerNonZeroByte

	// Fill the transaction with accessList, drain the gas and make it
	// insufficient for account-creation cost.
	al[0].StorageKeys = make([]common.Hash, (params.MaxTxGas-regularLeft-baseIntrinsic)/perKey)
	alIntrinsic, err := IntrinsicGas(nil, al, nil, senderAddr, nil, new(uint256.Int), rules8037)
	if err != nil {
		t.Fatal(err)
	}
	if left := params.MaxTxGas - alIntrinsic; left+reservoir >= newAccountState {
		t.Fatalf("setup: regular %d + reservoir %d must not cover the creation charge %d", left, reservoir, newAccountState)
	}
	alCreateTx := types.MustSignNewTx(senderKey, signer8037,
		&types.DynamicFeeTx{
			ChainID:    cfg8037.ChainID,
			Nonce:      0,
			To:         nil,
			Value:      big.NewInt(0),
			Gas:        params.MaxTxGas + reservoir,
			GasFeeCap:  big.NewInt(0),
			GasTipCap:  big.NewInt(0),
			AccessList: al,
		})

	cases := []struct {
		name     string
		tx       *types.Transaction
		wantUsed uint64 // = gas − preserved reservoir
	}{
		// Gas below MaxTxGas: no reservoir, the whole limit is burnt.
		{"no-reservoir", createTx(0, plainIntrinsic+regularLeft, nil), plainIntrinsic + regularLeft},

		// Gas above MaxTxGas: the reservoir survives the halt untouched and
		// is returned to the sender.
		{"with-reservoir", alCreateTx, params.MaxTxGas},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sdb := mkState(senderAlloc(nil))
			res, gp, err := applyMsg(t, sdb, tc.tx)
			if err != nil {
				t.Fatalf("transaction should remain valid: %v", err)
			}
			if res.Err != vm.ErrOutOfGas {
				t.Fatalf("expected out of gas, got %v", res.Err)
			}
			if res.UsedGas != tc.wantUsed {
				t.Fatalf("used gas = %d, want %d", res.UsedGas, tc.wantUsed)
			}
			if gp.cumulativeRegular != tc.wantUsed {
				t.Fatalf("regular gas = %d, want %d (burnt in full)", gp.cumulativeRegular, tc.wantUsed)
			}
			if gp.cumulativeState != 0 {
				t.Fatalf("state gas = %d, want 0 (charge never applied)", gp.cumulativeState)
			}
			if derived := crypto.CreateAddress(senderAddr, 0); sdb.Exist(derived) {
				t.Fatal("target account should not be created when the charge cannot be paid")
			}
			if sdb.GetNonce(senderAddr) != 1 {
				t.Fatal("sender nonce not consumed")
			}
		})
	}
}

// TestEIP2780AuthorityAccountWrite pins the first-write ACCOUNT_WRITE rule for
// authorities: the surcharge applies to the first paid write to the account
// within the transaction, regardless of whether the account exists, and is
// skipped when the write is already paid for: by TX_BASE_COST for the sender,
// by TX_VALUE_COST for the recipient of a value-bearing transaction, or by a
// preceding valid authorization.
func TestEIP2780AuthorityAccountWrite(t *testing.T) {
	const (
		base     = params.TxBaseCost2780
		cold     = params.ColdAccountAccessAmsterdam
		aw       = params.AccountWriteAmsterdam
		perAuth  = params.RegularPerAuthBaseCost
		valueCst = params.TxValueCost2780 + params.TransferLogCost2780
	)
	existingEOA := common.HexToAddress("0xe0a0000000000000000000000000000000000002")

	auth0, authority := signAuth(t, authKeyA, delegate8037, 0)
	auth1, _ := signAuth(t, authKeyA, delegate8037, 1)
	authBadNonce, _ := signAuth(t, authKeyA, delegate8037, 5)

	// Self-sponsored authorization: the sender's nonce is bumped before the
	// authorization list is processed, hence nonce 1.
	senderAuth, err := types.SignSetCode(senderKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(cfg8037.ChainID), Address: delegate8037, Nonce: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	// tx builds a SetCode transaction with an explicit value.
	tx := func(to common.Address, value uint64, auths ...types.SetCodeAuthorization) *types.Transaction {
		return types.MustSignNewTx(senderKey, signer8037, &types.SetCodeTx{
			ChainID: uint256.MustFromBig(cfg8037.ChainID), Nonce: 0, To: to,
			Value: uint256.NewInt(value), Gas: 1_000_000,
			GasFeeCap: new(uint256.Int), GasTipCap: new(uint256.Int), AuthList: auths,
		})
	}
	fundedAuthority := types.GenesisAlloc{authority: {Balance: big.NewInt(1)}}

	cases := []struct {
		name                   string
		alloc                  types.GenesisAlloc
		tx                     *types.Transaction
		wantRegular, wantState uint64
	}{
		{
			// Materializing a fresh authority pays the first-write surcharge
			// alongside the new-account state gas and the indicator bytes.
			name:        "fresh authority",
			tx:          tx(existingEOA, 0, auth0),
			wantRegular: base + cold + perAuth + aw,
			wantState:   authWorstState,
		},
		{
			// An existing authority still pays the surcharge: the nonce and
			// indicator stores are the first write to the account within the
			// transaction.
			name:        "existing authority",
			alloc:       fundedAuthority,
			tx:          tx(existingEOA, 0, auth0),
			wantRegular: base + cold + perAuth + aw,
			wantState:   authBaseState,
		},
		{
			// Self-sponsored: the sender's account write is prepaid by
			// TX_BASE_COST, no surcharge.
			name:        "authority is sender",
			tx:          tx(existingEOA, 0, senderAuth),
			wantRegular: base + cold + perAuth,
			wantState:   authBaseState,
		},
		{
			// authority == tx.to with zero value: no TX_VALUE_COST was paid,
			// so the authorization write is the first paid write and the
			// surcharge applies. The recipient becomes delegated, adding a
			// cold delegation-target access at runtime.
			name:        "authority is recipient, zero value",
			alloc:       fundedAuthority,
			tx:          tx(authority, 0, auth0),
			wantRegular: base + cold + perAuth + aw + cold,
			wantState:   authBaseState,
		},
		{
			// authority == tx.to with value: TX_VALUE_COST prepaid the
			// recipient write, so no surcharge is due.
			name:        "authority is recipient, value",
			alloc:       fundedAuthority,
			tx:          tx(authority, 1, auth0),
			wantRegular: base + cold + valueCst + perAuth + cold,
			wantState:   authBaseState,
		},
		{
			// Fresh authority == tx.to with value: the authorization pays the
			// new-account state gas, and the recipient charge then sees an
			// existing account, so the leaf is not paid for twice.
			name:        "authority is fresh recipient, value",
			tx:          tx(authority, 1, auth0),
			wantRegular: base + cold + valueCst + perAuth + cold,
			wantState:   authWorstState,
		},
		{
			// The same authority twice: only the first valid authorization
			// carries the surcharge, the account creation and the indicator.
			name:        "same authority twice",
			tx:          tx(existingEOA, 0, auth0, auth1),
			wantRegular: base + cold + 2*perAuth + aw,
			wantState:   authWorstState,
		},
		{
			// An invalid authorization performs no write and does not count
			// as the first write; the following valid one pays in full. The
			// per-auth intrinsic base is still paid for the invalid tuple.
			name:        "invalid then valid",
			tx:          tx(existingEOA, 0, authBadNonce, auth0),
			wantRegular: base + cold + 2*perAuth + aw,
			wantState:   authWorstState,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			alloc := types.GenesisAlloc{existingEOA: {Balance: big.NewInt(1)}}
			for addr, acc := range tc.alloc {
				alloc[addr] = acc
			}
			res, gp, err := applyMsg(t, mkState(senderAlloc(alloc)), tc.tx)
			if err != nil {
				t.Fatalf("consensus error: %v", err)
			}
			if res.Err != nil {
				t.Fatalf("execution failed: %v", res.Err)
			}
			if gp.cumulativeRegular != tc.wantRegular {
				t.Errorf("regular gas = %d, want %d", gp.cumulativeRegular, tc.wantRegular)
			}
			if gp.cumulativeState != tc.wantState {
				t.Errorf("state gas = %d, want %d", gp.cumulativeState, tc.wantState)
			}
		})
	}
}

// TestEIP2780DelegationTargetPrewarmed pins the warm rate for delegation
// targets that are already in accessed_addresses when the recipient is
// loaded.
func TestEIP2780DelegationTargetPrewarmed(t *testing.T) {
	const (
		base    = params.TxBaseCost2780
		cold    = params.ColdAccountAccessAmsterdam
		warm    = params.WarmAccountAccessAmsterdam
		aw      = params.AccountWriteAmsterdam
		perAuth = params.RegularPerAuthBaseCost
	)
	delegatedAcct := common.HexToAddress("0xde1e000000000000000000000000000000000002")

	t.Run("target is sender", func(t *testing.T) {
		sdb := mkState(senderAlloc(types.GenesisAlloc{
			delegatedAcct: {Code: types.AddressToDelegation(senderAddr)},
		}))
		res, gp, err := applyMsg(t, sdb, callTx(0, delegatedAcct, 0, 100_000, nil))
		if err != nil {
			t.Fatalf("consensus error: %v", err)
		}
		if res.Err != nil {
			t.Fatalf("execution failed: %v", res.Err)
		}
		if want := base + cold + warm; gp.cumulativeRegular != want {
			t.Errorf("regular gas = %d, want %d (warm delegation target)", gp.cumulativeRegular, want)
		}
		if gp.cumulativeState != 0 {
			t.Errorf("state gas = %d, want 0", gp.cumulativeState)
		}
	})

	t.Run("target warmed by authorization", func(t *testing.T) {
		// A clearing authorization from a fresh authority: it creates the
		// authority account (nonce bump) and warms it, without installing an
		// indicator.
		//
		// The recipient's pre-existing delegation then resolves to
		// the freshly warmed, codeless authority at the warm rate.
		authClear, authority := signAuth(t, authKeyA, common.Address{}, 0)
		sdb := mkState(senderAlloc(types.GenesisAlloc{
			delegatedAcct: {Code: types.AddressToDelegation(authority)},
		}))
		res, gp, err := applyMsg(t, sdb, setCodeTx(0, delegatedAcct, []types.SetCodeAuthorization{authClear}))
		if err != nil {
			t.Fatalf("consensus error: %v", err)
		}
		if res.Err != nil {
			t.Fatalf("execution failed: %v", res.Err)
		}
		if want := base + cold + perAuth + aw + warm; gp.cumulativeRegular != want {
			t.Errorf("regular gas = %d, want %d (auth-warmed delegation target)", gp.cumulativeRegular, want)
		}
		if gp.cumulativeState != newAccountState {
			t.Errorf("state gas = %d, want %d (authority account created)", gp.cumulativeState, newAccountState)
		}
	})
}
