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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
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
	// Drive the stateTransition directly (as ApplyMessage does) so the test can
	// inspect the final tx-level GasBudget vector via st.gasRemaining.
	evm.SetTxContext(NewEVMTxContext(msg))
	st := newStateTransition(evm, msg, gp)
	res, err := st.execute()
	if err == nil && res != nil {
		assertPoolSane(t, res, gp)
		// The budget is seeded with the post-intrinsic remainder: the intrinsic
		// cost counts towards the MaxTxGas regular cap and the execution gas
		// exceeding the regular budget forms the reservoir.
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
//	pre-refund: cumulativeRegular + cumulativeState <= res.MaxUsedGas (peak)
//	bottleneck: Used() == max(cumulativeRegular, cumulativeState) <= initial
func assertPoolSane(t *testing.T, res *ExecutionResult, gp *GasPool) {
	t.Helper()
	if gp.cumulativeUsed != res.UsedGas {
		t.Fatalf("receipt scalar = %d, want UsedGas %d", gp.cumulativeUsed, res.UsedGas)
	}
	if res.UsedGas > res.MaxUsedGas {
		t.Fatalf("post-refund gas %d exceeds peak %d", res.UsedGas, res.MaxUsedGas)
	}
	if sum := gp.cumulativeRegular + gp.cumulativeState; sum > res.MaxUsedGas {
		t.Fatalf("regular+state %d exceeds peak %d", sum, res.MaxUsedGas)
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
	tx := types.MustSignNewTx(senderKey, signer8037, &types.DynamicFeeTx{
		ChainID: cfg8037.ChainID, Nonce: 0, To: &senderAddr, Value: big.NewInt(0),
		Gas: 25_000_000, GasFeeCap: big.NewInt(0), GasTipCap: big.NewInt(0), AccessList: al,
	})
	if _, _, err := applyMsg(t, mkState(senderAlloc(nil)), tx); err == nil {
		t.Fatal("expected rejection for intrinsic regular over MaxTxGas")
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

// 0->a->0 in one tx: the indicator created by an earlier auth and cleared by a
// later one writes zero net bytes; the earlier indicator charge is refilled.
func TestAuthClearSameTxDoubleRefill(t *testing.T) {
	set, authority := signAuth(t, authKeyA, delegate8037, 0)
	clr, _ := signAuth(t, authKeyA, common.Address{}, 1)
	sdb := mkState(senderAlloc(nil))
	_, gp, err := applyMsg(t, sdb, setCodeTx(0, senderAddr, []types.SetCodeAuthorization{set, clr}))
	if err != nil {
		t.Fatal(err)
	}
	_ = authority
	if want := newAccountState; gp.cumulativeState != want {
		t.Fatalf("state gas = %d, want %d (net-zero delegation)", gp.cumulativeState, want)
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
