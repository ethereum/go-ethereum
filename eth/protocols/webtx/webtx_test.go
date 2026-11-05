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

package webtx

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
)

// stubTxPool records submitted transactions and returns err for each of them.
// Only Add is exercised by the browser tx subprotocol.
type stubTxPool struct {
	added []*types.Transaction
	err   error
}

func (s *stubTxPool) Add(txs []*types.Transaction, sync bool) []error {
	s.added = append(s.added, txs...)
	errs := make([]error, len(txs))
	for i := range errs {
		errs[i] = s.err
	}
	return errs
}

func TestBrowserTxSubprotocol(t *testing.T) {
	key, _ := crypto.GenerateKey()
	signer := types.LatestSignerForChainID(big.NewInt(1))
	tx, err := types.SignTx(types.NewTransaction(0, common.Address{1}, big.NewInt(0), params.TxGas, big.NewInt(1), nil), signer, key)
	if err != nil {
		t.Fatal(err)
	}

	pool := &stubTxPool{}
	proto := NewProtocol(pool)
	if !proto.AllowBrowser {
		t.Fatal("browser tx protocol must set AllowBrowser")
	}

	app, srv := p2p.MsgPipe()
	defer app.Close()
	runErr := make(chan error, 1)
	go func() { runErr <- proto.Run(nil, srv) }()

	if err := p2p.Send(app, txSubmitMsg, []*types.Transaction{tx}); err != nil {
		t.Fatal(err)
	}
	msg, err := app.ReadMsg()
	if err != nil {
		t.Fatal(err)
	}
	if msg.Code != txResultMsg {
		t.Fatalf("got reply code %d, want %d", msg.Code, txResultMsg)
	}
	var results []TxResult
	if err := msg.Decode(&results); err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Result != txResultOK {
		t.Fatalf("got result %d, want %d", results[0].Result, txResultOK)
	}
	if len(pool.added) != 1 || pool.added[0].Hash() != tx.Hash() {
		t.Fatal("transaction was not added to the pool")
	}
	if err := <-runErr; err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func TestBrowserTxSubprotocolInvalid(t *testing.T) {
	key, _ := crypto.GenerateKey()
	signer := types.LatestSignerForChainID(big.NewInt(1))
	tx, err := types.SignTx(types.NewTransaction(0, common.Address{1}, big.NewInt(0), params.TxGas, big.NewInt(1), nil), signer, key)
	if err != nil {
		t.Fatal(err)
	}

	pool := &stubTxPool{err: types.ErrInvalidSig}
	proto := NewProtocol(pool)

	app, srv := p2p.MsgPipe()
	defer app.Close()
	runErr := make(chan error, 1)
	go func() { runErr <- proto.Run(nil, srv) }()

	if err := p2p.Send(app, txSubmitMsg, []*types.Transaction{tx}); err != nil {
		t.Fatal(err)
	}
	msg, err := app.ReadMsg()
	if err != nil {
		t.Fatal(err)
	}
	var results []TxResult
	if err := msg.Decode(&results); err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Result != txResultInvalidTx {
		t.Fatalf("got results %v, want [%d]", results, txResultInvalidTx)
	}
	if err := <-runErr; err != p2p.DiscSubprotocolError {
		t.Fatalf("Run returned %v, want %v", err, p2p.DiscSubprotocolError)
	}
}

// makeTestTxs creates count signed transactions from a single account.
func makeTestTxs(t *testing.T, count int) []*types.Transaction {
	t.Helper()

	key, _ := crypto.GenerateKey()
	signer := types.LatestSignerForChainID(big.NewInt(1))
	txs := make([]*types.Transaction, count)
	for i := range txs {
		tx, err := types.SignTx(types.NewTransaction(uint64(i), common.Address{1}, big.NewInt(0), params.TxGas, big.NewInt(1), nil), signer, key)
		if err != nil {
			t.Fatal(err)
		}
		txs[i] = tx
	}
	return txs
}

// runSubmission hands txs to a fresh subprotocol instance and reports the
// result codes sent back to the peer, if any, along with the error Run returned.
func runSubmission(t *testing.T, pool TxPool, txs []*types.Transaction) ([]TxResult, error) {
	t.Helper()

	proto := NewProtocol(pool)
	app, srv := p2p.MsgPipe()
	defer app.Close()

	runErr := make(chan error, 1)
	go func() { runErr <- proto.Run(nil, srv) }()

	type readResult struct {
		results []TxResult
		err     error
	}
	reply := make(chan readResult, 1)
	go func() {
		msg, err := app.ReadMsg()
		if err != nil {
			reply <- readResult{err: err}
			return
		}
		var results []TxResult
		err = msg.Decode(&results)
		reply <- readResult{results, err}
	}()

	if err := p2p.Send(app, txSubmitMsg, txs); err != nil {
		t.Fatal(err)
	}
	err := <-runErr
	if err != nil {
		// The peer was dropped without a reply.
		return nil, err
	}
	r := <-reply
	if r.err != nil {
		t.Fatalf("failed to read reply: %v", r.err)
	}
	return r.results, nil
}

// This test checks that a submission cannot carry an unbounded number of
// transactions. Transactions are small enough that the 1MB size limit admits
// thousands of them, and the pool has to validate each one.
func TestBrowserTxSubprotocolSubmitLimit(t *testing.T) {
	for _, test := range []struct {
		name    string
		count   int
		wantErr error
	}{
		{"at-limit", maxSubmitCount, nil},
		{"over-limit", maxSubmitCount + 1, p2p.DiscProtocolError},
	} {
		t.Run(test.name, func(t *testing.T) {
			pool := &stubTxPool{}
			results, err := runSubmission(t, pool, makeTestTxs(t, test.count))
			if err != test.wantErr {
				t.Fatalf("Run returned %v, want %v", err, test.wantErr)
			}
			want := 0
			if test.wantErr == nil {
				want = test.count
			}
			if len(pool.added) != want {
				t.Fatalf("pool received %d transactions, want %d", len(pool.added), want)
			}
			if len(results) != want {
				t.Fatalf("got %d result codes, want %d", len(results), want)
			}
		})
	}
}
