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
	"bytes"
	"crypto/ecdsa"
	"maps"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// EIP-7928 BAL inclusion tests.
//
// Each test exercises a single rule from the spec and asserts both presence
// and absence in the resulting block access list.

// balChainConfig returns a MergedTestChainConfig clone with Amsterdam active from genesis.
func balChainConfig() *params.ChainConfig {
	cfg := *params.MergedTestChainConfig
	cfg.AmsterdamTime = new(uint64)
	return &cfg
}

// balTestEnv bundles common identities used across the tests.
type balTestEnv struct {
	cfg    *params.ChainConfig
	signer types.Signer
	key    *ecdsa.PrivateKey
	from   common.Address
	gspec  *Genesis
}

// newBALTestEnv builds an Amsterdam chain config, funds a sender and pre-deploys
// the EIP-7928 system contracts. Extra accounts can be merged into Alloc.
func newBALTestEnv(extra types.GenesisAlloc) *balTestEnv {
	cfg := balChainConfig()
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	from := crypto.PubkeyToAddress(key.PublicKey)

	alloc := types.GenesisAlloc{
		from:                             {Balance: newGwei(1_000_000_000)},
		params.BeaconRootsAddress:        {Nonce: 1, Code: params.BeaconRootsCode, Balance: common.Big0},
		params.HistoryStorageAddress:     {Nonce: 1, Code: params.HistoryStorageCode, Balance: common.Big0},
		params.WithdrawalQueueAddress:    {Nonce: 1, Code: params.WithdrawalQueueCode, Balance: common.Big0},
		params.ConsolidationQueueAddress: {Nonce: 1, Code: params.ConsolidationQueueCode, Balance: common.Big0},
		params.BuilderDepositAddress:     {Nonce: 1, Code: params.BuilderDepositCode, Balance: common.Big0},
		params.BuilderExitAddress:        {Nonce: 1, Code: params.BuilderExitCode, Balance: common.Big0},
	}
	maps.Copy(alloc, extra)
	return &balTestEnv{
		cfg:    cfg,
		signer: types.LatestSigner(cfg),
		key:    key,
		from:   from,
		gspec:  &Genesis{Config: cfg, Alloc: alloc},
	}
}

// run generates exactly one Amsterdam block and returns its BAL.
func (e *balTestEnv) run(t *testing.T, gen func(*BlockGen)) (*bal.BlockAccessList, types.Receipts) {
	t.Helper()
	engine := beacon.New(ethash.NewFaker())
	_, blocks, receipts := GenerateChainWithGenesis(e.gspec, engine, 1, func(_ int, b *BlockGen) {
		gen(b)
	})
	if blocks[0].AccessList() == nil {
		t.Fatal("expected non-nil block access list")
	}
	return blocks[0].AccessList(), receipts[0]
}

// --- assertion helpers ---

func findAccount(b *bal.BlockAccessList, addr common.Address) *bal.AccountAccess {
	for i := range *b {
		if (*b)[i].Address == addr {
			return &(*b)[i]
		}
	}
	return nil
}

func hasSlotIn(slots []*uint256.Int, key common.Hash) bool {
	want := new(uint256.Int).SetBytes(key[:])
	for _, s := range slots {
		if s.Cmp(want) == 0 {
			return true
		}
	}
	return false
}

func hasStorageWrite(b *bal.BlockAccessList, addr common.Address, key common.Hash) bool {
	aa := findAccount(b, addr)
	if aa == nil {
		return false
	}
	want := new(uint256.Int).SetBytes(key[:])
	for _, w := range aa.StorageChanges {
		if w.Slot.Cmp(want) == 0 {
			return true
		}
	}
	return false
}

func assertPresent(t *testing.T, b *bal.BlockAccessList, addr common.Address) *bal.AccountAccess {
	t.Helper()
	aa := findAccount(b, addr)
	if aa == nil {
		t.Fatalf("address %x missing from BAL\n%s", addr, b.PrettyPrint())
	}
	return aa
}

func assertAbsent(t *testing.T, b *bal.BlockAccessList, addr common.Address) {
	t.Helper()
	if findAccount(b, addr) != nil {
		t.Fatalf("address %x must NOT be in BAL\n%s", addr, b.PrettyPrint())
	}
}

func assertEmpty(t *testing.T, aa *bal.AccountAccess) {
	t.Helper()
	if len(aa.StorageChanges) != 0 || len(aa.StorageReads) != 0 ||
		len(aa.BalanceChanges) != 0 || len(aa.NonceChanges) != 0 || len(aa.CodeChanges) != 0 {
		t.Fatalf("expected empty change set for %x, got %+v", aa.Address, aa)
	}
}

type balStorageChangeExpectation struct {
	index uint32
	value common.Hash
}

func assertStorageChanges(t *testing.T, aa *bal.AccountAccess, key common.Hash, want []balStorageChangeExpectation) {
	t.Helper()
	wantKey := new(uint256.Int).SetBytes(key[:])
	for _, slot := range aa.StorageChanges {
		if slot.Slot.Cmp(wantKey) != 0 {
			continue
		}
		if len(slot.SlotChanges) != len(want) {
			t.Fatalf("slot %x changes: have %+v, want %d entries", key, slot.SlotChanges, len(want))
		}
		for i, expected := range want {
			if slot.SlotChanges[i].BlockAccessIndex != expected.index {
				t.Fatalf("slot %x change %d index: have %d, want %d", key, i, slot.SlotChanges[i].BlockAccessIndex, expected.index)
			}
			wantValue := new(uint256.Int).SetBytes(expected.value[:])
			if slot.SlotChanges[i].PostValue.Cmp(wantValue) != 0 {
				t.Fatalf("slot %x change %d value: have %s, want %s", key, i, slot.SlotChanges[i].PostValue, wantValue)
			}
		}
		return
	}
	t.Fatalf("slot %x missing from storage_changes", key)
}

func assertStorageChangeAt(t *testing.T, aa *bal.AccountAccess, key common.Hash, index uint32) {
	t.Helper()
	wantKey := new(uint256.Int).SetBytes(key[:])
	for _, slot := range aa.StorageChanges {
		if slot.Slot.Cmp(wantKey) != 0 {
			continue
		}
		for _, change := range slot.SlotChanges {
			if change.BlockAccessIndex == index {
				return
			}
		}
		t.Fatalf("slot %x has no change at index %d: %+v", key, index, slot.SlotChanges)
	}
	t.Fatalf("slot %x missing from storage_changes", key)
}

// txGasNewAccount covers the base tx cost plus the EIP-8037 account-creation
// state-gas charge (STATE_BYTES_PER_NEW_ACCOUNT × CPSB ≈ 183,600) that is
// incurred when value is transferred to a non-existent account under Amsterdam.
// params.TxGas (21,000) alone is insufficient: the transfer would run out of
// gas, the credit would revert, and the recipient would never get a balance
// change recorded in the BAL.
const txGasNewAccount = 250_000

// --- tx builders ---

func (e *balTestEnv) tx(nonce uint64, to *common.Address, value *big.Int, gas uint64, tipGwei int64, data []byte) *types.Transaction {
	return types.MustSignNewTx(e.key, e.signer, &types.DynamicFeeTx{
		ChainID:   e.cfg.ChainID,
		Nonce:     nonce,
		To:        to,
		Value:     value,
		Gas:       gas,
		GasFeeCap: newGwei(10),
		GasTipCap: newGwei(tipGwei),
		Data:      data,
	})
}

// ============================== Account inclusion ==============================

// TestBALTxSenderAndRecipient: a value transfer records balance+nonce for sender
// and a balance entry for the recipient.
func TestBALTxSenderAndRecipient(t *testing.T) {
	to := common.HexToAddress("0xc0ffee")
	env := newBALTestEnv(nil)

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &to, big.NewInt(1000), txGasNewAccount, 0, nil))
	})

	sender := assertPresent(t, b, env.from)
	if len(sender.NonceChanges) == 0 || sender.NonceChanges[0].PostNonce != 1 {
		t.Fatalf("sender nonce not bumped: %+v", sender.NonceChanges)
	}
	if len(sender.BalanceChanges) == 0 {
		t.Fatalf("sender missing balance change")
	}
	recipient := assertPresent(t, b, to)
	if len(recipient.BalanceChanges) != 1 || recipient.BalanceChanges[0].PostBalance.Uint64() != 1000 {
		t.Fatalf("recipient balance: %+v", recipient.BalanceChanges)
	}
}

// TestBALZeroValueRecipient: a tx with value 0 still lists the recipient,
// but without a balance entry.
func TestBALZeroValueRecipient(t *testing.T) {
	to := common.HexToAddress("0x0123456789abcdef")
	env := newBALTestEnv(nil)

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &to, big.NewInt(0), params.TxGas, 0, nil))
	})

	r := assertPresent(t, b, to)
	if len(r.BalanceChanges) != 0 {
		t.Fatalf("zero-value recipient should have no balance entry: %+v", r.BalanceChanges)
	}
}

// TestBALEmptyBlockExcludesCoinbase: an empty block (no txs, no withdrawals)
// never touches the coinbase, so it must NOT appear in the BAL — the zero
// block reward alone does not trigger inclusion.
func TestBALEmptyBlockExcludesCoinbase(t *testing.T) {
	coinbase := common.Address{0xc0}
	env := newBALTestEnv(nil)

	b, _ := env.run(t, func(g *BlockGen) {
		// SetCoinbase initialises b.bal but does not record any access.
		g.SetCoinbase(coinbase)
	})
	assertAbsent(t, b, coinbase)
}

// TestBALCoinbasePerTxBalance checks that fee-recipient balances are recorded
// after each transaction, rather than only once at the end of the block.
func TestBALCoinbasePerTxBalance(t *testing.T) {
	coinbase := common.HexToAddress("0xc01babe")
	to := common.HexToAddress("0xc0ffee")
	env := newBALTestEnv(nil)

	b, receipts := env.run(t, func(g *BlockGen) {
		g.SetCoinbase(coinbase)
		g.AddTx(env.tx(0, &to, big.NewInt(0), 100_000, 1, nil))
		g.AddTx(env.tx(1, &to, big.NewInt(0), 100_000, 1, nil))
	})

	feeRecipient := assertPresent(t, b, coinbase)
	if len(feeRecipient.BalanceChanges) != 2 {
		t.Fatalf("fee recipient must have one balance per transaction: %+v", feeRecipient.BalanceChanges)
	}
	first := new(big.Int).Mul(new(big.Int).SetUint64(receipts[0].GasUsed), newGwei(1))
	second := new(big.Int).Add(first, new(big.Int).Mul(new(big.Int).SetUint64(receipts[1].GasUsed), newGwei(1)))
	for i, want := range []*big.Int{first, second} {
		change := feeRecipient.BalanceChanges[i]
		if change.BlockAccessIndex != uint32(i+1) || change.PostBalance.ToBig().Cmp(want) != 0 {
			t.Fatalf("fee recipient change %d: have %+v, want index %d balance %s", i, change, i+1, want)
		}
	}
}

// TestBALSystemAddressExcluded: SYSTEM_ADDRESS (0xff…fe) is not in the BAL
// for a regular block.
func TestBALSystemAddressExcluded(t *testing.T) {
	to := common.HexToAddress("0xabba")
	env := newBALTestEnv(nil)

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &to, big.NewInt(0), params.TxGas, 0, nil))
	})
	assertAbsent(t, b, params.SystemAddress)
}

// TestBALSystemAddressIncludedWhenTouched: SYSTEM_ADDRESS becomes a regular
// account in the BAL once it experiences state access (here: receives value).
func TestBALSystemAddressIncludedWhenTouched(t *testing.T) {
	sys := params.SystemAddress
	env := newBALTestEnv(nil)

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &sys, big.NewInt(1000), txGasNewAccount, 0, nil))
	})

	aa := assertPresent(t, b, sys)
	if len(aa.BalanceChanges) != 1 || aa.BalanceChanges[0].PostBalance.Uint64() != 1000 {
		t.Fatalf("system-address balance change missing: %+v", aa.BalanceChanges)
	}
}

