// Copyright 2020 The go-ethereum Authors
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

package snap

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// Constants to match up protocol versions and messages
const (
	snap1 = 1
)

// protocolName is the official short name of the `snap` protocol used during
// devp2p capability negotiation.
const protocolName = "snap"

// protocolVersions are the supported versions of the `snap` protocol (first
// is primary).
var protocolVersions = []uint{snap1}

// protocolLengths are the number of implemented message corresponding to
// different protocol versions.
var protocolLengths = map[uint]uint64{snap1: 8}

// maxMessageSize is the maximum cap on the size of a protocol message.
const maxMessageSize = 10 * 1024 * 1024

const (
	getAccountRangeMsg  = 0x00
	accountRangeMsg     = 0x01
	getStorageRangesMsg = 0x02
	storageRangesMsg    = 0x03
	getByteCodesMsg     = 0x04
	byteCodesMsg        = 0x05
	getTrieNodesMsg     = 0x06
	trieNodesMsg        = 0x07
)

var (
	errMsgTooLarge    = errors.New("message too long")
	errDecode         = errors.New("invalid message")
	errInvalidMsgCode = errors.New("invalid message code")
	errBadRequest     = errors.New("bad request")
)

// getAccountRangeData represents an account query.
type getAccountRangeData struct {
	ID     uint64      // Request ID to match up responses with
	Root   common.Hash // Root hash of the account trie to serve
	Origin common.Hash // Hash of the first account to retrieve
	Limit  common.Hash // Hash of the last account to retrieve
	Bytes  uint64      // Soft limit at which to stop returning data
}

// accountRangeData represents an account query response.
type accountRangeData struct {
	ID       uint64         // ID of the request this is a response for
	Accounts []*accountData // List of consecutive accounts from the trie
	Proof    [][]byte       // List of trie nodes proving the account range
}

// accountData represents a single account in a query response.
type accountData struct {
	Hash common.Hash  // Hash of the account
	Body rlp.RawValue // Account body in slim format
}

// getStorageRangesData represents an storage slot query.
type getStorageRangesData struct {
	ID       uint64        // Request ID to match up responses with
	Root     common.Hash   // Root hash of the account trie to serve
	Accounts []common.Hash // Account hashes of the storage tries to serve
	Origin   []byte        // Hash of the first storage slot to retrieve (large contract mode)
	Limit    []byte        // Hash of the last storage slot to retrieve (large contract mode)
	Bytes    uint64        // Soft limit at which to stop returning data
}

// storageRangesData represents a storage slot query response.
type storageRangesData struct {
	ID    uint64           // ID of the request this is a response for
	Slots [][]*storageData // Lists of consecutive storage slots for the requested accounts
	Proof [][]byte         // Merkle proofs for the *last* slot range, if it's incomplete
}

// storageData represents a single storage slot in a query response.
type storageData struct {
	Hash common.Hash // Hash of the storage slot
	Body []byte      // Data content of the slot
}

// getByteCodesData represents a contract bytecode query.
type getByteCodesData struct {
	ID     uint64        // Request ID to match up responses with
	Hashes []common.Hash // Code hashes to retrieve the code for
	Bytes  uint64        // Soft limit at which to stop returning data
}

// byteCodesData represents a contract bytecode query response.
type byteCodesData struct {
	ID    uint64   // ID of the request this is a response for
	Codes [][]byte // Requested contract bytecodes
}

// getTrieNodesData represents a state trie node query.
type getTrieNodesData struct {
	ID    uint64            // Request ID to match up responses with
	Root  common.Hash       // Root hash of the account trie to serve
	Paths []trieNodePathSet // Trie node hashes to retrieve the nodes for
	Bytes uint64            // Soft limit at which to stop returning data
}

// trieNodePathSet is a list of trie node paths to retrieve. A naive way to
// represent trie nodes would be a simple list of `account || storage` path
// segments concatenated, but that would be very wasteful on the network.
//
// Instead, this array special cases the first element as the path in the
// account trie and the remaining elements as paths in the storage trie. To
// address an account node, the slice should have a length of 1 consisting
// of only the account path. There's no need to be able to address both an
// account node and a storage node in the same request as it cannot happen
// that a slot is accessed before the account path is fully expanded.
type trieNodePathSet [][]byte

// trieNodesData represents a state trie node query response.
type trieNodesData struct {
	ID    uint64   // ID of the request this is a response for
	Nodes [][]byte // Requested state trie nodes
}
