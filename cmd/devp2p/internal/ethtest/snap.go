// Copyright 2022 The go-ethereum Authors
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
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/protocols/snap"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"golang.org/x/crypto/sha3"
)

func (c *Conn) snapRequest(code uint64, msg any) (any, error) {
	if err := c.Write(snapProto, code, msg); err != nil {
		return nil, fmt.Errorf("could not write to connection: %v", err)
	}
	return c.ReadSnap()
}

func (s *Suite) TestSnapStatus(t *utesting.T) {
	conn, err := s.dialSnap()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err := conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}
}

type accRangeTest struct {
	nBytes       uint64
	root         common.Hash
	startingHash common.Hash
	limitHash    common.Hash

	expAccounts int
	expFirst    common.Hash
	expLast     common.Hash

	desc string
}

// TestSnapGetAccountRange various forms of GetAccountRange requests.
func (s *Suite) TestSnapGetAccountRange(t *utesting.T) {
	var (
		ffHash = common.MaxHash
		zero   = common.Hash{}

		// test values derived from chain/ account dump
		root        = s.chain.Head().Root()
		headstate   = s.chain.AccountsInHashOrder()
		firstKey    = common.BytesToHash(headstate[0].AddressHash)
		secondKey   = common.BytesToHash(headstate[1].AddressHash)
		storageRoot = findNonEmptyStorageRoot(headstate)
	)

	tests := []accRangeTest{
		// Tests decreasing the number of bytes
		{
			nBytes:       4000,
			root:         root,
			startingHash: zero,
			limitHash:    ffHash,
			expAccounts:  86,
			expFirst:     firstKey,
			expLast:      common.HexToHash("0x445cb5c1278fdce2f9cbdb681bdd76c52f8e50e41dbd9e220242a69ba99ac099"),
			desc:         "In this test, we request the entire state range, but limit the response to 4000 bytes.",
		},
		{
			nBytes:       3000,
			root:         root,
			startingHash: zero,
			limitHash:    ffHash,
			expAccounts:  65,
			expFirst:     firstKey,
			expLast:      common.HexToHash("0x2e6fe1362b3e388184fd7bf08e99e74170b26361624ffd1c5f646da7067b58b6"),
			desc:         "In this test, we request the entire state range, but limit the response to 3000 bytes.",
		},
		{
			nBytes:       2000,
			root:         root,
			startingHash: zero,
			limitHash:    ffHash,
			expAccounts:  44,
			expFirst:     firstKey,
			expLast:      common.HexToHash("0x1c3f74249a4892081ba0634a819aec9ed25f34c7653f5719b9098487e65ab595"),
			desc:         "In this test, we request the entire state range, but limit the response to 2000 bytes.",
		},
		{
			nBytes:       1,
			root:         root,
			startingHash: zero,
			limitHash:    ffHash,
			expAccounts:  1,
			expFirst:     firstKey,
			expLast:      firstKey,
			desc: `In this test, we request the entire state range, but limit the response to 1 byte.
The server should return the first account of the state.`,
		},
		{
			nBytes:       0,
			root:         root,
			startingHash: zero,
			limitHash:    ffHash,
			expAccounts:  1,
			expFirst:     firstKey,
			expLast:      firstKey,
			desc: `Here we request with a responseBytes limit of zero.
The server should return one account.`,
		},

		// Tests variations of the range
		{
			nBytes:       4000,
			root:         root,
			startingHash: hashAdd(firstKey, -500),
			limitHash:    hashAdd(firstKey, 1),
			expAccounts:  2,
			expFirst:     firstKey,
			expLast:      secondKey,
			desc: `In this test, we request a range where startingHash is before the first available
account key, and limitHash is after. The server should return the first and second
account of the state (because the second account is the 'next available').`,
		},

		{
			nBytes:       4000,
			root:         root,
			startingHash: hashAdd(firstKey, -500),
			limitHash:    hashAdd(firstKey, -450),
			expAccounts:  1,
			expFirst:     firstKey,
			expLast:      firstKey,
			desc: `Here we request range where both bounds are before the first available account key.
This should return the first account (even though it's out of bounds).`,
		},

		// More range tests:
		{
			nBytes:       4000,
			root:         root,
			startingHash: zero,
			limitHash:    zero,
			expAccounts:  1,
			expFirst:     firstKey,
			expLast:      firstKey,
			desc: `In this test, both startingHash and limitHash are zero.
The server should return the first available account.`,
		},
		{
			nBytes:       4000,
			root:         root,
			startingHash: firstKey,
			limitHash:    ffHash,
			expAccounts:  86,
			expFirst:     firstKey,
			expLast:      common.HexToHash("0x445cb5c1278fdce2f9cbdb681bdd76c52f8e50e41dbd9e220242a69ba99ac099"),
			desc: `In this test, startingHash is exactly the first available account key.
The server should return the first available account of the state as the first item.`,
		},
		{
			nBytes:       4000,
			root:         root,
			startingHash: hashAdd(firstKey, 1),
			limitHash:    ffHash,
			expAccounts:  86,
			expFirst:     secondKey,
			expLast:      common.HexToHash("0x4615e5f5df5b25349a00ad313c6cd0436b6c08ee5826e33a018661997f85ebaa"),
			desc: `In this test, startingHash is after the first available key.
The server should return the second account of the state as the first item.`,
		},

		// Test different root hashes

		{
			nBytes:       4000,
			root:         common.Hash{0x13, 0x37},
			startingHash: zero,
			limitHash:    ffHash,
			expAccounts:  0,
			expFirst:     zero,
			expLast:      zero,
			desc:         `This test requests a non-existent state root.`,
		},

		// The genesis stateroot (we expect it to not be served)
		{
			nBytes:       4000,
			root:         s.chain.RootAt(0),
			startingHash: zero,
			limitHash:    ffHash,
			expAccounts:  0,
			expFirst:     zero,
			expLast:      zero,
			desc: `This test requests data at the state root of the genesis block. We expect the
server to return no data because genesis is older than 127 blocks.`,
		},

		{
			nBytes:       4000,
			root:         s.chain.RootAt(int(s.chain.Head().Number().Uint64()) - 127),
			startingHash: zero,
			limitHash:    ffHash,
			expAccounts:  84,
			expFirst:     firstKey,
			expLast:      common.HexToHash("0x580aa878e2f92d113a12c0a3ce3c21972b03dbe80786858d49a72097e2c491a3"),
			desc: `This test requests data at a state root that is 127 blocks old.
We expect the server to have this state available.`,
		},

		{
			nBytes:       4000,
			root:         storageRoot,
			startingHash: zero,
			limitHash:    ffHash,
			expAccounts:  0,
			expFirst:     zero,
			expLast:      zero,
			desc: `This test requests data at a state root that is actually the storage root of
an existing account. The server is supposed to ignore this request.`,
		},

		// And some non-sensical requests

		{
			nBytes:       4000,
			root:         root,
			startingHash: ffHash,
			limitHash:    zero,
			expAccounts:  0,
			expFirst:     zero,
			expLast:      zero,
			desc: `In this test, the startingHash is after limitHash (wrong order). The server
should ignore this invalid request.`,
		},

		{
			nBytes:       4000,
			root:         root,
			startingHash: firstKey,
			limitHash:    hashAdd(firstKey, -1),
			expAccounts:  1,
			expFirst:     firstKey,
			expLast:      firstKey,
			desc: `In this test, the startingHash is the first available key, and limitHash is
a key before startingHash (wrong order). The server should return the first available key.`,
		},

		// range from [firstkey, 0], wrong order. Expect to get first key.
		{
			nBytes:       4000,
			root:         root,
			startingHash: firstKey,
			limitHash:    zero,
			expAccounts:  1,
			expFirst:     firstKey,
			expLast:      firstKey,
			desc: `In this test, the startingHash is the first available key and limitHash is zero.
(wrong order). The server should return the first available key.`,
		},
	}

	for i, tc := range tests {
		tc := tc
		if i > 0 {
			t.Log("\n")
		}
		t.Logf("-- Test %d", i)
		t.Log(tc.desc)
		t.Log("  request:")
		t.Logf("      root: %x", tc.root)
		t.Logf("      range: %#x - %#x", tc.startingHash, tc.limitHash)
		t.Logf("      responseBytes: %d", tc.nBytes)
		if err := s.snapGetAccountRange(t, &tc); err != nil {
			t.Errorf("test %d failed: %v", i, err)
		}
	}
}

