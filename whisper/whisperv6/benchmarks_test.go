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
	"crypto/sha256"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/pbkdf2"
)

func BenchmarkDeriveKeyMaterial(b *testing.B) {
	for i := 0; i < b.N; i++ {
		pbkdf2.Key([]byte("test"), nil, 65356, aesKeyLength, sha256.New)
	}
}

func BenchmarkEncryptionSym(b *testing.B) {
	InitSingleTest()

	params, err := generateMessageParams()
	if err != nil {
		b.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}

	for i := 0; i < b.N; i++ {
		msg, _ := NewSentMessage(params)
		_, err := msg.Wrap(params, time.Now())
		if err != nil {
			b.Errorf("failed Wrap with seed %d: %s.", seed, err)
			b.Errorf("i = %d, len(msg.Raw) = %d, params.Payload = %d.", i, len(msg.Raw), len(params.Payload))
			return
		}
	}
}

func BenchmarkEncryptionAsym(b *testing.B) {
	InitSingleTest()

	params, err := generateMessageParams()
	if err != nil {
		b.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}
	key, err := crypto.GenerateKey()
	if err != nil {
		b.Fatalf("failed GenerateKey with seed %d: %s.", seed, err)
	}
	params.KeySym = nil
	params.Dst = &key.PublicKey

	for i := 0; i < b.N; i++ {
		msg, _ := NewSentMessage(params)
		_, err := msg.Wrap(params, time.Now())
		if err != nil {
			b.Fatalf("failed Wrap with seed %d: %s.", seed, err)
		}
	}
}

func BenchmarkDecryptionSymValid(b *testing.B) {
	InitSingleTest()

	params, err := generateMessageParams()
	if err != nil {
		b.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}
	msg, _ := NewSentMessage(params)
	env, err := msg.Wrap(params, time.Now())
	if err != nil {
		b.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}
	f := Filter{KeySym: params.KeySym}

	for i := 0; i < b.N; i++ {
		msg := env.Open(&f)
		if msg == nil {
			b.Fatalf("failed to open with seed %d.", seed)
		}
	}
}

func BenchmarkDecryptionSymInvalid(b *testing.B) {
	InitSingleTest()

	params, err := generateMessageParams()
	if err != nil {
		b.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}
	msg, _ := NewSentMessage(params)
	env, err := msg.Wrap(params, time.Now())
	if err != nil {
		b.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}
	f := Filter{KeySym: []byte("arbitrary stuff here")}

	for i := 0; i < b.N; i++ {
		msg := env.Open(&f)
		if msg != nil {
			b.Fatalf("opened envelope with invalid key, seed: %d.", seed)
		}
	}
}

func BenchmarkDecryptionAsymValid(b *testing.B) {
	InitSingleTest()

	params, err := generateMessageParams()
	if err != nil {
		b.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}
	key, err := crypto.GenerateKey()
	if err != nil {
		b.Fatalf("failed GenerateKey with seed %d: %s.", seed, err)
	}
	f := Filter{KeyAsym: key}
	params.KeySym = nil
	params.Dst = &key.PublicKey
	msg, _ := NewSentMessage(params)
	env, err := msg.Wrap(params, time.Now())
	if err != nil {
		b.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}

	for i := 0; i < b.N; i++ {
		msg := env.Open(&f)
		if msg == nil {
			b.Fatalf("fail to open, seed: %d.", seed)
		}
	}
}

func BenchmarkDecryptionAsymInvalid(b *testing.B) {
	InitSingleTest()

	params, err := generateMessageParams()
	if err != nil {
		b.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}
	key, err := crypto.GenerateKey()
	if err != nil {
		b.Fatalf("failed GenerateKey with seed %d: %s.", seed, err)
	}
	params.KeySym = nil
	params.Dst = &key.PublicKey
	msg, _ := NewSentMessage(params)
	env, err := msg.Wrap(params, time.Now())
	if err != nil {
		b.Fatalf("failed Wrap with seed %d: %s.", seed, err)
	}

	key, err = crypto.GenerateKey()
	if err != nil {
		b.Fatalf("failed GenerateKey with seed %d: %s.", seed, err)
	}
	f := Filter{KeyAsym: key}

	for i := 0; i < b.N; i++ {
		msg := env.Open(&f)
		if msg != nil {
			b.Fatalf("opened envelope with invalid key, seed: %d.", seed)
		}
	}
}

func increment(x []byte) {
	for i := 0; i < len(x); i++ {
		x[i]++
		if x[i] != 0 {
			break
		}
	}
}

func BenchmarkPoW(b *testing.B) {
	InitSingleTest()

	params, err := generateMessageParams()
	if err != nil {
		b.Fatalf("failed generateMessageParams with seed %d: %s.", seed, err)
	}
	params.Payload = make([]byte, 32)
	params.PoW = 10.0
	params.TTL = 1

	for i := 0; i < b.N; i++ {
		increment(params.Payload)
		msg, _ := NewSentMessage(params)
		_, err := msg.Wrap(params, time.Now())
		if err != nil {
			b.Fatalf("failed Wrap with seed %d: %s.", seed, err)
		}
	}
}
