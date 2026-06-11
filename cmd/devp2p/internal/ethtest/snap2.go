// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package ethtest

import (
	"bytes"
	"fmt"
	"math/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/protocols/snap"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/rlp"
)

// Snap/2 (EIP-8189) replaces trie node healing with BAL-based state catch-up.
// It keeps 0x00..0x05 (AccountRange/StorageRanges/ByteCodes) unchanged, removes
// GetTrieNodes (0x06) / TrieNodes (0x07), and adds GetBlockAccessLists (0x08) /
// BlockAccessLists (0x09).
//
// The tests in this file focus on the wire behavior that is new or changed in
// snap/2. Tests for the unchanged messages are already covered by the snap/1
// suite in snap.go; the harness reuses the same code paths because those
// message formats are identical across versions.

// TestSnap2Status performs an RLPx+eth+snap/2 handshake against the node,
// verifying that the node advertises and negotiates snap/2.
func (s *Suite) TestSnap2Status(t *utesting.T) {
	t.Log(`This test performs a snap/2 (EIP-8189) handshake. The peer is expected to
advertise snap/2 as a p2p capability and accept the connection.`)

	conn, err := s.dialSnap2()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err := conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}
	if conn.negotiatedSnapProtoVersion != 2 {
		t.Fatalf("unexpected negotiated snap version: got %d, want 2", conn.negotiatedSnapProtoVersion)
	}
}

type accessListsTest struct {
	nBytes uint64
	hashes []common.Hash

	// minEntries/maxEntries bound the number of entries the response list
	// MUST contain. Per EIP-8189 the server may truncate from the tail when
	// the byte soft limit is reached, but MUST preserve request order.
	minEntries int
	maxEntries int

	desc string
}

// TestSnap2GetBlockAccessLists exercises various forms of GetBlockAccessLists
// requests defined in EIP-8189. Per the spec:
//
//   - Nodes MUST always respond.
//   - Unavailable BALs are returned as the RLP empty string (0x80) at the
//     matching position.
//   - The server MAY return fewer entries than requested (respecting the byte
//     soft limit or QoS limits), truncating from the tail.
//   - Returned entries MUST preserve request order.
//   - When a BAL is returned, its keccak256(rlp.encode(bal)) MUST match the
//     block-access-list-hash field of the corresponding block header.
func (s *Suite) TestSnap2GetBlockAccessLists(t *utesting.T) {
	var (
		head     = s.chain.Head()
		headHash = head.Hash()
		preHash  = s.chain.blocks[s.chain.Len()-2].Hash()
		unknown  = common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	)

	// Collect a window of recent canonical block hashes. Limit to at most 16
	// entries to keep the request small and well under any reasonable limit.
	var recent []common.Hash
	start := s.chain.Len() - 16
	if start < 1 {
		start = 1
	}
	for i := start; i < s.chain.Len(); i++ {
		recent = append(recent, s.chain.blocks[i].Hash())
	}

	tests := []accessListsTest{
		{
			desc: `An empty request. The server must respond with an empty list and must
not disconnect.`,
			nBytes:     softResponseLimitSnap,
			hashes:     nil,
			minEntries: 0,
			maxEntries: 0,
		},
		{
			desc: `A request for a single random/unknown block hash. Per the spec the
server must respond and include an RLP empty string (0x80) at that position.`,
			nBytes:     softResponseLimitSnap,
			hashes:     []common.Hash{unknown},
			minEntries: 1,
			maxEntries: 1,
		},
		{
			desc: `A request for multiple random/unknown block hashes. The server must
preserve request order and return an RLP empty string for each position.`,
			nBytes: softResponseLimitSnap,
			hashes: []common.Hash{
				unknown,
				common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
				common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002"),
			},
			minEntries: 3,
			maxEntries: 3,
		},
		{
			desc: `A request for the chain head. The server must respond. If the node is
post-Amsterdam and has the BAL for this block, the returned BAL must hash to
the block-access-list-hash in the header. Otherwise an empty entry is valid.`,
			nBytes:     softResponseLimitSnap,
			hashes:     []common.Hash{headHash},
			minEntries: 1,
			maxEntries: 1,
		},
		{
			desc: `A request for the chain head and its parent. The server must return
exactly two entries, in request order.`,
			nBytes:     softResponseLimitSnap,
			hashes:     []common.Hash{headHash, preHash},
			minEntries: 2,
			maxEntries: 2,
		},
		{
			desc: `A mixed request with known and unknown hashes. The server must
return entries in request order, with the RLP empty string at positions
corresponding to unknown hashes.`,
			nBytes: softResponseLimitSnap,
			hashes: []common.Hash{headHash, unknown, preHash, unknown},
			// We expect exactly 4 entries — mixed responses are small and well
			// under the byte limit, so truncation is not expected.
			minEntries: 4,
			maxEntries: 4,
		},
		{
			desc: `A request spanning the most recent canonical window. Implementations
may serve or drop individual entries, but the entries that are returned must
preserve request order.`,
			nBytes:     softResponseLimitSnap,
			hashes:     recent,
			minEntries: 0,
			maxEntries: len(recent),
		},
		{
			desc: `A request with a very small byte soft limit. The server must return
at least zero entries and no more than the requested number, truncating from
the tail. It must not disconnect.`,
			nBytes:     1,
			hashes:     recent,
			minEntries: 0,
			maxEntries: len(recent),
		},
		{
			desc: `A request with a zero byte soft limit. The server must still respond
(possibly with an empty list) and must not disconnect.`,
			nBytes:     0,
			hashes:     recent,
			minEntries: 0,
			maxEntries: len(recent),
		},
		{
			desc: `A request containing the same hash repeated. The server must treat
each position independently and preserve request order.`,
			nBytes:     softResponseLimitSnap,
			hashes:     []common.Hash{headHash, headHash, headHash},
			minEntries: 3,
			maxEntries: 3,
		},
	}

	for i, tc := range tests {
		if i > 0 {
			t.Log("\n")
		}
		t.Logf("-- Test %d", i)
		t.Log(tc.desc)
		t.Log("  request:")
		t.Logf("      hashes: %d", len(tc.hashes))
		t.Logf("      responseBytes: %d", tc.nBytes)
		if err := s.snapGetAccessLists(t, &tc); err != nil {
			t.Errorf("test %d failed: %v", i, err)
		}
	}
}

