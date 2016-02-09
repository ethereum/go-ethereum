// Copyright (c) 2013 Kyle Isom <kyle@tyrfingr.is>
// Copyright (c) 2012 The Go Authors. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//    * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//    * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//    * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package ecies

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
)

var dumpEnc bool

func init() {
	flDump := flag.Bool("dump", false, "write encrypted test message to file")
	flag.Parse()
	dumpEnc = *flDump
}

// Ensure the KDF generates appropriately sized keys.
func TestKDF(t *testing.T) {
	msg := []byte("Hello, world")
	h := sha256.New()

	k, err := concatKDF(h, msg, nil, 64)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}
	if len(k) != 64 {
		fmt.Printf("KDF: generated key is the wrong size (%d instead of 64\n",
			len(k))
		t.FailNow()
	}
}

var ErrBadSharedKeys = fmt.Errorf("ecies: shared keys don't match")

// cmpParams compares a set of ECIES parameters. We assume, as per the
// docs, that AES is the only supported symmetric encryption algorithm.
func cmpParams(p1, p2 *ECIESParams) bool {
	if p1.hashAlgo != p2.hashAlgo {
		return false
	} else if p1.KeyLen != p2.KeyLen {
		return false
	} else if p1.BlockSize != p2.BlockSize {
		return false
	}
	return true
}

// cmpPublic returns true if the two public keys represent the same pojnt.
func cmpPublic(pub1, pub2 PublicKey) bool {
	if pub1.X == nil || pub1.Y == nil {
		fmt.Println(ErrInvalidPublicKey.Error())
		return false
	}
	if pub2.X == nil || pub2.Y == nil {
		fmt.Println(ErrInvalidPublicKey.Error())
		return false
	}
	pub1Out := elliptic.Marshal(pub1.Curve, pub1.X, pub1.Y)
	pub2Out := elliptic.Marshal(pub2.Curve, pub2.X, pub2.Y)

	return bytes.Equal(pub1Out, pub2Out)
}

// cmpPrivate returns true if the two private keys are the same.
func cmpPrivate(prv1, prv2 *PrivateKey) bool {
	if prv1 == nil || prv1.D == nil {
		return false
	} else if prv2 == nil || prv2.D == nil {
		return false
	} else if prv1.D.Cmp(prv2.D) != 0 {
		return false
	} else {
		return cmpPublic(prv1.PublicKey, prv2.PublicKey)
	}
}

// Validate the ECDH component.
func TestSharedKey(t *testing.T) {
	prv1, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}
	skLen := MaxSharedKeyLength(&prv1.PublicKey) / 2

	prv2, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	sk1, err := prv1.GenerateShared(&prv2.PublicKey, skLen, skLen)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	sk2, err := prv2.GenerateShared(&prv1.PublicKey, skLen, skLen)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	if !bytes.Equal(sk1, sk2) {
		fmt.Println(ErrBadSharedKeys.Error())
		t.FailNow()
	}
}

