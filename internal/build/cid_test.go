// Copyright 2024 The go-ethereum Authors
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

package build

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestBase58Encode(t *testing.T) {
	tests := []struct {
		input    []byte
		expected string
	}{
		{[]byte{}, ""},
		{[]byte{0}, "1"},
		{[]byte{0, 0, 0}, "111"},
		{[]byte("Hello World!"), "2NEpo7TZRRrLZSi2U"},
	}

	for _, tt := range tests {
		result := base58Encode(tt.input)
		if result != tt.expected {
			t.Errorf("base58Encode(%v) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestComputeCID(t *testing.T) {
	tests := []struct {
		name        string
		content     []byte
		wantV1Start string
		wantMHStart string
		wantV1Len   int
		wantMHLen   int
	}{
		{
			name:        "empty content",
			content:     []byte{},
			wantV1Start: "bafkrei",
			wantMHStart: "Qm",
			wantV1Len:   59,
			wantMHLen:   46,
		},
		{
			name:        "hello world",
			content:     []byte("hello world"),
			wantV1Start: "bafkrei",
			wantMHStart: "Qm",
			wantV1Len:   59,
			wantMHLen:   46,
		},
		{
			name:        "binary content",
			content:     []byte{0x00, 0x01, 0x02, 0x03, 0xff, 0xfe, 0xfd},
			wantV1Start: "bafkrei",
			wantMHStart: "Qm",
			wantV1Len:   59,
			wantMHLen:   46,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cid, err := ComputeCID(bytes.NewReader(tt.content))
			if err != nil {
				t.Fatalf("ComputeCID() error = %v", err)
			}

			// Check CIDv1 format
			if !strings.HasPrefix(cid.V1, tt.wantV1Start) {
				t.Errorf("V1 = %q, want prefix %q", cid.V1, tt.wantV1Start)
			}
			if len(cid.V1) != tt.wantV1Len {
				t.Errorf("V1 length = %d, want %d", len(cid.V1), tt.wantV1Len)
			}

			// Check multihash format
			if !strings.HasPrefix(cid.Multihash, tt.wantMHStart) {
				t.Errorf("Multihash = %q, want prefix %q", cid.Multihash, tt.wantMHStart)
			}
			if len(cid.Multihash) != tt.wantMHLen {
				t.Errorf("Multihash length = %d, want %d", len(cid.Multihash), tt.wantMHLen)
			}

			// CIDv1 should be lowercase
			if cid.V1 != strings.ToLower(cid.V1) {
				t.Errorf("V1 should be lowercase: %q", cid.V1)
			}
		})
	}
}

func TestComputeCIDDeterministic(t *testing.T) {
	content := []byte("deterministic test content")

	cid1, err := ComputeCID(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("ComputeCID() error = %v", err)
	}

	cid2, err := ComputeCID(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("ComputeCID() error = %v", err)
	}

	if cid1.V1 != cid2.V1 {
		t.Errorf("V1 not deterministic: %q != %q", cid1.V1, cid2.V1)
	}
	if cid1.Multihash != cid2.Multihash {
		t.Errorf("Multihash not deterministic: %q != %q", cid1.Multihash, cid2.Multihash)
	}
}

// TestKnownCID verifies against a known IPFS CID.
// Verified with: echo -n "hello" | ipfs add --only-hash --raw-leaves -Q
// Output: bafkreibm6jg3ux5qumhcn2b3flc3tyu6dmlb4xa7u5bf44yegnrjhc4yeq
func TestKnownCID(t *testing.T) {
	content := []byte("hello")
	cid, err := ComputeCID(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("ComputeCID() error = %v", err)
	}

	// This is the CIDv1 for raw "hello" bytes
	// Verified with: echo -n "hello" | ipfs add --only-hash --raw-leaves -Q
	expectedV1 := "bafkreibm6jg3ux5qumhcn2b3flc3tyu6dmlb4xa7u5bf44yegnrjhc4yeq"
	if cid.V1 != expectedV1 {
		t.Errorf("V1 for 'hello' = %q, want %q", cid.V1, expectedV1)
	}

	t.Logf("V1 (CIDv1):    %s", cid.V1)
	t.Logf("Multihash:     %s", cid.Multihash)
}

// TestEmptyContent verifies the CID for empty content.
// Verified with: echo -n "" | ipfs add --only-hash --raw-leaves -Q
// Output: bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku
func TestEmptyContent(t *testing.T) {
	content := []byte{}
	cid, err := ComputeCID(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("ComputeCID() error = %v", err)
	}

	// This is the CIDv1 for empty content (SHA256 of nothing)
	expectedV1 := "bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku"
	if cid.V1 != expectedV1 {
		t.Errorf("V1 for empty = %q, want %q", cid.V1, expectedV1)
	}

	t.Logf("V1 (CIDv1):    %s", cid.V1)
	t.Logf("Multihash:     %s", cid.Multihash)
}

// TestReadmeFile verifies CID computation on an actual file in the repo.
// Run: ipfs add --only-hash --raw-leaves -Q ../../README.md
// to get the expected CID for comparison.
func TestReadmeFile(t *testing.T) {
	// This test only runs if the README.md exists (it should in the repo)
	path := "../../README.md"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("README.md not found, skipping file test")
	}

	cid, err := ComputeFileCID(path)
	if err != nil {
		t.Fatalf("ComputeFileCID() error = %v", err)
	}

	// Just verify it produces valid-looking CIDs
	if !strings.HasPrefix(cid.V1, "bafkrei") {
		t.Errorf("V1 should start with bafkrei: %s", cid.V1)
	}
	if !strings.HasPrefix(cid.Multihash, "Qm") {
		t.Errorf("Multihash should start with Qm: %s", cid.Multihash)
	}

	t.Logf("README.md CIDv1: %s", cid.V1)
	t.Logf("To verify: ipfs add --only-hash --raw-leaves -Q ../../README.md")
}
