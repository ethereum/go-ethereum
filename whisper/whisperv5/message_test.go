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
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

func copyFromBuf(dst []byte, src []byte, beg int) int {
	copy(dst, src[beg:])
	return beg + len(dst)
}

func generateMessageParams(seed int64) (*MessageParams, error) {
	var err error
	var p MessageParams

	buf := make([]byte, 1024)
	_, err = rand.Read(buf)
	if err != nil {
		return nil, err
	}

	p.TTL = uint32(rand.Intn(1024))
	sz := rand.Intn(400)
	p.Payload = make([]byte, sz)
	p.Padding = make([]byte, padSizeLimitUpper)
	p.KeySym = make([]byte, aesKeyLength)

	var b int
	b = copyFromBuf(p.Payload, buf, b)
	b = copyFromBuf(p.Padding, buf, b)
	b = copyFromBuf(p.KeySym, buf, b)
	p.Topic = BytesToTopic(buf[b:])

	p.Src, err = crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	// p.Dst, p.PoW, p.WorkTime are not set
	return &p, nil
}

func singleMessageTest(x *testing.T, seed int64, symmetric bool) {
	params, err := generateMessageParams(seed)
	if err != nil {
		x.Errorf("failed generateMessageParams with seed %d: %s.", seed, err)
		return
	}

	key, err := crypto.GenerateKey()
	if err != nil {
		x.Errorf("failed GenerateKey with seed %d: %s.", seed, err)
		return
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
		x.Errorf("failed Wrap with seed %d: %s.", seed, err)
		return
	}

	var decrypted *ReceivedMessage
	if symmetric {
		decrypted, err = env.OpenSymmetric(params.KeySym)
	} else {
		decrypted, err = env.OpenAsymmetric(key)
	}

	if err != nil {
		x.Errorf("failed to encrypt with seed %d: %s.", seed, err)
		return
	}

	if !decrypted.Validate() {
		x.Errorf("failed to validate with seed %d.", seed)
		return
	}

	padsz := len(decrypted.Padding)
	if bytes.Compare(steg[:padsz], decrypted.Padding) != 0 {
		x.Errorf("failed with seed %d: compare padding.", seed)
		return
	}
	if bytes.Compare(text, decrypted.Payload) != 0 {
		x.Errorf("failed with seed %d: compare payload.", seed)
		return
	}
	if !isMessageSigned(decrypted.Raw[0]) {
		x.Errorf("failed with seed %d: unsigned.", seed)
		return
	}
	if len(decrypted.Signature) != signatureLength {
		x.Errorf("failed with seed %d: signature len %d.", seed, len(decrypted.Signature))
		return
	}
	if !isPubKeyEqual(decrypted.Src, &params.Src.PublicKey) {
		x.Errorf("failed with seed %d: signature mismatch.", seed)
		return
	}
}

func TestMessageEncryption(x *testing.T) {
	seed := time.Now().Unix()
	rand.Seed(seed)
	var symmetric bool

	for i := 0; i < 256; i++ {
		singleMessageTest(x, seed, symmetric)
		symmetric = !symmetric
	}
}

func TestMessageWrap(x *testing.T) {
	seed := int64(1777444222)
	rand.Seed(seed)
	target := 128.0

	params, err := generateMessageParams(seed)
	if err != nil {
		x.Errorf("failed generateMessageParams with seed %d: %s.", seed, err)
		return
	}

	msg := NewSentMessage(params)
	params.TTL = 1
	params.WorkTime = 12
	params.PoW = target
	env, err := msg.Wrap(params)
	if err != nil {
		x.Errorf("failed Wrap with seed %d: %s.", seed, err)
		return
	}

	pow := env.PoW()
	if pow < target {
		x.Errorf("failed Wrap with seed %d: pow < target (%f vs. %f).", seed, pow, target)
		return
	}
}

func TestMessageSeal(x *testing.T) {
	seed := int64(1976726903)
	rand.Seed(seed)

	params, err := generateMessageParams(seed)
	if err != nil {
		x.Errorf("failed generateMessageParams with seed %d: %s.", seed, err)
		return
	}

	msg := NewSentMessage(params)
	params.TTL = 1
	aesnonce := make([]byte, 12)
	salt := make([]byte, 12)
	_, err = rand.Read(aesnonce)
	if err != nil {
		x.Errorf("failed generate aesnonce with seed %d: %s.", seed, err)
		return
	}
	_, err = rand.Read(salt)
	if err != nil {
		x.Errorf("failed generate salt with seed %d: %s.", seed, err)
		return
	}

	env := NewEnvelope(params.TTL, params.Topic, salt, aesnonce, msg)
	if err != nil {
		x.Errorf("failed Wrap with seed %d: %s.", seed, err)
		return
	}

	env.Expiry = uint32(seed) // make it deterministic
	target := 32.0
	params.WorkTime = 4
	params.PoW = target
	env.Seal(params)

	env.calculatePoW(0)
	pow := env.PoW()
	if pow < target {
		x.Errorf("failed Wrap with seed %d: pow < target (%f vs. %f).", seed, pow, target)
		return
	}

	params.WorkTime = 1
	params.PoW = 1000000000.0
	env.Seal(params)
	env.calculatePoW(0)
	pow = env.PoW()
	if pow < 1000 {
		// this depends on deterministic choice of seed (1976726903)
		x.Errorf("failed Wrap with seed %d: pow too small %f.", seed, pow)
		return
	}
}
