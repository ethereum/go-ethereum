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

package whisperv6

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	mrand "math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/pbkdf2"
)

func TestWhisperBasic(t *testing.T) {
	w := New(&DefaultConfig)
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
	if w.GetFilter("non-existent") != nil {
		t.Fatalf("failed GetFilter.")
	}

	peerID := make([]byte, 64)
	mrand.Read(peerID)
	peer, _ := w.getPeer(peerID)
	if peer != nil {
		t.Fatal("found peer for random key.")
	}
	if err := w.AllowP2PMessagesFromPeer(peerID); err == nil {
		t.Fatalf("failed MarkPeerTrusted.")
	}
	exist := w.HasSymKey("non-existing")
	if exist {
		t.Fatalf("failed HasSymKey.")
	}
	key, err := w.GetSymKey("non-existing")
	if err == nil {
		t.Fatalf("failed GetSymKey(non-existing): false positive.")
	}
	if key != nil {
		t.Fatalf("failed GetSymKey: false positive.")
	}
	mail := w.Envelopes()
	if len(mail) != 0 {
		t.Fatalf("failed w.Envelopes().")
	}
	m := w.Messages("non-existent")
	if len(m) != 0 {
		t.Fatalf("failed w.Messages.")
	}

	derived := pbkdf2.Key([]byte(peerID), nil, 65356, aesKeyLength, sha256.New)
	if !validateDataIntegrity(derived, aesKeyLength) {
		t.Fatalf("failed validateSymmetricKey with param = %v.", derived)
	}
	if containsOnlyZeros(derived) {
		t.Fatalf("failed containsOnlyZeros with param = %v.", derived)
	}

	buf := []byte{0xFF, 0xE5, 0x80, 0x2, 0}
	le := bytesToUintLittleEndian(buf)
	be := BytesToUintBigEndian(buf)
	if le != uint64(0x280e5ff) {
		t.Fatalf("failed bytesToIntLittleEndian: %d.", le)
	}
	if be != uint64(0xffe5800200) {
		t.Fatalf("failed BytesToIntBigEndian: %d.", be)
	}

	id, err := w.NewKeyPair()
	if err != nil {
		t.Fatalf("failed to generate new key pair: %s.", err)
	}
	pk, err := w.GetPrivateKey(id)
	if err != nil {
		t.Fatalf("failed to retrieve new key pair: %s.", err)
	}
	if !validatePrivateKey(pk) {
		t.Fatalf("failed validatePrivateKey: %v.", pk)
	}
	if !ValidatePublicKey(&pk.PublicKey) {
		t.Fatalf("failed ValidatePublicKey: %v.", pk)
	}
}

func TestWhisperAsymmetricKeyImport(t *testing.T) {
	var (
		w           = New(&DefaultConfig)
		privateKeys []*ecdsa.PrivateKey
	)

	for i := 0; i < 50; i++ {
		id, err := w.NewKeyPair()
		if err != nil {
			t.Fatalf("could not generate key: %v", err)
		}

		pk, err := w.GetPrivateKey(id)
		if err != nil {
			t.Fatalf("could not export private key: %v", err)
		}

		privateKeys = append(privateKeys, pk)

		if !w.DeleteKeyPair(id) {
			t.Fatalf("could not delete private key")
		}
	}

	for _, pk := range privateKeys {
		if _, err := w.AddKeyPair(pk); err != nil {
			t.Fatalf("could not import private key: %v", err)
		}
	}
}

