// Copyright 2014 The go-ethereum Authors
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
)

func TestBloom(t *testing.T) {
	positive := []string{
		"testtest",
		"test",
		"hallo",
		"other",
	}
	negative := []string{
		"tes",
		"lo",
	}

	var bloom Bloom
	for _, data := range positive {
		bloom.Add(new(big.Int).SetBytes([]byte(data)))
	}

	for _, data := range positive {
		if !bloom.TestBytes([]byte(data)) {
			t.Error("expected", data, "to test true")
		}
	}
	for _, data := range negative {
		if bloom.TestBytes([]byte(data)) {
			t.Error("did not expect", data, "to test true")
		}
	}
}

func TestBloom9(t *testing.T) {
	tests := []string{
		"testtest",
		"test",
		"hallo",
		"other",
		"tes",
		"lo",
	}
	for _, test := range tests {
		a := bloom9([]byte(test))
		b := new(big.Int).SetBytes(bloom10([]byte(test)))
		if a.Cmp(b) != 0 {
			t.Fatalf("Different results \n %v \n %v \n %v \n", test, a, b)
		}
	}
}

type Byter []byte

func (b Byter) Bytes() []byte {
	return b
}
func TestBloomLookup(t *testing.T) {
	tests := []Byter{
		Byter([]byte("testtest")),
		Byter([]byte("test")),
		Byter([]byte("te")),
		Byter([]byte("asdf")),
		Byter([]byte("asdfasdf")),
		Byter([]byte("asdfasdf")),
		Byter([]byte("asdfasdf")),
		Byter([]byte("12344")),
	}
	aBloom := new(Bloom)
	bBloom := new(Bloom)
	for _, test := range tests {
		a := BloomLookup(*aBloom, test)
		b := Bloom10Lookup(*bBloom, test)
		aBloom.Add(new(big.Int).SetBytes(test))
		bBloom.Add(new(big.Int).SetBytes(test))
		if a != b {
			t.Fatalf("Different results \n %v \n %v \n %v \n", test, a, b)
		}
	}
}

func BenchmarkBloom9(b *testing.B) {
	test := []byte("testestestest")
	for i := 0; i < b.N; i++ {
		bloom9(test)
	}
}

func BenchmarkBloom10(b *testing.B) {
	test := []byte("testestestest")
	for i := 0; i < b.N; i++ {
		bloom10(test)
	}
}

func BenchmarkBloom9Lookup(b *testing.B) {
	test := Byter([]byte("testtest"))
	bloom := new(Bloom)
	for i := 0; i < b.N; i++ {
		BloomLookup(*bloom, test)
	}
}

func BenchmarkBloom10Lookup(b *testing.B) {
	test := Byter([]byte("testtest"))
	bloom := new(Bloom)
	for i := 0; i < b.N; i++ {
		Bloom10Lookup(*bloom, test)
	}
}

var txs = Transactions{
	NewContractCreation(1, big.NewInt(1), 1, big.NewInt(1), nil),
	NewTransaction(2, common.HexToAddress("0x2"), big.NewInt(2), 2, big.NewInt(2), nil),
}
var receipts = Receipts{
	&Receipt{
		Status:            ReceiptStatusFailed,
		CumulativeGasUsed: 1,
		Logs: []*Log{
			{Address: common.BytesToAddress([]byte{0x11})},
			{Address: common.BytesToAddress([]byte{0x01, 0x11})},
		},
		TxHash:          txs[0].Hash(),
		ContractAddress: common.BytesToAddress([]byte{0x01, 0x11, 0x11}),
		GasUsed:         1,
	},
	&Receipt{
		PostState:         common.Hash{2}.Bytes(),
		CumulativeGasUsed: 3,
		Logs: []*Log{
			{Address: common.BytesToAddress([]byte{0x22})},
			{Address: common.BytesToAddress([]byte{0x02, 0x22})},
		},
		TxHash:          txs[1].Hash(),
		ContractAddress: common.BytesToAddress([]byte{0x02, 0x22, 0x22}),
		GasUsed:         2,
	},
}

func TestCreateBloom(t *testing.T) {
	// Create a few transactions to have receipts for
	if CreateBloom(receipts) != CreateBloom10(receipts) {
		t.Fatal("wrong")
	}
}

func BenchmarkCreateBloom(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CreateBloom(receipts)
	}
}

func BenchmarkCreateBloom10(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CreateBloom10(receipts)
	}
}

/*
import (
	"testing"

	"github.com/ethereum/go-ethereum/core/state"
)

func TestBloom9(t *testing.T) {
	testCase := []byte("testtest")
	bin := LogsBloom([]state.Log{
		{testCase, [][]byte{[]byte("hellohello")}, nil},
	}).Bytes()
	res := BloomLookup(bin, testCase)

	if !res {
		t.Errorf("Bloom lookup failed")
	}
}


func TestAddress(t *testing.T) {
	block := &Block{}
	block.Coinbase = common.Hex2Bytes("22341ae42d6dd7384bc8584e50419ea3ac75b83f")
	fmt.Printf("%x\n", crypto.Keccak256(block.Coinbase))

	bin := CreateBloom(block)
	fmt.Printf("bin = %x\n", common.LeftPadBytes(bin, 64))
}
*/
