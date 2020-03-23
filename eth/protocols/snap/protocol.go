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
	getAccountRangeMsg = 0x00
	accountRangeMsg    = 0x01
	getStorageRangeMsg = 0x02
	storageRangeMsg    = 0x03
	getCodeMsg         = 0x04
	codeMsg            = 0x05
	getTrieNodesMsg    = 0x06
	trieNodesMsg       = 0x07
)

var (
	errMsgTooLarge    = errors.New("message too long")
	errDecode         = errors.New("invalid message")
	errInvalidMsgCode = errors.New("invalid message code")
)

// getAccountRangeData represents an account query.
type getAccountRangeData struct {
	ID     uint64      // Request ID to match up responses with
	Root   common.Hash // Root hash of the account trie to serve
	Origin common.Hash // Account hash of the first to retrieve
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

// getStorageRangeData represents an storage slot query.
type getStorageRangeData struct {
	ID          uint64      // Request ID to match up responses with
	TrieRoot    common.Hash // Root hash of the account trie to serve
	AccountRoot common.Hash // Account hash of the storage trie to serve
	Origin      common.Hash // Storage slot hash of the first to retrieve
	Bytes       uint64      // Soft limit at which to stop returning data
}

// storageRangeData represents a storage slot query response.
type storageRangeData struct {
	ID    uint64        // ID of the request this is a response for
	Slots []storageData // LList of consecutive slots from the trie
	Proof [][]byte      // List of trie nodes proving the slot range
}

// storageData represents a single storage slot in a query response.
type storageData struct {
	Hash common.Hash // Hash of the storage slot
	Body []byte      // Data content of the slot
}

// getCodeData represents a contract bytecode query.
type getCodeData struct {
	ID     uint64        // Request ID to match up responses with
	Hashes []common.Hash // Code hashes to retrieve the code for
}

// codeData represents a contract bytecode query response.
type codeData struct {
	ID    uint64   // ID of the request this is a response for
	Codes [][]byte // Requested contract bytecodes
}

// getTrieNodesData represents a state trie node query.
type getTrieNodesData struct {
	ID     uint64        // Request ID to match up responses with
	Hashes []common.Hash // Trie node hashes to retrieve the nodes for
}

// trieNodesData represents a state trie node query response.
type trieNodesData struct {
	ID    uint64   // ID of the request this is a response for
	Codes [][]byte // Requested state trie nodes
}
