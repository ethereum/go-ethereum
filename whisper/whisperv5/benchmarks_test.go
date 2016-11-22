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
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func BenchmarkDeriveKeyMaterial(b *testing.B) {
	for i := 0; i < b.N; i++ {
		deriveKeyMaterial([]byte("test"), 0)
	}
}

func BenchmarkDeriveOneTimeKey(b *testing.B) {
	for i := 0; i < b.N; i++ {
		DeriveOneTimeKey([]byte("test value 1"), []byte("test value 2"), 0)
	}
}

//func TestEncryptionSym(b *testing.T) {
func BenchmarkEncryptionSym(b *testing.B) {
	InitSingleTest()

	params, err := generateMessageParams()
	if err != nil {
		b.Errorf("failed generateMessageParams with seed %d: %s.", seed, err)
		return
	}

	for i := 0; i < b.N; i++ {
		msg := NewSentMessage(params)
		_, err := msg.Wrap(params)
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
		b.Errorf("failed generateMessageParams with seed %d: %s.", seed, err)
		return
	}
	key, err := crypto.GenerateKey()
	if err != nil {
		b.Errorf("failed GenerateKey with seed %d: %s.", seed, err)
		return
	}
	params.KeySym = nil
	params.Dst = &key.PublicKey

	for i := 0; i < b.N; i++ {
		msg := NewSentMessage(params)
		_, err := msg.Wrap(params)
		if err != nil {
			b.Errorf("failed Wrap with seed %d: %s.", seed, err)
			return
		}
	}
}

func BenchmarkDecryptionSymValid(b *testing.B) {
	InitSingleTest()

	params, err := generateMessageParams()
	if err != nil {
		b.Errorf("failed generateMessageParams with seed %d: %s.", seed, err)
		return
	}
	msg := NewSentMessage(params)
	env, err := msg.Wrap(params)
	if err != nil {
		b.Errorf("failed Wrap with seed %d: %s.", seed, err)
		return
	}
	f := Filter{KeySym: params.KeySym}

	for i := 0; i < b.N; i++ {
		msg := env.Open(&f)
		if msg == nil {
			b.Errorf("failed to open with seed %d.", seed)
			return
		}
	}
}

func BenchmarkDecryptionSymInvalid(b *testing.B) {
	InitSingleTest()

	params, err := generateMessageParams()
	if err != nil {
		b.Errorf("failed generateMessageParams with seed %d: %s.", seed, err)
		return
	}
	msg := NewSentMessage(params)
	env, err := msg.Wrap(params)
	if err != nil {
		b.Errorf("failed Wrap with seed %d: %s.", seed, err)
		return
	}
	f := Filter{KeySym: []byte("arbitrary stuff here")}

	for i := 0; i < b.N; i++ {
		msg := env.Open(&f)
		if msg != nil {
			b.Errorf("opened envelope with invalid key, seed: %d.", seed)
			return
		}
	}
}

func BenchmarkDecryptionAsymValid(b *testing.B) {
	InitSingleTest()

	params, err := generateMessageParams()
	if err != nil {
		b.Errorf("failed generateMessageParams with seed %d: %s.", seed, err)
		return
	}
	key, err := crypto.GenerateKey()
	if err != nil {
		b.Errorf("failed GenerateKey with seed %d: %s.", seed, err)
		return
	}
	f := Filter{KeyAsym: key}
	params.KeySym = nil
	params.Dst = &key.PublicKey
	msg := NewSentMessage(params)
	env, err := msg.Wrap(params)
	if err != nil {
		b.Errorf("failed Wrap with seed %d: %s.", seed, err)
		return
	}

	for i := 0; i < b.N; i++ {
		msg := env.Open(&f)
		if msg == nil {
			b.Errorf("fail to open, seed: %d.", seed)
			return
		}
	}
}

func BenchmarkDecryptionAsymInvalid(b *testing.B) {
	InitSingleTest()

	params, err := generateMessageParams()
	if err != nil {
		b.Errorf("failed generateMessageParams with seed %d: %s.", seed, err)
		return
	}
	key, err := crypto.GenerateKey()
	if err != nil {
		b.Errorf("failed GenerateKey with seed %d: %s.", seed, err)
		return
	}
	params.KeySym = nil
	params.Dst = &key.PublicKey
	msg := NewSentMessage(params)
	env, err := msg.Wrap(params)
	if err != nil {
		b.Errorf("failed Wrap with seed %d: %s.", seed, err)
		return
	}

	key, err = crypto.GenerateKey()
	if err != nil {
		b.Errorf("failed GenerateKey with seed %d: %s.", seed, err)
		return
	}
	f := Filter{KeyAsym: key}

	for i := 0; i < b.N; i++ {
		msg := env.Open(&f)
		if msg != nil {
			b.Errorf("opened envelope with invalid key, seed: %d.", seed)
			return
		}
	}
}
