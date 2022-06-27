package types

import (
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types/zktrie"
)

func TestAccountMarshalling(t *testing.T) {
	//ensure the hash scheme consistent with designation
	example1 := &StateAccount{
		Nonce:    5,
		Balance:  big.NewInt(0).SetBytes(common.Hex2Bytes("01fffffffffffffffffffffffffffffffffffffffffffffffff9c8672c6bc7b3")),
		CodeHash: common.Hex2Bytes("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"),
	}

	example2 := &StateAccount{
		Nonce:    2,
		Balance:  big.NewInt(0),
		CodeHash: common.Hex2Bytes("cc0a77f6e063b4b62eb7d9ed6f427cf687d8d0071d751850cfe5d136bc60d3ab"),
		Root:     common.HexToHash("22fb59aa5410ed465267023713ab42554c250f394901455a3366e223d5f7d147"),
	}

	for i, example := range []*StateAccount{example1, example2} {
		fields, flag := example.MarshalFields()

		h1, err := zktrie.PreHandlingElems(flag, fields)
		if err != nil {
			t.Fatal(err)
		}

		h2, err := example.Hash()
		if err != nil {
			t.Fatal(err)
		}

		if h1.BigInt().Cmp(h2) != 0 {
			t.Errorf("hash <%d> unmatched, expected [%x], get [%x]", i, h2.Bytes(), h1.Bytes())
		}
	}

}
