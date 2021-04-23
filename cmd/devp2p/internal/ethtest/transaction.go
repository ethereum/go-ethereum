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
	defer sendConn.Close()
	sendSuccessfulTxWithConn(t, s, tx, sendConn)
}

func sendSuccessfulTxWithConn(t *utesting.T, s *Suite, tx *types.Transaction, sendConn *Conn) {
	t.Logf("sending tx: %v %v %v\n", tx.Hash().String(), tx.GasPrice(), tx.Gas())
	// Send the transaction
	if err := sendConn.Write(&Transactions{tx}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	recvConn := s.setupConnection(t)
	// Wait for the transaction announcement
	switch msg := recvConn.ReadAndServe(s.chain, timeout).(type) {
	case *Transactions:
		recTxs := *msg
		for _, gotTx := range recTxs {
			if gotTx.Hash() == tx.Hash() {
				// Ok
				return
			}
		}
		t.Fatalf("missing transaction: got %v missing %v", recTxs, tx.Hash())
	case *NewPooledTransactionHashes:
		txHashes := *msg
		for _, gotHash := range txHashes {
			if gotHash == tx.Hash() {
				return
			}
		}
		t.Fatalf("missing transaction announcement: got %v missing %v", txHashes, tx.Hash())
	default:
		t.Fatalf("unexpected message in sendSuccessfulTx: %s", pretty.Sdump(msg))
	}
}

func waitForTxPropagation(t *utesting.T, s *Suite, txs []*types.Transaction, recvConn *Conn) {
	// Wait for another transaction announcement
	switch msg := recvConn.ReadAndServe(s.chain, time.Second*8).(type) {
	case *Transactions:
		// check to see if any of the failing txs were in the announcement
		recvTxs := make([]common.Hash, len(*msg))
		for i, recvTx := range *msg {
			recvTxs[i] = recvTx.Hash()
		}
		badTxs := containsTxs(recvTxs, txs)
		if len(badTxs) > 0 {
			for _, tx := range badTxs {
				t.Logf("received bad tx: %v", tx)
			}
			t.Fatalf("received %d bad txs", len(badTxs))
		}
	case *NewPooledTransactionHashes:
		badTxs := containsTxs(*msg, txs)
		if len(badTxs) > 0 {
			for _, tx := range badTxs {
				t.Logf("received bad tx: %v", tx)
			}
			t.Fatalf("received %d bad txs", len(badTxs))
		}
	case *Error:
		// Transaction should not be announced -> wait for timeout
		return
	default:
		t.Fatalf("unexpected message in sendFailingTx: %s", pretty.Sdump(msg))
	}
}

// containsTxs checks whether the hashes of the received transactions are present in
// the given set of txs
func containsTxs(recvTxs []common.Hash, txs []*types.Transaction) []common.Hash {
	containedTxs := make([]common.Hash, 0)
	for _, recvTx := range recvTxs {
		for _, tx := range txs {
			if recvTx == tx.Hash() {
				containedTxs = append(containedTxs, recvTx)
			}
		}
	}
	return containedTxs
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