func TestWhisperIdentityManagement(t *testing.T) {
	w := New(&DefaultConfig)
	id1, err := w.NewKeyPair()
	if err != nil {
		t.Fatalf("failed to generate new key pair: %s.", err)
	}
	id2, err := w.NewKeyPair()
	if err != nil {
		t.Fatalf("failed to generate new key pair: %s.", err)
	}
	pk1, err := w.GetPrivateKey(id1)
	if err != nil {
		t.Fatalf("failed to retrieve the key pair: %s.", err)
	}
	pk2, err := w.GetPrivateKey(id2)
	if err != nil {
		t.Fatalf("failed to retrieve the key pair: %s.", err)
	}

	if !w.HasKeyPair(id1) {
		t.Fatalf("failed HasIdentity(pk1).")
	}
	if !w.HasKeyPair(id2) {
		t.Fatalf("failed HasIdentity(pk2).")
	}
	if pk1 == nil {
		t.Fatalf("failed GetIdentity(pk1).")
	}
	if pk2 == nil {
		t.Fatalf("failed GetIdentity(pk2).")
	}

	if !validatePrivateKey(pk1) {
		t.Fatalf("pk1 is invalid.")
	}
	if !validatePrivateKey(pk2) {
		t.Fatalf("pk2 is invalid.")
	}

	// Delete one identity
	done := w.DeleteKeyPair(id1)
	if !done {
		t.Fatalf("failed to delete id1.")
	}
	pk1, err = w.GetPrivateKey(id1)
	if err == nil {
		t.Fatalf("retrieve the key pair: false positive.")
	}
	pk2, err = w.GetPrivateKey(id2)
	if err != nil {
		t.Fatalf("failed to retrieve the key pair: %s.", err)
	}
	if w.HasKeyPair(id1) {
		t.Fatalf("failed DeleteIdentity(pub1): still exist.")
	}
	if !w.HasKeyPair(id2) {
		t.Fatalf("failed DeleteIdentity(pub1): pub2 does not exist.")
	}
	if pk1 != nil {
		t.Fatalf("failed DeleteIdentity(pub1): first key still exist.")
	}
	if pk2 == nil {
		t.Fatalf("failed DeleteIdentity(pub1): second key does not exist.")
	}

	// Delete again non-existing identity
	done = w.DeleteKeyPair(id1)
	if done {
		t.Fatalf("delete id1: false positive.")
	}
	pk1, err = w.GetPrivateKey(id1)
	if err == nil {
		t.Fatalf("retrieve the key pair: false positive.")
	}
	pk2, err = w.GetPrivateKey(id2)
	if err != nil {
		t.Fatalf("failed to retrieve the key pair: %s.", err)
	}
	if w.HasKeyPair(id1) {
		t.Fatalf("failed delete non-existing identity: exist.")
	}
	if !w.HasKeyPair(id2) {
		t.Fatalf("failed delete non-existing identity: pub2 does not exist.")
	}
	if pk1 != nil {
		t.Fatalf("failed delete non-existing identity: first key exist.")
	}
	if pk2 == nil {
		t.Fatalf("failed delete non-existing identity: second key does not exist.")
	}

	// Delete second identity
	done = w.DeleteKeyPair(id2)
	if !done {
		t.Fatalf("failed to delete id2.")
	}
	pk1, err = w.GetPrivateKey(id1)
	if err == nil {
		t.Fatalf("retrieve the key pair: false positive.")
	}
	pk2, err = w.GetPrivateKey(id2)
	if err == nil {
		t.Fatalf("retrieve the key pair: false positive.")
	}
	if w.HasKeyPair(id1) {
		t.Fatalf("failed delete second identity: first identity exist.")
	}
	if w.HasKeyPair(id2) {
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

	var err error
	var k1, k2 []byte
	w := New(&DefaultConfig)
	id1 := string("arbitrary-string-1")
	id2 := string("arbitrary-string-2")

	id1, err = w.GenerateSymKey()
	if err != nil {
		t.Fatalf("failed GenerateSymKey with seed %d: %s.", seed, err)
	}

	k1, err = w.GetSymKey(id1)
	if err != nil {
		t.Fatalf("failed GetSymKey(id1).")
	}
	k2, err = w.GetSymKey(id2)
	if err == nil {
		t.Fatalf("failed GetSymKey(id2): false positive.")
	}
	if !w.HasSymKey(id1) {
		t.Fatalf("failed HasSymKey(id1).")
	}
	if w.HasSymKey(id2) {
		t.Fatalf("failed HasSymKey(id2): false positive.")
	}
	if k1 == nil {
		t.Fatalf("first key does not exist.")
	}
	if k2 != nil {
		t.Fatalf("second key still exist.")
	}

	// add existing id, nothing should change
	randomKey := make([]byte, aesKeyLength)
	mrand.Read(randomKey)
	id1, err = w.AddSymKeyDirect(randomKey)
	if err != nil {
		t.Fatalf("failed AddSymKey with seed %d: %s.", seed, err)
	}

	k1, err = w.GetSymKey(id1)
	if err != nil {
		t.Fatalf("failed w.GetSymKey(id1).")
	}
	k2, err = w.GetSymKey(id2)
	if err == nil {
		t.Fatalf("failed w.GetSymKey(id2): false positive.")
	}
	if !w.HasSymKey(id1) {
		t.Fatalf("failed w.HasSymKey(id1).")
	}
	if w.HasSymKey(id2) {
		t.Fatalf("failed w.HasSymKey(id2): false positive.")
	}
	if k1 == nil {
		t.Fatalf("first key does not exist.")
	}
	if !bytes.Equal(k1, randomKey) {
		t.Fatalf("k1 != randomKey.")
	}
	if k2 != nil {
		t.Fatalf("second key already exist.")
	}

	id2, err = w.AddSymKeyDirect(randomKey)
	if err != nil {
		t.Fatalf("failed AddSymKey(id2) with seed %d: %s.", seed, err)
	}
	k1, err = w.GetSymKey(id1)
	if err != nil {
		t.Fatalf("failed w.GetSymKey(id1).")
	}
	k2, err = w.GetSymKey(id2)
	if err != nil {
		t.Fatalf("failed w.GetSymKey(id2).")
	}
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
	if !bytes.Equal(k1, k2) {
		t.Fatalf("k1 != k2.")
	}
	if !bytes.Equal(k1, randomKey) {
		t.Fatalf("k1 != randomKey.")
	}
	if len(k1) != aesKeyLength {
		t.Fatalf("wrong length of k1.")
	}
	if len(k2) != aesKeyLength {
		t.Fatalf("wrong length of k2.")
	}

	w.DeleteSymKey(id1)
	k1, err = w.GetSymKey(id1)
	if err == nil {
		t.Fatalf("failed w.GetSymKey(id1): false positive.")
	}
	if k1 != nil {
		t.Fatalf("failed GetSymKey(id1): false positive.")
	}
	k2, err = w.GetSymKey(id2)
	if err != nil {
		t.Fatalf("failed w.GetSymKey(id2).")
	}
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
	k1, err = w.GetSymKey(id1)
	if err == nil {
		t.Fatalf("failed w.GetSymKey(id1): false positive.")
	}
	k2, err = w.GetSymKey(id2)
	if err == nil {
		t.Fatalf("failed w.GetSymKey(id2): false positive.")
	}
	if k1 != nil || k2 != nil {
		t.Fatalf("k1 or k2 is not nil")
	}
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

	randomKey = make([]byte, aesKeyLength+1)
	mrand.Read(randomKey)
	_, err = w.AddSymKeyDirect(randomKey)
	if err == nil {
		t.Fatalf("added the key with wrong size, seed %d.", seed)
	}

	const password = "arbitrary data here"
	id1, err = w.AddSymKeyFromPassword(password)
	if err != nil {
		t.Fatalf("failed AddSymKeyFromPassword(id1) with seed %d: %s.", seed, err)
	}
	id2, err = w.AddSymKeyFromPassword(password)
	if err != nil {
		t.Fatalf("failed AddSymKeyFromPassword(id2) with seed %d: %s.", seed, err)
	}
	k1, err = w.GetSymKey(id1)
	if err != nil {
		t.Fatalf("failed w.GetSymKey(id1).")
	}
	k2, err = w.GetSymKey(id2)
	if err != nil {
		t.Fatalf("failed w.GetSymKey(id2).")
	}
	if !w.HasSymKey(id1) {
		t.Fatalf("HasSymKey(id1) failed.")
	}
	if !w.HasSymKey(id2) {
		t.Fatalf("HasSymKey(id2) failed.")
	}
	if !validateDataIntegrity(k2, aesKeyLength) {
		t.Fatalf("key validation failed.")
	}
	if !bytes.Equal(k1, k2) {
		t.Fatalf("k1 != k2.")
	}
}

func TestExpiry(t *testing.T) {
	InitSingleTest()

	w := New(&DefaultConfig)
	w.SetMinimumPowTest(0.0000001)
	defer w.SetMinimumPowTest(DefaultMinimumPoW)
	w.Start(nil)
	defer w.Stop()

	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}

	params.TTL = 1
	msg, err := NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
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

func TestCustomization(t *testing.T) {
	InitSingleTest()

	w := New(&DefaultConfig)
	defer w.SetMinimumPowTest(DefaultMinimumPoW)
	defer w.SetMaxMessageSize(DefaultMaxMessageSize)
	w.Start(nil)
	defer w.Stop()

	const smallPoW = 0.00001

	f, err := generateFilter(t, true)
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}
	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}

	params.KeySym = f.KeySym
	params.Topic = BytesToTopic(f.Topics[2])
	params.PoW = smallPoW
	params.TTL = 3600 * 24 // one day
	msg, err := NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	env, err := msg.Wrap(params)
	if err != nil {
		t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}

	err = w.Send(env)
	if err == nil {
		t.Fatalf("successfully sent envelope with PoW %.06f, false positive (seed %d).", env.PoW(), seed)
	}

	w.SetMinimumPowTest(smallPoW / 2)
	err = w.Send(env)
	if err != nil {
		t.Fatalf("failed to send envelope with seed %d: %s.", seed, err)
	}

	params.TTL++
	msg, err = NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	env, err = msg.Wrap(params)
	if err != nil {
		t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}
	w.SetMaxMessageSize(uint32(env.size() - 1))
	err = w.Send(env)
	if err == nil {
		t.Fatalf("successfully sent oversized envelope (seed %d): false positive.", seed)
	}

	w.SetMaxMessageSize(DefaultMaxMessageSize)
	err = w.Send(env)
	if err != nil {
		t.Fatalf("failed to send second envelope with seed %d: %s.", seed, err)
	}

	// wait till received or timeout
	var received bool
	for j := 0; j < 20; j++ {
		time.Sleep(100 * time.Millisecond)
		if len(w.Envelopes()) > 1 {
			received = true
			break
		}
	}

	if !received {
		t.Fatalf("did not receive the sent envelope, seed: %d.", seed)
	}

	// check w.messages()
	id, err := w.Subscribe(f)
	if err != nil {
		t.Fatalf("failed subscribe with seed %d: %s.", seed, err)
	}
	time.Sleep(5 * time.Millisecond)
	mail := f.Retrieve()
	if len(mail) > 0 {
		t.Fatalf("received premature mail")
	}

	mail = w.Messages(id)
	if len(mail) != 2 {
		t.Fatalf("failed to get whisper messages")
	}
}

