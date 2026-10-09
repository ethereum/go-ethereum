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

package framepool

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func rootFrame(slot uint64, count int) types.Frame {
	data := make([]byte, count*recentRootTupleBytes)
	for i := range count {
		tuple := data[i*recentRootTupleBytes : (i+1)*recentRootTupleBytes]
		tuple[31] = byte(i + 1)
		binary.BigEndian.PutUint64(tuple[32:40], slot)
		tuple[71] = byte(i + 2)
	}
	return types.Frame{Mode: types.ModeVerify, Target: &params.RecentRootAddress, Value: new(uint256.Int), GasLimits: types.Limits{Execution: 50_000}, Data: data}
}

func installRoots(s *state.StateDB, frame types.Frame) {
	s.SetCode(params.RecentRootAddress, params.RecentRootCode, tracing.CodeChangeUnspecified)
	for offset := 0; offset < len(frame.Data); offset += recentRootTupleBytes {
		key, entry := recentRootDependency(frame.Data[offset : offset+recentRootTupleBytes])
		s.SetState(params.RecentRootAddress, key, entry)
	}
}

// setInitialSlot changes only the synthetic initial header, before admissions or
// descendants exist. Timestamps deliberately remain independent of slots.
func setInitialSlot(p *FramePool, c *testBlockChain, slot uint64) {
	c.lock.Lock()
	defer c.lock.Unlock()
	head := types.CopyHeader(c.head)
	head.SlotNumber = &slot
	delete(c.blocks, c.head.Hash())
	c.head = head
	c.blocks[head.Hash()] = types.NewBlockWithHeader(head)
	p.head.Store(head)
}

func TestRecentRootReferenceVector(t *testing.T) {
	tuple := common.FromHex("b9382d35273c75a50631a3e84d3c75ec9266e2b18c35a627e16cdbf26a18ca8500000000000000010000000000000000000000000000000000000000000000000000000000000002")
	key, entry := recentRootDependency(tuple)
	if key != common.HexToHash("5f027aa1cbe2df279bf6518edd4b44ea5409fd800189ec35224e10ab05e574c3") || entry != common.HexToHash("0a0d1254c851be5a133b4c9a9e300f5602fc0f43dbe65aa6a66930d4ca0a51b8") {
		t.Fatalf("reference vector: key %s entry %s", key, entry)
	}
	frame := rootFrame(1, 1)
	frame.Data = tuple
	pool, chain, _ := setupFramePool(t, 10, func(s *state.StateDB) { installRoots(s, frame) })
	setInitialSlot(pool, chain, 1)
	tx := signedPoolTx(t, 1, 0, 0, 20, 2, nil, func(tx *types.FrameTx) { tx.Frames = append([]types.Frame{frame}, tx.Frames...) })
	addPoolTx(t, pool, tx, nil)
	assertFramePoolConsistent(t, pool)
}

