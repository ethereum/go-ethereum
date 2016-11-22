// Copyright 2016 The go-ethereum Authors
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

package whisperv5

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestWhisperBasic(x *testing.T) {
	w := NewWhisper(nil)
	p := w.Protocols()
	shh := p[0]
	if shh.Name != ProtocolName {
		x.Fatalf("failed Protocol Name: %v.", shh.Name)
	}
	if uint64(shh.Version) != ProtocolVersion {
		x.Fatalf("failed Protocol Version: %v.", shh.Version)
	}
	if shh.Length != NumberOfMessageCodes {
		x.Fatalf("failed Protocol Length: %v.", shh.Length)
	}
	if shh.Run == nil {
		x.Fatalf("failed shh.Run.")
	}
	if uint64(w.Version()) != ProtocolVersion {
		x.Fatalf("failed whisper Version: %v.", shh.Version)
	}
	if w.GetFilter(0) != nil {
		x.Fatalf("failed GetFilter.")
	}

	peerID := make([]byte, 64)
	randomize(peerID)
	peer, err := w.getPeer(peerID)
	if peer != nil {
		x.Fatalf("failed GetPeer.")
	}
	err = w.MarkPeerTrusted(peerID)
	if err == nil {
		x.Fatalf("failed MarkPeerTrusted.")
	}
	err = w.RequestHistoricMessages(peerID, peerID)
	if err == nil {
		x.Fatalf("failed RequestHistoricMessages.")
	}
	err = w.SendP2PMessage(peerID, nil)
	if err == nil {
		x.Fatalf("failed SendP2PMessage.")
	}
	exist := w.HasSymKey("non-existing")
	if exist {
		x.Fatalf("failed HasSymKey.")
	}
	key := w.GetSymKey("non-existing")
	if key != nil {
		x.Fatalf("failed GetSymKey.")
	}
	mail := w.Envelopes()
	if len(mail) != 0 {
		x.Fatalf("failed w.Envelopes().")
	}
	m := w.Messages(0)
	if len(m) != 0 {
		x.Fatalf("failed w.Messages.")
	}

	var derived []byte
	ver := uint64(0xDEADBEEF)
	derived, err = deriveKeyMaterial(peerID, ver)
	if err != unknownVersionError(ver) {
		x.Fatalf("failed deriveKeyMaterial test case 1 with param = %v: %s.", peerID, err)
	}
	derived, err = deriveKeyMaterial(peerID, 0)
	if err != nil {
		x.Fatalf("failed deriveKeyMaterial test case 2 with param = %v: %s.", peerID, err)
	}
	if !validateSymmetricKey(derived) {
		x.Fatalf("failed validateSymmetricKey with param = %v.", derived)
	}
	if containsOnlyZeros(derived) {
		x.Fatalf("failed containsOnlyZeros with param = %v.", derived)
	}

	buf := []byte{0xFF, 0xE5, 0x80, 0x2, 0}
	le := bytesToIntLittleEndian(buf)
	be := BytesToIntBigEndian(buf)
	if le != uint64(0x280e5ff) {
		x.Fatalf("failed bytesToIntLittleEndian: %d.", le)
	}
	if be != uint64(0xffe5800200) {
		x.Fatalf("failed BytesToIntBigEndian: %d.", be)
	}

	pk := w.NewIdentity()
	if !validatePrivateKey(pk) {
		x.Fatalf("failed validatePrivateKey: %v.", pk)
	}
	if !ValidatePublicKey(&pk.PublicKey) {
		x.Fatalf("failed ValidatePublicKey: %v.", pk)
	}
}

func TestWhisperIdentityManagement(x *testing.T) {
	w := NewWhisper(nil)
	id1 := w.NewIdentity()
	id2 := w.NewIdentity()
	pub1 := common.ToHex(crypto.FromECDSAPub(&id1.PublicKey))
	pub2 := common.ToHex(crypto.FromECDSAPub(&id2.PublicKey))
	pk1 := w.GetIdentity(pub1)
	pk2 := w.GetIdentity(pub2)
	if !w.HasIdentity(pub1) {
		x.Fatalf("failed test case 1.")
	}
	if !w.HasIdentity(pub2) {
		x.Fatalf("failed test case 2.")
	}
	if pk1 != id1 {
		x.Fatalf("failed test case 3.")
	}
	if pk2 != id2 {
		x.Fatalf("failed test case 4.")
	}

	// Delete one identity
	w.DeleteIdentity(pub1)
	pk1 = w.GetIdentity(pub1)
	pk2 = w.GetIdentity(pub2)
	if w.HasIdentity(pub1) {
		x.Fatalf("failed test case 11.")
	}
	if !w.HasIdentity(pub2) {
		x.Fatalf("failed test case 12.")
	}
	if pk1 != nil {
		x.Fatalf("failed test case 13.")
	}
	if pk2 != id2 {
		x.Fatalf("failed test case 14.")
	}

	// Delete again non-existing identity
	w.DeleteIdentity(pub1)
	pk1 = w.GetIdentity(pub1)
	pk2 = w.GetIdentity(pub2)
	if w.HasIdentity(pub1) {
		x.Fatalf("failed test case 21.")
	}
	if !w.HasIdentity(pub2) {
		x.Fatalf("failed test case 22.")
	}
	if pk1 != nil {
		x.Fatalf("failed test case 23.")
	}
	if pk2 != id2 {
		x.Fatalf("failed test case 24.")
	}

	// Delete second identity
	w.DeleteIdentity(pub2)
	pk1 = w.GetIdentity(pub1)
	pk2 = w.GetIdentity(pub2)
	if w.HasIdentity(pub1) {
		x.Fatalf("failed test case 31.")
	}
	if w.HasIdentity(pub2) {
		x.Fatalf("failed test case 32.")
	}
	if pk1 != nil {
		x.Fatalf("failed test case 33.")
	}
	if pk2 != nil {
		x.Fatalf("failed test case 34.")
	}
}

