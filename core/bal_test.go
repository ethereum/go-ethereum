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
	"github.com/ethereum/go-ethereum/core/rawdb"
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
	blob := *cfg.BlobScheduleConfig
	blob.Amsterdam = blob.Osaka
	cfg.BlobScheduleConfig = &blob
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

// TestBALCoinbaseTipCapturesBalance: positive priority fee credits coinbase
// and the balance change appears in the BAL.
func TestBALCoinbaseTipCapturesBalance(t *testing.T) {
	coinbase := common.Address{0xc0}
	to := common.HexToAddress("0xabba")
	env := newBALTestEnv(nil)

	b, _ := env.run(t, func(g *BlockGen) {
		g.SetCoinbase(coinbase)
		g.AddTx(env.tx(0, &to, big.NewInt(0), params.TxGas, 2 /* gwei tip */, nil))
	})

	cb := assertPresent(t, b, coinbase)
	if len(cb.BalanceChanges) == 0 || cb.BalanceChanges[0].PostBalance.Sign() == 0 {
		t.Fatalf("coinbase missing positive balance change: %+v", cb.BalanceChanges)
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

// ============================== Revert behaviour ==============================

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

// ============================== Storage inclusion ==============================

// TestBALStorageWriteRecorded: SSTORE places the slot in storage_changes and
// keeps it out of storage_reads.
func TestBALStorageWriteRecorded(t *testing.T) {
	contract := common.HexToAddress("0xc1")
	slot := common.BigToHash(big.NewInt(0x01))
	// PUSH1 0x42 PUSH1 0x01 SSTORE STOP
	code := []byte{0x60, 0x42, 0x60, 0x01, 0x55, 0x00}
	env := newBALTestEnv(types.GenesisAlloc{contract: {Code: code, Balance: common.Big0}})

	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.tx(0, &contract, big.NewInt(0), 1_000_000, 0, nil))
	})

	aa := assertPresent(t, b, contract)
	if !hasStorageWrite(b, contract, slot) {
		t.Fatalf("expected slot 0x01 in storage_changes\n%s", b.PrettyPrint())
	}
	if hasSlotIn(aa.StorageReads, slot) {
		t.Fatalf("slot 0x01 must NOT appear in storage_reads")
	}
}

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

// ============================== Mid-tx balance round-trip ==============================

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

// ============================== System contracts (pre/post-execution) ==============================

// TestBALSystemContractsPresent: per EIP-7928, "System contract addresses
// accessed during pre/post-execution" MUST be included in the BAL. That
// means all four of the post-merge system contracts touched by every
// Amsterdam block:
//
//   - EIP-4788 beacon roots         (pre-execution, when ParentBeaconRoot is set)
//   - EIP-2935 history storage      (pre-execution)
//   - EIP-7002 withdrawal queue     (post-execution)
//   - EIP-7251 consolidation queue  (post-execution)
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
	} {
		if findAccount(b, sys.addr) == nil {
			t.Errorf("%s (%x) MUST appear in BAL but is missing\n%s", sys.name, sys.addr, b.PrettyPrint())
		}
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

// ============================== Preimages ==============================

// TestBALPreimages tests that preimage tracking works when executing a block
// with an access list.
func TestBALPreimages(t *testing.T) {
	// Runtime code: store 0x42 at memory[0], then KECCAK256 over memory[0:1].
	//   PUSH1 0x42 ; PUSH1 0x00 ; MSTORE8 ; PUSH1 0x01 ; PUSH1 0x00 ; KECCAK256 ; POP ; STOP
	contract := common.HexToAddress("0xca11ee")
	runtime := []byte{0x60, 0x42, 0x60, 0x00, 0x53, 0x60, 0x01, 0x60, 0x00, 0x20, 0x50, 0x00}

	env := newBALTestEnv(types.GenesisAlloc{
		contract: {Code: runtime, Balance: common.Big0},
	})

	engine := beacon.New(ethash.NewFaker())
	_, blocks, _ := GenerateChainWithGenesis(env.gspec, engine, 1, func(_ int, g *BlockGen) {
		// Run the EIP-4788 beacon-root system call during generation so the
		// generated BAL matches what the processor recomputes on import.
		g.SetParentBeaconRoot(common.Hash{})
		g.AddTx(env.tx(0, &contract, big.NewInt(0), 200_000, 0, nil))
	})
	// Import the block through the parallel BAL processor with preimage
	// recording enabled, so the KECCAK256 above is captured.
	db := rawdb.NewMemoryDatabase()
	cfg := DefaultConfig()
	cfg.VmConfig.EnablePreimageRecording = true
	chain, err := NewBlockChain(db, env.gspec, engine, cfg)
	if err != nil {
		t.Fatalf("new blockchain: %v", err)
	}
	defer chain.Stop()

	if _, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("insert chain: %v", err)
	}

	// The transaction hashed the single byte 0x42; the preimage must have been
	// accumulated by the state transition and written to disk on commit.
	want := []byte{0x42}
	hash := crypto.Keccak256Hash(want)
	if got := rawdb.ReadPreimage(db, hash); !bytes.Equal(got, want) {
		t.Fatalf("preimage for %x: got %x, want %x", hash, got, want)
	}
}
