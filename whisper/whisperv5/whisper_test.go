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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestWhisperBasic(t *testing.T) {
	w := NewWhisper(nil)
	p := w.Protocols()
	shh := p[0]
	if shh.Name != ProtocolName {
		t.Fatalf("failed Protocol Name: %v.", shh.Name)
	}
	if uint64(shh.Version) != ProtocolVersion {
		t.Fatalf("failed Protocol Version: %v.", shh.Version)
	}
	if shh.Length != NumberOfMessageCodes {
		t.Fatalf("failed Protocol Length: %v.", shh.Length)
	}
	if shh.Run == nil {
		t.Fatalf("failed shh.Run.")
	}
	if uint64(w.Version()) != ProtocolVersion {
		t.Fatalf("failed whisper Version: %v.", shh.Version)
	}
	if w.GetFilter(0) != nil {
		t.Fatalf("failed GetFilter.")
	}

	peerID := make([]byte, 64)
	randomize(peerID)
	peer, _ := w.getPeer(peerID)
	if peer != nil {
		t.Fatal("found peer for random key.")
	}
	if err := w.MarkPeerTrusted(peerID); err == nil {
		t.Fatalf("failed MarkPeerTrusted.")
	}
	if err := w.RequestHistoricMessages(peerID, peerID); err == nil {
		t.Fatalf("failed RequestHistoricMessages.")
	}
	if err := w.SendP2PMessage(peerID, nil); err == nil {
		t.Fatalf("failed SendP2PMessage.")
	}
	exist := w.HasSymKey("non-existing")
	if exist {
		t.Fatalf("failed HasSymKey.")
	}
	key := w.GetSymKey("non-existing")
	if key != nil {
		t.Fatalf("failed GetSymKey.")
	}
	mail := w.Envelopes()
	if len(mail) != 0 {
		t.Fatalf("failed w.Envelopes().")
	}
	m := w.Messages(0)
	if len(m) != 0 {
		t.Fatalf("failed w.Messages.")
	}

	var derived []byte
	ver := uint64(0xDEADBEEF)
	if _, err := deriveKeyMaterial(peerID, ver); err != unknownVersionError(ver) {
		t.Fatalf("failed deriveKeyMaterial with param = %v: %s.", peerID, err)
	}
	derived, err := deriveKeyMaterial(peerID, 0)
	if err != nil {
		t.Fatalf("failed second deriveKeyMaterial with param = %v: %s.", peerID, err)
	}
	if !validateSymmetricKey(derived) {
		t.Fatalf("failed validateSymmetricKey with param = %v.", derived)
	}
	if containsOnlyZeros(derived) {
		t.Fatalf("failed containsOnlyZeros with param = %v.", derived)
	}

	buf := []byte{0xFF, 0xE5, 0x80, 0x2, 0}
	le := bytesToIntLittleEndian(buf)
	be := BytesToIntBigEndian(buf)
	if le != uint64(0x280e5ff) {
		t.Fatalf("failed bytesToIntLittleEndian: %d.", le)
	}
	if be != uint64(0xffe5800200) {
		t.Fatalf("failed BytesToIntBigEndian: %d.", be)
	}

	pk := w.NewIdentity()
	if !validatePrivateKey(pk) {
		t.Fatalf("failed validatePrivateKey: %v.", pk)
	}
	if !ValidatePublicKey(&pk.PublicKey) {
		t.Fatalf("failed ValidatePublicKey: %v.", pk)
	}
}

func TestWhisperIdentityManagement(t *testing.T) {
	w := NewWhisper(nil)
	id1 := w.NewIdentity()
	id2 := w.NewIdentity()
	pub1 := common.ToHex(crypto.FromECDSAPub(&id1.PublicKey))
	pub2 := common.ToHex(crypto.FromECDSAPub(&id2.PublicKey))
	pk1 := w.GetIdentity(pub1)
	pk2 := w.GetIdentity(pub2)
	if !w.HasIdentity(pub1) {
		t.Fatalf("failed HasIdentity(pub1).")
	}
	if !w.HasIdentity(pub2) {
		t.Fatalf("failed HasIdentity(pub2).")
	}
	if pk1 != id1 {
		t.Fatalf("failed GetIdentity(pub1).")
	}
	if pk2 != id2 {
		t.Fatalf("failed GetIdentity(pub2).")
	}

	// Delete one identity
	w.DeleteIdentity(pub1)
	pk1 = w.GetIdentity(pub1)
	pk2 = w.GetIdentity(pub2)
	if w.HasIdentity(pub1) {
		t.Fatalf("failed DeleteIdentity(pub1): still exist.")
	}
	if !w.HasIdentity(pub2) {
		t.Fatalf("failed DeleteIdentity(pub1): pub2 does not exist.")
	}
	if pk1 != nil {
		t.Fatalf("failed DeleteIdentity(pub1): first key still exist.")
	}
	if pk2 != id2 {
		t.Fatalf("failed DeleteIdentity(pub1): second key does not exist.")
	}

	// Delete again non-existing identity
	w.DeleteIdentity(pub1)
	pk1 = w.GetIdentity(pub1)
	pk2 = w.GetIdentity(pub2)
	if w.HasIdentity(pub1) {
		t.Fatalf("failed delete non-existing identity: exist.")
	}
	if !w.HasIdentity(pub2) {
		t.Fatalf("failed delete non-existing identity: pub2 does not exist.")
	}
	if pk1 != nil {
		t.Fatalf("failed delete non-existing identity: first key exist.")
	}
	if pk2 != id2 {
		t.Fatalf("failed delete non-existing identity: second key does not exist.")
	}

	// Delete second identity
	w.DeleteIdentity(pub2)
	pk1 = w.GetIdentity(pub1)
	pk2 = w.GetIdentity(pub2)
	if w.HasIdentity(pub1) {
		t.Fatalf("failed delete second identity: first identity exist.")
	}
	if w.HasIdentity(pub2) {
		t.Fatalf("failed delete second identity: still exist.")
	}
	if pk1 != nil {
		t.Fatalf("failed delete second identity: first key exist.")
	}
	if pk2 != nil {
		t.Fatalf("failed delete second identity: second key exist.")
	}
}

