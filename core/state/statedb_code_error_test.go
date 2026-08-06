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

package state

import (
	"errors"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
)

// errCodeTrie fails every code write and delegates everything else. Embedding
// the interface keeps the override to the one method under test.
type errCodeTrie struct {
	Trie
	err error
}

func (t errCodeTrie) UpdateContractCode(common.Address, common.Hash, []byte) error {
	return t.err
}

// TestUpdateContractCodeErrorSurfaces pins that a failed code write aborts the
// block instead of being committed over.
//
// Under the merkle-patricia trie UpdateContractCode is a no-op returning nil,
// so dropping its error is invisible. Under the binary tree the call writes the
// code out as leaves - one code-zone stem per 256 chunks - which resolves
// nodes and can fail on a disk read. A dropped error there means
// the state root is computed over a partially written code zone and committed
// as if nothing went wrong, because s.dbErr stays nil.
func TestUpdateContractCodeErrorSurfaces(t *testing.T) {
	sdb, err := New(types.EmptyRootHash, NewDatabaseForTesting())
	if err != nil {
		t.Fatal(err)
	}
	// The main trie is opened lazily inside IntermediateRoot, so force it open
	// before wrapping - otherwise the wrapper embeds a nil interface.
	sdb.IntermediateRoot(true)
	if sdb.trie == nil {
		t.Fatal("the state trie was not opened; the wrapper below would embed nil")
	}
	boom := errors.New("code write failed")
	sdb.trie = errCodeTrie{Trie: sdb.trie, err: boom}

	addr := common.Address{0xc0, 0xde}
	sdb.CreateAccount(addr)
	sdb.SetCode(addr, []byte{0x60, 0x00}, tracing.CodeChangeUnspecified)

	// IntermediateRoot runs the update loop, which is what writes the code.
	sdb.IntermediateRoot(true)

	got := sdb.Error()
	if got == nil {
		t.Fatal("a failed contract-code write left no error on the state; the block would commit over it")
	}
	if !strings.Contains(got.Error(), boom.Error()) {
		t.Fatalf("state error does not mention the underlying failure: %v", got)
	}
}
