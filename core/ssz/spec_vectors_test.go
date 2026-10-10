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

package ssz_test

// Conformance harness for the ssz_generic suite of consensus-spec-tests.
//
// Every valid case is checked three ways per tests/formats/ssz_generic/
// README.md: the value from value.yaml must serialize to serialized.ssz_snappy,
// those bytes must decode and re-serialize to themselves, and both objects
// must hash to the root in meta.yaml. Every invalid case must be rejected with
// an error wrapping a named sentinel of the package, and where the case name
// pins down the failure, with that specific sentinel.
//
// The six classic handlers are covered. The progressive handlers, and the
// ProgressiveTestStruct and ProgressiveBitsStruct cases inside the containers
// handler, need the progressive merkleization of EIP-7916 and are skipped
// until it lands.
//
// The vectors are not committed. CI downloads them through build/ci.go into
// tests/consensus-spec-tests, pinned and checksummed in build/checksums.txt.
// Set GETH_SSZ_SPEC_TESTS to point at an ssz_generic directory extracted
// elsewhere. The test is skipped when neither location exists.

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core/ssz"
	"github.com/golang/snappy"
	"gopkg.in/yaml.v3"
)

// specTestVersion is the consensus-specs release the harness was last run
// against.
const specTestVersion = "v1.7.0-alpha.12"

// classicHandlers are the ssz_generic handlers without progressive types.
var classicHandlers = []string{"boolean", "uints", "bitvector", "basic_vector", "bitlist", "containers"}

// sentinels is the complete rejection contract of the package: every decode
// error wraps exactly one of these.
var sentinels = []error{
	ssz.ErrSize, ssz.ErrOffset, ssz.ErrTrailing, ssz.ErrBadBoolean,
	ssz.ErrBadBitlist, ssz.ErrExcessBits, ssz.ErrTooBig,
}

