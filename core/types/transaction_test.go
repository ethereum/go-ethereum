package types

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// The values in those tests are from the Transaction Tests
// at github.com/ethereum/tests.

var (
	emptyTx = NewTransactionMessage(
		common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"),
		big.NewInt(0), big.NewInt(0), big.NewInt(0),
		nil,
	)

	rightvrsTx = &Transaction{
		Recipient:    common.HexToAddress("b94f5374fce5edbc8e2a8697c15331677e6ebf0b"),
		AccountNonce: 3,
		Price:        big.NewInt(1),
		GasLimit:     big.NewInt(2000),
		Amount:       big.NewInt(10),
		Payload:      common.FromHex("5544"),
		V:            28,
		R:            common.FromHex("98ff921201554726367d2be8c804a7ff89ccf285ebc57dff8ae4c44b9c19ac4a"),
		S:            common.FromHex("8887321be575c8095f789dd4c743dfe42c1820f9231f98a962b210e3ac2452a3"),
	}
)

func TestTransactionHash(t *testing.T) {
	// "EmptyTransaction"
	if emptyTx.Hash() != common.HexToHash("c775b99e7ad12f50d819fcd602390467e28141316969f4b57f0626f74fe3b386") {
		t.Errorf("empty transaction hash mismatch, got %x", emptyTx.Hash())
	}

	// "RightVRSTest"
	if rightvrsTx.Hash() != common.HexToHash("fe7a79529ed5f7c3375d06b26b186a8644e0e16c373d7a12be41c62d6042b77a") {
		t.Errorf("RightVRS transaction hash mismatch, got %x", rightvrsTx.Hash())
	}
}

func TestTransactionEncode(t *testing.T) {
	// "RightVRSTest"
	txb, err := rlp.EncodeToBytes(rightvrsTx)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	should := common.FromHex("f86103018207d094b94f5374fce5edbc8e2a8697c15331677e6ebf0b0a8255441ca098ff921201554726367d2be8c804a7ff89ccf285ebc57dff8ae4c44b9c19ac4aa08887321be575c8095f789dd4c743dfe42c1820f9231f98a962b210e3ac2452a3")
	if !bytes.Equal(txb, should) {
		t.Errorf("encoded RLP mismatch, got %x", txb)
	}
}