func TestRecentRootPrefixShapes(t *testing.T) {
	sender, sponsor := simulationSender, simulationHelper
	self := verifyFrame(types.ApproveExecutionAndPayment)
	self.GasLimits.Execution = 10_000
	only, pay := verifyFrame(types.ApproveExecution), verifyFrame(types.ApprovePayment)
	only.GasLimits.Execution, pay.GasLimits.Execution = 10_000, 10_000
	pay.Target = &sponsor
	deploy := types.Frame{Mode: types.ModeDefault}
	expiry := types.Frame{Mode: types.ModeVerify, Target: &params.FrameTxExpiryVerifier, Data: make([]byte, 8)}
	root := rootFrame(1, 1)
	for i, shape := range [][]types.Frame{{self}, {deploy, self}, {only, pay}, {deploy, only, pay}} {
		for _, withExpiry := range []bool{false, true} {
			for _, withRoot := range []bool{false, true} {
				t.Run(fmt.Sprintf("shape%d/expiry%t/root%t", i, withExpiry, withRoot), func(t *testing.T) {
					var frames []types.Frame
					if withExpiry {
						frames = append(frames, expiry)
					}
					rootIndex := -1
					if withRoot {
						rootIndex = len(frames)
						frames = append(frames, root)
					}
					frames = append(frames, shape...)
					end := len(frames) - 1
					frames = append(frames, types.Frame{Mode: types.ModeSender}, types.Frame{Mode: types.ModePostTx, GasLimits: types.Limits{Execution: math.MaxUint64}})
					got, err := ClassifyPrefix(frames, sender, nil)
					if err != nil || got.End != end || got.RecentRootFrame != rootIndex {
						t.Fatalf("prefix %+v: %v", got, err)
					}
				})
			}
		}
	}
	for _, tc := range []struct {
		name   string
		frames []types.Frame
	}{
		{"duplicate", []types.Frame{root, root, self}},
		{"reversed", []types.Frame{root, expiry, self}},
		{"duplicate expiry", []types.Frame{expiry, expiry, root, self}},
		{"after deploy", []types.Frame{deploy, root, self}},
		{"after validation", []types.Frame{self, root}},
		{"post tx inside prefix", []types.Frame{{Mode: types.ModePostTx}, self}},
		{"body after post tx", []types.Frame{self, {Mode: types.ModePostTx}, {Mode: types.ModeSender}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ClassifyPrefix(tc.frames, sender, nil); !errors.Is(err, ErrInvalidPrefix) {
				t.Fatalf("got %v", err)
			}
		})
	}
	for _, length := range []int{0, 64, 71, 72, 73, 16 * 72, 17 * 72} {
		t.Run(fmt.Sprintf("length%d", length), func(t *testing.T) {
			frame := root
			frame.Data = make([]byte, length)
			_, err := ClassifyPrefix([]types.Frame{frame, self}, sender, nil)
			valid := length == 72 || length == 16*72
			if (err == nil) != valid {
				t.Fatalf("length %d: %v", length, err)
			}
		})
	}
	for _, tc := range []struct {
		name  string
		alter func(*types.Frame)
	}{
		{"approval flags", func(f *types.Frame) { f.Flags = 1 }},
		{"atomic flag", func(f *types.Frame) { f.Flags = types.AtomicBatchFlag }},
		{"value", func(f *types.Frame) { f.Value = uint256.NewInt(1) }},
		{"state gas", func(f *types.Frame) { f.GasLimits.State = 1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			frame := root
			tc.alter(&frame)
			if _, err := ClassifyPrefix([]types.Frame{frame, self}, sender, nil); !errors.Is(err, ErrInvalidPrefix) {
				t.Fatalf("got %v", err)
			}
		})
	}
	sigs := types.SignatureList{{Scheme: types.FrameTxSchemeSecp256k1}}
	root.GasLimits.Execution = MaxVerifyGas - self.GasLimits.Execution - types.FrameTxSignatureGas(&sigs[0])
	if _, err := ClassifyPrefix([]types.Frame{root, self}, sender, sigs); err != nil {
		t.Fatal(err)
	}
	root.GasLimits.Execution++
	if _, err := ClassifyPrefix([]types.Frame{root, self}, sender, sigs); !errors.Is(err, ErrPrefixGasLimit) {
		t.Fatal(err)
	}
}

func TestRecentRootAdmission(t *testing.T) {
	for _, tc := range []struct {
		name  string
		slot  uint64
		count int
		alter func(*types.Frame)
		state func(*state.StateDB)
		valid bool
	}{
		{name: "previous", slot: 8191, count: 1, valid: true},
		{name: "sixteen cold", slot: 8191, count: 16, valid: true},
		{name: "duplicates", slot: 8191, count: 1, alter: func(f *types.Frame) { f.Data = append(f.Data, f.Data...) }, valid: true},
		{name: "age8191", slot: 1, count: 1, valid: true},
		{name: "age8192", slot: 0, count: 1},
		{name: "current", slot: 8192, count: 1},
		{name: "future", slot: 8193, count: 1},
		{name: "wrong root", slot: 8191, count: 1, alter: func(f *types.Frame) { f.Data[71]++ }},
		{name: "wrong source", slot: 8191, count: 1, alter: func(f *types.Frame) { f.Data[31]++ }},
		{name: "wrong slot", slot: 8191, count: 1, alter: func(f *types.Frame) { binary.BigEndian.PutUint64(f.Data[32:40], 8190) }},
		{name: "missing code", slot: 8191, count: 1, state: func(s *state.StateDB) { s.SetCode(params.RecentRootAddress, nil, tracing.CodeChangeUnspecified) }},
		{name: "noncanonical code", slot: 8191, count: 1, state: func(s *state.StateDB) {
			s.SetCode(params.RecentRootAddress, []byte{byte(vm.STOP)}, tracing.CodeChangeUnspecified)
		}},
		{name: "insufficient gas", slot: 8191, count: 1, alter: func(f *types.Frame) { f.GasLimits.Execution = 2600 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			frame := rootFrame(tc.slot, tc.count)
			pool, chain, _ := setupFramePool(t, 10, func(s *state.StateDB) {
				installRoots(s, frame)
				if tc.state != nil {
					tc.state(s)
				}
			})
			setInitialSlot(pool, chain, 8191)
			if tc.alter != nil {
				tc.alter(&frame)
			}
			tx := signedPoolTx(t, 1, 0, 0, 20, 2, nil, func(tx *types.FrameTx) { tx.Frames = append([]types.Frame{frame}, tx.Frames...) })
			err := pool.Add([]*types.Transaction{tx}, true)[0]
			if (err == nil) != tc.valid {
				t.Fatalf("admission: %v, valid=%v", err, tc.valid)
			}
			assertFramePoolConsistent(t, pool)
		})
	}
}

