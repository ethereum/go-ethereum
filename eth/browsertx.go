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
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/txpool/blobpool"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
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

// Result codes reported in TxResult
const (
	txResultOK                = iota // accepted into the pool
	txResultAlreadyKnown             // already in the pool
	txResultInvalidNonce             // nonce too low or too high
	txResultInvalidFee               // fee fields invalid or replacement underpriced
	txResultGasPriceTooLow           // tip below the node's minimum
	txResultInsufficientFunds        // balance can't cover value and fees
	txResultInvalidGasLimit          // gas limit too low or too high
	txResultPoolError                // rejected due to local pool state, may succeed if submitted elsewhere
	txResultInvalidTx                // any other error
)

// TxResult reports the outcome of one submitted transaction.
type TxResult struct {
	Result uint8
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
	invalid := false
	for i, err := range errs {
		results[i].Result = txResultCode(err)
		if results[i].Result != txResultOK {
			invalid = true
		}
	}
	if err := p2p.Send(rw, txResultMsg, results); err != nil {
		return err
	}
	// Submitting an invalid transaction penalizes the peer.
	if invalid {
		return p2p.DiscSubprotocolError
	}
	return nil
}

// txResultCode maps a pool error to a result code.
func txResultCode(err error) uint8 {
	switch {
	case err == nil:
		return txResultOK
	case errors.Is(err, txpool.ErrAlreadyKnown):
		return txResultAlreadyKnown
	case errors.Is(err, core.ErrNonceTooLow),
		errors.Is(err, core.ErrNonceTooHigh),
		errors.Is(err, core.ErrNonceMax):
		return txResultInvalidNonce
	case errors.Is(err, core.ErrTipAboveFeeCap),
		errors.Is(err, core.ErrFeeCapVeryHigh),
		errors.Is(err, core.ErrTipVeryHigh),
		errors.Is(err, txpool.ErrReplaceUnderpriced):
		return txResultInvalidFee
	case errors.Is(err, txpool.ErrTxGasPriceTooLow):
		return txResultGasPriceTooLow
	case errors.Is(err, core.ErrInsufficientFunds),
		errors.Is(err, txpool.ErrNegativeValue):
		return txResultInsufficientFunds
	case errors.Is(err, txpool.ErrGasLimit),
		errors.Is(err, core.ErrGasLimitTooHigh),
		errors.Is(err, core.ErrIntrinsicGas),
		errors.Is(err, core.ErrFloorDataGas):
		return txResultInvalidGasLimit
	case errors.Is(err, txpool.ErrUnderpriced),
		errors.Is(err, legacypool.ErrTxPoolOverflow),
		errors.Is(err, txpool.ErrAlreadyReserved),
		errors.Is(err, txpool.ErrInflightTxLimitReached),
		errors.Is(err, legacypool.ErrAuthorityReserved),
		errors.Is(err, legacypool.ErrOutOfOrderTxFromDelegated),
		errors.Is(err, legacypool.ErrFutureReplacePending),
		errors.Is(err, txpool.ErrAccountLimitExceeded),
		errors.Is(err, blobpool.ErrConversionQueueClosed),
		errors.Is(err, blobpool.ErrConversionInterrupted),
		errors.Is(err, blobpool.ErrConversionQueueOverloaded):
		return txResultPoolError
	default:
		return txResultInvalidTx
	}
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