func TestSharedKeyPadding(t *testing.T) {
	// sanity checks
	prv0 := hexKey("1adf5c18167d96a1f9a0b1ef63be8aa27eaf6032c233b2b38f7850cf5b859fd9")
	prv1 := hexKey("97a076fc7fcd9208240668e31c9abee952cbb6e375d1b8febc7499d6e16f1a")
	x0, _ := new(big.Int).SetString("1a8ed022ff7aec59dc1b440446bdda5ff6bcb3509a8b109077282b361efffbd8", 16)
	x1, _ := new(big.Int).SetString("6ab3ac374251f638d0abb3ef596d1dc67955b507c104e5f2009724812dc027b8", 16)
	y0, _ := new(big.Int).SetString("e040bd480b1deccc3bc40bd5b1fdcb7bfd352500b477cb9471366dbd4493f923", 16)
	y1, _ := new(big.Int).SetString("8ad915f2b503a8be6facab6588731fefeb584fd2dfa9a77a5e0bba1ec439e4fa", 16)

	if prv0.PublicKey.X.Cmp(x0) != 0 {
		t.Errorf("mismatched prv0.X:\nhave: %x\nwant: %x\n", prv0.PublicKey.X.Bytes(), x0.Bytes())
	}
	if prv0.PublicKey.Y.Cmp(y0) != 0 {
		t.Errorf("mismatched prv0.Y:\nhave: %x\nwant: %x\n", prv0.PublicKey.Y.Bytes(), y0.Bytes())
	}
	if prv1.PublicKey.X.Cmp(x1) != 0 {
		t.Errorf("mismatched prv1.X:\nhave: %x\nwant: %x\n", prv1.PublicKey.X.Bytes(), x1.Bytes())
	}
	if prv1.PublicKey.Y.Cmp(y1) != 0 {
		t.Errorf("mismatched prv1.Y:\nhave: %x\nwant: %x\n", prv1.PublicKey.Y.Bytes(), y1.Bytes())
	}

	// test shared secret generation
	sk1, err := prv0.GenerateShared(&prv1.PublicKey, 16, 16)
	if err != nil {
		fmt.Println(err.Error())
	}

	sk2, err := prv1.GenerateShared(&prv0.PublicKey, 16, 16)
	if err != nil {
		t.Fatal(err.Error())
	}

	if !bytes.Equal(sk1, sk2) {
		t.Fatal(ErrBadSharedKeys.Error())
	}
}

// Verify that the key generation code fails when too much key data is
// requested.
func TestTooBigSharedKey(t *testing.T) {
	prv1, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	prv2, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	_, err = prv1.GenerateShared(&prv2.PublicKey, 32, 32)
	if err != ErrSharedKeyTooBig {
		fmt.Println("ecdh: shared key should be too large for curve")
		t.FailNow()
	}

	_, err = prv2.GenerateShared(&prv1.PublicKey, 32, 32)
	if err != ErrSharedKeyTooBig {
		fmt.Println("ecdh: shared key should be too large for curve")
		t.FailNow()
	}
}

// Ensure a public key can be successfully marshalled and unmarshalled, and
// that the decoded key is the same as the original.
func TestMarshalPublic(t *testing.T) {
	prv, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		t.Fatalf("GenerateKey error: %s", err)
	}

	out, err := MarshalPublic(&prv.PublicKey)
	if err != nil {
		t.Fatalf("MarshalPublic error: %s", err)
	}

	pub, err := UnmarshalPublic(out)
	if err != nil {
		t.Fatalf("UnmarshalPublic error: %s", err)
	}

	if !cmpPublic(prv.PublicKey, *pub) {
		t.Fatal("ecies: failed to unmarshal public key")
	}
}

// Ensure that a private key can be encoded into DER format, and that
// the resulting key is properly parsed back into a public key.
func TestMarshalPrivate(t *testing.T) {
	prv, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	out, err := MarshalPrivate(prv)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	if dumpEnc {
		ioutil.WriteFile("test.out", out, 0644)
	}

	prv2, err := UnmarshalPrivate(out)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	if !cmpPrivate(prv, prv2) {
		fmt.Println("ecdh: private key import failed")
		t.FailNow()
	}
}

// Ensure that a private key can be successfully encoded to PEM format, and
// the resulting key is properly parsed back in.
func TestPrivatePEM(t *testing.T) {
	prv, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	out, err := ExportPrivatePEM(prv)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	if dumpEnc {
		ioutil.WriteFile("test.key", out, 0644)
	}

	prv2, err := ImportPrivatePEM(out)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	} else if !cmpPrivate(prv, prv2) {
		fmt.Println("ecdh: import from PEM failed")
		t.FailNow()
	}
}

// Ensure that a public key can be successfully encoded to PEM format, and
// the resulting key is properly parsed back in.
func TestPublicPEM(t *testing.T) {
	prv, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	out, err := ExportPublicPEM(&prv.PublicKey)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	if dumpEnc {
		ioutil.WriteFile("test.pem", out, 0644)
	}

	pub2, err := ImportPublicPEM(out)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	} else if !cmpPublic(prv.PublicKey, *pub2) {
		fmt.Println("ecdh: import from PEM failed")
		t.FailNow()
	}
}

