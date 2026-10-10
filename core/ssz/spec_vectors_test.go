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
// Only the handlers whose root needs no decoder are checked here. For a
// basic value, a bitvector or a vector of basic values, hash_tree_root is
// Merkleize(Pack(serialized), limit) with the limit fixed by the type, so the
// serialized bytes can be fed to the merkleizer directly. The remaining
// handlers, and every invalid case, exercise the decoder and arrive with it.
//
// The vectors are not committed. CI downloads them through build/ci.go into
// tests/consensus-spec-tests, pinned and checksummed in build/checksums.txt.
// Set GETH_SSZ_SPEC_TESTS to point at an ssz_generic directory extracted
// elsewhere. The test is skipped when neither location exists.

import (
	"encoding/hex"
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

// rootOnlyHandlers are the ssz_generic handlers whose valid cases can be
// checked without a decoder.
var rootOnlyHandlers = []string{"boolean", "uints", "bitvector", "basic_vector"}

// basicSizes maps the basic type names used in case names to their
// serialized size in bytes.
var basicSizes = map[string]uint64{
	"bool": 1, "uint8": 1, "uint16": 2, "uint32": 4, "uint64": 8, "uint128": 16, "uint256": 32,
}

// chunkLimit derives the merkleization limit of a case from its handler and
// name, following the naming scheme of tests/formats/ssz_generic/README.md.
func chunkLimit(handler, name string) (uint64, error) {
	parts := strings.Split(name, "_")
	switch handler {
	case "boolean", "uints":
		return 1, nil
	case "bitvector":
		// bitvec_<N>_<variant>
		n, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return 0, err
		}
		return (n + 255) / 256, nil
	case "basic_vector":
		// vec_<type>_<N>_<variant>
		size, ok := basicSizes[parts[1]]
		if !ok {
			return 0, fmt.Errorf("unknown basic type %q", parts[1])
		}
		n, err := strconv.ParseUint(parts[2], 10, 64)
		if err != nil {
			return 0, err
		}
		return (n*size + 31) / 32, nil
	}
	return 0, fmt.Errorf("unknown handler %q", handler)
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

func TestSpecVectors(t *testing.T) {
	root := vectorsDir(t)
	for _, handler := range rootOnlyHandlers {
		t.Run(handler, func(t *testing.T) {
			cases, err := os.ReadDir(filepath.Join(root, handler, "valid"))
			if err != nil {
				t.Fatal(err)
			}
			if len(cases) == 0 {
				t.Fatal("no valid cases found")
			}
			for _, c := range cases {
				if !c.IsDir() {
					continue
				}
				name := c.Name()
				t.Run(name, func(t *testing.T) {
					dir := filepath.Join(root, handler, "valid", name)
					limit, err := chunkLimit(handler, name)
					if err != nil {
						t.Fatalf("deriving limit: %v", err)
					}
					got := ssz.Merkleize(ssz.Pack(readSerialized(t, dir)), limit)
					if want := readMetaRoot(t, dir); got != want {
						t.Errorf("root %x, want %x", got, want)
					}
				})
			}
		})
	}
}