// makeSpecObject constructs the zero value of the SSZ type a case name
// declares, per the naming scheme of tests/formats/ssz_generic/README.md.
func makeSpecObject(handler, name string) (specObject, error) {
	tokens := strings.Split(name, "_")
	switch handler {
	case "boolean":
		return &testBasic{kind: kindBool}, nil

	case "uints":
		// uint_<size>_<variant>
		if len(tokens) < 2 {
			return nil, fmt.Errorf("bad uints case %q", name)
		}
		kind, err := parseBasicKind("uint" + tokens[1])
		if err != nil {
			return nil, err
		}
		return &testBasic{kind: kind}, nil

	case "basic_vector":
		// vec_<type>_<N>_<variant>
		if len(tokens) < 3 {
			return nil, fmt.Errorf("bad basic_vector case %q", name)
		}
		kind, err := parseBasicKind(tokens[1])
		if err != nil {
			return nil, err
		}
		length, err := strconv.Atoi(tokens[2])
		if err != nil {
			return nil, fmt.Errorf("bad vector length in %q: %v", name, err)
		}
		return &testBasicVector{kind: kind, length: length}, nil

	case "bitvector":
		// bitvec_<N>_<variant>
		if len(tokens) < 2 {
			return nil, fmt.Errorf("bad bitvector case %q", name)
		}
		nbits, err := strconv.ParseUint(tokens[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("bad bitvector length in %q: %v", name, err)
		}
		return &testBitvector{nbits: nbits}, nil

	case "bitlist":
		// bitlist_<limit>_<variant>
		if len(tokens) < 2 {
			return nil, fmt.Errorf("bad bitlist case %q", name)
		}
		limit, err := strconv.ParseUint(tokens[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("bad bitlist limit in %q: %v", name, err)
		}
		return &testBitlist{limit: limit}, nil

	case "containers":
		// <StructName>_<variant>
		switch tokens[0] {
		case "SingleFieldTestStruct":
			return new(singleFieldTestStruct), nil
		case "SmallTestStruct":
			return new(smallTestStruct), nil
		case "FixedTestStruct":
			return new(fixedTestStruct), nil
		case "VarTestStruct":
			return new(varTestStruct), nil
		case "ComplexTestStruct":
			return new(complexTestStruct), nil
		case "BitsStruct":
			return new(bitsStruct), nil
		}
		return nil, fmt.Errorf("unknown container in case %q", name)
	}
	return nil, fmt.Errorf("unknown handler %q", handler)
}

// expectedError names the sentinel an invalid case must produce, derived from
// how tests/generators/ssz_generic builds the case. It returns nil when the
// name does not pin the failure down: a corrupted container offset is caught
// either at the offset table or inside the section it misaligns, and which
// one depends on the field type.
func expectedError(handler, name string) (error, error) {
	tokens := strings.Split(name, "_")
	switch handler {
	case "boolean":
		// byte_<value>: a byte other than 0x00 or 0x01.
		return ssz.ErrBadBoolean, nil

	case "uints":
		// uint_<size>_one_byte_shorter | one_byte_longer | one_too_high,
		// the last serialized at the wider width the value needs.
		if strings.HasSuffix(name, "one_byte_shorter") {
			return ssz.ErrSize, nil
		}
		return ssz.ErrTrailing, nil

	case "bitvector":
		// bitvec_0 (illegal type), or bitvec_<N>_<mode>_<M>: a Bitvector[N]
		// given M bits with bit N set.
		if name == "bitvec_0" {
			return ssz.ErrSize, nil
		}
		if len(tokens) < 4 {
			return nil, fmt.Errorf("bad bitvector case %q", name)
		}
		n, err := strconv.ParseUint(tokens[1], 10, 64)
		if err != nil {
			return nil, err
		}
		m, err := strconv.ParseUint(tokens[3], 10, 64)
		if err != nil {
			return nil, err
		}
		if ssz.BitvectorSize(n) != ssz.BitvectorSize(m) {
			return ssz.ErrSize, nil
		}
		return ssz.ErrExcessBits, nil

	case "bitlist":
		// bitlist_<limit>_but_<M> (M bits over the limit), or
		// bitlist_<limit>_no_delimiter_<empty|zero_byte|zeroes>.
		if strings.Contains(name, "no_delimiter") {
			return ssz.ErrBadBitlist, nil
		}
		return ssz.ErrTooBig, nil

	case "basic_vector":
		// vec_<T>_0 (illegal type), vec_bool_<N>_<mode>_<byte> (a byte that
		// is not a boolean), or vec_<T>_<N>_<mode>_one_<byte|element>_<less|more>.
		for _, bad := range []string{"_0x80", "_0xff", "_2", "_rev_nibble"} {
			if strings.HasSuffix(name, bad) {
				return ssz.ErrBadBoolean, nil
			}
		}
		return ssz.ErrSize, nil

	case "containers":
		if !strings.HasSuffix(name, "_extra_byte") {
			return nil, nil // offset mutation
		}
		// One zero byte appended to the serialization.
		switch tokens[0] {
		case "SingleFieldTestStruct", "SmallTestStruct", "FixedTestStruct":
			return ssz.ErrTrailing, nil // no variable part to absorb it
		case "VarTestStruct", "ComplexTestStruct":
			return ssz.ErrSize, nil // lands in the last uint16 list, an odd byte count
		case "BitsStruct":
			return ssz.ErrBadBitlist, nil // lands in the last bitlist, whose last byte is now zero
		}
	}
	return nil, fmt.Errorf("no expectation for %s/%s", handler, name)
}

func readSerialized(t *testing.T, dir string) []byte {
	compressed, err := os.ReadFile(filepath.Join(dir, "serialized.ssz_snappy"))
	if err != nil {
		t.Fatalf("reading serialized.ssz_snappy: %v", err)
	}
	data, err := snappy.Decode(nil, compressed)
	if err != nil {
		t.Fatalf("snappy: %v", err)
	}
	return data
}

func readMetaRoot(t *testing.T, dir string) [32]byte {
	blob, err := os.ReadFile(filepath.Join(dir, "meta.yaml"))
	if err != nil {
		t.Fatalf("reading meta.yaml: %v", err)
	}
	var meta struct {
		Root string `yaml:"root"`
	}
	if err := yaml.Unmarshal(blob, &meta); err != nil {
		t.Fatalf("parsing meta.yaml: %v", err)
	}
	b, err := hex.DecodeString(strings.TrimPrefix(meta.Root, "0x"))
	if err != nil || len(b) != 32 {
		t.Fatalf("bad meta root %q", meta.Root)
	}
	var root [32]byte
	copy(root[:], b)
	return root
}

func readValueNode(t *testing.T, dir string) *yaml.Node {
	blob, err := os.ReadFile(filepath.Join(dir, "value.yaml"))
	if err != nil {
		t.Fatalf("reading value.yaml: %v", err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(blob, &doc); err != nil {
		t.Fatalf("parsing value.yaml: %v", err)
	}
	if len(doc.Content) != 1 {
		t.Fatalf("value.yaml has %d documents", len(doc.Content))
	}
	return doc.Content[0]
}

// vectorsDir locates the extracted ssz_generic directory or skips the test.
func vectorsDir(t *testing.T) string {
	if dir := os.Getenv("GETH_SSZ_SPEC_TESTS"); dir != "" {
		return dir
	}
	dir := filepath.Join("..", "..", "tests", "consensus-spec-tests",
		"tests", "general", "phase0", "ssz_generic")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("consensus-spec-tests %s not found; run go run build/ci.go test, or set GETH_SSZ_SPEC_TESTS", specTestVersion)
	}
	return dir
}

// runValidCase checks encoding, decoding and hash-tree-root.
func runValidCase(t *testing.T, handler, name, dir string) {
	obj, err := makeSpecObject(handler, name)
	if err != nil {
		t.Fatalf("type construction: %v", err)
	}
	serialized := readSerialized(t, dir)

	// Encoding: value.yaml -> object -> bytes must equal serialized.
	if err := obj.fromYAML(readValueNode(t, dir)); err != nil {
		t.Fatalf("loading value.yaml: %v", err)
	}
	if size := obj.SizeSSZ(); size != len(serialized) {
		t.Errorf("SizeSSZ = %d, serialized length %d", size, len(serialized))
	}
	encoded, err := ssz.Encode(obj)
	if err != nil {
		t.Fatalf("encoding: %v", err)
	}
	if !bytes.Equal(encoded, serialized) {
		t.Errorf("encoding mismatch:\n  got  %x\n  want %x", encoded, serialized)
	}

	// Decoding: serialized -> object -> bytes must round-trip.
	decoded, _ := makeSpecObject(handler, name)
	if err := ssz.Decode(serialized, decoded); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	reencoded, err := ssz.Encode(decoded)
	if err != nil {
		t.Fatalf("re-encoding: %v", err)
	}
	if !bytes.Equal(reencoded, serialized) {
		t.Errorf("decode round-trip mismatch:\n  got  %x\n  want %x", reencoded, serialized)
	}

	// Hash-tree-root of both the YAML-built and the decoded object.
	wantRoot := readMetaRoot(t, dir)
	if got := obj.HashTreeRoot(); got != wantRoot {
		t.Errorf("root (from yaml) = %x, want %x", got, wantRoot)
	}
	if got := decoded.HashTreeRoot(); got != wantRoot {
		t.Errorf("root (from decode) = %x, want %x", got, wantRoot)
	}
}

// runInvalidCase requires a clean rejection: never a panic, never success,
// and an error wrapping the expected sentinel.
func runInvalidCase(t *testing.T, handler, name, dir string) {
	obj, err := makeSpecObject(handler, name)
	if err != nil {
		t.Fatalf("type construction: %v", err)
	}
	want, err := expectedError(handler, name)
	if err != nil {
		t.Fatalf("expectation: %v", err)
	}
	serialized := readSerialized(t, dir)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("decoder panicked on invalid input: %v", r)
		}
	}()
	err = ssz.Decode(serialized, obj)
	if err == nil {
		t.Fatalf("invalid input of %d bytes decoded without error", len(serialized))
	}
	if want != nil {
		if !errors.Is(err, want) {
			t.Errorf("error %q does not wrap %q", err, want)
		}
		return
	}
	for _, s := range sentinels {
		if errors.Is(err, s) {
			return
		}
	}
	t.Errorf("error %q wraps no named sentinel", err)
}

func TestSpecVectors(t *testing.T) {
	root := vectorsDir(t)
	for _, handler := range classicHandlers {
		t.Run(handler, func(t *testing.T) {
			for _, suite := range []string{"valid", "invalid"} {
				t.Run(suite, func(t *testing.T) {
					cases, err := os.ReadDir(filepath.Join(root, handler, suite))
					if err != nil {
						t.Fatal(err)
					}
					if len(cases) == 0 {
						t.Fatalf("no %s cases found", suite)
					}
					for _, c := range cases {
						if !c.IsDir() {
							continue
						}
						name := c.Name()
						dir := filepath.Join(root, handler, suite, name)
						t.Run(name, func(t *testing.T) {
							if strings.HasPrefix(name, "Progressive") {
								t.Skip("needs EIP-7916 progressive merkleization")
							}
							if suite == "valid" {
								runValidCase(t, handler, name, dir)
							} else {
								runInvalidCase(t, handler, name, dir)
							}
						})
					}
				})
			}
		})
	}
}
