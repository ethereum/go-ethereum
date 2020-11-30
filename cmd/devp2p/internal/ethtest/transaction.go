// Copyright 2020 The go-ethereum Authors
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

package ethtest

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/utesting"
)

//var faucetAddr = common.HexToAddress("0x71562b71999873DB5b286dF957af199Ec94617F7")
var faucetKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

func sendSuccessfulTx(t *utesting.T, s *Suite, tx *types.Transaction) {
	sendConn := s.setupConnection(t)
	t.Logf("sending tx: %v %v %v\n", tx.Hash().String(), tx.GasPrice(), tx.Gas())
	// Send the transaction
	if err := sendConn.Write(Transactions([]*types.Transaction{tx})); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	recvConn := s.setupConnection(t)
	// Wait for the transaction announcement
	switch msg := recvConn.ReadAndServe(s.chain, timeout).(type) {
	case *Transactions:
		recTxs := *msg
		if len(recTxs) < 1 {
			t.Fatalf("received transactions do not match send: %v", recTxs)
		}
		if tx.Hash() != recTxs[len(recTxs)-1].Hash() {
			t.Fatalf("received transactions do not match send: got %v want %v", recTxs, tx)
		}
	case *NewPooledTransactionHashes:
		txHashes := *msg
		if len(txHashes) < 1 {
			t.Fatalf("received transactions do not match send: %v", txHashes)
		}
		if tx.Hash() != txHashes[len(txHashes)-1] {
			t.Fatalf("wrong announcement received, wanted %v got %v", tx, txHashes)
		}
	default:
		t.Fatalf("unexpected message in sendSuccessfulTx: %s", pretty.Sdump(msg))
	}
}

func sendFailingTx(t *utesting.T, s *Suite, tx *types.Transaction) {
	sendConn, recvConn := s.setupConnection(t), s.setupConnection(t)
	// Wait for a transaction announcement
	switch msg := recvConn.ReadAndServe(s.chain, timeout).(type) {
	case *NewPooledTransactionHashes:
		break
	default:
		t.Logf("unexpected message, logging: %v", pretty.Sdump(msg))
	}
	// Send the transaction
	if err := sendConn.Write(Transactions([]*types.Transaction{tx})); err != nil {
		t.Fatal(err)
	}
	// Wait for another transaction announcement
	switch msg := recvConn.ReadAndServe(s.chain, timeout).(type) {
	case *Transactions:
		t.Fatalf("Received unexpected transaction announcement: %v", msg)
	case *NewPooledTransactionHashes:
		t.Fatalf("Received unexpected pooledTx announcement: %v", msg)
	case *Error:
		// Transaction should not be announced -> wait for timeout
		return
	default:
		t.Fatalf("unexpected message in sendFailingTx: %s", pretty.Sdump(msg))
	}
}

func unknownTx(t *utesting.T, s *Suite) *types.Transaction {
	tx := getNextTxFromChain(t, s)
	var to common.Address
	if tx.To() != nil {
		to = *tx.To()
	}
	txNew := types.NewTransaction(tx.Nonce()+1, to, tx.Value(), tx.Gas(), tx.GasPrice(), tx.Data())
	return signWithFaucet(t, txNew)
}

func getNextTxFromChain(t *utesting.T, s *Suite) *types.Transaction {
	// Get a new transaction
	var tx *types.Transaction
	for _, blocks := range s.fullChain.blocks[s.chain.Len():] {
		txs := blocks.Transactions()
		if txs.Len() != 0 {
			tx = txs[0]
			break
		}
	}
	if tx == nil {
		t.Fatal("could not find transaction")
	}
	return tx
}

func getOldTxFromChain(t *utesting.T, s *Suite) *types.Transaction {
	var tx *types.Transaction
	for _, blocks := range s.fullChain.blocks[:s.chain.Len()-1] {
		txs := blocks.Transactions()
		if txs.Len() != 0 {
			tx = txs[0]
			break
		}
	}
	if tx == nil {
		t.Fatal("could not find transaction")
	}
	return tx
}

func invalidNonceTx(t *utesting.T, s *Suite) *types.Transaction {
	tx := getNextTxFromChain(t, s)
	var to common.Address
	if tx.To() != nil {
		to = *tx.To()
	}
	txNew := types.NewTransaction(tx.Nonce()-2, to, tx.Value(), tx.Gas(), tx.GasPrice(), tx.Data())
	return signWithFaucet(t, txNew)
}

func hugeAmount(t *utesting.T, s *Suite) *types.Transaction {
	tx := getNextTxFromChain(t, s)
	amount := largeNumber(2)
	var to common.Address
	if tx.To() != nil {
		to = *tx.To()
	}
	txNew := types.NewTransaction(tx.Nonce(), to, amount, tx.Gas(), tx.GasPrice(), tx.Data())
	return signWithFaucet(t, txNew)
}

func hugeGasPrice(t *utesting.T, s *Suite) *types.Transaction {
	tx := getNextTxFromChain(t, s)
	gasPrice := largeNumber(2)
	var to common.Address
	if tx.To() != nil {
		to = *tx.To()
	}
	txNew := types.NewTransaction(tx.Nonce(), to, tx.Value(), tx.Gas(), gasPrice, tx.Data())
	return signWithFaucet(t, txNew)
}

func hugeData(t *utesting.T, s *Suite) *types.Transaction {
	tx := getNextTxFromChain(t, s)
	var to common.Address
	if tx.To() != nil {
		to = *tx.To()
	}
	txNew := types.NewTransaction(tx.Nonce(), to, tx.Value(), tx.Gas(), tx.GasPrice(), largeBuffer(2))
	return signWithFaucet(t, txNew)
}

func signWithFaucet(t *utesting.T, tx *types.Transaction) *types.Transaction {
	signer := types.HomesteadSigner{}
	signedTx, err := types.SignTx(tx, signer, faucetKey)
	if err != nil {
		t.Fatalf("could not sign tx: %v\n", err)
	}
	return signedTx
}
