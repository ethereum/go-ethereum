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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/crypto"
)

var template8298 = []byte{byte(vm.PUSH1), 0x2a, byte(vm.STOP)}

func newEIP8298TestEnv(extra types.GenesisAlloc) *balTestEnv {
	env := newBALTestEnv(extra)
	env.cfg.BogotaTime = new(uint64)
	return env
}

func setCodeFrom8298(source common.Address) []byte {
	return program.New().Push(source).Op(vm.SETCODEFROM).Bytes()
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
