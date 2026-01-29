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
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rlp"
)

// XDPoS protocol version constants
const (
	xdpos2 = 100 // XDPoS 2.0 protocol version
)

// XDC protocol message codes (extensions to standard eth protocol)
const (
	// XDPoS consensus messages (starting at 0xe0 to avoid conflicts)
	VoteMsgCode     = 0xe0
	TimeoutMsgCode  = 0xe1
	SyncInfoMsgCode = 0xe2

	// Order/Lending transaction messages
	OrderTxMsgCode   = 0x08
	LendingTxMsgCode = 0x09
)

// XDC error codes
const (
	ErrMsgTooLargeXDC = iota + 100
	ErrDecodeXDC
	ErrInvalidMsgCodeXDC
	ErrSuspendedPeerXDC
)

func errCodeXDCString(e int) string {
	switch e {
	case ErrMsgTooLargeXDC:
		return "XDC: Message too long"
	case ErrDecodeXDC:
		return "XDC: Invalid message"
	case ErrInvalidMsgCodeXDC:
		return "XDC: Invalid message code"
	case ErrSuspendedPeerXDC:
		return "XDC: Suspended peer"
	default:
		return fmt.Sprintf("XDC: Unknown error %d", e)
	}
}

// OrderPool interface for XDCx order pool
type OrderPool interface {
	// AddRemotes should add the given transactions to the pool.
	AddRemotes([]*types.OrderTransaction) []error

	// Pending should return pending transactions.
	Pending() (map[common.Address]types.OrderTransactions, error)

	// SubscribeTxPreEvent should return an event subscription of
	// TxPreEvent and send events to the given channel.
	SubscribeTxPreEvent(chan<- core.OrderTxPreEvent) event.Subscription
}

// LendingPool interface for XDCx lending pool
type LendingPool interface {
	// AddRemotes should add the given transactions to the pool.
	AddRemotes([]*types.LendingTransaction) []error

	// Pending should return pending transactions.
	Pending() (map[common.Address]types.LendingTransactions, error)

	// SubscribeTxPreEvent should return an event subscription of
	// TxPreEvent and send events to the given channel.
	SubscribeTxPreEvent(chan<- core.LendingTxPreEvent) event.Subscription
}

// XDCStatusData extends statusData with XDPoS-specific fields
type XDCStatusData struct {
	ProtocolVersion uint32
	NetworkId       uint64
	TD              *big.Int
	CurrentBlock    common.Hash
	GenesisBlock    common.Hash
	Epoch           uint64 // Current epoch number
}

// hashOrNumberXDC is a combined field for specifying an origin block.
// Duplicated from protocol.go to avoid circular imports in some cases
type hashOrNumberXDC struct {
	Hash   common.Hash // Block hash from which to retrieve headers (excludes Number)
	Number uint64      // Block number from which to retrieve headers (excludes Hash)
}

// EncodeRLP is a specialized encoder for hashOrNumberXDC
func (hn *hashOrNumberXDC) EncodeRLP(w io.Writer) error {
	if hn.Hash == (common.Hash{}) {
		return rlp.Encode(w, hn.Number)
	}
	if hn.Number != 0 {
		return fmt.Errorf("both origin hash (%x) and number (%d) provided", hn.Hash, hn.Number)
	}
	return rlp.Encode(w, hn.Hash)
}

// DecodeRLP is a specialized decoder for hashOrNumberXDC
func (hn *hashOrNumberXDC) DecodeRLP(s *rlp.Stream) error {
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
