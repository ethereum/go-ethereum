// Copyright 2016 The go-ethereum Authors
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

package types

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestEIP155Signing(t *testing.T) {
	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)

	signer := NewEIP155Signer(big.NewInt(18))
	tx, err := SignTx(NewTransaction(0, addr, new(big.Int), 0, new(big.Int), nil), signer, key)
	if err != nil {
		t.Fatal(err)
	}

	from, err := Sender(signer, tx)
	if err != nil {
		t.Fatal(err)
	}
	if from != addr {
		t.Errorf("exected from and address to be equal. Got %x want %x", from, addr)
	}
}

func TestEIP155ChainId(t *testing.T) {
	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)

	signer := NewEIP155Signer(big.NewInt(18))
	tx, err := SignTx(NewTransaction(0, addr, new(big.Int), 0, new(big.Int), nil), signer, key)
	if err != nil {
		t.Fatal(err)
	}
	if !tx.Protected() {
		t.Fatal("expected tx to be protected")
	}

	if tx.ChainId().Cmp(signer.chainId) != 0 {
		t.Error("expected chainId to be", signer.chainId, "got", tx.ChainId())
	}

	tx = NewTransaction(0, addr, new(big.Int), 0, new(big.Int), nil)
	tx, err = SignTx(tx, HomesteadSigner{}, key)
	if err != nil {
		t.Fatal(err)
	}

	if tx.Protected() {
		t.Error("didn't expect tx to be protected")
	}

	if tx.ChainId().Sign() != 0 {
		t.Error("expected chain id to be 0 got", tx.ChainId())
	}
}

func TestEIP155SigningVitalik(t *testing.T) {
	// Test vectors come from http://vitalik.ca/files/eip155_testvec.txt
	for i, test := range []struct {
		txRlp, addr string
	}{
		{"f865808504a817c800825208942323232323232323232323232323232323232323808025a0044852b2a670ade5407e78fb2863c51de9fcb96542a07186fe3aeda6bb8a116da0044852b2a670ade5407e78fb2863c51de9fcb96542a07186fe3aeda6bb8a116d80", "0xc109873becfe6bfc287ea44fc03a630c04de7cf2"},
		{"f86e018504a817c80182a410942323232323232323232323232323232323232323018025a0489efdaa54c0f20c7adf612882df0950f5a951637e0307cdcb4c672f298b8bcaa0489efdaa54c0f20c7adf612882df0950f5a951637e0307cdcb4c672f298b8bca89456c6f6e204d61736b", "0x55888d8f398aad8a82cea09ecde3b9c5903e46da"},
		{"f873028504a817c80282f618942323232323232323232323232323232323232323088025a02d7c5bef027816a800da1736444fb58a807ef4c9603b7848673f7e3a68eb14a5a02d7c5bef027816a800da1736444fb58a807ef4c9603b7848673f7e3a68eb14a58e566f7661205a656c656e736b6969", "2cc3fb68ecb99563cf87266b7d97dc5d2c739df7"},
		{"f86a038504a817c803830148209423232323232323232323232323232323232323231b8025a02a80e1ef1d7842f27f2e6be0972bb708b9a135c38860dbe73c27c3486c34f4e0a02a80e1ef1d7842f27f2e6be0972bb708b9a135c38860dbe73c27c3486c34f4e0844c656e61", "0xede031eba233e637abe9a14b9e4649c01bf03302"},
		{"f86d048504a817c80483019a28942323232323232323232323232323232323232323408025a013600b294191fc92924bb3ce4b969c1e7e2bab8f4c93c3fc6d0a51733df3c063a013600b294191fc92924bb3ce4b969c1e7e2bab8f4c93c3fc6d0a51733df3c06387526f6e616c646f", "0x9963e2f0c6e97dc1d9b360b02cb324c894399e93"},
		{"f86b058504a817c8058301ec309423232323232323232323232323232323232323237d8025a04eebf77a833b30520287ddd9478ff51abbdffa30aa90a8d655dba0e8a79ce0c1a04eebf77a833b30520287ddd9478ff51abbdffa30aa90a8d655dba0e8a79ce0c1853132333435", "0xcdf8ec1a488aa82e694d9a960e2dc90f64c0ce01"},
		{"f86d068504a817c80683023e3894232323232323232323232323232323232323232381d88025a06455bf8ea6e7463a1046a0b52804526e119b4bf5136279614e0b1e8e296a4e2fa06455bf8ea6e7463a1046a0b52804526e119b4bf5136279614e0b1e8e296a4e2f864a6f6e373737", "0x2230c4fac77529a0a49be7d8ac1a46ba02142ea5"},
		{"f86e078504a817c807830290409423232323232323232323232323232323232323238201578025a052f1a9b320cab38e5da8a8f97989383aab0a49165fc91c737310e4f7e9821021a052f1a9b320cab38e5da8a8f97989383aab0a49165fc91c737310e4f7e982102186404040404040", "0xd72f08e54a9043c0a0b9e24a18182c159c082f2f"},
		{"f86b088504a817c8088302e2489423232323232323232323232323232323232323238202008025a064b1702d9298fee62dfeccc57d322a463ad55ca201256d01f62b45b2e1c21c12a064b1702d9298fee62dfeccc57d322a463ad55ca201256d01f62b45b2e1c21c1283373737", "0xfd962ebf109b96984b5873796b91f2c0f674a119"},
		{"f86c098504a817c809830334509423232323232323232323232323232323232323238202d98025a04eebf77a833b30520287ddd9478ff51abbdffa30aa90a8d655dba0e8a79ce0c1a04eebf77a833b30520287ddd9478ff51abbdffa30aa90a8d655dba0e8a79ce0c18468683131", "0x689ac45b838931145b0c25accec4f12e26cfbdd3"},
	} {
		signer := NewEIP155Signer(big.NewInt(1))

		var tx *Transaction
		err := rlp.DecodeBytes(common.Hex2Bytes(test.txRlp), &tx)
		if err != nil {
			t.Errorf("%d: %v", i, err)
			continue
		}

		from, err := Sender(signer, tx)
		if err != nil {
			t.Errorf("%d: %v", i, err)
			continue
		}

		addr := common.HexToAddress(test.addr)
		if from != addr {
			t.Errorf("%d: expected %x got %x", i, addr, from)
		}

	}
}

func TestChainId(t *testing.T) {
	key, _ := defaultTestKey()

	tx := NewTransaction(0, common.Address{}, new(big.Int), 0, new(big.Int), nil)

	var err error
	tx, err = SignTx(tx, NewEIP155Signer(big.NewInt(1)), key)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Sender(NewEIP155Signer(big.NewInt(2)), tx)
	if err != ErrInvalidChainId {
		t.Error("expected error:", ErrInvalidChainId)
	}

	_, err = Sender(NewEIP155Signer(big.NewInt(1)), tx)
	if err != nil {
		t.Error("expected no error")
	}
}