func hashAdd(h common.Hash, n int64) common.Hash {
	hb := h.Big()
	return common.BigToHash(hb.Add(hb, big.NewInt(n)))
}

func findNonEmptyStorageRoot(accounts []state.DumpAccount) common.Hash {
	for i := range accounts {
		if len(accounts[i].Storage) != 0 {
			return common.BytesToHash(accounts[i].Root)
		}
	}
	panic("can't find account with non-empty storage")
}

type stRangesTest struct {
	root     common.Hash
	accounts []common.Hash
	origin   []byte
	limit    []byte
	nBytes   uint64

	expSlots [][]*snap.StorageData

	desc string
}

// TestSnapGetStorageRanges various forms of GetStorageRanges requests.
func (s *Suite) TestSnapGetStorageRanges(t *utesting.T) {
	var (
		acct      = common.HexToAddress("0x8bebc8ba651aee624937e7d897853ac30c95a067")
		acctHash  = common.BytesToHash(s.chain.state[acct].AddressHash)
		ffHash    = common.MaxHash
		zero      = common.Hash{}
		blockroot = s.chain.Head().Root()
	)

	// These are the storage slots of the test account, encoded as snap response data.
	acctSlots := []*snap.StorageData{
		{
			Hash: common.HexToHash("0x405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ace"),
			Body: []byte{0x02},
		},
		{
			Hash: common.HexToHash("0xb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6"),
			Body: []byte{0x01},
		},
		{
			Hash: common.HexToHash("0xc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b"),
			Body: []byte{0x03},
		},
	}

	tests := []stRangesTest{
		/*
			Some tests against this account:

			"0x8bebc8ba651aee624937e7d897853ac30c95a067": {
				"balance": "1",
				"nonce": 1,
				"root": "0xe318dff15b33aa7f2f12d5567d58628e3e3f2e8859e46b56981a4083b391da17",
				"codeHash": "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
				"storage": {
					// Note: keys below are hashed!!!
					"0x405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ace": "02",
					"0xb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6": "01",
					"0xc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b": "03"
				},
				"key": "0x445cb5c1278fdce2f9cbdb681bdd76c52f8e50e41dbd9e220242a69ba99ac099"
			}
		*/

		{ // [:] -> [slot1, slot2, slot3]
			desc: `This request has a range of 00..ff.
The server should return all storage slots of the test account.`,
			root:     blockroot,
			accounts: []common.Hash{acctHash},
			origin:   zero[:],
			limit:    ffHash[:],
			nBytes:   500,
			expSlots: [][]*snap.StorageData{acctSlots},
		},

		{ // [slot1:] -> [slot1, slot2, slot3]
			desc: `This test requests slots starting at the first available key.
The server should return all storage slots of the test account.`,
			root:     blockroot,
			accounts: []common.Hash{acctHash},
			origin:   common.FromHex("0x405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ace"),
			limit:    ffHash[:],
			nBytes:   1000,
			expSlots: [][]*snap.StorageData{acctSlots},
		},

		{ // [slot1+:] -> [slot2, slot3]
			desc: `This test requests slots starting at a key one past the first available key.
The server should return the remaining two slots of the test account.`,
			root:     blockroot,
			accounts: []common.Hash{acctHash},
			origin:   common.FromHex("0x405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5acf"),
			limit:    ffHash[:],
			nBytes:   500,
			expSlots: [][]*snap.StorageData{acctSlots[1:]},
		},

		{ // [slot1:slot2] -> [slot1, slot2]
			desc:     `This test requests a range which is exactly the first and second available key.`,
			root:     blockroot,
			accounts: []common.Hash{acctHash},
			origin:   common.FromHex("0x405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5ace"),
			limit:    common.FromHex("0xb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf6"),
			nBytes:   500,
			expSlots: [][]*snap.StorageData{acctSlots[:2]},
		},

		{ // [slot1+:slot2+] -> [slot2, slot3]
			desc: `This test requests a range where limitHash is after the second, but before the third slot
of the test account. The server should return slots [2,3] (i.e. the 'next available' needs to be returned).`,
			root:     blockroot,
			accounts: []common.Hash{acctHash},
			origin:   common.FromHex("0x4fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
			limit:    common.FromHex("0xb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf7"),
			nBytes:   500,
			expSlots: [][]*snap.StorageData{acctSlots[1:]},
		},
	}

	for i, tc := range tests {
		tc := tc
		if i > 0 {
			t.Log("\n")
		}
		t.Logf("-- Test %d", i)
		t.Log(tc.desc)
		t.Log("  request:")
		t.Logf("      root: %x", tc.root)
		t.Logf("      accounts: %x", tc.accounts)
		t.Logf("      range: %#x - %#x", tc.origin, tc.limit)
		t.Logf("      responseBytes: %d", tc.nBytes)
		if err := s.snapGetStorageRanges(t, &tc); err != nil {
			t.Errorf("  failed: %v", err)
		}
	}
}

type byteCodesTest struct {
	nBytes uint64
	hashes []common.Hash

	expHashes int

	desc string
}

// TestSnapGetByteCodes various forms of GetByteCodes requests.
func (s *Suite) TestSnapGetByteCodes(t *utesting.T) {
	var (
		allHashes   = s.chain.CodeHashes()
		headRoot    = s.chain.Head().Root()
		genesisRoot = s.chain.RootAt(0)
	)

	tests := []byteCodesTest{
		// A few stateroots
		{
			desc:      `Here we request state roots as code hashes. The server should deliver an empty response with no items.`,
			nBytes:    10000,
			hashes:    []common.Hash{genesisRoot, headRoot},
			expHashes: 0,
		},
		{
			desc:      `Here we request the genesis state root (which is not an existing code hash) two times. The server should deliver an empty response with no items.`,
			nBytes:    10000,
			hashes:    []common.Hash{genesisRoot, genesisRoot},
			expHashes: 0,
		},
		// Empties
		{
			desc:      `Here we request the empty state root (which is not an existing code hash). The server should deliver an empty response with no items.`,
			nBytes:    10000,
			hashes:    []common.Hash{types.EmptyRootHash},
			expHashes: 0,
		},
		{
			desc:      `Here we request the empty code hash. The server should deliver an empty response item.`,
			nBytes:    10000,
			hashes:    []common.Hash{types.EmptyCodeHash},
			expHashes: 1,
		},
		{
			desc:      `In this test, we request the empty code hash three times. The server should deliver the empty item three times.`,
			nBytes:    10000,
			hashes:    []common.Hash{types.EmptyCodeHash, types.EmptyCodeHash, types.EmptyCodeHash},
			expHashes: 3,
		},
		// The existing bytecodes
		{
			desc:      `Here we request all available contract codes. The server should deliver them all in one response.`,
			nBytes:    100000,
			hashes:    allHashes,
			expHashes: len(allHashes),
		},
		// The existing, with limited byte arg
		{
			desc:      `In this test, the request has a bytes limit of one. The server should deliver one item.`,
			nBytes:    1,
			hashes:    allHashes,
			expHashes: 1,
		},
		{
			desc:      `In this test, the request has a bytes limit of zero. The server should deliver one item.`,
			nBytes:    0,
			hashes:    allHashes,
			expHashes: 1,
		},
		// Request the same hash multiple times.
		{
			desc:      `This test requests the same code hash multiple times. The server should deliver it multiple times.`,
			nBytes:    1000,
			hashes:    []common.Hash{allHashes[0], allHashes[0], allHashes[0], allHashes[0]},
			expHashes: 4,
		},
	}

	for i, tc := range tests {
		tc := tc
		if i > 0 {
			t.Log("\n")
		}
		t.Logf("-- Test %d", i)
		t.Log(tc.desc)
		t.Log("  request:")
		t.Logf("      hashes: %x", tc.hashes)
		t.Logf("      responseBytes: %d", tc.nBytes)
		if err := s.snapGetByteCodes(t, &tc); err != nil {
			t.Errorf("failed: %v", err)
		}
	}
}

type trieNodesTest struct {
	root   common.Hash
	paths  []snap.TrieNodePathSet
	nBytes uint64

	expHashes []common.Hash // expected response
	expReject bool          // if true, request should be rejected

	desc string
}

func decodeNibbles(nibbles []byte, bytes []byte) {
	for bi, ni := 0, 0; ni < len(nibbles); bi, ni = bi+1, ni+2 {
		bytes[bi] = nibbles[ni]<<4 | nibbles[ni+1]
	}
}

// hasTerm returns whether a hex key has the terminator flag.
func hasTerm(s []byte) bool {
	return len(s) > 0 && s[len(s)-1] == 16
}

func keybytesToHex(str []byte) []byte {
	l := len(str)*2 + 1
	var nibbles = make([]byte, l)
	for i, b := range str {
		nibbles[i*2] = b / 16
		nibbles[i*2+1] = b % 16
	}
	nibbles[l-1] = 16
	return nibbles
}

func hexToCompact(hex []byte) []byte {
	terminator := byte(0)
	if hasTerm(hex) {
		terminator = 1
		hex = hex[:len(hex)-1]
	}
	buf := make([]byte, len(hex)/2+1)
	buf[0] = terminator << 5 // the flag byte
	if len(hex)&1 == 1 {
		buf[0] |= 1 << 4 // odd flag
		buf[0] |= hex[0] // first nibble is contained in the first byte
		hex = hex[1:]
	}
	decodeNibbles(hex, buf[1:])
	return buf
}

// TestSnapTrieNodes various forms of GetTrieNodes requests.
func (s *Suite) TestSnapTrieNodes(t *utesting.T) {
	var (
		// This is the known address of the snap storage testing contract.
		storageAcct     = common.HexToAddress("0x8bebc8ba651aee624937e7d897853ac30c95a067")
		storageAcctHash = common.BytesToHash(s.chain.state[storageAcct].AddressHash)
		// This is the known address of an existing account.
		key      = common.FromHex("0xa87387b50b481431c6ccdb9ae99a54d4dcdd4a3eff75d7b17b4818f7bbfc21e9")
		empty    = types.EmptyCodeHash
		accPaths []snap.TrieNodePathSet
	)
	for i := 1; i <= 65; i++ {
		accPaths = append(accPaths, makeSnapPath(key, i))
	}

	tests := []trieNodesTest{
		{
			desc:      `In this test, we send an empty request to the node.`,
			root:      s.chain.Head().Root(),
			paths:     nil,
			nBytes:    500,
			expHashes: nil,
		},

		{
			desc: `In this test, we send a request containing an empty path-set.
The server should reject the request.`,
			root: s.chain.Head().Root(),
			paths: []snap.TrieNodePathSet{
				{}, // zero-length pathset should 'abort' and kick us off
				{[]byte{0}},
			},
			nBytes:    5000,
			expHashes: []common.Hash{},
			expReject: true,
		},

		{
			desc: `Here we request the root node of the trie. The server should respond with the root node.`,
			root: s.chain.RootAt(int(s.chain.Head().NumberU64() - 1)),
			paths: []snap.TrieNodePathSet{
				{[]byte{0}},
				{[]byte{1}, []byte{0}},
			},
			nBytes:    5000,
			expHashes: []common.Hash{s.chain.RootAt(int(s.chain.Head().NumberU64() - 1))},
		},

		{ // nonsensically long path
			desc: `In this test, we request a very long trie node path. The server should respond with an empty node (keccak256("")).`,
			root: s.chain.Head().Root(),
			paths: []snap.TrieNodePathSet{
				{[]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 0, 1, 2, 3, 4, 5, 6, 7, 8, 0, 1, 2, 3, 4, 5, 6, 7, 8,
					0, 1, 2, 3, 4, 5, 6, 7, 8, 0, 1, 2, 3, 4, 5, 6, 7, 8, 0, 1, 2, 3, 4, 5, 6, 7, 8}},
			},
			nBytes:    5000,
			expHashes: []common.Hash{common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470")},
		},

		{
			// The leaf is only a couple of levels down, so the continued trie traversal causes lookup failures.
			desc:   `Here we request some known accounts from the state.`,
			root:   s.chain.Head().Root(),
			paths:  accPaths,
			nBytes: 5000,
			expHashes: []common.Hash{
				// It's a bit unfortunate these are hard-coded, but the result depends on
				// a lot of aspects of the state trie and can't be guessed in a simple
				// way. So you'll have to update this when the test chain is changed.
				common.HexToHash("0x3e963a69401a70224cbfb8c0cc2249b019041a538675d71ccf80c9328d114e2e"),
				common.HexToHash("0xd0670d09cdfbf3c6320eb3e92c47c57baa6c226551a2d488c05581091e6b1689"),
				empty, empty, empty, empty, empty, empty, empty, empty, empty, empty, empty, empty,
				empty, empty, empty, empty, empty, empty, empty, empty, empty, empty, empty, empty,
				empty, empty, empty, empty, empty, empty, empty, empty, empty, empty, empty, empty,
				empty, empty, empty, empty, empty, empty, empty, empty, empty, empty, empty, empty,
				empty, empty, empty, empty, empty, empty, empty, empty, empty, empty, empty, empty,
				empty, empty, empty},
		},

		{
			desc: `In this test, we request some known accounts in state. The requested paths are NOT in key order.`,
			root: s.chain.Head().Root(),
			paths: []snap.TrieNodePathSet{
				accPaths[10], accPaths[1], accPaths[0],
			},
			nBytes: 5000,
			// As with the previous test, this result depends on the whole tree and will have to
			// be updated when the test chain is changed.
			expHashes: []common.Hash{
				empty,
				common.HexToHash("0xd0670d09cdfbf3c6320eb3e92c47c57baa6c226551a2d488c05581091e6b1689"),
				common.HexToHash("0x3e963a69401a70224cbfb8c0cc2249b019041a538675d71ccf80c9328d114e2e"),
			},
		},

		// Storage tests.
		// These use the known storage test account.

		{
			desc: `This test requests the storage root node of a known account.`,
			root: s.chain.Head().Root(),
			paths: []snap.TrieNodePathSet{
				{
					storageAcctHash[:],
					[]byte{0},
				},
			},
			nBytes: 5000,
			expHashes: []common.Hash{
				common.HexToHash("0xbe3d75a1729be157e79c3b77f00206db4d54e3ea14375a015451c88ec067c790"),
			},
		},

		{
			desc: `This test requests multiple storage nodes of a known account.`,
			root: s.chain.Head().Root(),
			paths: []snap.TrieNodePathSet{
				{
					storageAcctHash[:],
					[]byte{0},
					[]byte{0x1b},
				},
			},
			nBytes: 5000,
			expHashes: []common.Hash{
				common.HexToHash("0xbe3d75a1729be157e79c3b77f00206db4d54e3ea14375a015451c88ec067c790"),
				common.HexToHash("0xf4984a11f61a2921456141df88de6e1a710d28681b91af794c5a721e47839cd7"),
			},
		},
	}

	for i, tc := range tests {
		tc := tc
		if i > 0 {
			t.Log("\n")
		}
		t.Logf("-- Test %d", i)
		t.Log(tc.desc)
		t.Log("  request:")
		t.Logf("    root: %x", tc.root)
		t.Logf("    paths: %x", tc.paths)
		t.Logf("    responseBytes: %d", tc.nBytes)

		if err := s.snapGetTrieNodes(t, &tc); err != nil {
			t.Errorf("  failed: %v", err)
		}
	}
}

func makeSnapPath(key []byte, length int) snap.TrieNodePathSet {
	hex := keybytesToHex(key)[:length]
	hex[len(hex)-1] = 0 // remove term flag
	hKey := hexToCompact(hex)
	return snap.TrieNodePathSet{hKey}
}

func (s *Suite) snapGetAccountRange(t *utesting.T, tc *accRangeTest) error {
	conn, err := s.dialSnap()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err = conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}
	// write request
	req := &snap.GetAccountRangePacket{
		ID:     uint64(rand.Int63()),
		Root:   tc.root,
		Origin: tc.startingHash,
		Limit:  tc.limitHash,
		Bytes:  tc.nBytes,
	}
	msg, err := conn.snapRequest(snap.GetAccountRangeMsg, req)
	if err != nil {
		return fmt.Errorf("account range request failed: %v", err)
	}
	res, ok := msg.(*snap.AccountRangePacket)
	if !ok {
		return fmt.Errorf("account range response wrong: %T %v", msg, msg)
	}
	if exp, got := tc.expAccounts, len(res.Accounts); exp != got {
		return fmt.Errorf("expected %d accounts, got %d", exp, got)
	}
	// Check that the encoding order is correct
	for i := 1; i < len(res.Accounts); i++ {
		if bytes.Compare(res.Accounts[i-1].Hash[:], res.Accounts[i].Hash[:]) >= 0 {
			return fmt.Errorf("accounts not monotonically increasing: #%d [%x] vs #%d [%x]", i-1, res.Accounts[i-1].Hash[:], i, res.Accounts[i].Hash[:])
		}
	}
	var (
		hashes   []common.Hash
		accounts [][]byte
		proof    = res.Proof
	)
	hashes, accounts, err = res.Unpack()
	if err != nil {
		return err
	}
	if len(hashes) == 0 && len(accounts) == 0 && len(proof) == 0 {
		return nil
	}
	if len(hashes) > 0 {
		if exp, got := tc.expFirst, res.Accounts[0].Hash; exp != got {
			return fmt.Errorf("expected first account %#x, got %#x", exp, got)
		}
		if exp, got := tc.expLast, res.Accounts[len(res.Accounts)-1].Hash; exp != got {
			return fmt.Errorf("expected last account %#x, got %#x", exp, got)
		}
	}
	// Reconstruct a partial trie from the response and verify it
	keys := make([][]byte, len(hashes))
	for i, key := range hashes {
		keys[i] = common.CopyBytes(key[:])
	}
	nodes := make(trienode.ProofList, len(proof))
	for i, node := range proof {
		nodes[i] = node
	}
	proofdb := nodes.Set()

	_, err = trie.VerifyRangeProof(tc.root, tc.startingHash[:], keys, accounts, proofdb)
	return err
}