func TestWhisperSymKeyManagement(x *testing.T) {
	InitSingleTest()

	var k1, k2 []byte
	w := NewWhisper(nil)
	id1 := string("arbitrary-string-1")
	id2 := string("arbitrary-string-2")

	err := w.GenerateSymKey(id1)
	if err != nil {
		x.Fatalf("failed test case 1 with seed %d: %s.", seed, err)
	}

	k1 = w.GetSymKey(id1)
	k2 = w.GetSymKey(id2)
	if !w.HasSymKey(id1) {
		x.Fatalf("failed test case 2.")
	}
	if w.HasSymKey(id2) {
		x.Fatalf("failed test case 3.")
	}
	if k1 == nil {
		x.Fatalf("failed test case 4.")
	}
	if k2 != nil {
		x.Fatalf("failed test case 5.")
	}

	// add existing id, nothing should change
	randomKey := make([]byte, 16)
	randomize(randomKey)
	err = w.AddSymKey(id1, randomKey)
	if err == nil {
		x.Fatalf("failed test case 10 with seed %d.", seed)
	}

	k1 = w.GetSymKey(id1)
	k2 = w.GetSymKey(id2)
	if !w.HasSymKey(id1) {
		x.Fatalf("failed test case 12.")
	}
	if w.HasSymKey(id2) {
		x.Fatalf("failed test case 13.")
	}
	if k1 == nil {
		x.Fatalf("failed test case 14.")
	}
	if bytes.Compare(k1, randomKey) == 0 {
		x.Fatalf("failed test case 15: k1 == randomKey.")
	}
	if k2 != nil {
		x.Fatalf("failed test case 16.")
	}

	err = w.AddSymKey(id2, randomKey) // add non-existing (yet)
	if err != nil {
		x.Fatalf("failed test case 21 with seed %d: %s.", seed, err)
	}
	k1 = w.GetSymKey(id1)
	k2 = w.GetSymKey(id2)
	if !w.HasSymKey(id1) {
		x.Fatalf("failed test case 22.")
	}
	if !w.HasSymKey(id2) {
		x.Fatalf("failed test case 23.")
	}
	if k1 == nil {
		x.Fatalf("failed test case 24.")
	}
	if k2 == nil {
		x.Fatalf("failed test case 25.")
	}
	if bytes.Compare(k1, k2) == 0 {
		x.Fatalf("failed test case 26.")
	}
	if bytes.Compare(k1, randomKey) == 0 {
		x.Fatalf("failed test case 27.")
	}
	if len(k1) != aesKeyLength {
		x.Fatalf("failed test case 28.")
	}
	if len(k2) != aesKeyLength {
		x.Fatalf("failed test case 29.")
	}

	w.DeleteSymKey(id1)
	k1 = w.GetSymKey(id1)
	k2 = w.GetSymKey(id2)
	if w.HasSymKey(id1) {
		x.Fatalf("failed test case 31.")
	}
	if !w.HasSymKey(id2) {
		x.Fatalf("failed test case 32.")
	}
	if k1 != nil {
		x.Fatalf("failed test case 33.")
	}
	if k2 == nil {
		x.Fatalf("failed test case 34.")
	}

	w.DeleteSymKey(id1)
	w.DeleteSymKey(id2)
	k1 = w.GetSymKey(id1)
	k2 = w.GetSymKey(id2)
	if w.HasSymKey(id1) {
		x.Fatalf("failed test case 41.")
	}
	if w.HasSymKey(id2) {
		x.Fatalf("failed test case 42.")
	}
	if k1 != nil {
		x.Fatalf("failed test case 43.")
	}
	if k2 != nil {
		x.Fatalf("failed test case 44.")
	}
}