func TestSymmetricSendCycle(t *testing.T) {
	InitSingleTest()

	w := New(&DefaultConfig)
	defer w.SetMinimumPowTest(DefaultMinimumPoW)
	defer w.SetMaxMessageSize(DefaultMaxMessageSize)
	w.Start(nil)
	defer w.Stop()

	filter1, err := generateFilter(t, true)
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}
	filter1.PoW = DefaultMinimumPoW

	// Copy the first filter since some of its fields
	// are randomly gnerated.
	filter2 := &Filter{
		KeySym:   filter1.KeySym,
		Topics:   filter1.Topics,
		PoW:      filter1.PoW,
		AllowP2P: filter1.AllowP2P,
		Messages: make(map[common.Hash]*ReceivedMessage),
	}

	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}

	filter1.Src = &params.Src.PublicKey
	filter2.Src = &params.Src.PublicKey

	params.KeySym = filter1.KeySym
	params.Topic = BytesToTopic(filter1.Topics[2])
	params.PoW = filter1.PoW
	params.WorkTime = 10
	params.TTL = 50
	msg, err := NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	env, err := msg.Wrap(params)
	if err != nil {
		t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}

	_, err = w.Subscribe(filter1)
	if err != nil {
		t.Fatalf("failed subscribe 1 with seed %d: %s.", seed, err)
	}

	_, err = w.Subscribe(filter2)
	if err != nil {
		t.Fatalf("failed subscribe 2 with seed %d: %s.", seed, err)
	}

	err = w.Send(env)
	if err != nil {
		t.Fatalf("Failed sending envelope with PoW %.06f (seed %d): %s", env.PoW(), seed, err)
	}

	// wait till received or timeout
	var received bool
	for j := 0; j < 200; j++ {
		time.Sleep(10 * time.Millisecond)
		if len(w.Envelopes()) > 0 {
			received = true
			break
		}
	}

	if !received {
		t.Fatalf("did not receive the sent envelope, seed: %d.", seed)
	}

	// check w.messages()
	time.Sleep(5 * time.Millisecond)
	mail1 := filter1.Retrieve()
	mail2 := filter2.Retrieve()
	if len(mail2) == 0 {
		t.Fatalf("did not receive any email for filter 2")
	}
	if len(mail1) == 0 {
		t.Fatalf("did not receive any email for filter 1")
	}

}

