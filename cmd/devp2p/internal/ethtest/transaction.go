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
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/params"
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
	// update last nonce seen
	nonce = tx.Nonce()

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

var nonce = uint64(99)

func sendMultipleSuccessfulTxs(t *utesting.T, s *Suite, sendConn *Conn, txs []*types.Transaction) {
	txMsg := Transactions(txs)
	t.Logf("sending %d txs\n", len(txs))

	recvConn := s.setupConnection(t)
	defer recvConn.Close()

	// Send the transactions
	if err := sendConn.Write(&txMsg); err != nil {
		t.Fatal(err)
	}
	// update nonce
	nonce = txs[len(txs)-1].Nonce()
	// Wait for the transaction announcement(s) and make sure all sent txs are being propagated
	recvHashes := make([]common.Hash, 0)
	// all txs should be announced within 3 announcements
	for i := 0; i < 3; i++ {
		switch msg := recvConn.ReadAndServe(s.chain, timeout).(type) {
		case *Transactions:
			for _, tx := range *msg {
				recvHashes = append(recvHashes, tx.Hash())
			}
		case *NewPooledTransactionHashes:
			recvHashes = append(recvHashes, *msg...)
		default:
			if !strings.Contains(pretty.Sdump(msg), "i/o timeout") {
				t.Fatalf("unexpected message while waiting to receive txs: %s", pretty.Sdump(msg))
			}
		}
		// break once all 2000 txs have been received
		if len(recvHashes) == 2000 {
			break
		}
		if len(recvHashes) > 0 {
			_, missingTxs := compareReceivedTxs(recvHashes, txs)
			if len(missingTxs) > 0 {
				continue
			} else {
				t.Logf("successfully received all %d txs", len(txs))
				return
			}
		}
	}
	_, missingTxs := compareReceivedTxs(recvHashes, txs)
	if len(missingTxs) > 0 {
		for _, missing := range missingTxs {
			t.Logf("missing tx: %v", missing.Hash())
		}
		t.Fatalf("missing %d txs", len(missingTxs))
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
		badTxs, _ := compareReceivedTxs(recvTxs, txs)
		if len(badTxs) > 0 {
			for _, tx := range badTxs {
				t.Logf("received bad tx: %v", tx)
			}
			t.Fatalf("received %d bad txs", len(badTxs))
		}
	case *NewPooledTransactionHashes:
		badTxs, _ := compareReceivedTxs(*msg, txs)
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

// compareReceivedTxs compares the received set of txs against the given set of txs,
// returning both the set received txs that were present within the given txs, and
// the set of txs that were missing from the set of received txs
func compareReceivedTxs(recvTxs []common.Hash, txs []*types.Transaction) (present []*types.Transaction, missing []*types.Transaction) {
	// create a map of the hashes received from node
	recvHashes := make(map[common.Hash]common.Hash)
	for _, hash := range recvTxs {
		recvHashes[hash] = hash
	}

	// collect present txs and missing txs separately
	present = make([]*types.Transaction, 0)
	missing = make([]*types.Transaction, 0)
	for _, tx := range txs {
		if _, exists := recvHashes[tx.Hash()]; exists {
			present = append(present, tx)
		} else {
			missing = append(missing, tx)
		}
	}
	return present, missing
}

func unknownTx(t *utesting.T, s *Suite) *types.Transaction {
	tx := getNextTxFromChain(t, s)
	var to common.Address
	if tx.To() != nil {
		to = *tx.To()
	}
	txNew := types.NewTransaction(tx.Nonce()+1, to, tx.Value(), tx.Gas(), tx.GasPrice(), tx.Data())
	return signWithFaucet(t, s.chain.chainConfig, txNew)
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

func generateTxs(t *utesting.T, s *Suite, numTxs int) (map[common.Hash]common.Hash, []*types.Transaction) {
	txHashMap := make(map[common.Hash]common.Hash, numTxs)
	txs := make([]*types.Transaction, numTxs)

	nextTx := getNextTxFromChain(t, s)
	gas := nextTx.Gas()

	nonce = nonce + 1
	// generate txs
	for i := 0; i < numTxs; i++ {
		tx := generateTx(t, s.chain.chainConfig, nonce, gas)
		txHashMap[tx.Hash()] = tx.Hash()
		txs[i] = tx
		nonce = nonce + 1
	}
	return txHashMap, txs
}

func generateTx(t *utesting.T, chainConfig *params.ChainConfig, nonce uint64, gas uint64) *types.Transaction {
	var to common.Address
	tx := types.NewTransaction(nonce, to, big.NewInt(1), gas, big.NewInt(1), []byte{})
	return signWithFaucet(t, chainConfig, tx)
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
	return signWithFaucet(t, s.chain.chainConfig, txNew)
}

func hugeAmount(t *utesting.T, s *Suite) *types.Transaction {
	tx := getNextTxFromChain(t, s)
	amount := largeNumber(2)
	var to common.Address
	if tx.To() != nil {
		to = *tx.To()
	}
	txNew := types.NewTransaction(tx.Nonce(), to, amount, tx.Gas(), tx.GasPrice(), tx.Data())
	return signWithFaucet(t, s.chain.chainConfig, txNew)
}

func hugeGasPrice(t *utesting.T, s *Suite) *types.Transaction {
	tx := getNextTxFromChain(t, s)
	gasPrice := largeNumber(2)
	var to common.Address
	if tx.To() != nil {
		to = *tx.To()
	}
	txNew := types.NewTransaction(tx.Nonce(), to, tx.Value(), tx.Gas(), gasPrice, tx.Data())
	return signWithFaucet(t, s.chain.chainConfig, txNew)
}

func hugeData(t *utesting.T, s *Suite) *types.Transaction {
	tx := getNextTxFromChain(t, s)
	var to common.Address
	if tx.To() != nil {
		to = *tx.To()
	}
	txNew := types.NewTransaction(tx.Nonce(), to, tx.Value(), tx.Gas(), tx.GasPrice(), largeBuffer(2))
	return signWithFaucet(t, s.chain.chainConfig, txNew)
}

func signWithFaucet(t *utesting.T, chainConfig *params.ChainConfig, tx *types.Transaction) *types.Transaction {
	signer := types.LatestSigner(chainConfig)
	signedTx, err := types.SignTx(tx, signer, faucetKey)
	if err != nil {
		t.Fatalf("could not sign tx: %v\n", err)
	}
	return signedTx
}