func TestRecentRootReset(t *testing.T) {
	for _, reason := range []string{"storage", "code", "expiry", "reorg root", "reorg code", "reorg future"} {
		t.Run(reason, func(t *testing.T) {
			slot := uint64(0)
			if reason == "reorg future" {
				slot = 1
			}
			frame := rootFrame(slot, 1)
			key, _ := recentRootDependency(frame.Data)
			pool, chain, other := setupFramePool(t, 10, func(s *state.StateDB) {
				installRoots(s, frame)
				if reason == "reorg root" {
					s.SetState(params.RecentRootAddress, key, common.Hash{})
				}
				if reason == "reorg code" {
					s.SetCode(params.RecentRootAddress, nil, tracing.CodeChangeUnspecified)
				}
			})
			if reason == "expiry" {
				setInitialSlot(pool, chain, 8190)
			}
			origin := chain.CurrentBlock()
			head := origin
			if reason == "reorg root" || reason == "reorg code" || reason == "reorg future" {
				accesses := bal.NewConstructionBlockAccessList()
				entryKey, entryHash := recentRootDependency(frame.Data)
				accesses.StorageWrite(1, params.RecentRootAddress, entryKey, entryHash)
				accesses.CodeChange(params.RecentRootAddress, 1, params.RecentRootCode)
				head = chain.extend(origin, 101, nil, accesses, func(s *state.StateDB) { installRoots(s, frame) })
				pool.Reset(origin, head)
			}
			tx := signedPoolTx(t, 1, 0, 0, 20, 2, nil, func(tx *types.FrameTx) { tx.Frames = append([]types.Frame{frame}, tx.Frames...) })
			unrelated := signedPoolTx(t, 2, 0, 0, 20, 2, nil, nil)
			addPoolTx(t, pool, tx, nil)
			addPoolTx(t, pool, unrelated, nil)
			assertSimulations(t, pool, 2)
			accesses := bal.NewConstructionBlockAccessList()
			var change func(*state.StateDB)
			switch reason {
			case "storage":
				accesses.StorageWrite(1, params.RecentRootAddress, key, common.Hash{})
				change = func(s *state.StateDB) { s.SetState(params.RecentRootAddress, key, common.Hash{}) }
			case "code":
				accesses.CodeChange(params.RecentRootAddress, 1, nil)
				change = func(s *state.StateDB) { s.SetCode(params.RecentRootAddress, nil, tracing.CodeChangeUnspecified) }
			}
			parent := head
			if reason == "reorg root" || reason == "reorg code" {
				parent = origin
			}
			next := chain.extend(parent, 102, nil, accesses, change)
			if reason == "reorg future" {
				next = origin
			}
			pool.Reset(head, next)
			assertLive(t, pool, tx, false)
			assertLive(t, pool, unrelated, true)
			assertReleased(t, other, poolAddress(t, 1))
			assertFramePoolConsistent(t, pool)
			if reason == "storage" || reason == "code" {
				assertSimulations(t, pool, 1)
			}
			if reason == "expiry" {
				assertSimulations(t, pool, 0)
			}
		})
	}
}

func keyedTx(t *testing.T, keys []uint256.Int, seq, fee, tip uint64, alter func(*types.FrameTx)) *types.Transaction {
	return signedPoolTx(t, 1, 0, seq, fee, tip, nil, func(tx *types.FrameTx) {
		tx.NonceKeys = keys
		tx.Frames[0].GasLimits.State = MaxVerifyStateGas
		if alter != nil {
			alter(tx)
		}
	})
}

