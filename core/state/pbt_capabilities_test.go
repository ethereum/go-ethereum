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
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
)

// Operations the binary tree does not support must refuse, not return an answer
// that happens to be empty. An empty dump and a dump of an empty state are the
// same bytes; only the error distinguishes them.

// TestPBTRefusesAccountDump pins that the account-shaped dump reports failure.
//
// It used to log and return (nil, nil), which the signature reads as a
// completed dump - so a caller received an empty result and no indication that
// anything was wrong. The binary tree keeps no RLP account leaves to decode;
// its state is spread over per-zone leaves, and DumpPBTLeaves is what dumps
// those.
func TestPBTRefusesAccountDump(t *testing.T) {
	sdb, _ := newPBTState(t)

	// Put something in the state, so an empty result cannot be explained away
	// as the state genuinely being empty.
	addr := common.Address{0xd0, 0x11}
	sdb.CreateAccount(addr)
	sdb.SetNonce(addr, 1, tracing.NonceChangeUnspecified)

	dump, err := sdb.RawDump(nil)
	if err == nil {
		t.Fatalf("account dumping reported success on a binary tree state, returning %d accounts", len(dump.Accounts))
	}
	if !strings.Contains(err.Error(), "binary tree") {
		t.Fatalf("the refusal does not say why: %v", err)
	}

	// The other two entry points must not swallow it either.
	if _, err := sdb.Dump(nil); err == nil {
		t.Fatal("Dump reported success on a binary tree state")
	}
	if err := sdb.IterativeDump(nil, json.NewEncoder(&bytes.Buffer{})); err == nil {
		t.Fatal("IterativeDump reported success on a binary tree state")
	}
}
