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

package vm

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

// allPrecompileSets returns every set a node can run with, labelled by a fork
// that selects it. It walks the flags of params.Rules and asks
// activePrecompiledContracts what each one activates, so a set added for a new
// fork is covered here without anyone remembering to list it.
func allPrecompileSets() map[string]PrecompiledContracts {
	var (
		rules = reflect.TypeOf(params.Rules{})
		seen  = make(map[*PrecompiledContracts]bool)
		sets  = make(map[string]PrecompiledContracts)
	)
	// The zero value picks whatever the switch falls through to.
	base := activePrecompiledContracts(params.Rules{})
	seen[base], sets["default"] = true, *base

	for i := range rules.NumField() {
		field := rules.Field(i)
		if field.Type.Kind() != reflect.Bool {
			continue
		}
		// Activate only this field so the fork resolves to the set it gates.
		var forked params.Rules
		reflect.ValueOf(&forked).Elem().Field(i).SetBool(true)

		// Later forks shadow earlier ones, so the first flag reaching a set is
		// the one that names it.
		if set := activePrecompiledContracts(forked); !seen[set] {
			seen[set], sets[strings.TrimPrefix(field.Name, "Is")] = true, *set
		}
	}
	return sets
}

// probeGasLimit is a generous stand-in for the block gas limit, bounding the
// probe corpus to invocations the EVM could actually pay to run.
const probeGasLimit = 1 << 30

// cacheProbeInputs builds inputs that stress normalization: the lengths each
// precompile cares about, either side of them, zero padded and non-zero padded
// variants, and a few random ones.
//
// Random bytes alone are not enough. They fail every signature and curve check,
// so a precompile whose failure path returns one fixed value looks consistent
// no matter how badly its inputs are merged. The corpus therefore also carries
// every input from testdata, which is where the succeeding cases live, and pads
// each of them so a valid call and its padded form can be caught colliding.
func cacheProbeInputs(t *testing.T, rng *rand.Rand) [][]byte {
	lengths := []int{0, 1, 31, 32, 63, 64, 65, 95, 96, 97, 127, 128, 129, 160, 161,
		191, 192, 193, 213, 214, 255, 256, 257, 288, 384, 385, 512, 576, 1920, 4096, 8192, 8193}

	var inputs [][]byte
	for _, n := range lengths {
		inputs = append(inputs, make([]byte, n)) // all zero

		filled := make([]byte, n)
		rng.Read(filled)
		inputs = append(inputs, filled)

		// A short body followed by zeros, which normalization is allowed to
		// drop, and by non-zeros, which it is not unless unread.
		if n >= 64 {
			zeroTail := make([]byte, n)
			copy(zeroTail, filled[:32])
			inputs = append(inputs, zeroTail)

			oneTail := make([]byte, n)
			copy(oneTail, filled[:32])
			for i := 32; i < n; i++ {
				oneTail[i] = 1
			}
			inputs = append(inputs, oneTail)
		}
	}
	for _, fixture := range precompileFixtureInputs(t) {
		inputs = append(inputs, fixture)
		for _, extra := range []int{1, 64, 512} {
			padded := make([]byte, len(fixture)+extra)
			copy(padded, fixture)
			inputs = append(inputs, padded)

			nonzero := make([]byte, len(fixture)+extra)
			copy(nonzero, fixture)
			for i := len(fixture); i < len(nonzero); i++ {
				nonzero[i] = 0xff
			}
			inputs = append(inputs, nonzero)
		}
	}
	return inputs
}

// precompileFixtureInputs reads every precompile test vector on disk. These are
// the inputs that actually exercise success paths.
func precompileFixtureInputs(t *testing.T) [][]byte {
	t.Helper()

	entries, err := os.ReadDir("testdata/precompiles")
	if err != nil {
		t.Fatalf("reading precompile fixtures: %v", err)
	}
	var inputs [][]byte
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		blob, err := os.ReadFile(filepath.Join("testdata/precompiles", entry.Name()))
		if err != nil {
			t.Fatalf("reading %s: %v", entry.Name(), err)
		}
		var cases []struct{ Input string }
		if err := json.Unmarshal(blob, &cases); err != nil {
			t.Fatalf("parsing %s: %v", entry.Name(), err)
		}
		for _, c := range cases {
			if in, err := hex.DecodeString(c.Input); err == nil && len(in) <= maxCacheablePrecompileInput {
				inputs = append(inputs, in)
			}
		}
	}
	if len(inputs) == 0 {
		t.Fatal("no precompile fixtures found")
	}
	return inputs
}

