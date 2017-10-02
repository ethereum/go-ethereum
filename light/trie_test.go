// Copyright 2017 The go-burnout Authors
// This file is part of the go-burnout library.
//
// The go-burnout library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-burnout library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-burnout library. If not, see <http://www.gnu.org/licenses/>.

package light

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/burnoutcoin/go-burnout/consensus/ethash"
	"github.com/burnoutcoin/go-burnout/core"
	"github.com/burnoutcoin/go-burnout/core/state"
	"github.com/burnoutcoin/go-burnout/core/vm"
	"github.com/burnoutcoin/go-burnout/brndb"
	"github.com/burnoutcoin/go-burnout/params"
	"github.com/burnoutcoin/go-burnout/trie"
)

func TestNodeIterator(t *testing.T) {
	var (
		fulldb, _  = brndb.NewMemDatabase()
		lightdb, _ = brndb.NewMemDatabase()
		gspec      = core.Genesis{Alloc: core.GenesisAlloc{testBankAddress: {Balance: testBankFunds}}}
		genesis    = gspec.MustCommit(fulldb)
	)
	gspec.MustCommit(lightdb)
	blockchain, _ := core.NewBlockChain(fulldb, params.TestChainConfig, ethash.NewFullFaker(), vm.Config{})
	gchain, _ := core.GenerateChain(params.TestChainConfig, genesis, fulldb, 4, testChainGen)
	if _, err := blockchain.InsertChain(gchain); err != nil {
		panic(err)
	}

	ctx := context.Background()
	odr := &testOdr{sdb: fulldb, ldb: lightdb}
	head := blockchain.CurrentHeader()
	lightTrie, _ := NewStateDatabase(ctx, head, odr).OpenTrie(head.Root)
	fullTrie, _ := state.NewDatabase(fulldb).OpenTrie(head.Root)
	if err := diffTries(fullTrie, lightTrie); err != nil {
		t.Fatal(err)
	}
}

func diffTries(t1, t2 state.Trie) error {
	i1 := trie.NewIterator(t1.NodeIterator(nil))
	i2 := trie.NewIterator(t2.NodeIterator(nil))
	for i1.Next() && i2.Next() {
		if !bytes.Equal(i1.Key, i2.Key) {
			spew.Dump(i2)
			return fmt.Errorf("tries have different keys %x, %x", i1.Key, i2.Key)
		}
		if !bytes.Equal(i2.Value, i2.Value) {
			return fmt.Errorf("tries differ at key %x", i1.Key)
		}
	}
	switch {
	case i1.Err != nil:
		return fmt.Errorf("full trie iterator error: %v", i1.Err)
	case i2.Err != nil:
		return fmt.Errorf("light trie iterator error: %v", i1.Err)
	case i1.Next():
		return fmt.Errorf("full trie iterator has more k/v pairs")
	case i2.Next():
		return fmt.Errorf("light trie iterator has more k/v pairs")
	}
	return nil
}