func (s *Suite) snapGetStorageRanges(t *utesting.T, tc *stRangesTest) error {
	conn, err := s.dialSnap()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err = conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}

	// write request
	req := &snap.GetStorageRangesPacket{
		ID:       uint64(rand.Int63()),
		Root:     tc.root,
		Accounts: tc.accounts,
		Origin:   tc.origin,
		Limit:    tc.limit,
		Bytes:    tc.nBytes,
	}
	msg, err := conn.snapRequest(snap.GetStorageRangesMsg, req)
	if err != nil {
		return fmt.Errorf("account range request failed: %v", err)
	}
	res, ok := msg.(*snap.StorageRangesPacket)
	if !ok {
		return fmt.Errorf("account range response wrong: %T %v", msg, msg)
	}

	// Ensure the ranges are monotonically increasing
	for i, slots := range res.Slots {
		for j := 1; j < len(slots); j++ {
			if bytes.Compare(slots[j-1].Hash[:], slots[j].Hash[:]) >= 0 {
				return fmt.Errorf("storage slots not monotonically increasing for account #%d: #%d [%x] vs #%d [%x]", i, j-1, slots[j-1].Hash[:], j, slots[j].Hash[:])
			}
		}
	}

	// Compute expected slot hashes.
	var expHashes [][]common.Hash
	for _, acct := range tc.expSlots {
		var list []common.Hash
		for _, s := range acct {
			list = append(list, s.Hash)
		}
		expHashes = append(expHashes, list)
	}

	// Check response.
	if !reflect.DeepEqual(res.Slots, tc.expSlots) {
		t.Log("  expected slot hashes:", expHashes)
		return fmt.Errorf("wrong storage slots in response: %#v", res.Slots)
	}
	return nil
}