// TestPrecompileCacheNormalizationSound is the load bearing test of the cache:
// two inputs that normalize to the same key are served the same entry, so they
// must run to the same result. A violation here is a consensus bug, not a
// performance one.
func TestPrecompileCacheNormalizationSound(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	inputs := cacheProbeInputs(t, rng)

	for fork, set := range allPrecompileSets() {
		for addr, p := range set {
			type outcome struct {
				input  []byte
				output []byte
				err    error
			}
			seen := make(map[string]outcome)
			for _, in := range inputs {
				// Skip what the EVM could never reach. A random modexp header
				// declares operands nobody can pay for, and RunPrecompiledContract
				// charges before it runs, so Run never sees them.
				if p.RequiredGas(in) > probeGasLimit {
					continue
				}
				key, ok := precompileCacheKey(p, in)
				if !ok {
					continue
				}
				output, err := p.Run(in)
				prev, dup := seen[string(key)]
				if !dup {
					seen[string(key)] = outcome{in, output, err}
					continue
				}
				if !bytes.Equal(output, prev.output) || !errEqual(err, prev.err) {
					t.Errorf("%s %s (%x): inputs %x and %x share key %x but run differently:\n  %x / %v\n  %x / %v",
						fork, p.Name(), addr, prev.input, in, key, prev.output, prev.err, output, err)
				}
			}
		}
	}
}

// TestPrecompileCacheNormalizationCollapsesPadding checks the other direction
// for the precompiles that read a fixed prefix: padding must land on the entry
// the unpadded call already made, otherwise every padded call mints its own.
func TestPrecompileCacheNormalizationCollapsesPadding(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	for _, tc := range []struct {
		name  string
		p     PrecompiledContract
		read  int
		valid int
	}{
		{"ecrecover", &ecrecover{}, ecRecoverInputLength, ecRecoverInputLength},
		{"bn256ScalarMul", &bn256ScalarMulIstanbul{}, bn256ScalarMulInputLength, bn256ScalarMulInputLength},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := make([]byte, tc.valid)
			rng.Read(body)

			want, ok := precompileCacheKey(tc.p, body)
			if !ok {
				t.Fatal("unpadded input is not cacheable")
			}
			for _, n := range []int{tc.read + 1, tc.read * 2, maxCacheablePrecompileInput} {
				padded := make([]byte, n)
				copy(padded, body)
				got, ok := precompileCacheKey(tc.p, padded)
				if !ok {
					t.Errorf("input padded to %d is not cacheable", n)
					continue
				}
				if !bytes.Equal(got, want) {
					t.Errorf("padding to %d changed the key: %x != %x", n, got, want)
				}
			}
		})
	}
	// modexp declares its own operand lengths, so padding past them collapses
	// too even though the read length is not a constant.
	modexp := &bigModExp{eip2565: true}
	header := make([]byte, modExpHeaderLength)
	header[31], header[63], header[95] = 1, 1, 1 // one byte each of base, exp, mod
	body := append(append([]byte{}, header...), 2, 3, 5)

	want, ok := precompileCacheKey(modexp, body)
	if !ok {
		t.Fatal("modexp body is not cacheable")
	}
	padded := make([]byte, 4096)
	copy(padded, body)
	got, ok := precompileCacheKey(modexp, padded)
	if !ok {
		t.Fatal("padded modexp is not cacheable")
	}
	if !bytes.Equal(got, want) {
		t.Errorf("modexp padding changed the key: %x != %x", got, want)
	}
}

// TestPrecompileCacheOptIn asserts that caching is opt in, so a new precompile
// cannot be enrolled by accident before anyone has checked that its output is a
// pure function of its input.
func TestPrecompileCacheOptIn(t *testing.T) {
	if _, ok := precompileCacheKey(&undeclaredPrecompile{}, nil); ok {
		t.Fatal("a precompile that does not implement CacheablePrecompile is cacheable")
	}
}

