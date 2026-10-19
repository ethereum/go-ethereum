// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// DepositTxVersion is the version of the DepositTx format.
const DepositTxVersion uint8 = 0

// DepositTx is a cross-shard deposit, a message sent to a shard by another shard
// or by the root chain. The destination shard credits it to From and runs it as
// a call to To, with Gas at GasPrice. A move only transfers Value to To, running
// no code: it has no Data, and its Gas is params.XShardSendGas. Only moves are
// sent for now.
//
// The source block hash and log index identify the origin of the deposit: the
// block that sent it and the index of the sending log in that block. A deposit
// from the root chain has the hash of the root block and index 0.
type DepositTx struct {
	Version         uint8
	SourceBlockHash common.Hash
	SourceLogIndex  uint64
	From            common.Address
	To              *common.Address `rlp:"nil"` // nil means contract creation
	Value           *uint256.Int
	GasPrice        *uint256.Int // gas price of the sending transaction
	Gas             uint64       // gas the deposit may use on the destination shard
	IsMove          bool         // whether the deposit only transfers Value to To
	Data            []byte
}