func TestSymmetricSendWithoutAKey(t *testing.T) {
	InitSingleTest()

	w := New(&DefaultConfig)
	defer w.SetMinimumPowTest(DefaultMinimumPoW)
	defer w.SetMaxMessageSize(DefaultMaxMessageSize)
	w.Start(nil)
	defer w.Stop()

	filter, err := generateFilter(t, true)
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}
	filter.PoW = DefaultMinimumPoW

	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}

	filter.Src = nil

	params.KeySym = filter.KeySym
	params.Topic = BytesToTopic(filter.Topics[2])
	params.PoW = filter.PoW
	params.WorkTime = 10
	params.TTL = 50
	msg, err := NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	env, err := msg.Wrap(params)
	if err != nil {
		t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}

	_, err = w.Subscribe(filter)
	if err != nil {
		t.Fatalf("failed subscribe 1 with seed %d: %s.", seed, err)
	}

	err = w.Send(env)
	if err != nil {
		t.Fatalf("Failed sending envelope with PoW %.06f (seed %d): %s", env.PoW(), seed, err)
	}

	// wait till received or timeout
	var received bool
	for j := 0; j < 200; j++ {
		time.Sleep(10 * time.Millisecond)
		if len(w.Envelopes()) > 0 {
			received = true
			break
		}
	}

	if !received {
		t.Fatalf("did not receive the sent envelope, seed: %d.", seed)
	}

	// check w.messages()
	time.Sleep(5 * time.Millisecond)
	mail := filter.Retrieve()
	if len(mail) == 0 {
		t.Fatalf("did not receive message in spite of not setting a public key")
	}
}