// TestPrecompileCacheHit runs a precompile twice through the cache and asserts
// the served result matches the computed one, gas included.
func TestPrecompileCacheHit(t *testing.T) {
	var (
		cache = NewPrecompileCache()
		addr  = common.BytesToAddress([]byte{8})
		p     = PrecompiledContractsOsaka[addr]
		rules = params.Rules{IsBerlin: true, IsIstanbul: true, IsByzantium: true}
		input = make([]byte, 192)
	)
	gasCost := p.RequiredGas(input)
	key, ok := precompileCacheKey(p, input)
	if !ok {
		t.Fatalf("%s is not cacheable, pick a different fixture", p.Name())
	}
	want, wantGas, wantErr := RunPrecompiledContract(nil, p, addr, input, NewGasBudget(gasCost, 0), nil, rules, cache)
	got, gotGas, gotErr := RunPrecompiledContract(nil, p, addr, input, NewGasBudget(gasCost, 0), nil, rules, cache)
	if !bytes.Equal(got, want) {
		t.Errorf("cached output %x, computed %x", got, want)
	}
	if gotGas != wantGas {
		t.Errorf("cached gas %v, computed %v", gotGas, wantGas)
	}
	if gotErr != wantErr {
		t.Errorf("cached error %v, computed %v", gotErr, wantErr)
	}
	// Prove the second call was served rather than recomputed, the assertions
	// above would also pass if the cache never stored anything.
	scope := precompileCacheScope{activePrecompiledContracts(rules), addr}
	if _, ok := cache.load(scope, key); !ok {
		t.Error("result was not stored under its key")
	}
}

func errEqual(a, b error) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Error() == b.Error()
}

// undeclaredPrecompile stands in for a precompile added without a caching
// decision.
type undeclaredPrecompile struct{}

func (c *undeclaredPrecompile) RequiredGas([]byte) uint64  { return 0 }
func (c *undeclaredPrecompile) Run([]byte) ([]byte, error) { return nil, nil }
func (c *undeclaredPrecompile) Name() string               { return "UNDECLARED" }

// precompileCacheSynthetic lists the precompiles with no test vectors on disk,
// benchmarked on generated inputs instead.
var precompileCacheSynthetic = []struct{ name, addr string }{
	{"SHA256", "02"}, {"RIPEMD160", "03"}, {"ID", "04"},
}

// precompileCacheFixtures maps a testdata fixture to the address its precompile
// sits at in allPrecompiles. The Byzantium bn256 variants share their Run with
// the Istanbul ones and only differ in price, so they would be duplicate rows.
var precompileCacheFixtures = []struct{ fixture, addr string }{
	{"ecRecover", "01"}, {"modexp_eip2565", "f5"}, {"bn256Add", "06"},
	{"bn256ScalarMul", "07"}, {"bn256Pairing", "08"}, {"blake2F", "09"},
	{"pointEvaluation", "0a"}, {"blsG1Add", "f0a"}, {"blsG2Add", "f0c"},
	{"blsG1MultiExp", "f0b"}, {"blsG2MultiExp", "f0d"}, {"blsPairing", "f0e"},
	{"blsMapG1", "f0f"}, {"blsMapG2", "f10"}, {"p256Verify", "0b"},
}

// BenchmarkPrecompileCacheHitVsRun compares serving a warm entry against just
// running the precompile, over the real test vectors. It is the evidence for
// which precompiles opt in: caching only belongs where the hit is the cheaper
// path, and for sha256 and blake2F that margin is thin enough to be worth
// rechecking on the machine you care about. Run it with -count and read
// medians, the first case measured absorbs the warm-up.
func BenchmarkPrecompileCacheHitVsRun(b *testing.B) {
	rules := params.Rules{IsByzantium: true, IsIstanbul: true, IsBerlin: true, IsCancun: true, IsPrague: true, IsOsaka: true}
	set := activePrecompiledContracts(rules)

	for _, bench := range precompileCacheFixtures {
		tests, err := loadJson(bench.fixture)
		if err != nil {
			b.Fatalf("%s: %v", bench.fixture, err)
		}
		addr := common.HexToAddress(bench.addr)
		p := allPrecompiles[addr]
		for _, tc := range tests {
			if tc.NoBenchmark {
				continue
			}
			in := common.Hex2Bytes(tc.Input)
			key, ok := precompileCacheKey(p, in)
			if !ok || p.RequiredGas(in) > probeGasLimit {
				continue
			}
			out, err := p.Run(in)
			if err != nil {
				continue // errors are never cached, there is no hit to measure
			}
			cache := NewPrecompileCache()
			scope := precompileCacheScope{set, addr}
			cache.store(scope, key, out)

			label := fmt.Sprintf("%s/%s/len=%d", bench.fixture, tc.Name, len(in))
			b.Run(label+"/hit", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					k, _ := precompileCacheKey(p, in)
					cache.load(scope, k)
				}
			})
			b.Run(label+"/run", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					p.Run(in)
				}
			})
		}
	}
}

