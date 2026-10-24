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
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/holiman/uint256"
)

// EIP-8298 (SETCODEFROM) block-level tests. They reuse the EIP-7928 harness,
// whose run also checks that parallel execution reproduces each block.

// template8298 is the code adopted in the tests.
var template8298 = []byte{byte(vm.PUSH1), 0x2a, byte(vm.STOP)}

// newEIP8298TestEnv returns a BAL test environment with Bogota active at genesis.
func newEIP8298TestEnv(extra types.GenesisAlloc) *balTestEnv {
	env := newBALTestEnv(extra)
	env.cfg.BogotaTime = new(uint64)
	return env
}

// setCodeFrom8298 returns code adopting the code of source.
func setCodeFrom8298(source common.Address) []byte {
	return program.New().Push(source).Op(vm.SETCODEFROM).Bytes()
}

// codeChanges8298 renders the code changes of an account in a block access
// list as " index:code", followed by "#hash" for adopted code.
func codeChanges8298(b *bal.BlockAccessList, addr common.Address) string {
	var s string
	if aa := findAccount(b, addr); aa != nil {
		for _, c := range aa.CodeChanges {
			s += fmt.Sprintf(" %d:%x", c.BlockAccessIndex, c.NewCode)
			if c.NewCodeHash != nil {
				s += fmt.Sprintf("#%x", *c.NewCodeHash)
			}
		}
	}
	return s
}

// TestEIP8298BAL checks how adoptions enter the block access list: the code
// change holds the adopted hash, and the source is included once its access is
// paid for, even if the opcode then runs out of gas.
func TestEIP8298BAL(t *testing.T) {
	tests := []struct {
		name   string
		code   []byte // source code, nil for a missing source
		same   bool   // the source holds the adopter's code
		static bool   // the adopter runs in a static call
		gas    uint64 // gas given to the adopter
		inBAL  bool
		adopt  bool
	}{
		{name: "adopted", code: template8298, gas: 100_000, inBAL: true, adopt: true},
		{name: "same hash", same: true, gas: 100_000, inBAL: true},
		{name: "invalid", gas: 100_000, inBAL: true},
		{name: "oog after source charge", code: template8298, gas: 5_000, inBAL: true},
		{name: "oog before source charge", code: template8298, gas: 2_000},
		{name: "static", code: template8298, static: true, gas: 100_000},
	}
	var (
		template = common.HexToAddress("0x7e3a")
		alloc    = types.GenesisAlloc{template: {Code: template8298}}
		adopted  = fmt.Sprintf("#%x", crypto.Keccak256Hash(template8298))
	)
	for i, tt := range tests {
		source, adopter, caller := common.Address{0x50, byte(i)}, common.Address{0xad, byte(i)}, common.Address{0xca, byte(i)}
		alloc[adopter] = types.Account{Code: setCodeFrom8298(source)}
		if tt.same {
			tt.code = setCodeFrom8298(source)
		}
		if tt.code != nil {
			alloc[source] = types.Account{Code: tt.code}
		}
		call := program.New()
		if tt.static {
			call.StaticCall(uint256.NewInt(tt.gas), adopter, 0, 0, 0, 0)
		} else {
			call.Call(uint256.NewInt(tt.gas), adopter, 0, 0, 0, 0, 0)
		}
		alloc[caller] = types.Account{Code: call.Bytes()}
	}
	env := newEIP8298TestEnv(alloc)
	b, receipts := env.run(t, func(g *BlockGen) {
		for i := range tests {
			g.AddTx(env.tx(uint64(i), &common.Address{0xca, byte(i)}, common.Big0, 1_000_000, 0, nil))
		}
		// A creation adopting code instead of depositing its return data.
		g.AddTx(env.tx(uint64(len(tests)), nil, common.Big0, 1_000_000, 0, setCodeFrom8298(template)))
	})
	for i, tt := range tests {
		if got := findAccount(b, common.Address{0x50, byte(i)}) != nil; got != tt.inBAL {
			t.Errorf("%s: source in BAL %v, want %v", tt.name, got, tt.inBAL)
		}
		var want string
		if tt.adopt {
			want = fmt.Sprintf(" %d:%s", i+1, adopted)
		}
		if got := codeChanges8298(b, common.Address{0xad, byte(i)}); got != want {
			t.Errorf("%s: adopter code changes %q, want %q", tt.name, got, want)
		}
	}
	created := receipts[len(tests)]
	if got, want := codeChanges8298(b, created.ContractAddress), fmt.Sprintf(" %d:%s", len(tests)+1, adopted); created.Status != types.ReceiptStatusSuccessful || got != want {
		t.Errorf("creation: status %d, code changes %q, want %q", created.Status, got, want)
	}
}

