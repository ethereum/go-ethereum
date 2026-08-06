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

package bintrie

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

// TestGetAccountRefusesBrokenDelegation pins that a delegation leaf beside a
// code size that cannot describe it is an error rather than an answer.
//
// GetAccount derives a delegated account's code hash from the leaf, taking
// its leading code_size bytes. A size of zero would hash nothing and yield
// the empty-code hash, so the account would read back as codeless and
// EIP-161-empty while plainly holding a delegation - and every caller that
// depends on the synthesis, from the txpools to EXTCODEHASH, would agree with
// it. Neither shape is reachable through UpdateAccount, which writes the two
// leaves in one walk; both are reachable from a corrupt or hand-built stem,
// which is exactly when a wrong answer is worst.
func TestGetAccountRefusesBrokenDelegation(t *testing.T) {
	addr := common.Address{1}
	designator := types.AddressToDelegation(common.Address{9})

	for _, tc := range []struct {
		name  string
		basic []byte
	}{
		{"no basic data at all", nil},
		{"basic data reporting no code", func() []byte {
			b, err := EncodeBasicData(0, 7, uint256.NewInt(1))
			if err != nil {
				t.Fatal(err)
			}
			return b[:]
		}()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := newTestTrie()
			subs := []byte{DelegationLeafKey}
			vals := [][]byte{EncodeDelegation(designator)}
			if tc.basic != nil {
				subs = append([]byte{BasicDataLeafKey}, subs...)
				vals = append([][]byte{tc.basic}, vals...)
			}
			if err := tr.UpdateStem(HeaderStem(addr), subs, vals); err != nil {
				t.Fatal(err)
			}
			if _, err := tr.GetAccount(addr); err == nil {
				t.Fatal("a delegation leaf with a zero code size was read as an account")
			}
		})
	}

	// The control: the same leaf with a size that does describe it reads back
	// as the designator's hash, so the guard above rejects the broken shape
	// rather than the mechanism.
	tr := newTestTrie()
	basic, err := EncodeBasicData(uint32(len(designator)), 7, uint256.NewInt(1))
	if err != nil {
		t.Fatal(err)
	}
	if err := tr.UpdateStem(HeaderStem(addr),
		[]byte{BasicDataLeafKey, DelegationLeafKey},
		[][]byte{basic[:], EncodeDelegation(designator)}); err != nil {
		t.Fatal(err)
	}
	acc, err := tr.GetAccount(addr)
	if err != nil {
		t.Fatal(err)
	}
	if want := crypto.Keccak256(designator); string(acc.CodeHash) != string(want) {
		t.Fatalf("code hash %x, want the designator's %x", acc.CodeHash, want)
	}
}