// BenchmarkPrecompileCacheSynthetic covers the precompiles that ship no test
// vectors, which are also the ones whose caching decision is closest. Their
// cost scales with input length rather than varying per vector, so synthetic
// inputs of a few sizes say more here than a fixture corpus would.
func BenchmarkPrecompileCacheSynthetic(b *testing.B) {
	rules := params.Rules{IsByzantium: true, IsIstanbul: true, IsBerlin: true, IsCancun: true, IsPrague: true, IsOsaka: true}
	set := activePrecompiledContracts(rules)

	for _, tc := range precompileCacheSynthetic {
		addr := common.HexToAddress(tc.addr)
		p := allPrecompiles[addr]
		for _, n := range []int{32, 128, 1024, 8192} {
			in := make([]byte, n)
			for i := range in {
				in[i] = byte(i)
			}
			out, err := p.Run(in)
			if err != nil {
				continue
			}
			var (
				scope = precompileCacheScope{set, addr}
				cache = NewPrecompileCache()
			)
			label := fmt.Sprintf("%s/len=%d", tc.name, n)

			// Only the precompiles that opt in get a hit measured, going through
			// the same path RunPrecompiledContract does. For the one that does
			// not, its run cost against the lookups below is the whole argument.
			if key, ok := precompileCacheKey(p, in); ok && len(out) <= maxCacheablePrecompileOutput {
				cache.store(scope, key, out)
				b.Run(label+"/hit", func(b *testing.B) {
					b.ReportAllocs()
					for b.Loop() {
						k, _ := precompileCacheKey(p, in)
						cache.load(scope, k)
					}
				})
			}
			b.Run(label+"/run", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					p.Run(in)
				}
			})
		}
	}
}

// TestPrecompileCacheBoundedByKeyBytes checks that the precompile result cache
// stays within its byte budget even when its entries carry empty outputs, such
// as a failed ECRECOVER. The cache key is the precompile input, so it must
// count towards the budget; otherwise a stream of distinct failing inputs would
// grow the cache without bound, since an empty output adds nothing to the size.
func TestPrecompileCacheBoundedByKeyBytes(t *testing.T) {
	cache := NewPrecompileCache()
	scope := precompileCacheScope{activePrecompiledContracts(params.Rules{}), common.BytesToAddress([]byte{0x1})}
	ec := &ecrecover{}

	// A failing signature runs to an empty output with no error, and is still
	// eligible for caching. Keep its key as the eviction witness.
	var first [128]byte
	first[63] = 0xff // invalid v
	out, err := ec.Run(first[:])
	if err != nil || len(out) != 0 {
		t.Fatalf("expected empty failed-ecrecover output, got len=%d err=%v", len(out), err)
	}
	firstKey, ok := precompileCacheKey(ec, first[:])
	if !ok {
		t.Fatal("failed ecrecover unexpectedly not cacheable")
	}
	cache.store(scope, firstKey, out)

	// Store many more distinct failing inputs. A trailing sentinel byte keeps
	// every normalized key a full 128 bytes, so their key bytes alone overrun
	// the 1 MiB budget several times over.
	const n = 20000
	for i := 1; i < n; i++ {
		var in [128]byte
		in[63] = 0xff
		in[127] = 0x01
		binary.BigEndian.PutUint64(in[64:72], uint64(i))
		key, ok := precompileCacheKey(ec, in[:])
		if !ok {
			t.Fatalf("input %d not cacheable", i)
		}
		cache.store(scope, key, nil)
	}
	if _, hit := cache.load(scope, firstKey); hit {
		t.Fatal("cache is unbounded: oldest empty-output entry was never evicted")
	}
}
