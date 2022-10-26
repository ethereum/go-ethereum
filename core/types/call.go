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
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
)

//go:generate go run github.com/fjl/gencodec -type Call -field-override callMarshaling -out gen_call_json.go

type CallType byte

// 0xf0 range - closures.
const (
	CREATE       CallType = 0xf0
	CALL         CallType = 0xf1
	CALLCODE     CallType = 0xf2
	RETURN       CallType = 0xf3
	DELEGATECALL CallType = 0xf4
	CREATE2      CallType = 0xf5

	STATICCALL   CallType = 0xfa
	REVERT       CallType = 0xfd
	INVALID      CallType = 0xfe
	SELFDESTRUCT CallType = 0xff
)

type Call struct {
	Type    CallType       `json:"type" gencodec:"required"`
	From    common.Address `json:"from" gencodec:"required"`
	To      common.Address `json:"to" gencodec:"required"`
	Gas     uint64         `json:"gas" gencodec:"required"`
	Value   *big.Int       `json:"value" gencodec:"required"`
	Data    []byte         `json:"data" gencodec:"required"`
	Success bool           `json:"success" gencodec:"required"`

	BlockNumber uint64      `json:"blockNumber"`
	TxHash      common.Hash `json:"transactionHash" gencodec:"required"`
	TxIndex     uint        `json:"transactionIndex"`
	BlockHash   common.Hash `json:"blockHash"`
	Index       uint        `json:"callIndex"`
	Removed     bool        `json:"removed"`
}

type callMarshaling struct {
	Type  hexutil.Uint64
	Gas   hexutil.Uint64
	Value *hexutil.Big
	Data  hexutil.Bytes

	BlockNumber hexutil.Uint64
	TxIndex     hexutil.Uint
	Index       hexutil.Uint
}

type rlpCall struct {
	Type    CallType
	From    common.Address
	To      common.Address
	Gas     uint64
	Value   *big.Int
	Data    []byte
	Success bool
}

// EncodeRLP implements rlp.Encoder.
func (c *Call) EncodeRLP(w io.Writer) error {
	rl := rlpCall{c.Type, c.From, c.To, c.Gas, c.Value, c.Data, c.Success}
	return rlp.Encode(w, &rl)
}

// DecodeRLP implements rlp.Decoder.
func (c *Call) DecodeRLP(s *rlp.Stream) error {
	var dec rlpCall
	err := s.Decode(&dec)
	if err == nil {
		c.Type, c.From, c.To, c.Gas, c.Value, c.Data, c.Success = dec.Type, dec.From, dec.To, dec.Gas, dec.Value, dec.Data, dec.Success
	}
	return err
}

type CallForStorage Call

func (c *CallForStorage) EncodeRLP(w io.Writer) error {
	rl := rlpCall{c.Type, c.From, c.To, c.Gas, c.Value, c.Data, c.Success}
	return rlp.Encode(w, &rl)
}

func (c *CallForStorage) DecodeRLP(s *rlp.Stream) error {
	blob, err := s.Raw()
	if err != nil {
		return err
	}
	var dec rlpCall
	err = rlp.DecodeBytes(blob, &dec)
	if err == nil {
		*c = CallForStorage{
			Type:    dec.Type,
			From:    dec.From,
			To:      dec.To,
			Gas:     dec.Gas,
			Value:   dec.Value,
			Data:    dec.Data,
			Success: dec.Success,
		}
	}
	return err
}
