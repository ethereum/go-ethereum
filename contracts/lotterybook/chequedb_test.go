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

package lotterybook

import (
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/lotterybook/merkletree"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestPersistCheque(t *testing.T) {
	db := newChequeDB(rawdb.NewMemoryDatabase())

	draweeKey, _ := crypto.GenerateKey()
	drawee := crypto.PubkeyToAddress(draweeKey.PublicKey)
	drawerKey, _ := crypto.GenerateKey()
	drawer := crypto.PubkeyToAddress(drawerKey.PublicKey)

	// Read non-existent data
	got := db.readCheque(drawee, drawer, common.HexToHash("deadbeef"), true)
	if got != nil {
		t.Fatalf("Should return nil for non-existent data")
	}
	entry := &merkletree.Entry{Value: drawee.Bytes(), Weight: 1}
	tree, _ := merkletree.NewMerkleTree([]*merkletree.Entry{entry})
	witness, err := tree.Prove(entry)
	if err != nil {
		t.Fatalf("Failed to build merkle proof: %v", err)
	}
	cheque, err := newCheque(witness, common.HexToAddress("cafebabe"), 10086, entry.Salt())
	if err != nil {
		t.Fatalf("Failed to create cheque: %v", err)
	}
	if err := cheque.signWithKey(func(digestHash []byte) ([]byte, error) {
		return crypto.Sign(digestHash, drawerKey)
	}); err != nil {
		t.Fatalf("Failed to sign, %v", err)
	}
	db.writeCheque(drawee, drawer, cheque, true)
	got = db.readCheque(drawee, drawer, cheque.LotteryId, true)
	if got == nil {
		t.Fatalf("Failed to retrieve cheque from db")
	}
	if !reflect.DeepEqual(cheque, got) {
		t.Fatalf("Mismatch between the written cheque with the read one")
	}

	// Try to read in the receiver side
	got = db.readCheque(drawee, drawer, cheque.LotteryId, false)
	if got != nil {
		t.Fatalf("Should return nil for non-existent data")
	}
	db.writeCheque(drawee, drawer, cheque, false)
	got = db.readCheque(drawee, drawer, cheque.LotteryId, false)
	if got == nil {
		t.Fatalf("Failed to retrieve cheque from db")
	}
	if !reflect.DeepEqual(cheque, got) {
		t.Fatalf("Mismatch between the written cheque with the read one")
	}
}

func TestPersistLottery(t *testing.T) {
	db := newChequeDB(rawdb.NewMemoryDatabase())
	drawerKey, _ := crypto.GenerateKey()
	drawer := crypto.PubkeyToAddress(drawerKey.PublicKey)
	lottery := &Lottery{
		Id:           common.HexToHash("deadbeef"),
		Amount:       10,
		RevealNumber: 10086,
	}
	db.writeLottery(drawer, lottery.Id, false, lottery)
	got := db.readLottery(drawer, lottery.Id)
	if !reflect.DeepEqual(got, lottery) {
		t.Fatalf("Mismatch between the written lottery with the read one")
	}
	got = db.readLottery(drawer, common.HexToHash("cafebabe"))
	if got != nil {
		t.Fatalf("Should return nil for non-existent data")
	}
}

func TestListLotteries(t *testing.T) {
	db := newChequeDB(rawdb.NewMemoryDatabase())
	drawerKey, _ := crypto.GenerateKey()
	drawer := crypto.PubkeyToAddress(drawerKey.PublicKey)

	var cases = []struct {
		id     common.Hash
		amount uint64
		reveal uint64
	}{
		{common.HexToHash("deadbeef"), 10086, 1},
		{common.HexToHash("deadbeef2"), 10086, 2},
		{common.HexToHash("deadbeef3"), 10086, 3},
		{common.HexToHash("deadbeef4"), 10086, 4},
	}
	for _, c := range cases {
		db.writeLottery(drawer, c.id, false, &Lottery{
			Id:           c.id,
			Amount:       c.amount,
			RevealNumber: c.reveal,
		})
	}
	got := db.listLotteries(drawer, false)
	for _, l := range got {
		var find bool
		for _, c := range cases {
			if c.id == l.Id {
				find = true
			}
		}
		if !find {
			t.Fatalf("Failed to iterate all lotteries")
		}
	}
	if len(got) != len(cases) {
		t.Fatalf("Lotteries number mismatch")
	}
}

func TestListCheques(t *testing.T) {
	db := newChequeDB(rawdb.NewMemoryDatabase())

	drawerKey, _ := crypto.GenerateKey()
	drawer := crypto.PubkeyToAddress(drawerKey.PublicKey)

	var cheques []*Cheque
	var entries []*merkletree.Entry
	for i := 0; i < 16; i++ {
		key, _ := crypto.GenerateKey()
		drawee := crypto.PubkeyToAddress(key.PublicKey)
		entries = append(entries, &merkletree.Entry{Value: drawee.Bytes(), Weight: 1})
	}
	tree, _ := merkletree.NewMerkleTree(entries)
	for _, e := range entries {
		witness, _ := tree.Prove(e)
		cheque, err := newCheque(witness, common.HexToAddress("cafebabe"), 10086, e.Salt())
		if err != nil {
			t.Fatalf("Failed to create cheque: %v", err)
		}
		cheque.signWithKey(func(digestHash []byte) ([]byte, error) {
			return crypto.Sign(digestHash, drawerKey)
		})
		cheques = append(cheques, cheque)
		db.writeCheque(common.BytesToAddress(e.Value), drawer, cheque, true)
	}
	dbCheques, addresses := db.listCheques(drawer, nil)
	if len(dbCheques) != len(cheques) {
		t.Fatalf("Failed to read all cheques")
	}
	for index, dbc := range dbCheques {
		var buff [8]byte
		binary.BigEndian.PutUint64(buff[:], dbc.ReceiverSalt)
		if crypto.Keccak256Hash(append(addresses[index].Bytes(), buff[:]...)) != dbc.Witness[0] {
			t.Fatalf("Invalid cheque")
		}
		var find bool
		for _, c := range cheques {
			if reflect.DeepEqual(c.Witness, dbc.Witness) {
				find = true
				if !reflect.DeepEqual(dbc, c) {
					t.Fatalf("Mismatch between the written cheque with the read one")
				}
				break
			}
		}
		if !find {
			t.Fatalf("Miss cheque in the database")
		}
	}
	dbCheques, addresses = db.listCheques(
		drawer,
		func(addr common.Address, id common.Hash, cheque *Cheque) bool {
			return addr == common.BytesToAddress(entries[0].Value)
		},
	)
	if len(dbCheques) != 1 || len(addresses) != 1 {
		t.Fatalf("Should only return 1 element")
	}
	if addresses[0] != common.BytesToAddress(entries[0].Value) {
		t.Fatalf("Drawee address mismatch")
	}
	// Read non-existent records
	_, got := db.listCheques(common.HexToAddress("deadbeef"), nil)
	if len(got) != 0 {
		t.Fatalf("Should return nil for non-existent data")
	}
}