func TestKeyedIdentityReplacement(t *testing.T) {
	pool, _, _ := setupFramePool(t, 10, nil)
	keys := []uint256.Int{*uint256.NewInt(1), *uint256.NewInt(2)}
	original := keyedTx(t, keys, 0, 20, 2, nil)
	addPoolTx(t, pool, original, nil)
	addPoolTx(t, pool, keyedTx(t, keys, 0, 21, 2, nil), txpool.ErrReplaceUnderpriced)
	for _, other := range [][]uint256.Int{{{}}, {*uint256.NewInt(1)}, {*uint256.NewInt(1), *uint256.NewInt(3)}} {
		addPoolTx(t, pool, keyedTx(t, other, 0, 40, 4, nil), ErrSenderPending)
	}
	addPoolTx(t, pool, keyedTx(t, keys, 1, 40, 4, nil), ErrSenderPending)
	replacement := keyedTx(t, keys, 0, 22, 3, nil)
	addPoolTx(t, pool, replacement, nil)
	assertLive(t, pool, original, false)
	assertLive(t, pool, replacement, true)
	if nonce := pool.Nonce(poolAddress(t, 1)); nonce != 0 {
		t.Fatalf("keyed pending advanced account nonce to %d", nonce)
	}
	assertFramePoolConsistent(t, pool)
}

func TestKeyedNonceReset(t *testing.T) {
	keys := []uint256.Int{*uint256.NewInt(1), *uint256.NewInt(2)}
	pool, chain, other := setupFramePool(t, 10, nil)
	tx := keyedTx(t, keys, 0, 20, 2, nil)
	addPoolTx(t, pool, tx, nil)
	assertSimulations(t, pool, 1)
	sender := poolAddress(t, 1)
	for i, key := range []uint256.Int{*uint256.NewInt(3), keys[1]} {
		slot := types.FrameTxNonceSlot(sender, &key)
		accesses := bal.NewConstructionBlockAccessList()
		accesses.StorageWrite(1, params.NonceManagerAddress, slot, common.HexToHash("01"))
		old := chain.CurrentBlock()
		next := chain.extend(old, old.Time+1, nil, accesses, func(s *state.StateDB) { s.SetState(params.NonceManagerAddress, slot, common.HexToHash("01")) })
		pool.Reset(old, next)
		assertSimulations(t, pool, uint64(i))
		assertLive(t, pool, tx, i == 0)
	}
	assertReleased(t, other, sender)
	assertFramePoolConsistent(t, pool)
}

func TestKeyedLegacyNonceDependency(t *testing.T) {
	for _, readsLegacy := range []bool{false, true} {
		t.Run(fmt.Sprint(readsLegacy), func(t *testing.T) {
			pool, chain, _ := setupFramePool(t, 10, func(s *state.StateDB) {
				p := program.New()
				if readsLegacy {
					p.Push(0x0d).Op(vm.TXPARAM, vm.ISZERO)
					p = continueIf(p)
				}
				s.SetCode(poolAddress(t, 1), approveCode(p, types.ApproveExecutionAndPayment), tracing.CodeChangeUnspecified)
			})
			tx := keyedTx(t, []uint256.Int{*uint256.NewInt(1)}, 0, 20, 2, nil)
			addPoolTx(t, pool, tx, nil)
			assertSimulations(t, pool, 1)
			sender := poolAddress(t, 1)
			accesses := bal.NewConstructionBlockAccessList()
			accesses.NonceChange(sender, 1, 1)
			old := chain.CurrentBlock()
			next := chain.extend(old, 101, nil, accesses, func(s *state.StateDB) { s.SetNonce(sender, 1, tracing.NonceChangeUnspecified) })
			pool.Reset(old, next)
			want := uint64(0)
			if readsLegacy {
				want = 1
			}
			assertSimulations(t, pool, want)
			assertLive(t, pool, tx, !readsLegacy)
			if nonce := pool.Nonce(sender); nonce != 1 {
				t.Fatalf("pending account nonce %d, want 1", nonce)
			}
			assertFramePoolConsistent(t, pool)
		})
	}
}