func TestWhisperSymKeyManagement(t *testing.T) {
	InitSingleTest()

	var k1, k2 []byte
	w := NewWhisper(nil)
	id1 := string("arbitrary-string-1")
	id2 := string("arbitrary-string-2")

	err := w.GenerateSymKey(id1)
	if err != nil {
		t.Fatalf("failed GenerateSymKey with seed %d: %s.", seed, err)
	}

	k1 = w.GetSymKey(id1)
	k2 = w.GetSymKey(id2)
	if !w.HasSymKey(id1) {
		t.Fatalf("failed HasSymKey(id1).")
	}
	if w.HasSymKey(id2) {
		t.Fatalf("failed HasSymKey(id2).")
	}
	if k1 == nil {
		t.Fatalf("first key does not exist.")
	}
	if k2 != nil {
		t.Fatalf("second key still exist.")
	}

	// add existing id, nothing should change
	randomKey := make([]byte, 16)
	randomize(randomKey)
	err = w.AddSymKey(id1, randomKey)
	if err == nil {
		t.Fatalf("failed AddSymKey with seed %d.", seed)
	}

	k1 = w.GetSymKey(id1)
	k2 = w.GetSymKey(id2)
	if !w.HasSymKey(id1) {
		t.Fatalf("failed w.HasSymKey(id1).")
	}
	if w.HasSymKey(id2) {
		t.Fatalf("failed w.HasSymKey(id2).")
	}
	if k1 == nil {
		t.Fatalf("first key does not exist.")
	}
	if bytes.Equal(k1, randomKey) {
		t.Fatalf("k1 == randomKey.")
	}
	if k2 != nil {
		t.Fatalf("second key already exist.")
	}

	err = w.AddSymKey(id2, randomKey) // add non-existing (yet)
	if err != nil {
		t.Fatalf("failed AddSymKey(id2) with seed %d: %s.", seed, err)
	}
	k1 = w.GetSymKey(id1)
	k2 = w.GetSymKey(id2)
	if !w.HasSymKey(id1) {
		t.Fatalf("HasSymKey(id1) failed.")
	}
	if !w.HasSymKey(id2) {
		t.Fatalf("HasSymKey(id2) failed.")
	}
	if k1 == nil {
		t.Fatalf("k1 does not exist.")
	}
	if k2 == nil {
		t.Fatalf("k2 does not exist.")
	}
	if bytes.Equal(k1, k2) {
		t.Fatalf("k1 == k2.")
	}
	if bytes.Equal(k1, randomKey) {
		t.Fatalf("k1 == randomKey.")
	}
	if len(k1) != aesKeyLength {
		t.Fatalf("wrong length of k1.")
	}
	if len(k2) != aesKeyLength {
		t.Fatalf("wrong length of k2.")
	}

	w.DeleteSymKey(id1)
	k1 = w.GetSymKey(id1)
	k2 = w.GetSymKey(id2)
	if w.HasSymKey(id1) {
		t.Fatalf("failed to delete first key: still exist.")
	}
	if !w.HasSymKey(id2) {
		t.Fatalf("failed to delete first key: second key does not exist.")
	}
	if k1 != nil {
		t.Fatalf("failed to delete first key.")
	}
	if k2 == nil {
		t.Fatalf("failed to delete first key: second key is nil.")
	}

	w.DeleteSymKey(id1)
	w.DeleteSymKey(id2)
	k1 = w.GetSymKey(id1)
	k2 = w.GetSymKey(id2)
	if w.HasSymKey(id1) {
		t.Fatalf("failed to delete second key: first key exist.")
	}
	if w.HasSymKey(id2) {
		t.Fatalf("failed to delete second key: still exist.")
	}
	if k1 != nil {
		t.Fatalf("failed to delete second key: first key is not nil.")
	}
	if k2 != nil {
		t.Fatalf("failed to delete second key: second key is not nil.")
	}
}

func TestExpiry(t *testing.T) {
	InitSingleTest()

	w := NewWhisper(nil)
	w.test = true
	w.Start(nil)
	defer w.Stop()

	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}

	params.TTL = 1
	msg := NewSentMessage(params)
	env, err := msg.Wrap(params)
	if err != nil {
		t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}

	err = w.Send(env)
	if err != nil {
		t.Fatalf("failed to send envelope with seed %d: %s.", seed, err)
	}

	// wait till received or timeout
	var received, expired bool
	for j := 0; j < 20; j++ {
		time.Sleep(100 * time.Millisecond)
		if len(w.Envelopes()) > 0 {
			received = true
			break
		}
	}

	if !received {
		t.Fatalf("did not receive the sent envelope, seed: %d.", seed)
	}

	// wait till expired or timeout
	for j := 0; j < 20; j++ {
		time.Sleep(100 * time.Millisecond)
		if len(w.Envelopes()) == 0 {
			expired = true
			break
		}
	}

	if !expired {
		t.Fatalf("expire failed, seed: %d.", seed)
	}
}
