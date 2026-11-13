// Copyright 2026 The go-ethereum Authors
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

package leansphincs

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256r1"
	"gopkg.in/yaml.v3"
)

// The known answer shared by EthLib and leanVM: seed[i] = 37i+11, msg[i] = 7i+3.
var (
	katSeed = hexutil.MustDecode("0x0b30557a9fc4e90e33587da2c7ec11365b80a5caef14395e83a8cdf2173c6186")
	katMsg  = hexutil.MustDecode("0x030a11181f262d343b424950575e656c737a81888f969da4abb2b9c0c7ced5dc")
	katPub  = hexutil.MustDecode("0xe316da87de2d05b3ccd0a39c30bfda68303b3741edeb4da5f028855bcd31b9ee")
)

func katSig(t testing.TB) []byte {
	data, err := os.ReadFile("testdata/ethlib/leanvm_known_answer.hex")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Fields(string(data))
	if len(lines) != 2 {
		t.Fatalf("want 2 lines in known answer, got %d", len(lines))
	}
	pub, err := hex.DecodeString(lines[0])
	if err != nil || !bytes.Equal(pub, katPub) {
		t.Fatalf("leanVM public key %x does not match EthLib %x", pub, katPub)
	}
	sig, err := hex.DecodeString(lines[1])
	if err != nil {
		t.Fatal(err)
	}
	return sig
}

type katCase struct {
	Input struct {
		Seed      string `yaml:"seed"`
		PublicKey string `yaml:"public_key"`
		Message   string `yaml:"message"`
		Signature string `yaml:"signature"`
	} `yaml:"input"`
	Output any `yaml:"output"` // hex string, bool, or nil when the input is rejected
}

func decodeHex(t *testing.T, s string) []byte {
	if s == "" {
		return nil
	}
	b, err := hexutil.Decode(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestVectors runs the EthLib vectors under testdata/ethlib.
func TestVectors(t *testing.T) {
	files, err := filepath.Glob("testdata/ethlib/*/*/data.yaml")
	if err != nil || len(files) == 0 {
		t.Fatalf("no vectors found: %v", err)
	}
	for _, file := range files {
		kind := filepath.Base(filepath.Dir(filepath.Dir(file)))
		name := filepath.Base(filepath.Dir(file))
		if testing.Short() && kind != "verify" && !strings.Contains(name, "known_answer") {
			continue
		}
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			var c katCase
			if err := yaml.Unmarshal(data, &c); err != nil {
				t.Fatal(err)
			}
			want, _ := c.Output.(string)
			switch kind {
			case "key_gen":
				pub, err := KeyGen(decodeHex(t, c.Input.Seed))
				if c.Output == nil {
					if err == nil {
						t.Fatal("KeyGen accepted a bad seed")
					}
					return
				}
				if err != nil || !bytes.Equal(pub, decodeHex(t, want)) {
					t.Fatalf("KeyGen = %x, %v; want %s", pub, err, want)
				}
			case "sign":
				sig, err := Sign(decodeHex(t, c.Input.Seed), decodeHex(t, c.Input.Message))
				if c.Output == nil {
					if err == nil {
						t.Fatal("Sign accepted a bad input")
					}
					return
				}
				if err != nil || !bytes.Equal(sig, decodeHex(t, want)) {
					t.Fatalf("Sign mismatch: %v", err)
				}
			case "verify":
				got := Verify(decodeHex(t, c.Input.PublicKey), decodeHex(t, c.Input.Message), decodeHex(t, c.Input.Signature))
				if got != (c.Output == true) {
					t.Fatalf("Verify = %v, want %v", got, c.Output)
				}
			default:
				t.Fatalf("unknown vector kind %q", kind)
			}
		})
	}
}

// TestKnownAnswer checks the leanVM known answer, an implementation
// independent of EthLib, against KeyGen, Sign and Verify.
func TestKnownAnswer(t *testing.T) {
	sig := katSig(t)
	if !Verify(katPub, katMsg, sig) {
		t.Fatal("known answer does not verify")
	}
	if testing.Short() {
		return
	}
	pub, err := KeyGen(katSeed)
	if err != nil || !bytes.Equal(pub, katPub) {
		t.Fatalf("KeyGen = %x, %v", pub, err)
	}
	got, err := Sign(katSeed, katMsg)
	if err != nil || !bytes.Equal(got, sig) {
		t.Fatalf("Sign does not reproduce the known answer: %v", err)
	}
}