// TestBALPrecompileInvokedFromContractIncluded: a precompile that is invoked
// indirectly — via STATICCALL from a regular contract — must still appear in
// the BAL with no balance entry.
func TestBALPrecompileInvokedFromContractIncluded(t *testing.T) {
	identity := common.BytesToAddress([]byte{0x04})
	caller := common.HexToAddress("0xca11")
	// PUSH1 0 (retSize) PUSH1 0 (retOff) PUSH1 0 (argsSize) PUSH1 0 (argsOff)
	// PUSH20 0x04 GAS STATICCALL POP STOP
	code := []byte{0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x73}
	code = append(code, identity.Bytes()...)
	code = append(code, 0x5a, 0xfa, 0x50, 0x00)

	env := newBALTestEnv(types.GenesisAlloc{caller: {Code: code, Balance: common.Big0}})
	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &caller, big.NewInt(0), 1_000_000, 0, nil))
	})

	aa := assertPresent(t, b, identity)
	if len(aa.BalanceChanges) != 0 {
		t.Fatalf("precompile invoked via STATICCALL must not record balance: %+v", aa.BalanceChanges)
	}
}

// TestBALPrecompileCalledNoValueIncluded: a tx targeting the identity precompile
// with zero value lists the precompile but records no balance entry.
func TestBALPrecompileCalledNoValueIncluded(t *testing.T) {
	identity := common.BytesToAddress([]byte{0x04})
	env := newBALTestEnv(nil)

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &identity, big.NewInt(0), 50_000, 0, []byte{0xde, 0xad}))
	})

	aa := assertPresent(t, b, identity)
	if len(aa.BalanceChanges) != 0 {
		t.Fatalf("precompile must not record balance change: %+v", aa.BalanceChanges)
	}
}

// TestBALPrecompileValueTransferRecordsBalance: a precompile receives ETH only
// in the form of a value transfer — the balance entry is then recorded.
func TestBALPrecompileValueTransferRecordsBalance(t *testing.T) {
	identity := common.BytesToAddress([]byte{0x04})
	env := newBALTestEnv(nil)

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &identity, big.NewInt(5), txGasNewAccount, 0, nil))
	})

	aa := assertPresent(t, b, identity)
	if len(aa.BalanceChanges) != 1 || aa.BalanceChanges[0].PostBalance.Uint64() != 5 {
		t.Fatalf("precompile balance change wrong: %+v", aa.BalanceChanges)
	}
}

// TestBALBalanceProbeOnNonExistent: BALANCE against a never-allocated address
// still adds it to the BAL with an empty change set.
func TestBALBalanceProbeOnNonExistent(t *testing.T) {
	probe := common.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	caller := common.HexToAddress("0xc1")
	code := append([]byte{0x73}, probe.Bytes()...) // PUSH20 probe
	code = append(code, 0x31, 0x50, 0x00)          // BALANCE, POP, STOP

	env := newBALTestEnv(types.GenesisAlloc{caller: {Code: code, Balance: common.Big0}})
	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &caller, big.NewInt(0), 1_000_000, 0, nil))
	})

	assertEmpty(t, assertPresent(t, b, probe))
}

// TestBALExtCodeSizeProbeOnNonExistent: EXTCODESIZE against a never-allocated
// address adds it to the BAL with an empty change set.
func TestBALExtCodeSizeProbeOnNonExistent(t *testing.T) {
	probe := common.HexToAddress("0xcafecafecafecafecafecafecafecafecafecafe")
	caller := common.HexToAddress("0xc1")
	code := append([]byte{0x73}, probe.Bytes()...) // PUSH20 probe
	code = append(code, 0x3b, 0x50, 0x00)          // EXTCODESIZE, POP, STOP

	env := newBALTestEnv(types.GenesisAlloc{caller: {Code: code, Balance: common.Big0}})
	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &caller, big.NewInt(0), 1_000_000, 0, nil))
	})

	assertEmpty(t, assertPresent(t, b, probe))
}

// TestBALExtCodeHashProbeOnNonExistent: EXTCODEHASH against a never-allocated
// address adds it to the BAL with an empty change set.
func TestBALExtCodeHashProbeOnNonExistent(t *testing.T) {
	probe := common.HexToAddress("0xfacefacefacefacefacefacefacefacefacefacE")
	caller := common.HexToAddress("0xc1")
	code := append([]byte{0x73}, probe.Bytes()...) // PUSH20 probe
	code = append(code, 0x3f, 0x50, 0x00)          // EXTCODEHASH, POP, STOP

	env := newBALTestEnv(types.GenesisAlloc{caller: {Code: code, Balance: common.Big0}})
	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &caller, big.NewInt(0), 1_000_000, 0, nil))
	})

	assertEmpty(t, assertPresent(t, b, probe))
}

// TestBALExtCodeCopyProbeOnNonExistent: EXTCODECOPY against a never-allocated
// address adds it to the BAL with an empty change set.
func TestBALExtCodeCopyProbeOnNonExistent(t *testing.T) {
	probe := common.HexToAddress("0xfeedfeedfeedfeedfeedfeedfeedfeedfeedfeed")
	caller := common.HexToAddress("0xc1")
	// PUSH1 0 (length) PUSH1 0 (codeOffset) PUSH1 0 (destOffset)
	// PUSH20 probe EXTCODECOPY STOP
	code := []byte{0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x73}
	code = append(code, probe.Bytes()...)
	code = append(code, 0x3c, 0x00) // EXTCODECOPY, STOP

	env := newBALTestEnv(types.GenesisAlloc{caller: {Code: code, Balance: common.Big0}})
	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &caller, big.NewInt(0), 1_000_000, 0, nil))
	})

	assertEmpty(t, assertPresent(t, b, probe))
}

// TestBALAccessListNotAutoPromoted: an EIP-2930 access-list entry that is
// never actually touched must NOT appear in the BAL.
func TestBALAccessListNotAutoPromoted(t *testing.T) {
	to := common.HexToAddress("0xabba")
	dormant := common.HexToAddress("0xd0d0")
	env := newBALTestEnv(nil)

	b, _ := env.run(t, func(g *BlockGen) {
		tx := types.MustSignNewTx(env.key, env.signer, &types.DynamicFeeTx{
			ChainID:    env.cfg.ChainID,
			Nonce:      0,
			To:         &to,
			Value:      big.NewInt(0),
			Gas:        params.TxGas + 4000,
			GasFeeCap:  newGwei(10),
			GasTipCap:  newGwei(0),
			AccessList: types.AccessList{{Address: dormant, StorageKeys: nil}},
		})
		g.AddTx(tx)
	})

	assertAbsent(t, b, dormant)
}

// ============================== CALL family ==============================

// makeStubCaller emits a single CALL-family op against `target` then STOPs,
// with zero call data and discarded return data.
//
//	op = 0xf1 (CALL) / 0xf2 (CALLCODE):
//	  stack = retSize, retOff, argsSize, argsOff, value, addr, gas
//	op = 0xf4 (DELEGATECALL) / 0xfa (STATICCALL):
//	  stack = retSize, retOff, argsSize, argsOff, addr, gas
func makeStubCaller(op byte, target common.Address) []byte {
	// retSize, retOff, argsSize, argsOff = 0
	prelude := []byte{0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00}
	if op == 0xf1 || op == 0xf2 { // CALL/CALLCODE need an extra value=0
		prelude = append(prelude, 0x60, 0x00)
	}
	prelude = append(prelude, 0x73) // PUSH20
	prelude = append(prelude, target.Bytes()...)
	prelude = append(prelude, 0x5a) // GAS
	prelude = append(prelude, op)
	prelude = append(prelude, 0x50, 0x00) // POP, STOP
	return prelude
}

// TestBALCallTargetWithEmptyChangeSet: a zero-value CALL to an existing
// contract that has no state changes lists the target with empty entries.
func TestBALCallTargetWithEmptyChangeSet(t *testing.T) {
	target := common.HexToAddress("0xbabe")
	env := newBALTestEnv(types.GenesisAlloc{
		target: {Code: []byte{0x00}, Balance: common.Big0}, // STOP
	})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &target, big.NewInt(0), 1_000_000, 0, nil))
	})

	assertEmpty(t, assertPresent(t, b, target))
}

// TestBALCallCodeTargetIncluded: CALLCODE puts the target in the BAL with an
// empty change set (CALLCODE executes target's code in the caller's storage
// context, so the target itself records no state changes).
func TestBALCallCodeTargetIncluded(t *testing.T) {
	target := common.HexToAddress("0xdeed")
	caller := common.HexToAddress("0xca11")
	env := newBALTestEnv(types.GenesisAlloc{
		caller: {Code: makeStubCaller(0xf2 /* CALLCODE */, target), Balance: common.Big0},
		target: {Code: []byte{0x00}, Balance: common.Big0},
	})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &caller, big.NewInt(0), 1_000_000, 0, nil))
	})

	assertPresent(t, b, caller)
	assertEmpty(t, assertPresent(t, b, target))
}

// TestBALDelegateCallTargetIncluded: DELEGATECALL puts both caller and target
// in the BAL even when neither produces state changes.
func TestBALDelegateCallTargetIncluded(t *testing.T) {
	target := common.HexToAddress("0xdeed")
	caller := common.HexToAddress("0xca11")
	env := newBALTestEnv(types.GenesisAlloc{
		caller: {Code: makeStubCaller(0xf4 /* DELEGATECALL */, target), Balance: common.Big0},
		target: {Code: []byte{0x00}, Balance: common.Big0},
	})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &caller, big.NewInt(0), 1_000_000, 0, nil))
	})

	assertPresent(t, b, caller)
	assertEmpty(t, assertPresent(t, b, target))
}

// TestBALStaticCallTargetIncluded: STATICCALL puts the target in the BAL with
// no balance entry recorded.
func TestBALStaticCallTargetIncluded(t *testing.T) {
	target := common.HexToAddress("0xdeed")
	caller := common.HexToAddress("0xca11")
	env := newBALTestEnv(types.GenesisAlloc{
		caller: {Code: makeStubCaller(0xfa /* STATICCALL */, target), Balance: common.Big0},
		target: {Code: []byte{0x00}, Balance: common.Big0},
	})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &caller, big.NewInt(0), 1_000_000, 0, nil))
	})

	assertPresent(t, b, caller)
	assertEmpty(t, assertPresent(t, b, target))
}

// makeValueCaller emits a single value-transferring CALL-family op (CALL 0xf1
// or CALLCODE 0xf2) against `target` with value=1, then STOPs. Used together
// with a zero-balance caller to make the value transfer fail CanTransfer.
func makeValueCaller(op byte, target common.Address) []byte {
	code := []byte{
		0x60, 0x00, // retSize
		0x60, 0x00, // retOff
		0x60, 0x00, // argsSize
		0x60, 0x00, // argsOff
		0x60, 0x01, // value = 1
		0x73, // PUSH20 target
	}
	code = append(code, target.Bytes()...)
	return append(code, 0x5a, op, 0x50, 0x00) // GAS, op, POP, STOP
}

