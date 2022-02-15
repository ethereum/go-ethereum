// Copyright 2019 The go-ethereum Authors
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

// Copyright 2021-2022 The go-xpayments Authors
// This file is part of go-xpayments.

package xps

import (
	"github.com/xpaymentsorg/go-xpayments/core"
	"github.com/xpaymentsorg/go-xpayments/core/forkid"
	"github.com/xpaymentsorg/go-xpayments/p2p/enode"
	"github.com/xpaymentsorg/go-xpayments/rlp"
	// "github.com/ethereum/go-ethereum/core"
	// "github.com/ethereum/go-ethereum/core/forkid"
	// "github.com/ethereum/go-ethereum/p2p/enode"
	// "github.com/ethereum/go-ethereum/rlp"
)

// xpsEntry is the "xps" ENR entry which advertises xps protocol
// on the discovery network.
type xpsEntry struct {
	ForkID forkid.ID // Fork identifier per EIP-2124

	// Ignore additional fields (for forward compatibility).
	Rest []rlp.RawValue `rlp:"tail"`
}

// ENRKey implements enr.Entry.
func (e xpsEntry) ENRKey() string {
	return "xps"
}

// startXpsEntryUpdate starts the ENR updater loop.
func (xps *xPayments) startXpsEntryUpdate(ln *enode.LocalNode) {
	var newHead = make(chan core.ChainHeadEvent, 10)
	sub := xps.blockchain.SubscribeChainHeadEvent(newHead)

	go func() {
		defer sub.Unsubscribe()
		for {
			select {
			case <-newHead:
				ln.Set(xps.currentXpsEntry())
			case <-sub.Err():
				// Would be nice to sync with xps.Stop, but there is no
				// good way to do that.
				return
			}
		}
	}()
}

func (xps *xPayments) currentXpsEntry() *xpsEntry {
	return &xpsEntry{ForkID: forkid.NewID(xps.blockchain.Config(), xps.blockchain.Genesis().Hash(),
		xps.blockchain.CurrentHeader().Number.Uint64())}
}