func TestRoundTrip(t *testing.T) {
	seed := make([]byte, SeedSize)
	msg := make([]byte, MessageSize)
	rand.Read(seed)
	rand.Read(msg)
	pub, err := KeyGen(seed)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := Sign(seed, msg)
	if err != nil {
		t.Fatal(err)
	}
	if !Verify(pub, msg, sig) {
		t.Fatal("signature does not verify")
	}
	again, _ := Sign(seed, msg)
	if !bytes.Equal(sig, again) {
		t.Fatal("signing is not deterministic")
	}
	// Flipping any bit of any input must fail. In short mode, only sample.
	step := 1
	if testing.Short() {
		step = 97
	}
	for _, in := range [][]byte{pub, msg, sig} {
		for i := 0; i < len(in); i += step {
			in[i] ^= 1
			if Verify(pub, msg, sig) {
				t.Fatalf("tampered input of length %d at byte %d verifies", len(in), i)
			}
			in[i] ^= 1
		}
	}
}

func TestMalformed(t *testing.T) {
	sig := katSig(t)
	cases := []struct{ pub, msg, sig []byte }{
		{nil, nil, nil},
		{katPub[:31], katMsg, sig},
		{append(katPub, 0), katMsg, sig},
		{katPub, katMsg[:31], sig},
		{katPub, nil, sig},
		{katPub, katMsg, sig[:SignatureSize-1]},
		{katPub, katMsg, append(sig, 0)},
	}
	for i, c := range cases {
		if Verify(c.pub, c.msg, c.sig) {
			t.Errorf("case %d: malformed input verifies", i)
		}
	}
	if _, err := KeyGen(katSeed[:31]); err == nil {
		t.Error("KeyGen accepted a short seed")
	}
	if _, err := Sign(katSeed, katMsg[:31]); err == nil {
		t.Error("Sign accepted a short message")
	}
}

// countingState counts the Keccak-f[1600] permutations a hash performs,
// which is len/136 + 1 for the single-rate legacy Keccak-256.
type countingState struct {
	crypto.KeccakState
	written, perms int
}

func (c *countingState) Reset() {
	c.written = 0
	c.KeccakState.Reset()
}

func (c *countingState) Write(p []byte) (int, error) {
	c.written += len(p)
	return c.KeccakState.Write(p)
}

func (c *countingState) Read(p []byte) (int, error) {
	c.perms += c.written/136 + 1
	return c.KeccakState.Read(p)
}

// TestVerifyCost checks the spec's figure of 515 permutations per
// verification, and that the inner loops do not allocate.
func TestVerifyCost(t *testing.T) {
	sig := katSig(t)
	counter := &countingState{KeccakState: crypto.NewKeccakState()}
	if !verify(&hasher{kh: counter}, katPub, katMsg, sig) {
		t.Fatal("known answer does not verify")
	}
	if counter.perms != 515 {
		t.Errorf("verify used %d permutations, want 515", counter.perms)
	}
	if allocs := testing.AllocsPerRun(10, func() { Verify(katPub, katMsg, sig) }); allocs > 2 {
		t.Errorf("Verify allocates %v times, want at most 2", allocs)
	}
}

func BenchmarkVerify(b *testing.B) {
	sig := katSig(b)
	for b.Loop() {
		if !Verify(katPub, katMsg, sig) {
			b.Fatal("verify error")
		}
	}
}

func BenchmarkSign(b *testing.B) {
	for b.Loop() {
		if _, err := Sign(katSeed, katMsg); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkKeyGen(b *testing.B) {
	for b.Loop() {
		if _, err := KeyGen(katSeed); err != nil {
			b.Fatal(err)
		}
	}
}

// The two benchmarks below are the schemes leanSPHINCS is priced against:
// ecrecover at 3000 gas and P256VERIFY at 6900 gas.

func BenchmarkEcrecover(b *testing.B) {
	msg := hexutil.MustDecode("0xce0677bb30baa8cf067c88db9811f4333d131bf8bcf12fe7065d211dce971008")
	sig := hexutil.MustDecode("0x90f27b8b488db00b00606796d2987f6a5f59ae62ea05effe84fef5b8b0e549984a691139ad57a3f0b906637673aa2f63d1f55cb1a69199d4009eea23ceaddc9301")
	for b.Loop() {
		if _, err := crypto.Ecrecover(msg, sig); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkP256Verify(b *testing.B) {
	data, err := os.ReadFile("../../core/vm/testdata/precompiles/p256Verify.json")
	if err != nil {
		b.Fatal(err)
	}
	var vectors []struct{ Input string }
	if err := json.Unmarshal(data, &vectors); err != nil {
		b.Fatal(err)
	}
	in, _ := hex.DecodeString(vectors[0].Input) // hash || r || s || x || y
	hash := in[:32]
	r, s := new(big.Int).SetBytes(in[32:64]), new(big.Int).SetBytes(in[64:96])
	x, y := new(big.Int).SetBytes(in[96:128]), new(big.Int).SetBytes(in[128:160])
	for b.Loop() {
		if !secp256r1.Verify(hash, r, s, x, y) {
			b.Fatal("verify error")
		}
	}
}
