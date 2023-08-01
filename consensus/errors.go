// Copyright 2017 The go-ethereum Authors
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

package consensus

import "errors"

var (
	// ErrUnknownAncestor is returned when validating a block requires an ancestor
	// that is unknown.
	ErrUnknownAncestor = errors.New("unknown ancestor")

	// ErrPrunedAncestor is returned when validating a block requires an ancestor
	// that is known, but the state of which is not available.
	ErrPrunedAncestor = errors.New("pruned ancestor")

	// ErrFutureBlock is returned when a block's timestamp is in the future according
	// to the current node.
	ErrFutureBlock = errors.New("block in the future")

	// ErrInvalidNumber is returned if a block's number doesn't equal its parent's
	// plus one.
	ErrInvalidNumber = errors.New("invalid block number")

	// ErrInvalidTxCount is returned if a block contains too many transactions.
	ErrInvalidTxCount = errors.New("invalid transaction count")

	// ErrMissingL1MessageData is returned if a block contains L1 messages that the
	// node has not synced yet. In this case we insert the block into the future
	// queue and process it again later.
	ErrMissingL1MessageData = errors.New("unknown L1 message data")

	// ErrInvalidL1MessageOrder is returned if a block contains L1 messages in the wrong
	// order. Possible scenarios are: (1) L1 messages do not follow their QueueIndex order,
	// (2) L1 messages are not included in a contiguous block at the front of the block.
	ErrInvalidL1MessageOrder = errors.New("invalid L1 message order")

	// ErrUnknownL1Message is returned if a block contains an L1 message that does not
	// match the corresponding message in the node's local database.
	ErrUnknownL1Message = errors.New("unknown L1 message")
)