// Benchmark the generation of P256 keys.
func BenchmarkGenerateKeyP256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := GenerateKey(rand.Reader, elliptic.P256(), nil); err != nil {
			fmt.Println(err.Error())
			b.FailNow()
		}
	}
}

// Benchmark the generation of P256 shared keys.
func BenchmarkGenSharedKeyP256(b *testing.B) {
	prv, err := GenerateKey(rand.Reader, elliptic.P256(), nil)
	if err != nil {
		fmt.Println(err.Error())
		b.FailNow()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := prv.GenerateShared(&prv.PublicKey, 16, 16)
		if err != nil {
			fmt.Println(err.Error())
			b.FailNow()
		}
	}
}

// Benchmark the generation of S256 shared keys.
func BenchmarkGenSharedKeyS256(b *testing.B) {
	prv, err := GenerateKey(rand.Reader, secp256k1.S256(), nil)
	if err != nil {
		fmt.Println(err.Error())
		b.FailNow()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := prv.GenerateShared(&prv.PublicKey, 16, 16)
		if err != nil {
			fmt.Println(err.Error())
			b.FailNow()
		}
	}
}

// Verify that an encrypted message can be successfully decrypted.
func TestEncryptDecrypt(t *testing.T) {
	prv1, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	prv2, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	message := []byte("Hello, world.")
	ct, err := Encrypt(rand.Reader, &prv2.PublicKey, message, nil, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	pt, err := prv2.Decrypt(rand.Reader, ct, nil, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	if !bytes.Equal(pt, message) {
		fmt.Println("ecies: plaintext doesn't match message")
		t.FailNow()
	}

	_, err = prv1.Decrypt(rand.Reader, ct, nil, nil)
	if err == nil {
		fmt.Println("ecies: encryption should not have succeeded")
		t.FailNow()
	}
}

func TestDecryptShared2(t *testing.T) {
	prv, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		t.Fatal(err)
	}
	message := []byte("Hello, world.")
	shared2 := []byte("shared data 2")
	ct, err := Encrypt(rand.Reader, &prv.PublicKey, message, nil, shared2)
	if err != nil {
		t.Fatal(err)
	}

	// Check that decrypting with correct shared data works.
	pt, err := prv.Decrypt(rand.Reader, ct, nil, shared2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(pt, message) {
		t.Fatal("ecies: plaintext doesn't match message")
	}

	// Decrypting without shared data or incorrect shared data fails.
	if _, err = prv.Decrypt(rand.Reader, ct, nil, nil); err == nil {
		t.Fatal("ecies: decrypting without shared data didn't fail")
	}
	if _, err = prv.Decrypt(rand.Reader, ct, nil, []byte("garbage")); err == nil {
		t.Fatal("ecies: decrypting with incorrect shared data didn't fail")
	}
}

// TestMarshalEncryption validates the encode/decode produces a valid
// ECIES encryption key.
func TestMarshalEncryption(t *testing.T) {
	prv1, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	out, err := MarshalPrivate(prv1)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	prv2, err := UnmarshalPrivate(out)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	message := []byte("Hello, world.")
	ct, err := Encrypt(rand.Reader, &prv2.PublicKey, message, nil, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	pt, err := prv2.Decrypt(rand.Reader, ct, nil, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	if !bytes.Equal(pt, message) {
		fmt.Println("ecies: plaintext doesn't match message")
		t.FailNow()
	}

	_, err = prv1.Decrypt(rand.Reader, ct, nil, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

}

type testCase struct {
	Curve    elliptic.Curve
	Name     string
	Expected bool
}

var testCases = []testCase{
	testCase{
		Curve:    elliptic.P256(),
		Name:     "P256",
		Expected: true,
	},
	testCase{
		Curve:    elliptic.P384(),
		Name:     "P384",
		Expected: true,
	},
	testCase{
		Curve:    elliptic.P521(),
		Name:     "P521",
		Expected: true,
	},
}

// Test parameter selection for each curve, and that P224 fails automatic
// parameter selection (see README for a discussion of P224). Ensures that
// selecting a set of parameters automatically for the given curve works.
func TestParamSelection(t *testing.T) {
	for _, c := range testCases {
		testParamSelection(t, c)
	}
}

func testParamSelection(t *testing.T, c testCase) {
	params := ParamsFromCurve(c.Curve)
	if params == nil && c.Expected {
		fmt.Printf("%s (%s)\n", ErrInvalidParams.Error(), c.Name)
		t.FailNow()
	} else if params != nil && !c.Expected {
		fmt.Printf("ecies: parameters should be invalid (%s)\n",
			c.Name)
		t.FailNow()
	}

	prv1, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Printf("%s (%s)\n", err.Error(), c.Name)
		t.FailNow()
	}

	prv2, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Printf("%s (%s)\n", err.Error(), c.Name)
		t.FailNow()
	}

	message := []byte("Hello, world.")
	ct, err := Encrypt(rand.Reader, &prv2.PublicKey, message, nil, nil)
	if err != nil {
		fmt.Printf("%s (%s)\n", err.Error(), c.Name)
		t.FailNow()
	}

	pt, err := prv2.Decrypt(rand.Reader, ct, nil, nil)
	if err != nil {
		fmt.Printf("%s (%s)\n", err.Error(), c.Name)
		t.FailNow()
	}

	if !bytes.Equal(pt, message) {
		fmt.Printf("ecies: plaintext doesn't match message (%s)\n",
			c.Name)
		t.FailNow()
	}

	_, err = prv1.Decrypt(rand.Reader, ct, nil, nil)
	if err == nil {
		fmt.Printf("ecies: encryption should not have succeeded (%s)\n",
			c.Name)
		t.FailNow()
	}

}

// Ensure that the basic public key validation in the decryption operation
// works.
func TestBasicKeyValidation(t *testing.T) {
	badBytes := []byte{0, 1, 5, 6, 7, 8, 9}

	prv, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	message := []byte("Hello, world.")
	ct, err := Encrypt(rand.Reader, &prv.PublicKey, message, nil, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	for _, b := range badBytes {
		ct[0] = b
		_, err := prv.Decrypt(rand.Reader, ct, nil, nil)
		if err != ErrInvalidPublicKey {
			fmt.Println("ecies: validated an invalid key")
			t.FailNow()
		}
	}
}

// Verify GenerateShared against static values - useful when
// debugging changes in underlying libs
func TestSharedKeyStatic(t *testing.T) {
	prv1 := hexKey("7ebbc6a8358bc76dd73ebc557056702c8cfc34e5cfcd90eb83af0347575fd2ad")
	prv2 := hexKey("6a3d6396903245bba5837752b9e0348874e72db0c4e11e9c485a81b4ea4353b9")

	skLen := MaxSharedKeyLength(&prv1.PublicKey) / 2

	sk1, err := prv1.GenerateShared(&prv2.PublicKey, skLen, skLen)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	sk2, err := prv2.GenerateShared(&prv1.PublicKey, skLen, skLen)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	if !bytes.Equal(sk1, sk2) {
		fmt.Println(ErrBadSharedKeys.Error())
		t.FailNow()
	}

	sk, _ := hex.DecodeString("167ccc13ac5e8a26b131c3446030c60fbfac6aa8e31149d0869f93626a4cdf62")
	if !bytes.Equal(sk1, sk) {
		t.Fatalf("shared secret mismatch: want: %x have: %x", sk, sk1)
	}
}

// TODO: remove after refactoring packages crypto and crypto/ecies
func hexKey(prv string) *PrivateKey {
	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = secp256k1.S256()
	priv.D, _ = new(big.Int).SetString(prv, 16)
	priv.PublicKey.X, priv.PublicKey.Y = secp256k1.S256().ScalarBaseMult(priv.D.Bytes())
	return ImportECDSA(priv)
}