// TestSnap2TrieNodesRemoved verifies that snap/2 no longer serves the
// GetTrieNodes message (0x06). Per EIP-8189, snap/2 removes GetTrieNodes and
// TrieNodes entirely. A server that negotiated snap/2 must not treat these
// codes as valid snap messages and should disconnect the peer that sends them.
func (s *Suite) TestSnap2TrieNodesRemoved(t *utesting.T) {
	t.Log(`This test verifies that sending a GetTrieNodes message over a snap/2
connection causes the peer to reject the request. Per EIP-8189, GetTrieNodes
is removed in snap/2.`)

	conn, err := s.dialSnap2()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err := conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}

	// Build a syntactically valid GetTrieNodes request to the head state root.
	paths, err := rlp.EncodeToRawList([]snap.TrieNodePathSet{{[]byte{0}}})
	if err != nil {
		t.Fatalf("failed to encode paths: %v", err)
	}
	req := &snap.GetTrieNodesPacket{
		ID:    uint64(rand.Int63()),
		Root:  s.chain.Head().Root(),
		Paths: paths,
		Bytes: 5000,
	}
	if err := conn.Write(snapProto, snap.GetTrieNodesMsg, req); err != nil {
		t.Fatalf("failed to write GetTrieNodes: %v", err)
	}

	// We expect either a disconnect or a read error/timeout. We must NOT
	// receive a valid TrieNodes response. Loop a few times to consume any
	// incidental messages the peer might send (e.g. block updates) before
	// deciding.
	for i := 0; i < 5; i++ {
		msg, err := conn.ReadSnap()
		if err != nil {
			// Disconnect or read error — the peer rejected the request.
			return
		}
		if _, ok := msg.(*snap.TrieNodesPacket); ok {
			t.Fatal("peer responded with TrieNodes over snap/2; GetTrieNodes must be unsupported")
		}
	}
	t.Fatal("peer did not reject GetTrieNodes over snap/2 within the observation window")
}

// softResponseLimitSnap mirrors the recommended 2 MiB soft limit for
// BlockAccessLists responses from EIP-8189 §"Response Size Limit".
const softResponseLimitSnap = 2 * 1024 * 1024