func TestKeyedNonceSequenceChecks(t *testing.T) {
	key := uint256.NewInt(1)
	sender := poolAddress(t, 1)
	slot := types.FrameTxNonceSlot(sender, key)
	for _, tc := range []struct {
		name   string
		stored *uint256.Int
		seq    uint64
		want   error
	}{
		{"equal", uint256.NewInt(7), 7, nil},
		{"low", uint256.NewInt(7), 6, core.ErrNonceTooLow},
		{"high", uint256.NewInt(7), 8, core.ErrNonceTooHigh},
		{"exhausted", uint256.NewInt(math.MaxUint64), math.MaxUint64 - 1, core.ErrNonceTooLow},
		{"wide", new(uint256.Int).Lsh(uint256.NewInt(1), 64), 0, core.ErrNonceTooLow},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool, _, _ := setupFramePool(t, 10, func(s *state.StateDB) {
				s.SetNonce(sender, 42, tracing.NonceChangeUnspecified)
				s.SetState(params.NonceManagerAddress, slot, common.Hash(tc.stored.Bytes32()))
			})
			tx := keyedTx(t, []uint256.Int{*key}, tc.seq, 20, 2, nil)
			if err := checkNonces(pool.state, tx); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestKeyedPrefixConsumption(t *testing.T) {
	keys := []uint256.Int{*uint256.NewInt(1), *uint256.NewInt(2)}
	pool, chain, _ := setupFramePool(t, 10, func(s *state.StateDB) { s.SetNonce(poolAddress(t, 1), 7, tracing.NonceChangeUnspecified) })
	tx := keyedTx(t, keys, 0, 20, 2, nil)
	prefix, err := pool.classify(tx, chain.CurrentBlock())
	if err != nil {
		t.Fatal(err)
	}
	s, err := chain.StateAt(chain.CurrentBlock())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.simulate(chain.CurrentBlock(), s, tx, prefix); err != nil {
		t.Fatal(err)
	}
	for i := range keys {
		if got := s.GetState(params.NonceManagerAddress, types.FrameTxNonceSlot(poolAddress(t, 1), &keys[i])); got != common.HexToHash("01") {
			t.Fatalf("key %s consumed to %s", keys[i].Hex(), got)
		}
	}
	if got := s.GetNonce(poolAddress(t, 1)); got != 7 {
		t.Fatalf("legacy nonce changed to %d", got)
	}
	addPoolTx(t, pool, tx, nil)
	if got := pool.Nonce(poolAddress(t, 1)); got != 7 {
		t.Fatalf("pending legacy nonce %d", got)
	}
}

func TestPostTxAdmission(t *testing.T) {
	target := simulationHelper
	pool, _, _ := setupFramePool(t, 10, func(s *state.StateDB) { s.SetCode(target, []byte{byte(vm.INVALID)}, tracing.CodeChangeUnspecified) })
	tx := signedPoolTx(t, 1, 0, 0, 20, 2, nil, func(tx *types.FrameTx) {
		tx.Frames = append(tx.Frames, types.Frame{Mode: types.ModeSender, GasLimits: types.Limits{Execution: 200_000}, Value: new(uint256.Int)}, types.Frame{Mode: types.ModePostTx, Target: &target, GasLimits: types.Limits{Execution: 200_000}, Value: new(uint256.Int)})
	})
	addPoolTx(t, pool, tx, nil)
	assertLive(t, pool, tx, true)
	assertSimulations(t, pool, 1)
}

func TestRecentRootGasBoundary(t *testing.T) {
	for _, count := range []int{1, 16} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			frame := rootFrame(0, count)
			pool, chain, _ := setupFramePool(t, 10, func(s *state.StateDB) { installRoots(s, frame) })
			makeTx := func() *types.Transaction {
				return signedPoolTx(t, 1, 0, 0, 20, 2, nil, func(tx *types.FrameTx) {
					tx.Frames = append([]types.Frame{frame}, tx.Frames...)
				})
			}
			tx := makeTx()
			head := chain.CurrentBlock()
			s, err := chain.StateAt(head)
			if err != nil {
				t.Fatal(err)
			}
			msg, err := core.TransactionToMessage(tx, pool.signer, head.BaseFee)
			if err != nil {
				t.Fatal(err)
			}
			var evm *vm.EVM
			var frameContext *vm.FrameContext
			evm = vm.NewEVM(vm.BlockContext{
				CanTransfer: core.CanTransfer, Transfer: core.Transfer,
				BlockNumber: head.Number, Time: head.Time, SlotNum: 1,
				Difficulty: head.Difficulty, Random: &head.MixDigest, BaseFee: head.BaseFee,
				GasLimit: head.GasLimit, CostPerStateByte: params.CostPerStateByte,
			}, s, chain.Config(), vm.Config{Tracer: &tracing.Hooks{
				OnOpcode: func(_ uint64, _ byte, _, _ uint64, _ tracing.OpContext, _ []byte, _ int, _ error) {
					frameContext = evm.TxContext.FrameContext
				},
			}})
			if _, err := core.ApplyFramePrefix(evm, msg, nil, 1); err != nil {
				t.Fatal(err)
			}
			// The receipt includes the cold target access, not just bytecode gas.
			required := frameContext.Receipts[0].GasUsed
			frame.GasLimits.Execution = required - 1
			addPoolTx(t, pool, makeTx(), core.ErrFrameTxInvalidExecution)
			frame.GasLimits.Execution = required
			addPoolTx(t, pool, makeTx(), nil)
		})
	}
}

