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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
)

// stubTxPool records submitted transactions and accepts all of them. Only Add is
// exercised by the browser tx subprotocol.
type stubTxPool struct {
	txPool
	added []*types.Transaction
}

func (s *stubTxPool) Add(txs []*types.Transaction, sync bool) []error {
	s.added = append(s.added, txs...)
	return make([]error, len(txs))
}

func TestBrowserTxSubprotocol(t *testing.T) {
	key, _ := crypto.GenerateKey()
	signer := types.LatestSignerForChainID(big.NewInt(1))
	tx, err := types.SignTx(types.NewTransaction(0, common.Address{1}, big.NewInt(0), params.TxGas, big.NewInt(1), nil), signer, key)
	if err != nil {
		t.Fatal(err)
	}

	pool := &stubTxPool{}
	proto := makeBrowserTxProtocol(pool)
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
	if results[0].Hash != tx.Hash() {
		t.Fatalf("result hash %x, want %x", results[0].Hash, tx.Hash())
	}
	if results[0].Error != "" {
		t.Fatalf("unexpected result error: %s", results[0].Error)
	}
	if len(pool.added) != 1 || pool.added[0].Hash() != tx.Hash() {
		t.Fatal("transaction was not added to the pool")
	}
	if err := <-runErr; err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}