func TestSymmetricSendKeyMismatch(t *testing.T) {
	InitSingleTest()

	w := New(&DefaultConfig)
	defer w.SetMinimumPowTest(DefaultMinimumPoW)
	defer w.SetMaxMessageSize(DefaultMaxMessageSize)
	w.Start(nil)
	defer w.Stop()

	filter, err := generateFilter(t, true)
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}
	filter.PoW = DefaultMinimumPoW

	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}

	params.KeySym = filter.KeySym
	params.Topic = BytesToTopic(filter.Topics[2])
	params.PoW = filter.PoW
	params.WorkTime = 10
	params.TTL = 50
	msg, err := NewSentMessage(params)
	if err != nil {
		t.Fatalf("failed to create new message with seed %d: %s.", seed, err)
	}
	env, err := msg.Wrap(params)
	if err != nil {
		t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}

	_, err = w.Subscribe(filter)
	if err != nil {
		t.Fatalf("failed subscribe 1 with seed %d: %s.", seed, err)
	}

	err = w.Send(env)
	if err != nil {
		t.Fatalf("Failed sending envelope with PoW %.06f (seed %d): %s", env.PoW(), seed, err)
	}

	// wait till received or timeout
	var received bool
	for j := 0; j < 200; j++ {
		time.Sleep(10 * time.Millisecond)
		if len(w.Envelopes()) > 0 {
			received = true
			break
		}
	}

	if !received {
		t.Fatalf("did not receive the sent envelope, seed: %d.", seed)
	}

	// check w.messages()
	time.Sleep(5 * time.Millisecond)
	mail := filter.Retrieve()
	if len(mail) > 0 {
		t.Fatalf("received a message when keys weren't matching")
	}
}

func TestBloom(t *testing.T) {
	topic := TopicType{0, 0, 255, 6}
	b := TopicToBloom(topic)
	x := make([]byte, bloomFilterSize)
	x[0] = byte(1)
	x[32] = byte(1)
	x[bloomFilterSize-1] = byte(128)
	if !bloomFilterMatch(x, b) || !bloomFilterMatch(b, x) {
		t.Fatalf("bloom filter does not match the mask")
	}

	_, err := mrand.Read(b)
	if err != nil {
		t.Fatalf("math rand error")
	}
	_, err = mrand.Read(x)
	if err != nil {
		t.Fatalf("math rand error")
	}
	if !bloomFilterMatch(b, b) {
		t.Fatalf("bloom filter does not match self")
	}
	x = addBloom(x, b)
	if !bloomFilterMatch(x, b) {
		t.Fatalf("bloom filter does not match combined bloom")
	}
	if !isFullNode(nil) {
		t.Fatalf("isFullNode did not recognize nil as full node")
	}
	x[17] = 254
	if isFullNode(x) {
		t.Fatalf("isFullNode false positive")
	}
	for i := 0; i < bloomFilterSize; i++ {
		b[i] = byte(255)
	}
	if !isFullNode(b) {
		t.Fatalf("isFullNode false negative")
	}
	if bloomFilterMatch(x, b) {
		t.Fatalf("bloomFilterMatch false positive")
	}
	if !bloomFilterMatch(b, x) {
		t.Fatalf("bloomFilterMatch false negative")
	}

	w := New(&DefaultConfig)
	f := w.BloomFilter()
	if f != nil {
		t.Fatalf("wrong bloom on creation")
	}
	err = w.SetBloomFilter(x)
	if err != nil {
		t.Fatalf("failed to set bloom filter: %s", err)
	}
	f = w.BloomFilter()
	if !bloomFilterMatch(f, x) || !bloomFilterMatch(x, f) {
		t.Fatalf("retireved wrong bloom filter")
	}
}
