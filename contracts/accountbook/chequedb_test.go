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

package accountbook

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestPersistContractAddr(t *testing.T) {
	db := newChequeDB(rawdb.NewMemoryDatabase())
	local, contract := common.HexToAddress("cafebabe"), common.HexToAddress("deadbeef")

	// Read non-existent data
	got := db.readContractAddr(local)
	if got != nil {
		t.Fatalf("Should return nil for non-existent data")
	}
	db.writeContractAddr(local, contract)
	got = db.readContractAddr(local)
	if got == nil {
		t.Fatalf("Can't read back the written addr")
	}
	if *got != contract {
		t.Fatalf("Mismatch between the written addr with read one, want: %s, got %s", contract.Hex(), got.Hex())
	}
}

func TestPersistCheque(t *testing.T) {
	db := newChequeDB(rawdb.NewMemoryDatabase())
	contract := common.HexToAddress("cafebabe")
	key, _ := crypto.GenerateKey()
	drawer := crypto.PubkeyToAddress(key.PublicKey)

	// Read non-existent data
	got := db.readCheque(contract, drawer)
	if got != nil {
		t.Fatalf("Should return nil for non-existent data")
	}
	cheque := &Cheque{
		Drawer:       drawer,
		ContractAddr: contract,
		Amount:       big.NewInt(1),
	}
	cheque.signWithKey(func(digestHash []byte) ([]byte, error) {
		sig, _ := crypto.Sign(digestHash, key)
		return sig, nil
	})
	db.writeCheque(contract, drawer, cheque)
	got = db.readCheque(contract, drawer)
	if got == nil {
		t.Fatalf("Failed to retrieve cheque from db")
	}
	if !reflect.DeepEqual(cheque, got) {
		t.Fatalf("Mismatch between the written cheque with the read one")
	}
	// Persist a unsigned cheque, it should be retrieved
	cheque2 := &Cheque{
		Drawer:       drawer,
		ContractAddr: contract,
		Amount:       big.NewInt(1),
	}
	db.writeCheque(contract, drawer, cheque2)
	got = db.readCheque(contract, drawer)
	if got == nil {
		t.Fatalf("Failed to retrieve cheque from db")
	}
	if cheque2.Amount.Cmp(got.Amount) != 0 || cheque2.ContractAddr != got.ContractAddr {
		t.Fatalf("Mismatch between the written cheque with the read one")
	}
}

func TestPersistLastIssued(t *testing.T) {
	db := newChequeDB(rawdb.NewMemoryDatabase())
	drawer, contract := common.HexToAddress("cafebabe"), common.HexToAddress("deadbeef")
	amount := big.NewInt(100)

	// Read non-existent data
	got := db.readLastIssued(drawer, contract)
	if got != nil {
		t.Fatalf("Should return nil for non-existent data")
	}
	db.writeLastIssued(drawer, contract, amount)
	got = db.readLastIssued(drawer, contract)
	if got == nil || got.Cmp(amount) != 0 {
		t.Fatalf("Mismatch between the written amount with the read one, want: %d, got: %d", amount, got)
	}
}

func TestListCheques(t *testing.T) {
	db := newChequeDB(rawdb.NewMemoryDatabase())
	contract := common.HexToAddress("cafebabe")

	var cheques []*Cheque
	for i := 0; i < 10; i++ {
		key, _ := crypto.GenerateKey()
		drawer := crypto.PubkeyToAddress(key.PublicKey)

		cheque := &Cheque{
			Drawer:       drawer,
			ContractAddr: contract,
			Amount:       big.NewInt(1),
		}
		cheque.signWithKey(func(digestHash []byte) ([]byte, error) {
			sig, _ := crypto.Sign(digestHash, key)
			return sig, nil
		})
		cheques = append(cheques, cheque)
		db.writeCheque(contract, drawer, cheque)
	}
	got := db.allCheques(contract)
	if len(got) != len(cheques) {
		t.Fatalf("Failed to read all cheques")
	}
	for _, c1 := range got {
		var find bool
		for _, c2 := range cheques {
			if c1.Drawer == c2.Drawer {
				find = true
				if !reflect.DeepEqual(c1, c2) {
					t.Fatalf("Mismatch between the written cheque with the read one")
				}
				break
			}
		}
		if !find {
			t.Fatalf("Miss cheque in the database")
		}
	}
	// Read non-existent records
	got = db.allCheques(common.HexToAddress("deadbeef"))
	if len(got) != 0 {
		t.Fatalf("Should return nil for non-existent data")
	}
}

func TestListAllIssued(t *testing.T) {
	db := newChequeDB(rawdb.NewMemoryDatabase())
	drawer := common.HexToAddress("cafebabe")

	var (
		addresses []common.Address
		amounts   []*big.Int
	)
	for i := 0; i < 10; i++ {
		c, amount := common.BytesToAddress([]byte{byte(i + 1)}), big.NewInt(int64(i+1))
		addresses = append(addresses, c)
		amounts = append(amounts, amount)
		db.writeLastIssued(drawer, c, amount)
	}
	addresses2, amounts2 := db.allIssued(drawer)
	if !reflect.DeepEqual(addresses, addresses2) {
		t.Fatalf("Addresses mismatch, want: %v, got: %v", addresses, addresses2)
	}
	if !reflect.DeepEqual(amounts, amounts2) {
		t.Fatalf("Amounts mismatch, want: %v, got: %v", amounts, amounts2)
	}
	// Read non-existent records
	addresses2, amounts2 = db.allIssued(common.HexToAddress("deadbeef"))
	if len(addresses2) != 0 || len(amounts2) != 0 {
		t.Fatalf("Should return nil for non-existent data")
	}
}