// TestBALCallToDelegatedTargetBalanceFail asserts the EIP-7928 rule revised in
// ethereum/EIPs#11838: when a CALL targets an EIP-7702 delegated account and the
// delegated address passes its access_cost gas check, the delegated
// (implementation) address MUST appear in the BAL even when the call then fails
// its sender-balance check, because the delegation is resolved before that
// check. CALL routes through the EIP-8037 gas path.
func TestBALCallToDelegatedTargetBalanceFail(t *testing.T) {
	delegated := common.HexToAddress("0xde1e9a7ed") // EOA carrying a 7702 designator
	impl := common.HexToAddress("0x111111")         // delegation target (implementation)
	caller := common.HexToAddress("0xca11")         // zero-balance contract issuing the CALL

	env := newBALTestEnv(types.GenesisAlloc{
		caller:    {Code: makeValueCaller(0xf1 /* CALL */, delegated), Balance: common.Big0},
		delegated: {Code: types.AddressToDelegation(impl), Balance: common.Big0},
		impl:      {Code: []byte{0x00}, Balance: common.Big0}, // STOP
	})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &caller, big.NewInt(0), 1_000_000, 0, nil))
	})

	assertPresent(t, b, caller)
	assertPresent(t, b, delegated)
	// The call failed its sender-balance check, so the implementation never
	// executed: it is recorded with an empty change set, but it MUST be present.
	assertEmpty(t, assertPresent(t, b, impl))
}

// TestBALCallCodeToDelegatedTargetBalanceFail is the CALLCODE analogue of
// TestBALCallToDelegatedTargetBalanceFail, exercising the EIP-7702 gas path
// (CALLCODE/STATICCALL/DELEGATECALL) rather than the EIP-8037 one.
func TestBALCallCodeToDelegatedTargetBalanceFail(t *testing.T) {
	delegated := common.HexToAddress("0xde1e9a7ed")
	impl := common.HexToAddress("0x111111")
	caller := common.HexToAddress("0xca11")

	env := newBALTestEnv(types.GenesisAlloc{
		caller:    {Code: makeValueCaller(0xf2 /* CALLCODE */, delegated), Balance: common.Big0},
		delegated: {Code: types.AddressToDelegation(impl), Balance: common.Big0},
		impl:      {Code: []byte{0x00}, Balance: common.Big0}, // STOP
	})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &caller, big.NewInt(0), 1_000_000, 0, nil))
	})

	assertPresent(t, b, caller)
	assertPresent(t, b, delegated)
	assertEmpty(t, assertPresent(t, b, impl))
}

// ============================== Reverts and exceptional halts ==============================

// TestBALRevertedTxStillIncluded: a tx whose top-level call REVERTs still
// records the touched contract in the BAL with an empty change set.
func TestBALRevertedTxStillIncluded(t *testing.T) {
	reverter := common.HexToAddress("0xbeef")
	// PUSH1 0 PUSH1 0 REVERT
	revertCode := []byte{0x60, 0x00, 0x60, 0x00, 0xfd}
	env := newBALTestEnv(types.GenesisAlloc{reverter: {Code: revertCode, Balance: common.Big0}})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &reverter, big.NewInt(0), 1_000_000, 0, nil))
	})

	assertEmpty(t, assertPresent(t, b, reverter))
}

// TestBALSenderRecordedOnRevert: even when the top-level call reverts, the
// sender's final nonce and balance MUST be recorded.
func TestBALSenderRecordedOnRevert(t *testing.T) {
	reverter := common.HexToAddress("0xbeef")
	revertCode := []byte{0x60, 0x00, 0x60, 0x00, 0xfd}
	env := newBALTestEnv(types.GenesisAlloc{reverter: {Code: revertCode, Balance: common.Big0}})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &reverter, big.NewInt(0), 1_000_000, 0, nil))
	})

	sender := assertPresent(t, b, env.from)
	if len(sender.NonceChanges) == 0 || sender.NonceChanges[0].PostNonce != 1 {
		t.Fatalf("sender nonce must be bumped even on revert: %+v", sender.NonceChanges)
	}
	if len(sender.BalanceChanges) == 0 {
		t.Fatalf("sender balance change (gas paid) must be present on revert")
	}
}

// TestBALDelegationTargetOOG exercises the exceptional-halt
// boundary in EIP-7928. The top-level recipient is loaded to discover its
// delegation, but the runtime budget is insufficient for the cold target
// access, so the implementation must not enter the BAL.
func TestBALDelegationTargetOOG(t *testing.T) {
	authority := common.HexToAddress("0xa7702")
	implementation := common.HexToAddress("0x1a11")
	env := newBALTestEnv(types.GenesisAlloc{
		authority:      {Code: types.AddressToDelegation(implementation), Balance: common.Big0},
		implementation: {Code: []byte{0x00}, Balance: common.Big0},
	})

	b, receipts := env.run(t, func(g *BlockGen) {
		// This transaction has exactly enough gas for its intrinsic charges, but
		// less than the 3,000 cold-account runtime charge needed to load target.
		g.AddTx(env.tx(0, &authority, big.NewInt(0), 15_000, 0, nil))
	})
	if receipts[0].Status != types.ReceiptStatusFailed {
		t.Fatalf("expected runtime out-of-gas receipt, have status %d", receipts[0].Status)
	}
	assertEmpty(t, assertPresent(t, b, authority))
	assertAbsent(t, b, implementation)
}

// ============================== Storage inclusion ==============================

// TestBALStorageSloadOnly: SLOAD without a write puts the slot in storage_reads.
func TestBALStorageSloadOnly(t *testing.T) {
	contract := common.HexToAddress("0xc1")
	slot := common.BigToHash(big.NewInt(0x07))
	// PUSH1 0x07 SLOAD POP STOP
	code := []byte{0x60, 0x07, 0x54, 0x50, 0x00}
	env := newBALTestEnv(types.GenesisAlloc{contract: {Code: code, Balance: common.Big0}})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &contract, big.NewInt(0), 1_000_000, 0, nil))
	})

	aa := assertPresent(t, b, contract)
	if !hasSlotIn(aa.StorageReads, slot) {
		t.Fatalf("expected slot in storage_reads\n%s", b.PrettyPrint())
	}
	if hasStorageWrite(b, contract, slot) {
		t.Fatalf("slot must NOT appear in storage_changes")
	}
}

// TestBALStorageReadThenWriteOnlyInWrites: SLOAD followed by SSTORE on the
// same slot drops the slot from storage_reads (write-wins invariant).
func TestBALStorageReadThenWriteOnlyInWrites(t *testing.T) {
	contract := common.HexToAddress("0xc1")
	slot := common.BigToHash(big.NewInt(0x05))
	// PUSH1 5 SLOAD POP   PUSH1 0x42 PUSH1 5 SSTORE   STOP
	code := []byte{
		0x60, 0x05, 0x54, 0x50,
		0x60, 0x42, 0x60, 0x05, 0x55,
		0x00,
	}
	env := newBALTestEnv(types.GenesisAlloc{contract: {Code: code, Balance: common.Big0}})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &contract, big.NewInt(0), 1_000_000, 0, nil))
	})

	aa := assertPresent(t, b, contract)
	if !hasStorageWrite(b, contract, slot) {
		t.Fatalf("slot must be in storage_changes\n%s", b.PrettyPrint())
	}
	if hasSlotIn(aa.StorageReads, slot) {
		t.Fatalf("slot must NOT appear in storage_reads (write-wins)\n%s", b.PrettyPrint())
	}
}

// TestBALNoOpSSTOREDemotesToRead: an SSTORE whose value equals the committed
// value lands the slot in storage_reads only.
func TestBALNoOpSSTOREDemotesToRead(t *testing.T) {
	contract := common.HexToAddress("0xc1")
	slot := common.BigToHash(big.NewInt(0x09))
	// SSTORE(0x09, 0x42) — slot pre-state is 0x42, so the write is a no-op.
	code := []byte{0x60, 0x42, 0x60, 0x09, 0x55, 0x00}
	env := newBALTestEnv(types.GenesisAlloc{
		contract: {
			Code:    code,
			Balance: common.Big0,
			Storage: map[common.Hash]common.Hash{slot: common.BigToHash(big.NewInt(0x42))},
		},
	})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &contract, big.NewInt(0), 1_000_000, 0, nil))
	})

	aa := assertPresent(t, b, contract)
	if !hasSlotIn(aa.StorageReads, slot) {
		t.Fatalf("no-op SSTORE should leave slot in storage_reads\n%s", b.PrettyPrint())
	}
	if hasStorageWrite(b, contract, slot) {
		t.Fatalf("no-op SSTORE must NOT register a write")
	}
}

// TestBALStorageWriteZeroIsAWrite: writing 0 to a non-zero slot is still a
// state change and lands in storage_changes.
func TestBALStorageWriteZeroIsAWrite(t *testing.T) {
	contract := common.HexToAddress("0xc1")
	slot := common.BigToHash(big.NewInt(0x03))
	// PUSH1 0 PUSH1 3 SSTORE STOP
	code := []byte{0x60, 0x00, 0x60, 0x03, 0x55, 0x00}
	env := newBALTestEnv(types.GenesisAlloc{
		contract: {
			Code:    code,
			Balance: common.Big0,
			Storage: map[common.Hash]common.Hash{slot: common.BigToHash(big.NewInt(0x42))},
		},
	})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &contract, big.NewInt(0), 1_000_000, 0, nil))
	})

	aa := assertPresent(t, b, contract)
	if !hasStorageWrite(b, contract, slot) {
		t.Fatalf("SSTORE to zero must record a write\n%s", b.PrettyPrint())
	}
	for _, w := range aa.StorageChanges {
		if w.Slot.Uint64() == 0x03 {
			if len(w.SlotChanges) != 1 || !w.SlotChanges[0].PostValue.IsZero() {
				t.Fatalf("expected post-value 0 for slot 0x03, got %+v", w.SlotChanges)
			}
		}
	}
}

// ============================== CREATE / contract deployment ==============================

// TestBALCreateDeploysCode: a successful contract-creation tx records the new
// address with nonce 0→1, a balance entry (value transferred), and a code entry.
func TestBALCreateDeploysCode(t *testing.T) {
	env := newBALTestEnv(nil)
	// Init: deploy runtime [0x00] (single STOP byte).
	// PUSH1 0 PUSH1 0 MSTORE8   PUSH1 1 PUSH1 0 RETURN
	init := []byte{0x60, 0x00, 0x60, 0x00, 0x53, 0x60, 0x01, 0x60, 0x00, 0xf3}

	b, receipts := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, nil, big.NewInt(7), 1_000_000, 0, init))
	})

	created := receipts[0].ContractAddress
	aa := assertPresent(t, b, created)
	if len(aa.NonceChanges) != 1 || aa.NonceChanges[0].PostNonce != 1 {
		t.Fatalf("expected nonce 0→1, got %+v", aa.NonceChanges)
	}
	if len(aa.CodeChanges) != 1 || !bytes.Equal(aa.CodeChanges[0].NewCode, []byte{0x00}) {
		t.Fatalf("expected code [0x00], got %+v", aa.CodeChanges)
	}
	if len(aa.BalanceChanges) != 1 || aa.BalanceChanges[0].PostBalance.Uint64() != 7 {
		t.Fatalf("expected balance 7, got %+v", aa.BalanceChanges)
	}
}

// TestBALCreateEmptyRuntimeNoCodeEntry: when init code returns 0 bytes the
// new address is still listed with nonce 0→1 but no code entry.
func TestBALCreateEmptyRuntimeNoCodeEntry(t *testing.T) {
	env := newBALTestEnv(nil)
	// Init: PUSH1 0 PUSH1 0 RETURN  → returns 0 bytes
	init := []byte{0x60, 0x00, 0x60, 0x00, 0xf3}

	b, receipts := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, nil, big.NewInt(0), 1_000_000, 0, init))
	})

	created := receipts[0].ContractAddress
	aa := assertPresent(t, b, created)
	if len(aa.NonceChanges) != 1 || aa.NonceChanges[0].PostNonce != 1 {
		t.Fatalf("expected nonce 0→1, got %+v", aa.NonceChanges)
	}
	if len(aa.CodeChanges) != 0 {
		t.Fatalf("empty runtime must NOT record a code entry, got %+v", aa.CodeChanges)
	}
}

