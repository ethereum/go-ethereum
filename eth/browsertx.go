// Copyright 2026 The go-ethereum Authors
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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/p2p"
)

// The browser tx-submission subprotocol lets anonymous browser peers submit
// signed transactions over QUIC. A peer sends a single TxSubmit message, the
// node adds the transactions to its pool and replies with TxResult, and the
// connection is then dropped.
const (
	browserTxName    = "btx"
	browserTxVersion = 1
	browserTxLength  = 2

	browserTxTimeout = 10 * time.Second
)

const (
	txSubmitMsg = 0x00
	txResultMsg = 0x01
)

// TxResult reports the outcome of one submitted transaction. Error is empty if
// the transaction was accepted into the pool.
type TxResult struct {
	Hash  common.Hash
	Error string
}

// makeBrowserTxProtocol returns the browser tx-submission subprotocol.
func makeBrowserTxProtocol(pool txPool) p2p.Protocol {
	return p2p.Protocol{
		Name:         browserTxName,
		Version:      browserTxVersion,
		Length:       browserTxLength,
		AllowBrowser: true,
		Run: func(_ *p2p.Peer, rw p2p.MsgReadWriter) error {
			return handleBrowserTx(pool, rw)
		},
	}
}

// handleBrowserTx reads one submission, adds it to the pool and replies. It
// returns after the reply, which disconnects the peer.
func handleBrowserTx(pool txPool, rw p2p.MsgReadWriter) error {
	type result struct {
		txs []*types.Transaction
		err error
	}
	ch := make(chan result, 1)
	go func() {
		txs, err := readBrowserTx(rw)
		ch <- result{txs, err}
	}()
	var txs []*types.Transaction
	select {
	case r := <-ch:
		if r.err != nil {
			return r.err
		}
		txs = r.txs
	case <-time.After(browserTxTimeout):
		return p2p.DiscReadTimeout
	}

	errs := pool.Add(txs, false)
	results := make([]TxResult, len(txs))
	for i, tx := range txs {
		results[i].Hash = tx.Hash()
		if errs[i] != nil {
			results[i].Error = errs[i].Error()
		}
	}
	return p2p.Send(rw, txResultMsg, results)
}

// readBrowserTx reads the submission message
func readBrowserTx(rw p2p.MsgReadWriter) ([]*types.Transaction, error) {
	msg, err := rw.ReadMsg()
	if err != nil {
		return nil, err
	}
	defer msg.Discard()
	if msg.Code != txSubmitMsg {
		return nil, p2p.DiscProtocolError
	}
	var txs []*types.Transaction
	if err := msg.Decode(&txs); err != nil {
		return nil, p2p.DiscProtocolError
	}
	return txs, nil
}
