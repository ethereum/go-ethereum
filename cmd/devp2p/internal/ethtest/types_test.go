// Copyright 2021 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

// TODO document everything

func Test_encodeEthMessage(t *testing.T) {
	chain := getChain(t)
	// create tx
	txs := getTx(chain)
	tx := Transactions{&txs}

	packet, err := encodeEthMessage(&tx)
	if err != nil {
		t.Fatal(err)
	}

	expected, err := rlp.EncodeToBytes(&txs)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, expected, packet)
}

func Test_decodeEthMessage(t *testing.T) {
	// load chain
	chain := getChain(t)
	// create tx
	txs := getTx(chain)
	expected := &Transactions{&txs}

	packet, err := rlp.EncodeToBytes(&expected.TransactionsPacket)
	if err != nil {
		t.Fatal(err)
	}

	msg := new(Transactions)
	txMessage := decodeEthMessage(packet, msg)

	actual, ok := txMessage.(*Transactions)
	if !ok {
		t.Fatalf("wrong message type: %v", msg)
	}

	for i, tx := range *actual.TransactionsPacket {
		expectedTxs := *expected.TransactionsPacket
		expectedTx := expectedTxs[i]
		assert.Equal(t, expectedTx.Data(), tx.Data())
	}
}

func getTx(chain *Chain) eth.TransactionsPacket {
	// Get some transactions
	transactions := make([]*types.Transaction, 0)
	for _, tx := range chain.blocks[1].Transactions() {
		transactions = append(transactions, tx)
	}
	return transactions
}

func getChain(t *testing.T) *Chain {
	// load chain
	chainFile, err := filepath.Abs("./testdata/chain.rlp")
	if err != nil {
		t.Fatal(err)
	}
	genesisFile, err := filepath.Abs("./testdata/genesis.json")
	if err != nil {
		t.Fatal(err)
	}
	chain, err := loadChain(chainFile, genesisFile)
	if err != nil {
		t.Fatal(err)
	}
	return chain
}