// TestBALCreateInitRevertEmptyChangeSet: when init code reverts, the would-be
// contract address is in the BAL with an empty change set.
func TestBALCreateInitRevertEmptyChangeSet(t *testing.T) {
	env := newBALTestEnv(nil)
	// PUSH1 0 PUSH1 0 REVERT
	init := []byte{0x60, 0x00, 0x60, 0x00, 0xfd}

	b, receipts := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, nil, big.NewInt(0), 1_000_000, 0, init))
	})

	created := receipts[0].ContractAddress
	assertEmpty(t, assertPresent(t, b, created))
}

// TestBALCreateInitOOGEmptyChangeSet: init code that runs out of gas leaves
// the deployed address in the BAL with an empty change set.
func TestBALCreateInitOOGEmptyChangeSet(t *testing.T) {
	env := newBALTestEnv(nil)
	// Infinite loop: JUMPDEST PUSH1 0 JUMP — burns gas until OOG. The
	// gas budget must cover EIP-8037 intrinsic state gas (account creation)
	// so the tx is accepted; OOG must happen inside the init code.
	init := []byte{0x5b, 0x60, 0x00, 0x56}

	b, receipts := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, nil, big.NewInt(0), 220_000, 0, init))
	})

	created := receipts[0].ContractAddress
	assertEmpty(t, assertPresent(t, b, created))
}

// TestBALCreateAddressCollisionStillIncluded: when CREATE targets an address
// that already holds a contract, the deployment fails but the address was
// probed during execution and MUST appear in the BAL with an empty change set.
func TestBALCreateAddressCollisionStillIncluded(t *testing.T) {
	env := newBALTestEnv(nil)
	// For a top-level CREATE tx the deployed address is CreateAddress(sender, 0).
	// Pre-allocate a contract at that address to provoke ErrContractAddressCollision.
	collide := crypto.CreateAddress(env.from, 0)
	env.gspec.Alloc[collide] = types.Account{
		Nonce:   1,
		Code:    []byte{0x00},
		Balance: common.Big0,
	}

	// Init code doesn't matter — execution never starts.
	init := []byte{0x60, 0x00, 0x60, 0x00, 0xf3}
	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, nil, big.NewInt(0), 1_000_000, 0, init))
	})

	aa := assertPresent(t, b, collide)
	// The address must be present but the pre-existing nonce/code MUST NOT
	// be overwritten by the failed creation.
	if len(aa.NonceChanges) != 0 {
		t.Fatalf("collision must not bump nonce: %+v", aa.NonceChanges)
	}
	if len(aa.CodeChanges) != 0 {
		t.Fatalf("collision must not write code: %+v", aa.CodeChanges)
	}
}

// TestBALInEVMCreatePreAccessAbortDestinationExcluded: if a CREATE frame
// aborts BEFORE the destination is read from state (here: the caller has 0
// balance and CREATE requests value > 0, tripping evm.create's CanTransfer
// check before GetCodeHash), the would-be address MUST NOT appear in the
// BAL — only "if target account is accessed" qualifies for inclusion.
func TestBALInEVMCreatePreAccessAbortDestinationExcluded(t *testing.T) {
	factory := common.HexToAddress("0xfac4")
	// PUSH1 0 (length) PUSH1 0 (offset) PUSH1 1 (value)  CREATE  POP STOP
	code := []byte{0x60, 0x00, 0x60, 0x00, 0x60, 0x01, 0xf0, 0x50, 0x00}
	env := newBALTestEnv(types.GenesisAlloc{
		factory: {Code: code, Balance: common.Big0, Nonce: 1}, // factory has no balance
	})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &factory, big.NewInt(0), 1_000_000, 0, nil))
	})

	// The address that WOULD have been deployed had the create succeeded.
	wouldBeDest := crypto.CreateAddress(factory, 1)
	assertAbsent(t, b, wouldBeDest)

	// The factory itself is in BAL (it ran), but its nonce MUST NOT have been
	// bumped because evm.create returned before the SetNonce call.
	aa := assertPresent(t, b, factory)
	if len(aa.NonceChanges) != 0 {
		t.Fatalf("factory nonce must not be bumped on pre-access abort: %+v", aa.NonceChanges)
	}
}

// TestBALInEVMCreateOOGDestination distinguishes a CREATE precheck abort from
// an account-creation runtime OOG. The latter calls StateDB.Empty on the
// destination to determine whether the creation charge is due, so the
// destination has been accessed and must appear in the BAL even though the
// failed charge halts the transaction before evm.create runs.
func TestBALInEVMCreateOOGDestination(t *testing.T) {
	factory := common.HexToAddress("0xfac4")
	// PUSH1 0 (length) PUSH1 0 (offset) PUSH1 0 (value) CREATE POP STOP.
	// The factory has enough regular gas for CREATE's opcode cost but not enough
	// combined gas to pay Amsterdam's 183,600 account-creation state charge.
	code := []byte{0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0xf0, 0x50, 0x00}
	env := newBALTestEnv(types.GenesisAlloc{
		factory: {Code: code, Balance: common.Big0, Nonce: 1},
	})

	b, receipts := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &factory, big.NewInt(0), 30_000, 0, nil))
	})
	if receipts[0].Status != types.ReceiptStatusFailed {
		t.Fatalf("expected account-creation runtime OOG, have status %d", receipts[0].Status)
	}

	wouldBeDest := crypto.CreateAddress(factory, 1)
	assertEmpty(t, assertPresent(t, b, wouldBeDest))

	// evm.create is never entered, so its creator-nonce bump does not occur.
	aa := assertPresent(t, b, factory)
	if len(aa.NonceChanges) != 0 {
		t.Fatalf("factory nonce must not be bumped before account-creation charge succeeds: %+v", aa.NonceChanges)
	}
}

// TestBALInEVMCreateDeploysContract: a CREATE issued by an existing contract
// (not a top-level CREATE tx) records the deployed address in the BAL.
func TestBALInEVMCreateDeploysContract(t *testing.T) {
	factory := common.HexToAddress("0xfac4")
	// Factory code:
	//   Write 5-byte init code (0x60 0x00 0x60 0x00 0xf3) into memory starting at offset 0.
	//   Then CREATE(value=0, offset=0, length=5).
	//
	// Layout: store the init code as a single 32-byte word at offset 0 via MSTORE
	// with leftmost 27 bytes garbage, then call CREATE with offset = 27, length = 5.
	initBlob := []byte{0x60, 0x00, 0x60, 0x00, 0xf3}
	var word [32]byte
	copy(word[32-len(initBlob):], initBlob)
	code := []byte{0x7f} // PUSH32
	code = append(code, word[:]...)
	code = append(code, 0x60, 0x00, 0x52) // PUSH1 0, MSTORE
	// CREATE expects [value, offset, length] with value on bottom of stack.
	code = append(code,
		0x60, 0x05, // PUSH1 5 (length)
		0x60, 0x1b, // PUSH1 27 (offset)
		0x60, 0x00, // PUSH1 0 (value)
		0xf0, // CREATE
		0x00, // STOP (discard result)
	)

	env := newBALTestEnv(types.GenesisAlloc{factory: {Code: code, Balance: common.Big0, Nonce: 1}})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &factory, big.NewInt(0), 1_000_000, 0, nil))
	})

	// Deployed address depends on the factory's nonce at the moment of CREATE,
	// which is the factory's genesis nonce (1).
	deployed := crypto.CreateAddress(factory, 1)
	aa := assertPresent(t, b, deployed)
	if len(aa.NonceChanges) != 1 || aa.NonceChanges[0].PostNonce != 1 {
		t.Fatalf("deployed contract nonce: %+v", aa.NonceChanges)
	}
}

// ============================== SELFDESTRUCT ==============================

// TestBALSelfDestructBeneficiaryWithZeroBalance: SELFDESTRUCT to a fresh
// beneficiary when the destructing account has 0 balance — both addresses are
// listed with empty change sets (no balance entry).
func TestBALSelfDestructBeneficiaryWithZeroBalance(t *testing.T) {
	beneficiary := common.HexToAddress("0xbeefbeef")
	env := newBALTestEnv(nil)
	// Init code performs SELFDESTRUCT to beneficiary inside the constructor,
	// so EIP-6780's same-tx requirement is satisfied. The destructing account
	// starts with balance 0 because the creation tx sends 0 value.
	//   PUSH20 <ben> SELFDESTRUCT
	init := append([]byte{0x73}, beneficiary.Bytes()...)
	init = append(init, 0xff)

	b, receipts := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, nil, big.NewInt(0), 1_000_000, 0, init))
	})

	created := receipts[0].ContractAddress
	ben := assertPresent(t, b, beneficiary)
	if len(ben.BalanceChanges) != 0 {
		t.Fatalf("zero-value SELFDESTRUCT must not credit beneficiary: %+v", ben.BalanceChanges)
	}
	cc := assertPresent(t, b, created)
	if len(cc.BalanceChanges) != 0 {
		t.Fatalf("destructing contract must not record a balance entry: %+v", cc.BalanceChanges)
	}
}

// TestBALSelfDestructBeneficiaryWithValueTransfer: SELFDESTRUCT from a freshly
// created contract that received positive value — beneficiary records the
// credit; destructing account's balance entry is omitted because its
// pre-transaction balance was 0.
func TestBALSelfDestructBeneficiaryWithValueTransfer(t *testing.T) {
	beneficiary := common.HexToAddress("0xbeefbeef")
	env := newBALTestEnv(nil)
	// Init code: PUSH20 <ben> SELFDESTRUCT
	init := append([]byte{0x73}, beneficiary.Bytes()...)
	init = append(init, 0xff)

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, nil, big.NewInt(100), 1_000_000, 0, init))
	})

	ben := assertPresent(t, b, beneficiary)
	if len(ben.BalanceChanges) != 1 || ben.BalanceChanges[0].PostBalance.Uint64() != 100 {
		t.Fatalf("beneficiary balance must be credited with 100: %+v", ben.BalanceChanges)
	}
}

// TestBALSelfDestructPreExistingContract: SELFDESTRUCT on a pre-existing
// contract with positive balance records balance→0 for the contract and the
// corresponding credit on the beneficiary. EIP-6780 means the contract is
// only credited and not deleted, but its balance moves regardless.
func TestBALSelfDestructPreExistingContract(t *testing.T) {
	suicidal := common.HexToAddress("0x5e1f")
	beneficiary := common.HexToAddress("0xbeefbeef")
	// PUSH20 <ben> SELFDESTRUCT
	code := append([]byte{0x73}, beneficiary.Bytes()...)
	code = append(code, 0xff)

	env := newBALTestEnv(types.GenesisAlloc{
		suicidal: {Code: code, Balance: big.NewInt(50)},
	})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &suicidal, big.NewInt(0), 1_000_000, 0, nil))
	})

	aa := assertPresent(t, b, suicidal)
	if len(aa.BalanceChanges) != 1 || !aa.BalanceChanges[0].PostBalance.IsZero() {
		t.Fatalf("suicidal contract balance should drop to 0: %+v", aa.BalanceChanges)
	}
	ben := assertPresent(t, b, beneficiary)
	if len(ben.BalanceChanges) != 1 || ben.BalanceChanges[0].PostBalance.Uint64() != 50 {
		t.Fatalf("beneficiary should receive 50: %+v", ben.BalanceChanges)
	}
}