func (s *Suite) snapGetByteCodes(t *utesting.T, tc *byteCodesTest) error {
	conn, err := s.dialSnap()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err = conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}
	// write request
	req := &snap.GetByteCodesPacket{
		ID:     uint64(rand.Int63()),
		Hashes: tc.hashes,
		Bytes:  tc.nBytes,
	}
	msg, err := conn.snapRequest(snap.GetByteCodesMsg, req)
	if err != nil {
		return fmt.Errorf("getBytecodes request failed: %v", err)
	}
	res, ok := msg.(*snap.ByteCodesPacket)
	if !ok {
		return fmt.Errorf("bytecodes response wrong: %T %v", msg, msg)
	}
	if exp, got := tc.expHashes, len(res.Codes); exp != got {
		for i, c := range res.Codes {
			t.Logf("%d. %#x\n", i, c)
		}
		return fmt.Errorf("expected %d bytecodes, got %d", exp, got)
	}
	// Cross reference the requested bytecodes with the response to find gaps
	// that the serving node is missing
	var (
		bytecodes = res.Codes
		hasher    = sha3.NewLegacyKeccak256().(crypto.KeccakState)
		hash      = make([]byte, 32)
		codes     = make([][]byte, len(req.Hashes))
	)

	for i, j := 0, 0; i < len(bytecodes); i++ {
		// Find the next hash that we've been served, leaving misses with nils
		hasher.Reset()
		hasher.Write(bytecodes[i])
		hasher.Read(hash)

		for j < len(req.Hashes) && !bytes.Equal(hash, req.Hashes[j][:]) {
			j++
		}
		if j < len(req.Hashes) {
			codes[j] = bytecodes[i]
			j++
			continue
		}
		// We've either ran out of hashes, or got unrequested data
		return errors.New("unexpected bytecode")
	}

	return nil
}