// snapGetAccessLists sends a GetBlockAccessLists request, validates the
// response structure against EIP-8189, and verifies BAL content against the
// block-access-list-hash field of the corresponding block header (when the
// block is known and a BAL was returned).
func (s *Suite) snapGetAccessLists(t *utesting.T, tc *accessListsTest) error {
	conn, err := s.dialSnap2()
	if err != nil {
		return fmt.Errorf("dial failed: %v", err)
	}
	defer conn.Close()
	if err = conn.peer(s.chain, nil); err != nil {
		return fmt.Errorf("peering failed: %v", err)
	}

	req := &snap.GetAccessListsPacket{
		ID:     uint64(rand.Int63()),
		Hashes: tc.hashes,
		Bytes:  tc.nBytes,
	}
	msg, err := conn.snapRequest(snap.GetAccessListsMsg, req)
	if err != nil {
		return fmt.Errorf("access list request failed: %v", err)
	}
	res, ok := msg.(*snap.AccessListsPacket)
	if !ok {
		return fmt.Errorf("unexpected response type: %T", msg)
	}
	if res.ID != req.ID {
		return fmt.Errorf("request id mismatch: got %d, want %d", res.ID, req.ID)
	}

	// Check list length bounds.
	got := res.AccessLists.Len()
	if got < tc.minEntries || got > tc.maxEntries {
		return fmt.Errorf("response has %d entries, want between %d and %d", got, tc.minEntries, tc.maxEntries)
	}

	// Build a map of request-index -> block so we can verify BAL hashes.
	blocks := make(map[int]*types.Block)
	for i, h := range tc.hashes {
		for _, b := range s.chain.blocks {
			if b.Hash() == h {
				blocks[i] = b
				break
			}
		}
	}

	// Iterate the response, validating each entry positionally.
	var (
		idx int
		it  = res.AccessLists.ContentIterator()
	)
	for it.Next() {
		raw := it.Value()
		block := blocks[idx]

		// Empty entry: per spec, indicates BAL is unavailable for that block.
		if bytes.Equal(raw, rlp.EmptyString) {
			if block != nil && block.Header().BlockAccessListHash != nil {
				// Not a failure — the server is allowed to legitimately not
				// have the BAL. But we log it so the test output is diagnosable.
				t.Logf("    entry %d: server returned empty for known post-Amsterdam block %x", idx, tc.hashes[idx])
			}
			idx++
			continue
		}

		// Non-empty entry. A BAL is only legitimate for a block we know
		// locally whose header commits to one; for any other hash the only
		// valid response is the RLP empty string, so receiving data here
		// means the server fabricated it.
		if block == nil {
			return fmt.Errorf("entry %d: server returned BAL data for unknown hash %x", idx, tc.hashes[idx])
		}
		if block.Header().BlockAccessListHash == nil {
			return fmt.Errorf("entry %d: server returned BAL data for a block with no expected BAL (hash %x)", idx, tc.hashes[idx])
		}

		// Per EIP-8189: compute keccak256(rlp.encode(bal)) against the raw
		// bytes actually received on the wire, and compare to the header
		// commitment. Hashing raw bytes (rather than re-encoding after a
		// decode round-trip) catches peers that send non-canonical BAL
		// encodings.
		have := crypto.Keccak256Hash(raw)
		want := *block.Header().BlockAccessListHash
		if have != want {
			return fmt.Errorf("entry %d: BAL hash mismatch: have %x, want %x", idx, have, want)
		}

		// Decode and validate the BAL's internal structure: ordering of
		// accounts/slots/changes, code-size limits, and per-entry access-index
		// bounds, against the known block.
		var accessList bal.BlockAccessList
		if err := rlp.DecodeBytes(raw, &accessList); err != nil {
			return fmt.Errorf("entry %d: invalid BAL RLP: %v", idx, err)
		}
		if err := accessList.Validate(block.GasLimit(), len(block.Transactions())); err != nil {
			return fmt.Errorf("entry %d: BAL failed validation: %v", idx, err)
		}
		idx++
	}

	// Sanity: iterator consumed exactly the reported number of entries.
	if idx != got {
		return fmt.Errorf("iterator visited %d entries, expected %d", idx, got)
	}
	return nil
}
