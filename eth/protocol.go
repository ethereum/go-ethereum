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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Supported versions of the eth protocol (first is primary).
var ProtocolVersions = []uint{61, 60}

// Number of implemented message corresponding to different protocol versions.
var ProtocolLengths = []uint64{9, 8}

const (
	NetworkId          = 0
	ProtocolMaxMsgSize = 10 * 1024 * 1024 // Maximum cap on the size of a protocol message
)

// eth protocol message codes
const (
	StatusMsg = iota
	NewBlockHashesMsg
	TxMsg
	GetBlockHashesMsg
	BlockHashesMsg
	GetBlocksMsg
	BlocksMsg
	NewBlockMsg
	GetBlockHashesFromNumberMsg
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

// getBlockHashesData is the network packet for the hash based block retrieval
// message.
type getBlockHashesData struct {
	Hash   common.Hash
	Amount uint64
}

// getBlockHashesFromNumberData is the network packet for the number based block
// retrieval message.
type getBlockHashesFromNumberData struct {
	Number uint64
	Amount uint64
}

// newBlockData is the network packet for the block propagation message.
type newBlockData struct {
	Block *types.Block
	TD    *big.Int
}