func (s *Suite) snapGetTrieNodes(t *utesting.T, tc *trieNodesTest) error {
	conn, err := s.dialSnap()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err = conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}

	// write0 request
	req := &snap.GetTrieNodesPacket{
		ID:    uint64(rand.Int63()),
		Root:  tc.root,
		Paths: tc.paths,
		Bytes: tc.nBytes,
	}
	msg, err := conn.snapRequest(snap.GetTrieNodesMsg, req)
	if err != nil {
		if tc.expReject {
			return nil
		}
		return fmt.Errorf("trienodes  request failed: %v", err)
	}
	res, ok := msg.(*snap.TrieNodesPacket)
	if !ok {
		return fmt.Errorf("trienodes response wrong: %T %v", msg, msg)
	}

	// Check the correctness

	// Cross reference the requested trienodes with the response to find gaps
	// that the serving node is missing
	hasher := sha3.NewLegacyKeccak256().(crypto.KeccakState)
	hash := make([]byte, 32)
	trienodes := res.Nodes
	if got, want := len(trienodes), len(tc.expHashes); got != want {
		return fmt.Errorf("wrong trienode count, got %d, want %d", got, want)
	}
	for i, trienode := range trienodes {
		hasher.Reset()
		hasher.Write(trienode)
		hasher.Read(hash)
		if got, want := hash, tc.expHashes[i]; !bytes.Equal(got, want[:]) {
			t.Logf("  hash %d wrong, got %#x, want %#x\n", i, got, want)
			err = fmt.Errorf("hash %d wrong, got %#x, want %#x", i, got, want)
		}
	}
	return err
}
