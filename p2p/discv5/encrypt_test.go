// Copyright 2015 The go-ethereum Authors
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

package discv5

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

const testMaxPacketLen = 1000

func testPacket(t *testing.T, encA, encB symmEncryption, packetLen int) {
	packet := make([]byte, packetLen)
	rand.Read(packet[:])
	enc := encA.encode(packet)
	if len(enc) > testMaxPacketLen {
		t.Errorf("Encoded packet is too long")
	}
	dec := encB.decode(enc)
	if !bytes.Equal(packet, dec) {
		t.Errorf("Decoded packet does not match original (packet = %x  size = %d  enc = %x  encSize = %d  dec = %x  decSize = %d)", packet, len(packet), enc, len(enc), dec, len(dec))
	}
}

func TestEcdhAes256Encryption(t *testing.T) {
	privKeyA, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	pubKeyA := &privKeyA.PublicKey
	privKeyB, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	pubKeyB := &privKeyB.PublicKey

	encA, err := newEcdhAes256Encryption(privKeyA, pubKeyB, testMaxPacketLen)
	if err != nil {
		panic(err)
	}
	encB, err := newEcdhAes256Encryption(privKeyB, pubKeyA, testMaxPacketLen)
	if err != nil {
		panic(err)
	}

	maxDecLen := encA.maxDecodedLength()
	for i := 0; i <= maxDecLen; i++ {
		testPacket(t, encA, encB, i)
		testPacket(t, encB, encA, i)
	}
}

func testPacketAsymm(t *testing.T, encA, encB asymmEncryption, pubKeyB *ecdsa.PublicKey, packetLen int) {
	packet := make([]byte, packetLen)
	rand.Read(packet[:])
	enc, _ := encA.encode(packet, pubKeyB)
	if len(enc) > testMaxPacketLen {
		t.Errorf("Encoded packet is too long")
	}
	dec := encB.decode(enc)
	if !bytes.Equal(packet, dec) {
		t.Errorf("Decoded packet does not match original (packet = %x  size = %d  enc = %x  encSize = %d  dec = %x  decSize = %d)", packet, len(packet), enc, len(enc), dec, len(dec))
	}
}

func TestEciesEncryption(t *testing.T) { //TODO fix this
	privKeyA, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	pubKeyA := &privKeyA.PublicKey
	privKeyB, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	pubKeyB := &privKeyB.PublicKey

	encA := newEciesEncryption(privKeyA, testMaxPacketLen)
	encB := newEciesEncryption(privKeyB, testMaxPacketLen)

	maxDecLen := encA.maxDecodedLength()
	for i := 0; i <= maxDecLen; i++ {
		testPacketAsymm(t, encA, encB, pubKeyB, i)
		testPacketAsymm(t, encB, encA, pubKeyA, i)
	}
}