// TestEIP8298SystemCall gives each system contract code running SETCODEFROM.
// The opcode halts before touching the source: the unchecked beacon roots and
// history calls keep the block valid, a request call invalidates it.
func TestEIP8298SystemCall(t *testing.T) {
	source := common.HexToAddress("0x5005ce")
	for _, addr := range []common.Address{params.BeaconRootsAddress, params.HistoryStorageAddress} {
		env := newEIP8298TestEnv(types.GenesisAlloc{
			addr:   {Nonce: 1, Code: setCodeFrom8298(source)},
			source: {Code: template8298},
		})
		b, _ := env.run(t, func(g *BlockGen) { g.SetParentBeaconRoot(common.Hash{0x01}) })
		assertAbsent(t, b, source)
		if got := codeChanges8298(b, addr); got != "" {
			t.Errorf("%x: code changes %q", addr, got)
		}
	}
	for _, addr := range []common.Address{params.WithdrawalQueueAddress, params.ConsolidationQueueAddress, params.BuilderDepositAddress, params.BuilderExitAddress} {
		env := newEIP8298TestEnv(types.GenesisAlloc{
			addr:   {Nonce: 1, Code: setCodeFrom8298(source)},
			source: {Code: template8298},
		})
		bc, err := NewBlockChain(rawdb.NewMemoryDatabase(), env.gspec, beacon.New(ethash.NewFaker()), nil)
		if err != nil {
			t.Fatal(err)
		}
		statedb, _ := bc.State()
		head := bc.CurrentBlock()
		blockContext := NewEVMBlockContext(head, bc, &head.Coinbase)
		blockContext.Random = &common.Hash{} // enable post-merge instruction set
		evm := vm.NewEVM(blockContext, statedb, env.cfg, vm.Config{})
		_, _, err = PostExecution(context.Background(), env.cfg, head.Number, head.Time, nil, nil, evm, 1)
		if err == nil || !strings.Contains(err.Error(), vm.ErrWriteProtection.Error()) {
			t.Errorf("%x: post-execution error %v, want %v", addr, err, vm.ErrWriteProtection)
		}
		bc.Stop()
	}
}

// TestEIP8298MigratedEOA lets an EOA delegated to a migrator adopt regular
// code. Its transactions are then rejected (EIP-3607) and later authorizations
// for it are skipped.
func TestEIP8298MigratedEOA(t *testing.T) {
	var (
		template = common.HexToAddress("0x7e3a")
		migrator = common.HexToAddress("0x3196")
		env      = newEIP8298TestEnv(types.GenesisAlloc{
			template: {Code: template8298},
			migrator: {Code: setCodeFrom8298(template)},
		})
		key, _ = crypto.HexToECDSA("0202020202020202020202020202020202020202020202020202002020202020")
		eoa    = crypto.PubkeyToAddress(key.PublicKey)
	)
	auth := func(nonce uint64) []types.SetCodeAuthorization {
		auth, err := types.SignSetCode(key, types.SetCodeAuthorization{
			ChainID: *uint256.MustFromBig(env.cfg.ChainID),
			Address: migrator,
			Nonce:   nonce,
		})
		if err != nil {
			t.Fatal(err)
		}
		return []types.SetCodeAuthorization{auth}
	}
	b, _ := env.run(t, func(g *BlockGen) {
		g.AddTx(env.newSetCodeTx(t, 0, eoa, auth(0))) // delegate to the migrator and run it
		g.AddTx(env.newSetCodeTx(t, 1, eoa, auth(1))) // skipped: the EOA holds code

		tx := types.MustSignNewTx(key, env.signer, &types.DynamicFeeTx{
			ChainID:   env.cfg.ChainID,
			Nonce:     1,
			To:        &eoa,
			Gas:       params.TxGas,
			GasFeeCap: newGwei(10),
		})
		msg, err := TransactionToMessage(tx, env.signer, g.header.BaseFee)
		if err != nil {
			t.Fatal(err)
		}
		evm := vm.NewEVM(NewEVMBlockContext(g.header, g.cm, &g.header.Coinbase), g.statedb.Copy(), env.cfg, vm.Config{})
		if _, err := ApplyMessage(evm, msg, NewGasPool(g.header.GasLimit)); !errors.Is(err, ErrSenderNoEOA) {
			t.Errorf("transaction of the migrated EOA: error %v, want %v", err, ErrSenderNoEOA)
		}
	})
	if got, want := codeChanges8298(b, eoa), fmt.Sprintf(" 1:#%x", crypto.Keccak256Hash(template8298)); got != want {
		t.Errorf("code changes %q, want %q", got, want)
	}
	if nonces := assertPresent(t, b, eoa).NonceChanges; len(nonces) != 1 || nonces[0].BlockAccessIndex != 1 {
		t.Errorf("nonce changes %+v, want only the first authorization", nonces)
	}
}

