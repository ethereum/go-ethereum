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
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func copyFromBuf(dst []byte, src []byte, beg int) int {
	copy(dst, src[beg:])
	return beg + len(dst)
}

func generateMessageParams() (*MessageParams, error) {
	// set all the parameters except p.Dst

	buf := make([]byte, 1024)
	randomize(buf)
	sz := rand.Intn(400)

	var p MessageParams
	p.PoW = 0.01
	p.WorkTime = 1
	p.TTL = uint32(rand.Intn(1024))
	p.Payload = make([]byte, sz)
	p.Padding = make([]byte, padSizeLimitUpper)
	p.KeySym = make([]byte, aesKeyLength)

	var b int
	b = copyFromBuf(p.Payload, buf, b)
	b = copyFromBuf(p.Padding, buf, b)
	b = copyFromBuf(p.KeySym, buf, b)
	p.Topic = BytesToTopic(buf[b:])

	var err error
	p.Src, err = crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func singleMessageTest(t *testing.T, symmetric bool) {
	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed GenerateKey with seed %d: %s.", seed, err)
	}

	if !symmetric {
		params.KeySym = nil
		params.Dst = &key.PublicKey
	}

	text := make([]byte, 0, 512)
	steg := make([]byte, 0, 512)
	raw := make([]byte, 0, 1024)
	text = append(text, params.Payload...)
	steg = append(steg, params.Padding...)
	raw = append(raw, params.Padding...)

	msg := NewSentMessage(params)
	env, err := msg.Wrap(params)
	if err != nil {
		t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}

	var decrypted *ReceivedMessage
	if symmetric {
		decrypted, err = env.OpenSymmetric(params.KeySym)
	} else {
		decrypted, err = env.OpenAsymmetric(key)
	}

	if err != nil {
		t.Fatalf("failed to encrypt with seed %d: %s.", seed, err)
	}

	if !decrypted.Validate() {
		t.Fatalf("failed to validate with seed %d.", seed)
	}

	padsz := len(decrypted.Padding)
	if !bytes.Equal(steg[:padsz], decrypted.Padding) {
		t.Fatalf("failed with seed %d: compare padding.", seed)
	}
	if !bytes.Equal(text, decrypted.Payload) {
		t.Fatalf("failed with seed %d: compare payload.", seed)
	}
	if !isMessageSigned(decrypted.Raw[0]) {
		t.Fatalf("failed with seed %d: unsigned.", seed)
	}
	if len(decrypted.Signature) != signatureLength {
		t.Fatalf("failed with seed %d: signature len %d.", seed, len(decrypted.Signature))
	}
	if !IsPubKeyEqual(decrypted.Src, &params.Src.PublicKey) {
		t.Fatalf("failed with seed %d: signature mismatch.", seed)
	}
}

func TestMessageEncryption(t *testing.T) {
	InitSingleTest()

	var symmetric bool
	for i := 0; i < 256; i++ {
		singleMessageTest(t, symmetric)
		symmetric = !symmetric
	}
}

func TestMessageWrap(t *testing.T) {
	seed = int64(1777444222)
	rand.Seed(seed)
	target := 128.0

	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}

	msg := NewSentMessage(params)
	params.TTL = 1
	params.WorkTime = 12
	params.PoW = target
	env, err := msg.Wrap(params)
	if err != nil {
		t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}

	pow := env.PoW()
	if pow < target {
		t.Fatalf("failed Wrap with seed %d: pow < target (%f vs. %f).", seed, pow, target)
	}

	// set PoW target too high, expect error
	msg2 := NewSentMessage(params)
	params.TTL = 1000000
	params.WorkTime = 1
	params.PoW = 10000000.0
	env, err = msg2.Wrap(params)
	if err == nil {
		t.Fatalf("unexpectedly reached the PoW target with seed %d.", seed)
	}
}

