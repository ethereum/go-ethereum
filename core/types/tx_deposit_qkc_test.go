// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
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
	"bytes"
	"math"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

func TestDepositTxRLP(t *testing.T) {
	deposits := []*DepositTx{
		{
			Version:         DepositTxVersion,
			SourceBlockHash: common.Hash{0xaa},
			SourceLogIndex:  7,
			From:            common.Address{0x01},
			To:              common.Address{0x02},
			Value:           uint256.NewInt(1000),
			GasPrice:        uint256.NewInt(10),
			Gas:             9000,
			IsMove:          true,
			Data:            []byte{},
		},
		{
			Version:         DepositTxVersion,
			SourceBlockHash: common.Hash{0xbb},
			SourceLogIndex:  math.MaxUint64,
			From:            common.Address{0x03},
			To:              common.Address{0x04},
			Value:           new(uint256.Int).SetAllOne(),
			GasPrice:        new(uint256.Int),
			Gas:             math.MaxUint64,
			Data:            bytes.Repeat([]byte{0xff}, 100),
		},
	}
	enc, err := rlp.EncodeToBytes(deposits)
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}
	var dec []*DepositTx
	if err := rlp.DecodeBytes(enc, &dec); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !reflect.DeepEqual(dec, deposits) {
		t.Errorf("decoded %+v, want %+v", dec, deposits)
	}
	// Pin the encoding: the fields, in order, as an RLP list.
	want := common.FromHex("0xf856" + "80" + "a0" + "aa" + "00000000000000000000000000000000000000000000000000000000000000" + "07" +
		"94" + "0100000000000000000000000000000000000000" + "94" + "0200000000000000000000000000000000000000" +
		"8203e8" + "0a" + "822328" + "01" + "80")
	if enc, _ := rlp.EncodeToBytes(deposits[0]); !bytes.Equal(enc, want) {
		t.Errorf("encoding %x, want %x", enc, want)
	}
}
