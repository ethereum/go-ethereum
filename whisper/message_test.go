// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package whisper

import (
	"bytes"
	"crypto/elliptic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

// Tests whether a message can be wrapped without any identity or encryption.
func TestMessageSimpleWrap(t *testing.T) {
	payload := []byte("hello world")

	msg := NewMessage(payload)
	if _, err := msg.Wrap(DefaultPoW, Options{}); err != nil {
		t.Fatalf("failed to wrap message: %v", err)
	}
	if msg.Flags&signatureFlag != 0 {
		t.Fatalf("signature flag mismatch: have %d, want %d", msg.Flags&signatureFlag, 0)
	}
	if len(msg.Signature) != 0 {
		t.Fatalf("signature found for simple wrapping: 0x%x", msg.Signature)
	}
	if bytes.Compare(msg.Payload, payload) != 0 {
		t.Fatalf("payload mismatch after wrapping: have 0x%x, want 0x%x", msg.Payload, payload)
	}
	if msg.TTL/time.Second != DefaultTTL/time.Second {
		t.Fatalf("message TTL mismatch: have %v, want %v", msg.TTL, DefaultTTL)
	}
}

// Tests whether a message can be signed, and wrapped in plain-text.
func TestMessageCleartextSignRecover(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to create crypto key: %v", err)
	}
	payload := []byte("hello world")

	msg := NewMessage(payload)
	if _, err := msg.Wrap(DefaultPoW, Options{
		From: key,
	}); err != nil {
		t.Fatalf("failed to sign message: %v", err)
	}
	if msg.Flags&signatureFlag != signatureFlag {
		t.Fatalf("signature flag mismatch: have %d, want %d", msg.Flags&signatureFlag, signatureFlag)
	}
	if bytes.Compare(msg.Payload, payload) != 0 {
		t.Fatalf("payload mismatch after signing: have 0x%x, want 0x%x", msg.Payload, payload)
	}

	pubKey := msg.Recover()
	if pubKey == nil {
		t.Fatalf("failed to recover public key")
	}
	p1 := elliptic.Marshal(crypto.S256(), key.PublicKey.X, key.PublicKey.Y)
	p2 := elliptic.Marshal(crypto.S256(), pubKey.X, pubKey.Y)
	if !bytes.Equal(p1, p2) {
		t.Fatalf("public key mismatch: have 0x%x, want 0x%x", p2, p1)
	}
}

// Tests whether a message can be encrypted and decrypted using an anonymous
// sender (i.e. no signature).
func TestMessageAnonymousEncryptDecrypt(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to create recipient crypto key: %v", err)
	}
	payload := []byte("hello world")

	msg := NewMessage(payload)
	envelope, err := msg.Wrap(DefaultPoW, Options{
		To: &key.PublicKey,
	})
	if err != nil {
		t.Fatalf("failed to encrypt message: %v", err)
	}
	if msg.Flags&signatureFlag != 0 {
		t.Fatalf("signature flag mismatch: have %d, want %d", msg.Flags&signatureFlag, 0)
	}
	if len(msg.Signature) != 0 {
		t.Fatalf("signature found for anonymous message: 0x%x", msg.Signature)
	}

	out, err := envelope.Open(key)
	if err != nil {
		t.Fatalf("failed to open encrypted message: %v", err)
	}
	if !bytes.Equal(out.Payload, payload) {
		t.Error("payload mismatch: have 0x%x, want 0x%x", out.Payload, payload)
	}
}

// Tests whether a message can be properly signed and encrypted.
func TestMessageFullCrypto(t *testing.T) {
	fromKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to create sender crypto key: %v", err)
	}
	toKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to create recipient crypto key: %v", err)
	}

	payload := []byte("hello world")
	msg := NewMessage(payload)
	envelope, err := msg.Wrap(DefaultPoW, Options{
		From: fromKey,
		To:   &toKey.PublicKey,
	})
	if err != nil {
		t.Fatalf("failed to encrypt message: %v", err)
	}
	if msg.Flags&signatureFlag != signatureFlag {
		t.Fatalf("signature flag mismatch: have %d, want %d", msg.Flags&signatureFlag, signatureFlag)
	}
	if len(msg.Signature) == 0 {
		t.Fatalf("no signature found for signed message")
	}

	out, err := envelope.Open(toKey)
	if err != nil {
		t.Fatalf("failed to open encrypted message: %v", err)
	}
	if !bytes.Equal(out.Payload, payload) {
		t.Error("payload mismatch: have 0x%x, want 0x%x", out.Payload, payload)
	}

	pubKey := out.Recover()
	if pubKey == nil {
		t.Fatalf("failed to recover public key")
	}
	p1 := elliptic.Marshal(crypto.S256(), fromKey.PublicKey.X, fromKey.PublicKey.Y)
	p2 := elliptic.Marshal(crypto.S256(), pubKey.X, pubKey.Y)
	if !bytes.Equal(p1, p2) {
		t.Fatalf("public key mismatch: have 0x%x, want 0x%x", p2, p1)
	}
}