func TestMessageSeal(t *testing.T) {
	// this test depends on deterministic choice of seed (1976726903)
	seed = int64(1976726903)
	rand.Seed(seed)

	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}

	msg := NewSentMessage(params)
	params.TTL = 1
	aesnonce := make([]byte, 12)
	salt := make([]byte, 12)
	randomize(aesnonce)
	randomize(salt)

	env := NewEnvelope(params.TTL, params.Topic, salt, aesnonce, msg)
	if err != nil {
		t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}

	env.Expiry = uint32(seed) // make it deterministic
	target := 32.0
	params.WorkTime = 4
	params.PoW = target
	env.Seal(params)

	env.calculatePoW(0)
	pow := env.PoW()
	if pow < target {
		t.Fatalf("failed Wrap with seed %d: pow < target (%f vs. %f).", seed, pow, target)
	}

	params.WorkTime = 1
	params.PoW = 1000000000.0
	env.Seal(params)
	env.calculatePoW(0)
	pow = env.PoW()
	if pow < 2*target {
		t.Fatalf("failed Wrap with seed %d: pow too small %f.", seed, pow)
	}
}

func TestEnvelopeOpen(t *testing.T) {
	InitSingleTest()

	var symmetric bool
	for i := 0; i < 256; i++ {
		singleEnvelopeOpenTest(t, symmetric)
		symmetric = !symmetric
	}
}

func singleEnvelopeOpenTest(t *testing.T, symmetric bool) {
	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed GenerateKey with seed %d: %s.", seed, err)
	}

	if !symmetric {
		params.KeySym = nil
		params.Dst = &key.PublicKey
	}

	text := make([]byte, 0, 512)
	steg := make([]byte, 0, 512)
	raw := make([]byte, 0, 1024)
	text = append(text, params.Payload...)
	steg = append(steg, params.Padding...)
	raw = append(raw, params.Padding...)

	msg := NewSentMessage(params)
	env, err := msg.Wrap(params)
	if err != nil {
		t.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}

	f := Filter{KeyAsym: key, KeySym: params.KeySym}
	decrypted := env.Open(&f)
	if decrypted == nil {
		t.Fatalf("failed to open with seed %d.", seed)
	}

	padsz := len(decrypted.Padding)
	if !bytes.Equal(steg[:padsz], decrypted.Padding) {
		t.Fatalf("failed with seed %d: compare padding.", seed)
	}
	if !bytes.Equal(text, decrypted.Payload) {
		t.Fatalf("failed with seed %d: compare payload.", seed)
	}
	if !isMessageSigned(decrypted.Raw[0]) {
		t.Fatalf("failed with seed %d: unsigned.", seed)
	}
	if len(decrypted.Signature) != signatureLength {
		t.Fatalf("failed with seed %d: signature len %d.", seed, len(decrypted.Signature))
	}
	if !IsPubKeyEqual(decrypted.Src, &params.Src.PublicKey) {
		t.Fatalf("failed with seed %d: signature mismatch.", seed)
	}
	if decrypted.isAsymmetricEncryption() == symmetric {
		t.Fatalf("failed with seed %d: asymmetric %v vs. %v.", seed, decrypted.isAsymmetricEncryption(), symmetric)
	}
	if decrypted.isSymmetricEncryption() != symmetric {
		t.Fatalf("failed with seed %d: symmetric %v vs. %v.", seed, decrypted.isSymmetricEncryption(), symmetric)
	}
	if !symmetric {
		if decrypted.Dst == nil {
			t.Fatalf("failed with seed %d: dst is nil.", seed)
		}
		if !IsPubKeyEqual(decrypted.Dst, &key.PublicKey) {
			t.Fatalf("failed with seed %d: Dst.", seed)
		}
	}
}

func TestEncryptWithZeroKey(t *testing.T) {
	InitSingleTest()

	params, err := generateMessageParams()
	if err != nil {
		t.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}

	msg := NewSentMessage(params)

	params.KeySym = make([]byte, aesKeyLength)
	_, err = msg.Wrap(params)
	if err == nil {
		t.Fatalf("wrapped with zero key, seed: %d.", seed)
	}

	params.KeySym = make([]byte, 0)
	_, err = msg.Wrap(params)
	if err == nil {
		t.Fatalf("wrapped with empty key, seed: %d.", seed)
	}

	params.KeySym = nil
	_, err = msg.Wrap(params)
	if err == nil {
		t.Fatalf("wrapped with nil key, seed: %d.", seed)
	}
}
