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

// TestEIP2780NewAccountFunded verifies that a value transfer creating a new
// account both materializes and funds the recipient.
func TestEIP2780NewAccountFunded(t *testing.T) {
	fresh := common.HexToAddress("0xbeef000000000000000000000000000000000002")
	sdb := mkState(senderAlloc(nil))
	if _, _, err := applyMsg(t, sdb, callTx(0, fresh, 1, 300_000, nil)); err != nil {
		t.Fatal(err)
	}
	if !sdb.Exist(fresh) || sdb.GetBalance(fresh).Cmp(uint256.NewInt(1)) != 0 {
		t.Fatalf("recipient not funded: exist=%v balance=%v", sdb.Exist(fresh), sdb.GetBalance(fresh))
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
// the transaction's access list is still charged the recipient touch at the
// cold rate: per EIP-2780 that touch is priced unconditionally at the intrinsic
// phase, so an access-list entry does not discount it. The total is the
// intrinsic cold recipient charge plus the access-list entry itself, with no
// separate runtime charge.
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

// TestEIP2780RecipientColdInIntrinsic exercises the validity boundary created
// by charging the recipient touch at the cold rate unconditionally in the
// intrinsic phase: a zero-value call funded one gas below the cold-inclusive
// intrinsic (TX_BASE_COST + COLD_ACCOUNT_ACCESS) is rejected as intrinsic-gas
// too low, while funding exactly that amount is valid and included with no
// further runtime charge.
func TestEIP2780RecipientColdInIntrinsic(t *testing.T) {
	to := common.HexToAddress("0xe0a000000000000000000000000000000000000a")
	intrinsic := params.TxBaseCost2780 + params.ColdAccountAccessAmsterdam // 15,000

	// One gas short of the intrinsic cost: the transaction is invalid.
	sdb := mkState(senderAlloc(types.GenesisAlloc{to: {Balance: big.NewInt(1)}}))
	if _, _, err := applyMsg(t, sdb, callTx(0, to, 0, intrinsic-1, nil)); err == nil {
		t.Fatal("expected intrinsic-gas-too-low error, got nil")
	}

	// Funded for exactly the intrinsic cost: valid, included, and fully
	// consumed with no runtime cold surcharge to halt on.
	sdb = mkState(senderAlloc(types.GenesisAlloc{to: {Balance: big.NewInt(1)}}))
	res, _, err := applyMsg(t, sdb, callTx(0, to, 0, intrinsic, nil))
	if err != nil {
		t.Fatalf("transaction should be valid: %v", err)
	}
	if res.Err != nil {
		t.Fatalf("unexpected execution error: %v", res.Err)
	}
	if res.UsedGas != intrinsic {
		t.Fatalf("used gas = %d, want %d", res.UsedGas, intrinsic)
	}
	if sdb.GetNonce(senderAddr) != 1 {
		t.Fatal("sender nonce not consumed")
	}
}

// TestEIP2780RuntimeOOGRevertsDelegations verifies that running out of gas on
// a runtime authorization charge halts the (still valid) transaction and
// reverts all state changes, including the already applied EIP-7702
// delegations — while the sender's nonce increment persists.
func TestEIP2780RuntimeOOGRevertsDelegations(t *testing.T) {
	auth, authority := signAuth(t, authKeyA, delegate8037, 0)
	sdb := mkState(senderAlloc(nil))
	// Gas covers the intrinsic cost (TX_BASE_COST + the cold-inclusive
	// per-authorization base for a self-call) but not the runtime authorization
	// charges (ACCOUNT_WRITE + account + indicator bytes).
	tx := types.MustSignNewTx(senderKey, signer8037, &types.SetCodeTx{
		ChainID: uint256.MustFromBig(cfg8037.ChainID), Nonce: 0, To: senderAddr,
		Value: new(uint256.Int), Gas: 30_000, GasFeeCap: new(uint256.Int),
		GasTipCap: new(uint256.Int), AuthList: []types.SetCodeAuthorization{auth},
	})
	res, _, err := applyMsg(t, sdb, tx)
	if err != nil {
		t.Fatalf("transaction should remain valid: %v", err)
	}
	if res.Err != vm.ErrOutOfGas {
		t.Fatalf("expected out of gas, got %v", res.Err)
	}
	if res.UsedGas != 30_000 {
		t.Fatalf("used gas = %d, want all 30000 burnt", res.UsedGas)
	}
	if code := sdb.GetCode(authority); len(code) != 0 {
		t.Fatalf("delegation persisted despite runtime OOG: %x", code)
	}
	if sdb.GetNonce(authority) != 0 {
		t.Fatal("authority nonce persisted despite runtime OOG")
	}
	if sdb.GetNonce(senderAddr) != 1 {
		t.Fatal("sender nonce not consumed")
	}
}

// TestEIP2780SelfTransferDelegated verifies that a self-transfer incurs no
// recipient touch or value charges (the account is warm and existent as the
// sender), while resolving the sender's own delegation is still paid for.
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
}

// TestEIP2780HaltKeepsAuthStateGas verifies that when the top-most frame halts
// exceptionally, the EIP-7702 delegations applied before the frame was persist.
func TestEIP2780HaltKeepsAuthStateGas(t *testing.T) {
	halting := common.HexToAddress("0xbad0000000000000000000000000000000000001")
	sdb := mkState(senderAlloc(types.GenesisAlloc{
		halting: {Code: []byte{0xfe}}, // INVALID
	}))
	auth, authority := signAuth(t, authKeyA, delegate8037, 0)

	// A gas limit above MaxTxGas so the state reservoir covers the runtime
	// authorization state charges (218,790) without spilling into regular gas.
	gasLimit := params.MaxTxGas + 300_000
	tx := types.MustSignNewTx(senderKey, signer8037, &types.SetCodeTx{
		ChainID: uint256.MustFromBig(cfg8037.ChainID), Nonce: 0, To: halting,
		Value: new(uint256.Int), Gas: gasLimit, GasFeeCap: new(uint256.Int),
		GasTipCap: new(uint256.Int), AuthList: []types.SetCodeAuthorization{auth},
	})
	res, gp, err := applyMsg(t, sdb, tx)
	if err != nil {
		t.Fatalf("transaction should remain valid: %v", err)
	}
	if res.Err == nil {
		t.Fatal("expected the frame to halt")
	}
	if code := sdb.GetCode(authority); len(code) == 0 {
		t.Fatal("delegation should persist through an in-frame halt")
	}
	// The regular dimension is burned in full by the halt: the intrinsic cost
	// plus the entire regular execution budget, which together equal the
	// MaxTxGas cap. The state dimension keeps the delegation's durable
	// growth: a new account leaf plus the 23-byte indicator.
	if gp.cumulativeRegular != params.MaxTxGas {
		t.Errorf("regular gas = %d, want %d", gp.cumulativeRegular, params.MaxTxGas)
	}
	if gp.cumulativeState != authWorstState {
		t.Errorf("state gas = %d, want %d (delegation state growth persisted)", gp.cumulativeState, authWorstState)
	}
	if want := params.MaxTxGas + authWorstState; res.UsedGas != want {
		t.Errorf("used gas = %d, want %d", res.UsedGas, want)
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
