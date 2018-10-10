// Copyright 2018 The go-ethereum Authors
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

package enode

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
)

var pyRecord, _ = hex.DecodeString("f884b8407098ad865b00a582051940cb9cf36836572411a47278783077011599ed5cd16b76f2635f4e234738f30813a89eb9137e3e3df5266e3a1f11df72ecf1145ccb9c01826964827634826970847f00000189736563703235366b31a103ca634cae0d49acb401d8a4c6b6fe8c55b70d115bf400769cc1400f3258cd31388375647082765f")

// TestPythonInterop checks that we can decode and verify a record produced by the Python
// implementation.
func TestPythonInterop(t *testing.T) {
	var r enr.Record
	if err := rlp.DecodeBytes(pyRecord, &r); err != nil {
		t.Fatalf("can't decode: %v", err)
	}
	n, err := New(ValidSchemes, &r)
	if err != nil {
		t.Fatalf("can't verify record: %v", err)
	}

	var (
		wantID  = HexID("a448f24c6d18e575453db13171562b71999873db5b286df957af199ec94617f7")
		wantSeq = uint64(1)
		wantIP  = enr.IP{127, 0, 0, 1}
		wantUDP = enr.UDP(30303)
	)
	if n.Seq() != wantSeq {
		t.Errorf("wrong seq: got %d, want %d", n.Seq(), wantSeq)
	}
	if n.ID() != wantID {
		t.Errorf("wrong id: got %x, want %x", n.ID(), wantID)
	}
	want := map[enr.Entry]interface{}{new(enr.IP): &wantIP, new(enr.UDP): &wantUDP}
	for k, v := range want {
		desc := fmt.Sprintf("loading key %q", k.ENRKey())
		if assert.NoError(t, n.Load(k), desc) {
			assert.Equal(t, k, v, desc)
		}
	}
}
