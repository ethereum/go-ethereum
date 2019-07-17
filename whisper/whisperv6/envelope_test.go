// Copyright 2017 The go-ethereum Authors
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

// Contains the tests associated with the Whisper protocol Envelope object.

package whisperv6

import (
	mrand "math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestPoWCalculationsWithNoLeadingZeros(t *testing.T) {
	e := Envelope{
		TTL:   1,
		Data:  []byte{0xde, 0xad, 0xbe, 0xef},
		Nonce: 100000,
	}

	e.calculatePoW(0)

	if e.pow != 0.07692307692307693 {
		t.Fatalf("invalid PoW calculation. Expected 0.07692307692307693, got %v", e.pow)
	}
}

func TestPoWCalculationsWith8LeadingZeros(t *testing.T) {
	e := Envelope{
		TTL:   1,
		Data:  []byte{0xde, 0xad, 0xbe, 0xef},
		Nonce: 276,
	}
	e.calculatePoW(0)

	if e.pow != 19.692307692307693 {
		t.Fatalf("invalid PoW calculation. Expected 19.692307692307693, got %v", e.pow)
	}
}

func TestEnvelopeOpenAcceptsOnlyOneKeyTypeInFilter(t *testing.T) {
	symKey := make([]byte, aesKeyLength)
	mrand.Read(symKey)

	asymKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed GenerateKey with seed %d: %s.", seed, err)
	}

	params := MessageParams{
		PoW:      0.01,
		WorkTime: 1,
		TTL:      uint32(mrand.Intn(1024)),
		Payload:  make([]byte, 50),
		KeySym:   symKey,
		Dst:      nil,
	}

	mrand.Read(params.Payload)

	msg, err := NewSentMessage(&params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}

	e, err := msg.Wrap(&params)
	if err != nil {
		t.Fatalf("Failed to Wrap the message in an envelope with seed %d: %s", seed, err)
	}

	f := Filter{KeySym: symKey, KeyAsym: asymKey}

	decrypted := e.Open(&f)
	if decrypted != nil {
		t.Fatalf("Managed to decrypt a message with an invalid filter, seed %d", seed)
	}
}
