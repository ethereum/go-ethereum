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
		x.Errorf("failed Protocol Name: %v.", shh.Name)
		return
	}
	if uint64(shh.Version) != ProtocolVersion {
		x.Errorf("failed Protocol Version: %v.", shh.Version)
		return
	}
	if shh.Length != NumberOfMessageCodes {
		x.Errorf("failed Protocol Length: %v.", shh.Length)
		return
	}
	if shh.Run == nil {
		x.Errorf("failed shh.Run.")
		return
	}
	if uint64(w.Version()) != ProtocolVersion {
		x.Errorf("failed whisper Version: %v.", shh.Version)
		return
	}
	if w.GetFilter(0) != nil {
		x.Errorf("failed GetFilter.")
		return
	}

	peerID := make([]byte, 64)
	randomize(peerID)
	peer, err := w.getPeer(peerID)
	if peer != nil {
		x.Errorf("failed GetPeer.")
		return
	}
	err = w.MarkPeerTrusted(peerID)
	if err == nil {
		x.Errorf("failed MarkPeerTrusted.")
		return
	}
	err = w.RequestHistoricMessages(peerID, peerID)
	if err == nil {
		x.Errorf("failed RequestHistoricMessages.")
		return
	}
	err = w.SendP2PMessage(peerID, nil)
	if err == nil {
		x.Errorf("failed SendP2PMessage.")
		return
	}
	exist := w.HasSymKey("non-existing")
	if exist {
		x.Errorf("failed HasSymKey.")
		return
	}
	key := w.GetSymKey("non-existing")
	if key != nil {
		x.Errorf("failed GetSymKey.")
		return
	}
	mail := w.Envelopes()
	if len(mail) != 0 {
		x.Errorf("failed w.Envelopes().")
		return
	}
	m := w.Messages(0)
	if len(m) != 0 {
		x.Errorf("failed w.Messages.")
		return
	}

	var derived []byte
	ver := uint64(0xDEADBEEF)
	derived, err = deriveKeyMaterial(peerID, ver)
	if err != unknownVersionError(ver) {
		x.Errorf("failed deriveKeyMaterial 1 with param = %v: %s.", peerID, err)
		return
	}
	derived, err = deriveKeyMaterial(peerID, 0)
	if err != nil {
		x.Errorf("failed deriveKeyMaterial 2 with param = %v: %s.", peerID, err)
		return
	}
	if !validateSymmetricKey(derived) {
		x.Errorf("failed validateSymmetricKey with param = %v.", derived)
		return
	}
	if containsOnlyZeros(derived) {
		x.Errorf("failed containsOnlyZeros with param = %v.", derived)
		return
	}

	buf := []byte{0xFF, 0xE5, 0x80, 0x2, 0}
	le := bytesToIntLittleEndian(buf)
	be := BytesToIntBigEndian(buf)
	if le != uint64(0x280e5ff) {
		x.Errorf("failed bytesToIntLittleEndian: %d.", le)
		return
	}
	if be != uint64(0xffe5800200) {
		x.Errorf("failed BytesToIntBigEndian: %d.", be)
		return
	}

	pk := w.NewIdentity()
	if !validatePrivateKey(pk) {
		x.Errorf("failed validatePrivateKey: %v.", pk)
		return
	}
	if !ValidatePublicKey(&pk.PublicKey) {
		x.Errorf("failed ValidatePublicKey: %v.", pk)
		return
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
		x.Errorf("failed HasIdentity 1.")
		return
	}
	if !w.HasIdentity(pub2) {
		x.Errorf("failed HasIdentity 2.")
		return
	}
	if pk1 != id1 {
		x.Errorf("failed GetIdentity 3.")
		return
	}
	if pk2 != id2 {
		x.Errorf("failed GetIdentity 4.")
		return
	}

	// Delete one identity
	w.DeleteIdentity(pub1)
	pk1 = w.GetIdentity(pub1)
	pk2 = w.GetIdentity(pub2)
	if w.HasIdentity(pub1) {
		x.Errorf("failed HasIdentity 11.")
		return
	}
	if !w.HasIdentity(pub2) {
		x.Errorf("failed HasIdentity 12.")
		return
	}
	if pk1 != nil {
		x.Errorf("failed GetIdentity 13.")
		return
	}
	if pk2 != id2 {
		x.Errorf("failed GetIdentity 14.")
		return
	}

	// Delete again non-existing identity
	w.DeleteIdentity(pub1)
	pk1 = w.GetIdentity(pub1)
	pk2 = w.GetIdentity(pub2)
	if w.HasIdentity(pub1) {
		x.Errorf("failed HasIdentity 21.")
		return
	}
	if !w.HasIdentity(pub2) {
		x.Errorf("failed HasIdentity 22.")
		return
	}
	if pk1 != nil {
		x.Errorf("failed GetIdentity 23.")
		return
	}
	if pk2 != id2 {
		x.Errorf("failed GetIdentity 24.")
		return
	}

	// Delete second identity
	w.DeleteIdentity(pub2)
	pk1 = w.GetIdentity(pub1)
	pk2 = w.GetIdentity(pub2)
	if w.HasIdentity(pub1) {
		x.Errorf("failed HasIdentity 31.")
		return
	}
	if w.HasIdentity(pub2) {
		x.Errorf("failed HasIdentity 32.")
		return
	}
	if pk1 != nil {
		x.Errorf("failed GetIdentity 33.")
		return
	}
	if pk2 != nil {
		x.Errorf("failed GetIdentity 34.")
		return
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
		x.Errorf("failed test case 1 with seed %d: %s.", seed, err)
		return
	}

	k1 = w.GetSymKey(id1)
	k2 = w.GetSymKey(id2)
	if !w.HasSymKey(id1) {
		x.Errorf("failed HasIdentity 2.")
		return
	}
	if w.HasSymKey(id2) {
		x.Errorf("failed HasIdentity 3.")
		return
	}
	if k1 == nil {
		x.Errorf("failed GetIdentity 4.")
		return
	}
	if k2 != nil {
		x.Errorf("failed GetIdentity 5.")
		return
	}

	// add existing id, nothing should change
	randomKey := make([]byte, 16)
	randomize(randomKey)
	err = w.AddSymKey(id1, randomKey)
	if err == nil {
		x.Errorf("failed test case 10 with seed %d.", seed)
		return
	}

	k1 = w.GetSymKey(id1)
	k2 = w.GetSymKey(id2)
	if !w.HasSymKey(id1) {
		x.Errorf("failed HasIdentity 12.")
		return
	}
	if w.HasSymKey(id2) {
		x.Errorf("failed HasIdentity 13.")
		return
	}
	if k1 == nil {
		x.Errorf("failed GetIdentity 14.")
		return
	}
	if bytes.Compare(k1, randomKey) == 0 {
		x.Errorf("failed GetIdentity 15: k1 == randomKey.")
		return
	}
	if k2 != nil {
		x.Errorf("failed GetIdentity 16.")
		return
	}

	err = w.AddSymKey(id2, randomKey) // add non-existing (yet)
	if err != nil {
		x.Errorf("failed test case 21 with seed %d: %s.", seed, err)
		return
	}
	k1 = w.GetSymKey(id1)
	k2 = w.GetSymKey(id2)
	if !w.HasSymKey(id1) {
		x.Errorf("failed HasIdentity 22.")
		return
	}
	if !w.HasSymKey(id2) {
		x.Errorf("failed HasIdentity 23.")
		return
	}
	if k1 == nil {
		x.Errorf("failed GetIdentity 24.")
		return
	}
	if k2 == nil {
		x.Errorf("failed GetIdentity 25.")
		return
	}
	if bytes.Compare(k1, k2) == 0 {
		x.Errorf("failed GetIdentity 26.")
		return
	}
	if bytes.Compare(k1, randomKey) == 0 {
		x.Errorf("failed GetIdentity 27.")
		return
	}
	if len(k1) != aesKeyLength {
		x.Errorf("failed GetIdentity 28.")
		return
	}
	if len(k2) != aesKeyLength {
		x.Errorf("failed GetIdentity 29.")
		return
	}

	w.DeleteSymKey(id1)
	k1 = w.GetSymKey(id1)
	k2 = w.GetSymKey(id2)
	if w.HasSymKey(id1) {
		x.Errorf("failed HasIdentity 31.")
		return
	}
	if !w.HasSymKey(id2) {
		x.Errorf("failed HasIdentity 32.")
		return
	}
	if k1 != nil {
		x.Errorf("failed GetIdentity 33.")
		return
	}
	if k2 == nil {
		x.Errorf("failed GetIdentity 34.")
		return
	}

	w.DeleteSymKey(id1)
	w.DeleteSymKey(id2)
	k1 = w.GetSymKey(id1)
	k2 = w.GetSymKey(id2)
	if w.HasSymKey(id1) {
		x.Errorf("failed HasIdentity 41.")
		return
	}
	if w.HasSymKey(id2) {
		x.Errorf("failed HasIdentity 42.")
		return
	}
	if k1 != nil {
		x.Errorf("failed GetIdentity 43.")
		return
	}
	if k2 != nil {
		x.Errorf("failed GetIdentity 44.")
		return
	}
}