// TestBALSelfDestructToSelfKeepsBalance: under EIP-8246 a freshly created
// contract that self-destructs to itself keeps its balance (it is not burnt and
// the account is not removed). The surviving balance-only account must therefore
// be recorded in the BAL with its preserved balance.
func TestBALSelfDestructToSelfKeepsBalance(t *testing.T) {
	env := newBALTestEnv(nil)
	// Init code: ADDRESS SELFDESTRUCT — the contract self-destructs to itself
	// during its own creation transaction (satisfying EIP-6780's same-tx rule).
	//   ADDRESS (0x30) ; SELFDESTRUCT (0xff)
	init := []byte{0x30, 0xff}

	b, receipts := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, nil, big.NewInt(100), 1_000_000, 0, init))
	})

	created := receipts[0].ContractAddress
	cc := assertPresent(t, b, created)
	// EIP-8246: balance preserved (not burnt), account survives -> the BAL must
	// record the created address with its retained balance.
	if len(cc.BalanceChanges) != 1 || cc.BalanceChanges[0].PostBalance.Uint64() != 100 {
		t.Fatalf("self-destruct-to-self must preserve balance 100 in the BAL: %+v", cc.BalanceChanges)
	}
}

// TestBALSelfDestructToSelfPrefundedUnchanged: a pre-funded address onto which a
// contract is deployed and which self-destructs to itself in the same
// transaction. Under EIP-8246 the account survives with its balance unchanged,
// so the BAL must list it only as an access (no balance/nonce/code change).
func TestBALSelfDestructToSelfPrefundedUnchanged(t *testing.T) {
	// The contract address created by the sender's nonce-0 transaction; it is
	// pre-funded in genesis (balance only: nonce 0, no code, no storage), which
	// EIP-7610 permits as a deployment target.
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	created := crypto.CreateAddress(crypto.PubkeyToAddress(key.PublicKey), 0)

	env := newBALTestEnv(types.GenesisAlloc{
		created: {Balance: big.NewInt(77)},
	})
	// Init code: ADDRESS SELFDESTRUCT, deployed with zero value so the balance is
	// untouched (stays at the pre-funded 77).
	init := []byte{0x30, 0xff}

	b, receipts := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, nil, big.NewInt(0), 1_000_000, 0, init))
	})

	if receipts[0].ContractAddress != created {
		t.Fatalf("unexpected created address: have %x want %x", receipts[0].ContractAddress, created)
	}
	aa := assertPresent(t, b, created)
	// EIP-8246: balance preserved and equal to the pre-transaction value, so no
	// balance change; nonce and code end where they started (0 / empty). The
	// account is only read, with an empty change set.
	assertEmpty(t, aa)
}

// TestBALSelfDestructStorageRead checks the in-transaction
// SELFDESTRUCT rule: storage changed by an account that is subsequently
// deleted is retained as a read footprint, not as a storage change. The
// deleted account also must not carry nonce or code changes.
func TestBALSelfDestructStorageRead(t *testing.T) {
	beneficiary := common.HexToAddress("0xbeefbeef")
	slot := common.BigToHash(big.NewInt(0x05))
	env := newBALTestEnv(nil)
	// SSTORE(5, 0x42); SELFDESTRUCT(beneficiary). This executes in init code,
	// so the created account is deleted under EIP-6780.
	init := []byte{0x60, 0x42, 0x60, 0x05, 0x55, 0x73}
	init = append(init, beneficiary.Bytes()...)
	init = append(init, 0xff)

	b, receipts := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, nil, big.NewInt(100), 1_000_000, 0, init))
	})

	deleted := assertPresent(t, b, receipts[0].ContractAddress)
	if !hasSlotIn(deleted.StorageReads, slot) {
		t.Fatalf("deleted account storage slot %x must be in storage_reads\n%s", slot, b.PrettyPrint())
	}
	if hasStorageWrite(b, deleted.Address, slot) {
		t.Fatalf("deleted account storage slot %x must not remain in storage_changes\n%s", slot, b.PrettyPrint())
	}
	if len(deleted.NonceChanges) != 0 || len(deleted.CodeChanges) != 0 {
		t.Fatalf("deleted account must not record nonce or code: %+v", deleted)
	}
}

// TestBALSelfDestructDelegatedBeneficiary checks that SELFDESTRUCT treats a
// delegated authority as its beneficiary account, without loading the
// implementation as executable code.
func TestBALSelfDestructDelegatedBeneficiary(t *testing.T) {
	victim := common.HexToAddress("0x5e1f")
	authority := common.HexToAddress("0xa7702")
	implementation := common.HexToAddress("0x1a11")
	victimCode := append([]byte{0x73}, authority.Bytes()...)
	victimCode = append(victimCode, 0xff) // SELFDESTRUCT
	env := newBALTestEnv(types.GenesisAlloc{
		victim:         {Code: victimCode, Balance: common.Big0},
		authority:      {Code: types.AddressToDelegation(implementation), Balance: common.Big0},
		implementation: {Code: []byte{0x00}, Balance: common.Big0},
	})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &victim, big.NewInt(0), 1_000_000, 0, nil))
	})

	assertEmpty(t, assertPresent(t, b, authority))
	assertAbsent(t, b, implementation)
}

// ============================== Balance accounting ==============================

// TestBALMidTxBalanceRoundTrip: when an address's balance changes during a
// transaction but returns to its pre-transaction value, the address is still
// listed in the BAL but MUST NOT have a balance entry.
func TestBALMidTxBalanceRoundTrip(t *testing.T) {
	bouncer := common.HexToAddress("0xb0unce")
	// On receiving value, the bouncer immediately CALLs CALLER with CALLVALUE
	// and zero data. Net effect: bouncer.balance returns to its pre-tx value.
	//
	//   PUSH1 0 (retSize)
	//   PUSH1 0 (retOff)
	//   PUSH1 0 (argsSize)
	//   PUSH1 0 (argsOff)
	//   CALLVALUE
	//   CALLER
	//   GAS
	//   CALL
	//   POP
	//   STOP
	code := []byte{
		0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00,
		0x34, // CALLVALUE
		0x33, // CALLER
		0x5a, // GAS
		0xf1, // CALL
		0x50, // POP
		0x00, // STOP
	}
	env := newBALTestEnv(types.GenesisAlloc{bouncer: {Code: code, Balance: common.Big0}})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &bouncer, big.NewInt(1234), 1_000_000, 0, nil))
	})

	aa := assertPresent(t, b, bouncer)
	if len(aa.BalanceChanges) != 0 {
		t.Fatalf("mid-tx round-trip must not record a balance entry: %+v", aa.BalanceChanges)
	}
}

// TestBALGasRefundSenderBalance ensures a gas-refunding SSTORE still records
// the sender's post-transaction balance, after the refund has been applied
// to the transaction's final gas charge.
func TestBALGasRefundSenderBalance(t *testing.T) {
	contract := common.HexToAddress("0xc1")
	slot := common.BigToHash(big.NewInt(0x05))
	env := newBALTestEnv(types.GenesisAlloc{
		contract: {
			Code:    []byte{0x5f, 0x60, 0x05, 0x55, 0x00}, // SSTORE(5, 0); STOP
			Balance: common.Big0,
			Storage: map[common.Hash]common.Hash{slot: common.BigToHash(big.NewInt(1))},
		},
	})

	_, blocks, receipts := GenerateChainWithGenesis(env.gspec, beacon.New(ethash.NewFaker()), 1, func(_ int, g *BlockGen) {
		g.AddTx(env.tx(0, &contract, big.NewInt(0), 1_000_000, 0, nil))
	})
	b := blocks[0].AccessList()
	if b == nil {
		t.Fatal("expected non-nil block access list")
	}
	sender := assertPresent(t, b, env.from)
	if len(sender.BalanceChanges) != 1 || sender.BalanceChanges[0].BlockAccessIndex != 1 {
		t.Fatalf("sender needs one post-tx balance at index 1: %+v", sender.BalanceChanges)
	}
	gasCost := new(big.Int).Mul(new(big.Int).SetUint64(receipts[0][0].GasUsed), blocks[0].BaseFee())
	want := new(big.Int).Sub(newGwei(1_000_000_000), gasCost)
	if sender.BalanceChanges[0].PostBalance.ToBig().Cmp(want) != 0 {
		t.Fatalf("sender post-refund balance: have %s, want %s", sender.BalanceChanges[0].PostBalance, want)
	}
}

// ============================== System contracts (pre/post-execution) ==============================

// TestBALSystemContractsPresent: per EIP-7928, "System contract addresses
// accessed during pre/post-execution" MUST be included in the BAL. That
// means all four of the post-merge system contracts touched by every
// Amsterdam block:
//
//   - EIP-4788 beacon roots          (pre-execution, when ParentBeaconRoot is set)
//   - EIP-2935 history storage       (pre-execution)
//   - EIP-7002 withdrawal queue      (post-execution)
//   - EIP-7251 consolidation queue   (post-execution)
//   - EIP-8282 builder-deposit queue (post-execution)
//   - EIP-8282 builder-exit queue    (post-execution)
func TestBALSystemContractsPresent(t *testing.T) {
	env := newBALTestEnv(nil)

	b, _ := env.run(t, func(g *BlockGen) {
		// SetCoinbase initialises b.bal; SetParentBeaconRoot triggers EIP-4788.
		g.SetCoinbase(common.Address{0xc0})
		g.SetParentBeaconRoot(common.Hash{0xbe, 0xac})
	})

	for _, sys := range []struct {
		name string
		addr common.Address
	}{
		{"BeaconRoots (4788)", params.BeaconRootsAddress},
		{"HistoryStorage (2935)", params.HistoryStorageAddress},
		{"WithdrawalQueue (7002)", params.WithdrawalQueueAddress},
		{"ConsolidationQueue (7251)", params.ConsolidationQueueAddress},
		{"BuilderDepositQueue (8282)", params.BuilderDepositAddress},
		{"BuilderExitQueue (8282)", params.BuilderExitAddress},
	} {
		if findAccount(b, sys.addr) == nil {
			t.Errorf("%s (%x) MUST appear in BAL but is missing\n%s", sys.name, sys.addr, b.PrettyPrint())
		}
	}
}

// TestBALPreExecutionStorage checks the precise pre-execution entries required
// by EIP-7928: both EIP-2935 and EIP-4788 changes belong to index zero, before
// any transaction in the block.
func TestBALPreExecutionStorage(t *testing.T) {
	beaconRoot := common.HexToHash("0xbeac")
	env := newBALTestEnv(nil)

	b, _ := env.run(t, func(g *BlockGen) {
		g.SetParentBeaconRoot(beaconRoot)
	})

	// The generated block is number 1 and timestamp 10. EIP-2935 stores the
	// parent hash at (block.number - 1) % 8191, i.e. slot zero here.
	history := assertPresent(t, b, params.HistoryStorageAddress)
	assertStorageChanges(t, history, common.Hash{}, []balStorageChangeExpectation{{
		index: 0,
		value: env.gspec.ToBlock().Hash(),
	}})

	// EIP-4788 stores timestamp at timestamp % 8191 and the parent beacon root
	// at that slot plus 8191.
	const timestamp = 10
	beacon := assertPresent(t, b, params.BeaconRootsAddress)
	assertStorageChanges(t, beacon, common.BigToHash(big.NewInt(timestamp)), []balStorageChangeExpectation{{
		index: 0,
		value: common.BigToHash(big.NewInt(timestamp)),
	}})
	assertStorageChanges(t, beacon, common.BigToHash(big.NewInt(timestamp+8191)), []balStorageChangeExpectation{{
		index: 0,
		value: beaconRoot,
	}})
}