// TestEIP8298Restart adopts code from a source created earlier in the block,
// which then replaces its own code. The adopted code must still be readable
// after a restart with cold caches, for sequential and parallel imports.
func TestEIP8298Restart(t *testing.T) {
	var (
		template = common.HexToAddress("0x7e3a")
		adopter  = common.HexToAddress("0xad09")
		env      = newEIP8298TestEnv(types.GenesisAlloc{template: {Code: template8298}})
		source   = crypto.CreateAddress(env.from, 0)
		created  = setCodeFrom8298(template) // the source's code, adopting the template
		engine   = beacon.New(ethash.NewFaker())
	)
	env.gspec.Alloc[adopter] = types.Account{Code: setCodeFrom8298(source)}
	_, blocks, _ := GenerateChainWithGenesis(env.gspec, engine, 1, func(_ int, g *BlockGen) {
		g.AddTx(env.tx(0, nil, common.Big0, 1_000_000, 0, program.New().ReturnViaCodeCopy(created).Bytes()))
		g.AddTx(env.tx(1, &adopter, common.Big0, 1_000_000, 0, nil))
		g.AddTx(env.tx(2, &source, common.Big0, 1_000_000, 0, nil))
	})
	assertParallelEquiv(t, env.gspec, engine, blocks[0])

	b := blocks[0].AccessList()
	if got, want := codeChanges8298(b, source), fmt.Sprintf(" 1:%x 3:#%x", created, crypto.Keccak256Hash(template8298)); got != want {
		t.Errorf("source code changes %q, want %q", got, want)
	}
	if got, want := codeChanges8298(b, adopter), fmt.Sprintf(" 2:#%x", crypto.Keccak256Hash(created)); got != want {
		t.Errorf("adopter code changes %q, want %q", got, want)
	}
	for _, parallel := range []bool{true, false} {
		db := rawdb.NewMemoryDatabase()
		cfg := DefaultConfig()
		cfg.VmConfig.DisableParallelExecution = !parallel
		bc, err := NewBlockChain(db, env.gspec, engine, cfg)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := bc.InsertChain(blocks); err != nil {
			t.Fatalf("parallel %v: import: %v", parallel, err)
		}
		bc.Stop()

		bc, err = NewBlockChain(db, env.gspec, engine, cfg)
		if err != nil {
			t.Fatal(err)
		}
		statedb, err := bc.State()
		if err != nil {
			t.Fatal(err)
		}
		if got := statedb.GetCode(adopter); !bytes.Equal(got, created) || statedb.Error() != nil {
			t.Errorf("parallel %v: adopter code %x (%v), want %x", parallel, got, statedb.Error(), created)
		}
		if got := statedb.GetCode(source); !bytes.Equal(got, template8298) {
			t.Errorf("parallel %v: source code %x, want %x", parallel, got, template8298)
		}
		bc.Stop()
	}
}

