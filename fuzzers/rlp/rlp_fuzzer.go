package rlp

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func decodeEncode(input []byte, val interface{}, i int) {
	if err := rlp.DecodeBytes(input, val); err == nil {
		output, err := rlp.EncodeToBytes(val)
		if err != nil {
			panic(err)
		}
		if bytes.Equal(input, output) {
			panic(fmt.Sprintf("case %d: encode-decode is not equal", i))
		}
	}
}

func Fuzz(input []byte) int {

	var i int
	{
		if len(input) > 0 {
			rlp.Split(input)
		}
	}
	{
		if len(input) > 0 {
			if elems, _, err := rlp.SplitList(input); err == nil {
				rlp.CountValues(elems)
			}
		}
	}

	{
		rlp.NewStream(bytes.NewReader(input), 0).Decode(new(interface{}))
	}

	{
		decodeEncode(input, new(interface{}), i)
		i++
	}
	{
		var v struct {
			Int    uint
			String string
			Bytes  []byte
		}
		decodeEncode(input, &v, i)
		i++
	}

	{
		type Types struct {
			Bool  bool
			Raw   rlp.RawValue
			Slice []*Types
			Iface []interface{}
		}
		var v Types
		decodeEncode(input, &v, i)
		i++
	}
	{
		type AllTypes struct {
			Int    uint
			String string
			Bytes  []byte
			Bool   bool
			Raw    rlp.RawValue
			Slice  []*AllTypes
			Array  [3]*AllTypes
			Iface  []interface{}
		}
		var v AllTypes
		decodeEncode(input, &v, i)
		i++
	}
	{
		decodeEncode(input, [10]byte{}, i)
		i++
	}
	{
		var v struct {
			Byte [10]byte
			Rool [10]bool
		}
		decodeEncode(input, &v, i)
		i++
	}
	{
		var h types.Header
		decodeEncode(input, &h, i)
		i++
		var b types.Block
		decodeEncode(input, &b, i)
		i++
		var t types.Transaction
		decodeEncode(input, &t, i)
		i++
		var txs types.Transactions
		decodeEncode(input, &txs, i)
		i++
		var rs types.Receipts
		decodeEncode(input, &rs, i)
		i++
	}

	return 0
}
