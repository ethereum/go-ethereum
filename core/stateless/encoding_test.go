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

package stateless

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestWitnessExtRoundTripPreservesKeys(t *testing.T) {
	ext := &ExtWitness{
		Headers: []*types.Header{{Number: big.NewInt(1)}},
		Codes:   []hexutil.Bytes{[]byte("code")},
		Keys:    []hexutil.Bytes{[]byte("key-a"), []byte("key-b")},
		State:   []hexutil.Bytes{[]byte("state")},
	}
	var witness Witness
	if err := witness.FromExtWitness(ext); err != nil {
		t.Fatalf("FromExtWitness error: %v", err)
	}
	if len(witness.Keys) != len(ext.Keys) {
		t.Fatalf("stored %d keys, want %d", len(witness.Keys), len(ext.Keys))
	}
	for _, key := range ext.Keys {
		if _, ok := witness.Keys[string(key)]; !ok {
			t.Fatalf("missing key %q after FromExtWitness", []byte(key))
		}
	}
	roundTrip := witness.ToExtWitness()
	if len(roundTrip.Keys) != len(ext.Keys) {
		t.Fatalf("encoded %d keys, want %d", len(roundTrip.Keys), len(ext.Keys))
	}
	encoded := make(map[string]struct{}, len(roundTrip.Keys))
	for _, key := range roundTrip.Keys {
		encoded[string(key)] = struct{}{}
	}
	for _, key := range ext.Keys {
		if _, ok := encoded[string(key)]; !ok {
			t.Fatalf("missing key %q after ToExtWitness", []byte(key))
		}
	}
}

func TestWitnessAddKey(t *testing.T) {
	witness := &Witness{}
	witness.AddKey([]byte("key-a"), nil, []byte("key-a"), []byte("key-b"))

	if len(witness.Keys) != 2 {
		t.Fatalf("stored %d keys, want 2", len(witness.Keys))
	}
	for _, key := range []string{"key-a", "key-b"} {
		if _, ok := witness.Keys[key]; !ok {
			t.Fatalf("missing key %q", key)
		}
	}
}