// TestEIP8298Archive checks that archive nodes serve the code before and from
// an adoption on.
func TestEIP8298Archive(t *testing.T) {
	var (
		template = common.HexToAddress("0x7e3a")
		adopter  = common.HexToAddress("0xad09")
		code     = setCodeFrom8298(template)
	)
	// The hash scheme archive keeps the trie of every block.
	t.Run("hash", func(t *testing.T) {
		env := newEIP8298TestEnv(types.GenesisAlloc{template: {Code: template8298}, adopter: {Code: code}})
		engine := beacon.New(ethash.NewFaker())
		_, blocks, _ := GenerateChainWithGenesis(env.gspec, engine, 1, func(_ int, g *BlockGen) {
			g.AddTx(env.tx(0, &adopter, common.Big0, 1_000_000, 0, nil))
		})
		bc, err := NewBlockChain(rawdb.NewMemoryDatabase(), env.gspec, engine, DefaultConfig().WithArchive(true))
		if err != nil {
			t.Fatal(err)
		}
		defer bc.Stop()
		if _, err := bc.InsertChain(blocks); err != nil {
			t.Fatal(err)
		}
		for number, want := range [][]byte{code, template8298} {
			header := bc.GetHeaderByNumber(uint64(number))
			statedb, err := bc.StateAt(header.Root, header.Number, header.Time)
			if err != nil {
				t.Fatalf("block %d: %v", number, err)
			}
			if got := statedb.GetCode(adopter); !bytes.Equal(got, want) {
				t.Errorf("block %d: code %x, want %x", number, got, want)
			}
		}
	})
	// The path scheme serves older states from the indexed state history, as
	// BlockChain.HistoricState does. The blockchain cannot skip the delay before
	// indexing, so the states are committed to a path database directly.
	t.Run("path", func(t *testing.T) {
		if testing.Short() {
			t.Skip("the state history indexer starts on a 15s heartbeat")
		}
		disk, err := rawdb.Open(rawdb.NewMemoryDatabase(), rawdb.OpenOptions{}) // the history needs an ancient store
		if err != nil {
			t.Fatal(err)
		}
		var (
			codedb = state.NewCodeDB(disk)
			tdb    = triedb.NewDatabase(disk, &triedb.Config{PathDB: &pathdb.Config{EnableStateIndexing: true, NoHistoryIndexDelay: true}})
			roots  []common.Hash
			root   = types.EmptyRootHash
		)
		defer tdb.Close()
		for i, apply := range []func(*state.StateDB){
			func(s *state.StateDB) {
				s.SetCode(template, template8298, tracing.CodeChangeUnspecified)
				s.SetCode(adopter, code, tracing.CodeChangeUnspecified)
			},
			func(s *state.StateDB) { s.SetCodeFrom(adopter, template) },
			func(s *state.StateDB) { s.SetNonce(template, 1, tracing.NonceChangeUnspecified) },
		} {
			statedb, err := state.New(root, state.NewDatabase(tdb, codedb))
			if err != nil {
				t.Fatal(err)
			}
			apply(statedb)
			if root, err = statedb.Commit(params.Rules{}, uint64(i+1)); err != nil {
				t.Fatal(err)
			}
			roots = append(roots, root)
		}
		// Flatten the layers: all but the last state now come from the history.
		if err := tdb.Commit(root, false); err != nil {
			t.Fatal(err)
		}
		historic := state.NewHistoricDatabase(tdb, codedb)
		for i, want := range [][]byte{code, template8298} {
			statedb, err := state.New(roots[i], historic)
			// The indexer starts in the background, after a heartbeat.
			for start := time.Now(); err != nil && time.Since(start) < 30*time.Second; {
				time.Sleep(10 * time.Millisecond)
				statedb, err = state.New(roots[i], historic)
			}
			if err != nil {
				t.Fatalf("state %d: %v", i, err)
			}
			if got := statedb.GetCode(adopter); !bytes.Equal(got, want) {
				t.Errorf("state %d: code %x, want %x", i, got, want)
			}
		}
	})
}