// TestBALPostExecutionQueueReads covers the EIP-7002 and EIP-7251 rule that
// a post-execution system call accesses queue metadata in slots 0..3 at index
// n+1, while the queued payload slots are read-only.
func TestBALPostExecutionQueueReads(t *testing.T) {
	withdrawalData := common.FromHex("b917cfdc0d25b72d55cf94db328e1629b7f4fde2c30cdacf873b664416f76a0c7f7cc50c9f72a3cb84be88144cde91250000000000000d80")
	consolidationData := common.FromHex("b917cfdc0d25b72d55cf94db328e1629b7f4fde2c30cdacf873b664416f76a0c7f7cc50c9f72a3cb84be88144cde9125b9812f7d0b1f2f969b52bbb2d316b0c2fa7c9dba85c428c5e6c27766bcc4b0c6e874702ff1eb1c7024b08524a9771601")

	for _, tc := range []struct {
		name string
		addr common.Address
		data []byte
	}{
		{"withdrawal queue (EIP-7002)", params.WithdrawalQueueAddress, withdrawalData},
		{"consolidation queue (EIP-7251)", params.ConsolidationQueueAddress, consolidationData},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := newBALTestEnv(nil)
			// A request transaction writes several new storage slots under
			// Amsterdam's state-gas schedule. Raise the test chain's gas limit so
			// all 17 requests fit in the first block.
			env.gspec.GasLimit = 200_000_000
			_, blocks, _ := GenerateChainWithGenesis(env.gspec, beacon.New(ethash.NewFaker()), 2, func(i int, g *BlockGen) {
				if i == 1 {
					// Make the post-execution system call occur after one ordinary
					// transaction, proving it uses n + 1 rather than a fixed index.
					to := common.HexToAddress("0xf00")
					g.AddTx(env.tx(17, &to, big.NewInt(0), 100_000, 0, nil))
					return
				}
				// The system call processes at most 16 requests per block. Leave one
				// payload for block 2 so its storage access is genuinely a read.
				for nonce := uint64(0); nonce < 17; nonce++ {
					g.AddTx(env.tx(nonce, &tc.addr, newGwei(1), 5_000_000, 0, tc.data))
				}
			})

			b := blocks[1].AccessList()
			if b == nil {
				t.Fatal("expected non-nil block access list")
			}
			aa := assertPresent(t, b, tc.addr)
			// Block 2 contains one user transaction, so its post-execution call is
			// index 2 (= n + 1).
			for slot := int64(0); slot < 4; slot++ {
				key := common.BigToHash(big.NewInt(slot))
				if hasSlotIn(aa.StorageReads, key) {
					continue // A metadata slot whose value is unchanged is read-only.
				}
				assertStorageChangeAt(t, aa, key, 2)
			}
			foundPayloadRead := false
			for _, slot := range aa.StorageReads {
				if slot.Uint64() >= 4 {
					foundPayloadRead = true
				}
			}
			if !foundPayloadRead {
				t.Fatalf("post-execution queue call must leave payload slots in storage_reads\n%s", b.PrettyPrint())
			}
			for _, change := range aa.StorageChanges {
				if change.Slot.Uint64() >= 4 {
					t.Fatalf("post-execution queue call must not write payload slot %s\n%s", change.Slot, b.PrettyPrint())
				}
			}
		})
	}
}

// ============================== Withdrawals ==============================

// TestBALWithdrawalZeroAmountIncluded: a withdrawal with amount 0 still puts
// the recipient in the BAL (with no balance entry).
func TestBALWithdrawalZeroAmountIncluded(t *testing.T) {
	recipient := common.HexToAddress("0xdada")
	env := newBALTestEnv(nil)

	b, _ := env.run(t, func(g *BlockGen) {
		g.SetCoinbase(common.Address{0xc0})
		g.AddWithdrawal(&types.Withdrawal{Validator: 1, Address: recipient, Amount: 0})
	})

	r := assertPresent(t, b, recipient)
	if len(r.BalanceChanges) != 0 {
		t.Fatalf("zero-amount withdrawal must not record balance: %+v", r.BalanceChanges)
	}
}

// TestBALWithdrawalNonZeroAmountRecordsBalance: a positive-amount withdrawal
// records a balance change for the recipient.
func TestBALWithdrawalNonZeroAmountRecordsBalance(t *testing.T) {
	recipient := common.HexToAddress("0xdada")
	env := newBALTestEnv(nil)

	b, _ := env.run(t, func(g *BlockGen) {
		g.SetCoinbase(common.Address{0xc0})
		g.AddWithdrawal(&types.Withdrawal{Validator: 1, Address: recipient, Amount: 7})
	})

	r := assertPresent(t, b, recipient)
	if len(r.BalanceChanges) != 1 || r.BalanceChanges[0].PostBalance.Sign() == 0 {
		t.Fatalf("withdrawal balance change missing: %+v", r.BalanceChanges)
	}
}

// ============================== EIP-7702 authority ==============================

// TestBALAuthorityIncludedOnSetCodeTx: the authority of an EIP-7702 set-code
// transaction is added to the BAL once its delegation is loaded, recording
// both the nonce bump and the delegation-pointer code entry.
func TestBALAuthorityIncludedOnSetCodeTx(t *testing.T) {
	env := newBALTestEnv(nil)
	authKey, _ := crypto.HexToECDSA("0202020202020202020202020202020202020202020202020202002020202020")
	authority := crypto.PubkeyToAddress(authKey.PublicKey)
	delegate := common.HexToAddress("0xdeadbeef")

	auth, err := types.SignSetCode(authKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(env.cfg.ChainID),
		Address: delegate,
		Nonce:   0,
	})
	if err != nil {
		t.Fatalf("sign auth: %v", err)
	}

	b, _ := env.run(t, func(g *BlockGen) {
		tx := types.MustSignNewTx(env.key, env.signer, &types.SetCodeTx{
			ChainID:   uint256.MustFromBig(env.cfg.ChainID),
			Nonce:     0,
			To:        env.from,
			Value:     new(uint256.Int),
			Gas:       1_000_000,
			GasFeeCap: uint256.NewInt(uint64(newGwei(10).Int64())),
			GasTipCap: new(uint256.Int),
			AuthList:  []types.SetCodeAuthorization{auth},
		})
		g.AddTx(tx)
	})

	aa := assertPresent(t, b, authority)
	if len(aa.NonceChanges) == 0 {
		t.Fatalf("authority nonce should be bumped by delegation: %+v", aa.NonceChanges)
	}
	if len(aa.CodeChanges) == 0 {
		t.Fatalf("authority code (delegation pointer) should be recorded: %+v", aa.CodeChanges)
	}
}

// TestBALDelegationTargetNotIncludedOnAuthOnly: the EIP-7702 delegation target
// MUST NOT appear in the BAL when only the authorization is installed and the
// target is never loaded as an execution target.
func TestBALDelegationTargetNotIncludedOnAuthOnly(t *testing.T) {
	env := newBALTestEnv(nil)
	authKey, _ := crypto.HexToECDSA("0202020202020202020202020202020202020202020202020202002020202020")
	delegate := common.HexToAddress("0xdeadbeef") // never accessed

	auth, err := types.SignSetCode(authKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(env.cfg.ChainID),
		Address: delegate,
		Nonce:   0,
	})
	if err != nil {
		t.Fatalf("sign auth: %v", err)
	}

	b, _ := env.run(t, func(g *BlockGen) {
		tx := types.MustSignNewTx(env.key, env.signer, &types.SetCodeTx{
			ChainID:   uint256.MustFromBig(env.cfg.ChainID),
			Nonce:     0,
			To:        env.from, // tx.to is an EOA with no code: delegate is never called
			Value:     new(uint256.Int),
			Gas:       1_000_000,
			GasFeeCap: uint256.NewInt(uint64(newGwei(10).Int64())),
			GasTipCap: new(uint256.Int),
			AuthList:  []types.SetCodeAuthorization{auth},
		})
		g.AddTx(tx)
	})

	assertAbsent(t, b, delegate)
}

// newSetCodeTx is a small constructor used by the multi-auth tests below.
func (e *balTestEnv) newSetCodeTx(t *testing.T, nonce uint64, to common.Address, auths []types.SetCodeAuthorization) *types.Transaction {
	t.Helper()
	tx, err := types.SignTx(types.NewTx(&types.SetCodeTx{
		ChainID:   uint256.MustFromBig(e.cfg.ChainID),
		Nonce:     nonce,
		To:        to,
		Value:     new(uint256.Int),
		Gas:       1_000_000,
		GasFeeCap: uint256.NewInt(uint64(newGwei(10).Int64())),
		GasTipCap: new(uint256.Int),
		AuthList:  auths,
	}), e.signer, e.key)
	if err != nil {
		t.Fatalf("sign SetCodeTx: %v", err)
	}
	return tx
}

// TestBALAuthFailedBeforeLoadExcluded: an EIP-7702 auth whose ChainID check
// fails returns before the authority is loaded, so the authority address
// MUST NOT appear in the BAL.
func TestBALAuthFailedBeforeLoadExcluded(t *testing.T) {
	env := newBALTestEnv(nil)
	authKey, _ := crypto.HexToECDSA("0202020202020202020202020202020202020202020202020202002020202020")
	authority := crypto.PubkeyToAddress(authKey.PublicKey)

	auth, err := types.SignSetCode(authKey, types.SetCodeAuthorization{
		ChainID: *uint256.NewInt(999), // wrong chain → fails ChainID check (pre-load)
		Address: common.HexToAddress("0xdeadbeef"),
		Nonce:   0,
	})
	if err != nil {
		t.Fatalf("sign auth: %v", err)
	}

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.newSetCodeTx(t, 0, env.from, []types.SetCodeAuthorization{auth}))
	})

	assertAbsent(t, b, authority)
}

// TestBALAuthFailedAfterLoadEmptyChangeSet: an EIP-7702 auth that fails the
// nonce check happens AFTER the authority's code is loaded (and the address
// added to accessed_addresses), so the authority MUST appear in the BAL —
// but with no nonce or code change.
func TestBALAuthFailedAfterLoadEmptyChangeSet(t *testing.T) {
	env := newBALTestEnv(nil)
	authKey, _ := crypto.HexToECDSA("0202020202020202020202020202020202020202020202020202002020202020")
	authority := crypto.PubkeyToAddress(authKey.PublicKey)

	// The authority's actual nonce is 0; supplying auth.Nonce=99 makes
	// validation fail only after the code has been loaded.
	auth, err := types.SignSetCode(authKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(env.cfg.ChainID),
		Address: common.HexToAddress("0xdeadbeef"),
		Nonce:   99,
	})
	if err != nil {
		t.Fatalf("sign auth: %v", err)
	}

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.newSetCodeTx(t, 0, env.from, []types.SetCodeAuthorization{auth}))
	})

	aa := assertPresent(t, b, authority)
	if len(aa.NonceChanges) != 0 {
		t.Fatalf("failed auth must not bump nonce: %+v", aa.NonceChanges)
	}
	if len(aa.CodeChanges) != 0 {
		t.Fatalf("failed auth must not record a code change: %+v", aa.CodeChanges)
	}
}

// TestBALAuthOOGRecipientExcluded covers the exceptional halt boundary where
// an EIP-7702 authorization exhausts runtime gas before the transaction's
// recipient is loaded. The authority was already loaded by authorization
// validation, but tx.to must not enter the BAL.
func TestBALAuthOOGRecipientExcluded(t *testing.T) {
	authKey, _ := crypto.HexToECDSA("0202020202020202020202020202020202020202020202020202002020202020")
	authority := crypto.PubkeyToAddress(authKey.PublicKey)
	recipient := common.HexToAddress("0xdec1a1")
	implementation := common.HexToAddress("0x1a11")
	env := newBALTestEnv(nil)
	auth, err := types.SignSetCode(authKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(env.cfg.ChainID),
		Address: implementation,
		Nonce:   0,
	})
	if err != nil {
		t.Fatalf("sign auth: %v", err)
	}

	b, receipts := env.run(t, func(g *BlockGen) {
		tx, err := types.SignTx(types.NewTx(&types.SetCodeTx{
			ChainID:   uint256.MustFromBig(env.cfg.ChainID),
			Nonce:     0,
			To:        recipient,
			Value:     new(uint256.Int),
			Gas:       30_000,
			GasFeeCap: uint256.NewInt(uint64(newGwei(10).Int64())),
			GasTipCap: new(uint256.Int),
			AuthList:  []types.SetCodeAuthorization{auth},
		}), env.signer, env.key)
		if err != nil {
			t.Fatalf("sign SetCodeTx: %v", err)
		}
		g.AddTx(tx)
	})
	if receipts[0].Status != types.ReceiptStatusFailed {
		t.Fatalf("expected authorization runtime OOG, have status %d", receipts[0].Status)
	}
	assertEmpty(t, assertPresent(t, b, authority))
	assertAbsent(t, b, recipient)
	assertAbsent(t, b, implementation)
}

