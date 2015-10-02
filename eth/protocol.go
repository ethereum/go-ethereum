// Copyright 2014 The go-ethereum Authors
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

package eth

import (
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// Mode represents the mode of operation of the eth client.
type Mode int

const (
	ArchiveMode Mode = iota // Maintain the entire blockchain history
	FullMode                // Maintain only a recent view of the blockchain
	LightMode               // Don't maintain any history, rather fetch on demand
)

// Constants to match up protocol versions and messages
const (
	eth61 = 61
	eth62 = 62
	eth63 = 63
	eth64 = 64
)

// minimumProtocolVersion is the minimum version of the protocol eth must run to
// support the desired mode of operation.
var minimumProtocolVersion = map[Mode]uint{
	ArchiveMode: eth61,
	FullMode:    eth63,
	LightMode:   eth64,
}

// Supported versions of the eth protocol (first is primary).
var ProtocolVersions = []uint{eth64, eth63, eth62, eth61}

// Number of implemented message corresponding to different protocol versions.
var ProtocolLengths = []uint64{19, 17, 8, 9}

const (
	NetworkId          = 1
	ProtocolMaxMsgSize = 10 * 1024 * 1024 // Maximum cap on the size of a protocol message
)

// eth protocol message codes
const (
	// Protocol messages belonging to eth/61
	StatusMsg                   = 0x00
	NewBlockHashesMsg           = 0x01
	TxMsg                       = 0x02
	GetBlockHashesMsg           = 0x03
	BlockHashesMsg              = 0x04
	GetBlocksMsg                = 0x05
	BlocksMsg                   = 0x06
	NewBlockMsg                 = 0x07
	GetBlockHashesFromNumberMsg = 0x08

	// Protocol messages belonging to eth/62 (new protocol from scratch)
	// StatusMsg          = 0x00 (uncomment after eth/61 deprecation)
	// NewBlockHashesMsg  = 0x01 (uncomment after eth/61 deprecation)
	// TxMsg              = 0x02 (uncomment after eth/61 deprecation)
	GetBlockHeadersMsg = 0x03
	BlockHeadersMsg    = 0x04
	GetBlockBodiesMsg  = 0x05
	BlockBodiesMsg     = 0x06
	// 	NewBlockMsg       = 0x07 (uncomment after eth/61 deprecation)

	// Protocol messages belonging to eth/63
	GetNodeDataMsg = 0x0d
	NodeDataMsg    = 0x0e
	GetReceiptsMsg = 0x0f
	ReceiptsMsg    = 0x10

	// Protocol messages belonging to eth/64
	GetAcctProofMsg     = 0x11
	GetStorageDataProof = 0x12
	Proof               = 0x13
)

type errCode int

const (
	ErrMsgTooLarge = iota
	ErrDecode
	ErrInvalidMsgCode
	ErrProtocolVersionMismatch
	ErrNetworkIdMismatch
	ErrGenesisBlockMismatch
	ErrNoStatusMsg
	ErrExtraStatusMsg
	ErrSuspendedPeer
)

func (e errCode) String() string {
	return errorToString[int(e)]
}

// XXX change once legacy code is out
var errorToString = map[int]string{
	ErrMsgTooLarge:             "Message too long",
	ErrDecode:                  "Invalid message",
	ErrInvalidMsgCode:          "Invalid message code",
	ErrProtocolVersionMismatch: "Protocol version mismatch",
	ErrNetworkIdMismatch:       "NetworkId mismatch",
	ErrGenesisBlockMismatch:    "Genesis block mismatch",
	ErrNoStatusMsg:             "No status message",
	ErrExtraStatusMsg:          "Extra status message",
	ErrSuspendedPeer:           "Suspended peer",
}

type txPool interface {
	// AddTransactions should add the given transactions to the pool.
	AddTransactions([]*types.Transaction)

	// GetTransactions should return pending transactions.
	// The slice should be modifiable by the caller.
	GetTransactions() types.Transactions
}

type chainManager interface {
	GetBlockHashesFromHash(hash common.Hash, amount uint64) (hashes []common.Hash)
	GetBlock(hash common.Hash) (block *types.Block)
	Status() (td *big.Int, currentBlock common.Hash, genesisBlock common.Hash)
}

// statusData is the network packet for the status message.
type statusData struct {
	ProtocolVersion uint32
	NetworkId       uint32
	TD              *big.Int
	CurrentBlock    common.Hash
	GenesisBlock    common.Hash
}

// newBlockHashesData is the network packet for the block announcements.
type newBlockHashesData []struct {
	Hash   common.Hash // Hash of one particular block being announced
	Number uint64      // Number of one particular block being announced
}

// getBlockHashesData is the network packet for the hash based hash retrieval.
type getBlockHashesData struct {
	Hash   common.Hash
	Amount uint64
}

// getBlockHashesFromNumberData is the network packet for the number based hash
// retrieval.
type getBlockHashesFromNumberData struct {
	Number uint64
	Amount uint64
}

// getBlockHeadersData represents a block header query.
type getBlockHeadersData struct {
	Origin  hashOrNumber // Block from which to retrieve headers
	Amount  uint64       // Maximum number of headers to retrieve
	Skip    uint64       // Blocks to skip between consecutive headers
	Reverse bool         // Query direction (false = rising towards latest, true = falling towards genesis)
}

// hashOrNumber is a combined field for specifying an origin block.
type hashOrNumber struct {
	Hash   common.Hash // Block hash from which to retrieve headers (excludes Number)
	Number uint64      // Block hash from which to retrieve headers (excludes Hash)
}

// EncodeRLP is a specialized encoder for hashOrNumber to encode only one of the
// two contained union fields.
func (hn *hashOrNumber) EncodeRLP(w io.Writer) error {
	if hn.Hash == (common.Hash{}) {
		return rlp.Encode(w, hn.Number)
	}
	if hn.Number != 0 {
		return fmt.Errorf("both origin hash (%x) and number (%d) provided", hn.Hash, hn.Number)
	}
	return rlp.Encode(w, hn.Hash)
}

// DecodeRLP is a specialized decoder for hashOrNumber to decode the contents
// into either a block hash or a block number.
func (hn *hashOrNumber) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	origin, err := s.Raw()
	if err == nil {
		switch {
		case size == 32:
			err = rlp.DecodeBytes(origin, &hn.Hash)
		case size <= 8:
			err = rlp.DecodeBytes(origin, &hn.Number)
		default:
			err = fmt.Errorf("invalid input size %d for origin", size)
		}
	}
	return err
}

// newBlockData is the network packet for the block propagation message.
type newBlockData struct {
	Block *types.Block
	TD    *big.Int
}

// blockBody represents the data content of a single block.
type blockBody struct {
	Transactions []*types.Transaction // Transactions contained within a block
	Uncles       []*types.Header      // Uncles contained within a block
}

// blockBodiesData is the network packet for block content distribution.
type blockBodiesData []*blockBody

// nodeDataData is the network response packet for a node data retrieval.
type nodeDataData []struct {
	Value []byte
}