func TestRecentRootNestedCallRejected(t *testing.T) {
	frame := rootFrame(0, 1)
	p := program.New()
	for offset := 0; offset < len(frame.Data); offset += 32 {
		var word [32]byte
		copy(word[:], frame.Data[offset:min(offset+32, len(frame.Data))])
		p.Push(word[:]).Push(offset).Op(vm.MSTORE)
	}
	p.Push(0).Push(0).Push(len(frame.Data)).Push(0).Push(params.RecentRootAddress).Op(vm.GAS, vm.STATICCALL, vm.POP)
	pool, _, _ := setupFramePool(t, 10, func(s *state.StateDB) {
		installRoots(s, frame)
		s.SetCode(poolAddress(t, 1), approveCode(p, types.ApproveExecutionAndPayment), tracing.CodeChangeUnspecified)
	})
	tx := signedPoolTx(t, 1, 0, 0, 20, 2, nil, func(tx *types.FrameTx) {
		tx.Frames = append([]types.Frame{frame}, tx.Frames...)
	})
	addPoolTx(t, pool, tx, ErrTraceViolation)
}

func TestRecentRootReplacement(t *testing.T) {
	oldFrame, newFrame := rootFrame(0, 1), rootFrame(0, 1)
	newFrame.Data[31]++
	pool, chain, _ := setupFramePool(t, 10, func(s *state.StateDB) {
		installRoots(s, oldFrame)
		installRoots(s, newFrame)
	})
	old := signedPoolTx(t, 1, 0, 0, 20, 2, nil, func(tx *types.FrameTx) { tx.Frames = append([]types.Frame{oldFrame}, tx.Frames...) })
	replacement := signedPoolTx(t, 1, 0, 0, 22, 3, nil, func(tx *types.FrameTx) { tx.Frames = append([]types.Frame{newFrame}, tx.Frames...) })
	addPoolTx(t, pool, old, nil)
	addPoolTx(t, pool, replacement, nil)
	assertSimulations(t, pool, 2)
	for i, frame := range []types.Frame{oldFrame, newFrame} {
		key, _ := recentRootDependency(frame.Data)
		accesses := bal.NewConstructionBlockAccessList()
		accesses.StorageWrite(1, params.RecentRootAddress, key, common.Hash{})
		head := chain.CurrentBlock()
		next := chain.extend(head, head.Time+1, nil, accesses, func(s *state.StateDB) { s.SetState(params.RecentRootAddress, key, common.Hash{}) })
		pool.Reset(head, next)
		assertSimulations(t, pool, uint64(i))
		assertLive(t, pool, replacement, i == 0)
		assertLive(t, pool, old, false)
		assertFramePoolConsistent(t, pool)
	}
}

func TestRecentRootActivationReorg(t *testing.T) {
	frame := rootFrame(0, 1)
	pool, chain, other := setupFramePool(t, 10, func(s *state.StateDB) { installRoots(s, frame) })
	activation := uint64(101)
	chain.config.BogotaTime = &activation
	origin := chain.CurrentBlock()
	head := chain.extend(origin, activation, nil, bal.NewConstructionBlockAccessList(), nil)
	pool.Reset(origin, head)
	tx := signedPoolTx(t, 1, 0, 0, 20, 2, nil, func(tx *types.FrameTx) { tx.Frames = append([]types.Frame{frame}, tx.Frames...) })
	addPoolTx(t, pool, tx, nil)
	pool.Reset(head, origin)
	assertLive(t, pool, tx, false)
	assertReleased(t, other, poolAddress(t, 1))
	assertFramePoolConsistent(t, pool)
}