// TestBALMultipleAuthsOnlyLoadedIncluded: a SetCode tx with a mix of valid and
// pre-load-failed auths lists only the loaded authorities in the BAL.
func TestBALMultipleAuthsOnlyLoadedIncluded(t *testing.T) {
	env := newBALTestEnv(nil)
	goodKey, _ := crypto.HexToECDSA("0202020202020202020202020202020202020202020202020202002020202020")
	badKey, _ := crypto.HexToECDSA("0303030303030303030303030303030303030303030303030303003030303030")
	good := crypto.PubkeyToAddress(goodKey.PublicKey)
	bad := crypto.PubkeyToAddress(badKey.PublicKey)
	delegate := common.HexToAddress("0xdeadbeef")

	goodAuth, err := types.SignSetCode(goodKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(env.cfg.ChainID),
		Address: delegate,
		Nonce:   0,
	})
	if err != nil {
		t.Fatalf("sign good auth: %v", err)
	}
	badAuth, err := types.SignSetCode(badKey, types.SetCodeAuthorization{
		ChainID: *uint256.NewInt(999), // fails before load
		Address: delegate,
		Nonce:   0,
	})
	if err != nil {
		t.Fatalf("sign bad auth: %v", err)
	}

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.newSetCodeTx(t, 0, env.from, []types.SetCodeAuthorization{goodAuth, badAuth}))
	})

	assertPresent(t, b, good) // loaded → in BAL
	assertAbsent(t, b, bad)   // never loaded → not in BAL
}

// TestBALAuthCodeRoundTripNoCodeEntry: two auths on the same authority that
// (1) install a delegation and (2) clear it again. Final code equals pre-tx
// code (empty), so the BAL records only the cumulative nonce bump and NO
// code change.
func TestBALAuthCodeRoundTripNoCodeEntry(t *testing.T) {
	env := newBALTestEnv(nil)
	authKey, _ := crypto.HexToECDSA("0202020202020202020202020202020202020202020202020202002020202020")
	authority := crypto.PubkeyToAddress(authKey.PublicKey)
	delegateA := common.HexToAddress("0xa11ce")

	auth1, err := types.SignSetCode(authKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(env.cfg.ChainID),
		Address: delegateA, // empty → A
		Nonce:   0,
	})
	if err != nil {
		t.Fatalf("sign auth1: %v", err)
	}
	auth2, err := types.SignSetCode(authKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(env.cfg.ChainID),
		Address: common.Address{}, // delegation to zero clears the code (A → empty)
		Nonce:   1,
	})
	if err != nil {
		t.Fatalf("sign auth2: %v", err)
	}

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.newSetCodeTx(t, 0, env.from, []types.SetCodeAuthorization{auth1, auth2}))
	})

	aa := assertPresent(t, b, authority)
	if len(aa.NonceChanges) != 1 || aa.NonceChanges[0].PostNonce != 2 {
		t.Fatalf("expected final nonce 2, got %+v", aa.NonceChanges)
	}
	if len(aa.CodeChanges) != 0 {
		t.Fatalf("code round-trip (empty→A→empty) must NOT record a code change: %+v", aa.CodeChanges)
	}
}

// TestBALAuthCodeOverwrittenFinalRecorded: two auths on the same authority
// switching delegation A → B record exactly one code change carrying the
// final delegation pointer (B), not the intermediate value.
func TestBALAuthCodeOverwrittenFinalRecorded(t *testing.T) {
	env := newBALTestEnv(nil)
	authKey, _ := crypto.HexToECDSA("0202020202020202020202020202020202020202020202020202002020202020")
	authority := crypto.PubkeyToAddress(authKey.PublicKey)
	delegateA := common.HexToAddress("0xa11ce")
	delegateB := common.HexToAddress("0xb0b0b0")

	auth1, err := types.SignSetCode(authKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(env.cfg.ChainID),
		Address: delegateA,
		Nonce:   0,
	})
	if err != nil {
		t.Fatalf("sign auth1: %v", err)
	}
	auth2, err := types.SignSetCode(authKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(env.cfg.ChainID),
		Address: delegateB,
		Nonce:   1,
	})
	if err != nil {
		t.Fatalf("sign auth2: %v", err)
	}

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.newSetCodeTx(t, 0, env.from, []types.SetCodeAuthorization{auth1, auth2}))
	})

	aa := assertPresent(t, b, authority)
	if len(aa.CodeChanges) != 1 {
		t.Fatalf("expected exactly 1 code change (final), got %+v", aa.CodeChanges)
	}
	want := types.AddressToDelegation(delegateB)
	if !bytes.Equal(aa.CodeChanges[0].NewCode, want) {
		t.Fatalf("final code mismatch: want %x, got %x", want, aa.CodeChanges[0].NewCode)
	}
	if len(aa.NonceChanges) != 1 || aa.NonceChanges[0].PostNonce != 2 {
		t.Fatalf("expected final nonce 2, got %+v", aa.NonceChanges)
	}
}

// TestBALReapplyDelegation covers an authorization whose final code equals its
// pre-transaction code. The authority nonce changes, but the unchanged
// delegation must not create a code change or implementation read.
func TestBALReapplyDelegation(t *testing.T) {
	authKey, _ := crypto.HexToECDSA("0202020202020202020202020202020202020202020202020202002020202020")
	authority := crypto.PubkeyToAddress(authKey.PublicKey)
	implementation := common.HexToAddress("0x1a11")
	env := newBALTestEnv(types.GenesisAlloc{
		authority:      {Nonce: 1, Code: types.AddressToDelegation(implementation), Balance: common.Big0},
		implementation: {Code: []byte{0x00}, Balance: common.Big0},
	})
	auth, err := types.SignSetCode(authKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(env.cfg.ChainID),
		Address: implementation,
		Nonce:   1,
	})
	if err != nil {
		t.Fatalf("sign auth: %v", err)
	}

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.newSetCodeTx(t, 0, env.from, []types.SetCodeAuthorization{auth}))
	})

	aa := assertPresent(t, b, authority)
	if len(aa.NonceChanges) != 1 || aa.NonceChanges[0].PostNonce != 2 {
		t.Fatalf("reapplied delegation nonce: %+v", aa.NonceChanges)
	}
	if len(aa.CodeChanges) != 0 {
		t.Fatalf("reapplying unchanged delegation must not record code: %+v", aa.CodeChanges)
	}
	assertAbsent(t, b, implementation)
}

// TestBALSetCodeTxDelegatedRecipient verifies that authorizations are applied
// before the top-level call. A newly delegated tx.to must immediately load and
// execute its implementation.
func TestBALSetCodeTxDelegatedRecipient(t *testing.T) {
	authKey, _ := crypto.HexToECDSA("0202020202020202020202020202020202020202020202020202002020202020")
	authority := crypto.PubkeyToAddress(authKey.PublicKey)
	implementation := common.HexToAddress("0x1a11")
	slot := common.BigToHash(big.NewInt(0x07))
	implementationCode := []byte{0x60, 0x07, 0x54, 0x50, 0x00} // SLOAD(7); POP; STOP.
	env := newBALTestEnv(types.GenesisAlloc{
		implementation: {Code: implementationCode, Balance: common.Big0},
	})
	auth, err := types.SignSetCode(authKey, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(env.cfg.ChainID),
		Address: implementation,
		Nonce:   0,
	})
	if err != nil {
		t.Fatalf("sign auth: %v", err)
	}

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.newSetCodeTx(t, 0, authority, []types.SetCodeAuthorization{auth}))
	})

	authorityAccess := assertPresent(t, b, authority)
	if len(authorityAccess.NonceChanges) != 1 || authorityAccess.NonceChanges[0].PostNonce != 1 {
		t.Fatalf("authority nonce after installation: %+v", authorityAccess.NonceChanges)
	}
	if len(authorityAccess.CodeChanges) != 1 || !bytes.Equal(authorityAccess.CodeChanges[0].NewCode, types.AddressToDelegation(implementation)) {
		t.Fatalf("authority delegation code after installation: %+v", authorityAccess.CodeChanges)
	}
	if !hasSlotIn(authorityAccess.StorageReads, slot) {
		t.Fatalf("same-transaction delegated execution must read authority storage\n%s", b.PrettyPrint())
	}
	assertEmpty(t, assertPresent(t, b, implementation))
}

// ============================== Read-list regression cases ==============================

// TestBALStorageWriteThenRead covers the opposite ordering of
// TestBALStorageReadThenWriteOnlyInWrites. Once an index changes a slot, all
// subsequent reads of that slot must be hidden by storage_changes.
func TestBALStorageWriteThenRead(t *testing.T) {
	contract := common.HexToAddress("0xc1")
	slot := common.BigToHash(big.NewInt(0x05))
	// SSTORE(0x05, 0x42); SLOAD(0x05); POP; STOP.
	code := []byte{0x60, 0x42, 0x60, 0x05, 0x55, 0x60, 0x05, 0x54, 0x50, 0x00}
	env := newBALTestEnv(types.GenesisAlloc{contract: {Code: code, Balance: common.Big0}})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &contract, big.NewInt(0), 1_000_000, 0, nil))
	})

	aa := assertPresent(t, b, contract)
	assertStorageChanges(t, aa, slot, []balStorageChangeExpectation{{index: 1, value: common.BigToHash(big.NewInt(0x42))}})
	if hasSlotIn(aa.StorageReads, slot) {
		t.Fatalf("written slot %x must not remain in storage_reads\n%s", slot, b.PrettyPrint())
	}
}

// TestBALStorageNoOpAfterWrite checks that the no-op decision
// is relative to the state before the current block-access index. Tx2 writes
// the value that Tx1 already committed, so it must not replace Tx1's change
// with a read entry.
func TestBALStorageNoOpAfterWrite(t *testing.T) {
	contract := common.HexToAddress("0xc1")
	slot := common.BigToHash(big.NewInt(0x05))
	// SSTORE(0x05, 0x42); STOP.
	code := []byte{0x60, 0x42, 0x60, 0x05, 0x55, 0x00}
	env := newBALTestEnv(types.GenesisAlloc{contract: {Code: code, Balance: common.Big0}})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &contract, big.NewInt(0), 1_000_000, 0, nil))
		g.AddTx(env.tx(1, &contract, big.NewInt(0), 1_000_000, 0, nil))
	})

	aa := assertPresent(t, b, contract)
	assertStorageChanges(t, aa, slot, []balStorageChangeExpectation{{index: 1, value: common.BigToHash(big.NewInt(0x42))}})
	if hasSlotIn(aa.StorageReads, slot) {
		t.Fatalf("later no-op must not add slot %x to storage_reads\n%s", slot, b.PrettyPrint())
	}
}

// TestBALStorageChangesAcrossTxs verifies that a slot changed at two
// distinct transaction indices retains both post-index values, including a
// later reset to its pre-block value.
func TestBALStorageChangesAcrossTxs(t *testing.T) {
	contract := common.HexToAddress("0xc1")
	slot := common.BigToHash(big.NewInt(0x05))
	// CALLDATALOAD(0); SSTORE(0x05, value); STOP.
	code := []byte{0x60, 0x00, 0x35, 0x60, 0x05, 0x55, 0x00}
	env := newBALTestEnv(types.GenesisAlloc{contract: {Code: code, Balance: common.Big0}})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &contract, big.NewInt(0), 1_000_000, 0, common.LeftPadBytes([]byte{0x42}, 32)))
		g.AddTx(env.tx(1, &contract, big.NewInt(0), 1_000_000, 0, make([]byte, 32)))
	})

	aa := assertPresent(t, b, contract)
	assertStorageChanges(t, aa, slot, []balStorageChangeExpectation{
		{index: 1, value: common.BigToHash(big.NewInt(0x42))},
		{index: 2, value: common.Hash{}},
	})
	if hasSlotIn(aa.StorageReads, slot) {
		t.Fatalf("slot with committed changes must not be in storage_reads\n%s", b.PrettyPrint())
	}
}

// TestBALRevertedSStoreRead verifies that SSTORE performs an implicit read once
// its access boundary is crossed. The later REVERT discards the write but not
// that read footprint.
func TestBALRevertedSStoreRead(t *testing.T) {
	contract := common.HexToAddress("0xc1")
	slot := common.BigToHash(big.NewInt(0x05))
	// SSTORE(0x05, 0x42); REVERT(0, 0).
	code := []byte{0x60, 0x42, 0x60, 0x05, 0x55, 0x60, 0x00, 0x60, 0x00, 0xfd}
	env := newBALTestEnv(types.GenesisAlloc{contract: {Code: code, Balance: common.Big0}})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &contract, big.NewInt(0), 1_000_000, 0, nil))
	})

	aa := assertPresent(t, b, contract)
	if !hasSlotIn(aa.StorageReads, slot) {
		t.Fatalf("reverted SSTORE must leave slot %x in storage_reads\n%s", slot, b.PrettyPrint())
	}
	if hasStorageWrite(b, contract, slot) {
		t.Fatalf("reverted SSTORE must not leave a storage change\n%s", b.PrettyPrint())
	}
}

// TestBALParentRevertSStoreRead checks the journal boundary between call frames:
// a child returns successfully after SSTORE, then its parent reverts the whole
// call tree. The child's slot is still a BAL read.
func TestBALParentRevertSStoreRead(t *testing.T) {
	child := common.HexToAddress("0xc1")
	parent := common.HexToAddress("0xc2")
	slot := common.BigToHash(big.NewInt(0x05))
	childCode := []byte{0x60, 0x42, 0x60, 0x05, 0x55, 0x00}
	parentCode := makeStubCaller(0xf1 /* CALL */, child)
	parentCode = append(parentCode[:len(parentCode)-1], 0x60, 0x00, 0x60, 0x00, 0xfd)
	env := newBALTestEnv(types.GenesisAlloc{
		child:  {Code: childCode, Balance: common.Big0},
		parent: {Code: parentCode, Balance: common.Big0},
	})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &parent, big.NewInt(0), 1_000_000, 0, nil))
	})

	aa := assertPresent(t, b, child)
	if !hasSlotIn(aa.StorageReads, slot) {
		t.Fatalf("parent-reverted child write must leave slot %x in storage_reads\n%s", slot, b.PrettyPrint())
	}
	if hasStorageWrite(b, child, slot) {
		t.Fatalf("parent-reverted child write must not leave a storage change\n%s", b.PrettyPrint())
	}
}

// TestBALStorageReadsSorted exercises the final encoding of a read-only set.
// Repeated reads must produce one key, and keys are ordered lexicographically
// regardless of execution order.
func TestBALStorageReadsSorted(t *testing.T) {
	contract := common.HexToAddress("0xc1")
	// SLOAD(9); SLOAD(1); SLOAD(9); STOP.
	code := []byte{
		0x60, 0x09, 0x54, 0x50,
		0x60, 0x01, 0x54, 0x50,
		0x60, 0x09, 0x54, 0x50,
		0x00,
	}
	env := newBALTestEnv(types.GenesisAlloc{contract: {Code: code, Balance: common.Big0}})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &contract, big.NewInt(0), 1_000_000, 0, nil))
	})

	aa := assertPresent(t, b, contract)
	if len(aa.StorageReads) != 2 || aa.StorageReads[0].Uint64() != 1 || aa.StorageReads[1].Uint64() != 9 {
		t.Fatalf("storage_reads must be deduplicated and sorted: %+v", aa.StorageReads)
	}
}

// TestBALAccessListSlotExcluded ensures an EIP-2930 storage-key warming entry
// changes gas only. It must not create a storage_reads entry unless the EVM
// actually executes an access to that slot.
func TestBALAccessListSlotExcluded(t *testing.T) {
	contract := common.HexToAddress("0xc1")
	slot := common.BigToHash(big.NewInt(0x07))
	env := newBALTestEnv(types.GenesisAlloc{contract: {Code: []byte{0x00}, Balance: common.Big0}})

	b, _ := env.run(t, func(g *BlockGen) {
		tx := types.MustSignNewTx(env.key, env.signer, &types.DynamicFeeTx{
			ChainID:   env.cfg.ChainID,
			Nonce:     0,
			To:        &contract,
			Value:     new(big.Int),
			Gas:       1_000_000,
			GasFeeCap: newGwei(10),
			GasTipCap: new(big.Int),
			AccessList: types.AccessList{{
				Address:     contract,
				StorageKeys: []common.Hash{slot},
			}},
		})
		g.AddTx(tx)
	})

	aa := assertPresent(t, b, contract)
	if hasSlotIn(aa.StorageReads, slot) || hasStorageWrite(b, contract, slot) {
		t.Fatalf("untouched access-list slot %x must be absent from BAL\n%s", slot, b.PrettyPrint())
	}
}

// ============================== EIP-7702 execution-time delegation ==============================

func makeDelegationProbe(op byte, target common.Address) []byte {
	if op == 0x3c { // EXTCODECOPY needs length, code offset and memory offset.
		code := []byte{0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x73}
		code = append(code, target.Bytes()...)
		return append(code, op, 0x00)
	}
	code := append([]byte{0x73}, target.Bytes()...)
	return append(code, op, 0x50, 0x00) // opcode, POP, STOP
}

// TestBALDelegationInspection ensures the code-reading opcodes observe an
// authority's 7702 indicator as its own code. They access the authority
// but MUST NOT load the implementation address.
func TestBALDelegationInspection(t *testing.T) {
	authority := common.HexToAddress("0xa7702")
	implementation := common.HexToAddress("0x1a11")
	for _, tc := range []struct {
		name string
		op   byte
	}{
		{"balance", 0x31},
		{"extcodesize", 0x3b},
		{"extcodecopy", 0x3c},
		{"extcodehash", 0x3f},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caller := common.HexToAddress("0xca11")
			env := newBALTestEnv(types.GenesisAlloc{
				caller:         {Code: makeDelegationProbe(tc.op, authority), Balance: common.Big0},
				authority:      {Code: types.AddressToDelegation(implementation), Balance: common.Big0},
				implementation: {Code: []byte{0x00}, Balance: common.Big0},
			})
			b, _ := env.run(t, func(g *BlockGen) {
				g.AddTx(env.tx(0, &caller, big.NewInt(0), 1_000_000, 0, nil))
			})
			assertEmpty(t, assertPresent(t, b, authority))
			assertAbsent(t, b, implementation)
		})
	}
}

// TestBALDelegatedCallStorage checks CALL's storage context. The implementation
// code is fetched from implementation, but its SLOAD executes in the delegated
// authority's storage, not implementation's.
func TestBALDelegatedCallStorage(t *testing.T) {
	authority := common.HexToAddress("0xa7702")
	implementation := common.HexToAddress("0x1a11")
	slot := common.BigToHash(big.NewInt(0x07))
	implCode := []byte{0x60, 0x07, 0x54, 0x50, 0x00} // SLOAD(7), POP, STOP
	env := newBALTestEnv(types.GenesisAlloc{
		authority:      {Code: types.AddressToDelegation(implementation), Balance: common.Big0},
		implementation: {Code: implCode, Balance: common.Big0},
	})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &authority, big.NewInt(0), 1_000_000, 0, nil))
	})

	aa := assertPresent(t, b, authority)
	if !hasSlotIn(aa.StorageReads, slot) {
		t.Fatalf("delegated CALL read must belong to authority\n%s", b.PrettyPrint())
	}
	assertEmpty(t, assertPresent(t, b, implementation))
}

// TestBALDelegatedCallFamilyStorage checks the storage context of the CALL
// family against a delegated authority: the resolved implementation supplies
// code only, while the storage read belongs to the executing context — the
// authority for CALL/STATICCALL, the caller for DELEGATECALL/CALLCODE.
func TestBALDelegatedCallFamilyStorage(t *testing.T) {
	caller := common.HexToAddress("0xca11")
	authority := common.HexToAddress("0xa7702")
	implementation := common.HexToAddress("0x1a11")
	slot := common.BigToHash(big.NewInt(0x07))
	implCode := []byte{0x60, 0x07, 0x54, 0x50, 0x00} // SLOAD(7), POP, STOP
	cases := []struct {
		name      string
		op        byte
		readOwner common.Address
	}{
		{"call", 0xf1, authority},
		{"callcode", 0xf2, caller},
		{"delegatecall", 0xf4, caller},
		{"staticcall", 0xfa, authority},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := newBALTestEnv(types.GenesisAlloc{
				caller:         {Code: makeStubCaller(tc.op, authority), Balance: common.Big0},
				authority:      {Code: types.AddressToDelegation(implementation), Balance: common.Big0},
				implementation: {Code: implCode, Balance: common.Big0},
			})

			b, _ := env.run(t, func(g *BlockGen) {
				g.AddTx(env.tx(0, &caller, big.NewInt(0), 1_000_000, 0, nil))
			})

			owner := assertPresent(t, b, tc.readOwner)
			if !hasSlotIn(owner.StorageReads, slot) {
				t.Fatalf("%s read must belong to %x\n%s", tc.name, tc.readOwner, b.PrettyPrint())
			}
			for _, other := range []common.Address{caller, authority} {
				if other != tc.readOwner {
					assertEmpty(t, assertPresent(t, b, other))
				}
			}
			assertEmpty(t, assertPresent(t, b, implementation))
		})
	}
}

// TestBALDelegationOneHop verifies that resolving A -> B does not
// recursively resolve B -> C. A and B are state accesses; C is not.
func TestBALDelegationOneHop(t *testing.T) {
	caller := common.HexToAddress("0xca11")
	authority := common.HexToAddress("0xa7702")
	delegated := common.HexToAddress("0xb7702")
	secondHop := common.HexToAddress("0xc7702")
	env := newBALTestEnv(types.GenesisAlloc{
		caller:    {Code: makeStubCaller(0xf1 /* CALL */, authority), Balance: common.Big0},
		authority: {Code: types.AddressToDelegation(delegated), Balance: common.Big0},
		delegated: {Code: types.AddressToDelegation(secondHop), Balance: common.Big0},
		secondHop: {Code: []byte{0x00}, Balance: common.Big0},
	})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &caller, big.NewInt(0), 1_000_000, 0, nil))
	})

	assertPresent(t, b, authority)
	assertPresent(t, b, delegated)
	assertAbsent(t, b, secondHop)
}

// TestBALDelegatedSender distinguishes transaction origination from execution.
// A sender may carry a valid delegation indicator, but its implementation is
// not accessed unless a call actually targets sender.
func TestBALDelegatedSender(t *testing.T) {
	to := common.HexToAddress("0xb0b")
	implementation := common.HexToAddress("0x1a11")
	env := newBALTestEnv(types.GenesisAlloc{
		implementation: {Code: []byte{0x00}, Balance: common.Big0},
	})
	sender := env.gspec.Alloc[env.from]
	sender.Code = types.AddressToDelegation(implementation)
	env.gspec.Alloc[env.from] = sender

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &to, big.NewInt(0), params.TxGas, 0, nil))
	})

	assertPresent(t, b, env.from)
	assertAbsent(t, b, implementation)
}
