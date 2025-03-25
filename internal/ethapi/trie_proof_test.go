package ethapi

import (
	"bytes"
	"errors"
	"math/big"
	"reflect"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/common/hexutil"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/rlp"
	"github.com/XinFinOrg/XDPoSChain/trie"
)

// implement interface only for testing verifyProof
func (n *proofPairList) Has(key []byte) (bool, error) {
	key_hex := hexutil.Encode(key)
	for _, k := range n.keys {
		if k == key_hex {
			return true, nil
		}
	}
	return false, nil
}

func (n *proofPairList) Get(key []byte) ([]byte, error) {
	key_hex := hexutil.Encode(key)
	for i, k := range n.keys {
		if k == key_hex {
			b, err := hexutil.Decode(n.values[i])
			if err != nil {
				return nil, err
			}
			return b, nil
		}
	}
	return nil, errors.New("key not found")
}

func TestTransactionProof(t *testing.T) {
	to1 := common.HexToAddress("0x01")
	to2 := common.HexToAddress("0x02")
	t1 := types.NewTransaction(1, to1, big.NewInt(1), 1, big.NewInt(1), []byte{})
	t2 := types.NewTransaction(2, to2, big.NewInt(2), 2, big.NewInt(2), []byte{})
	t3 := types.NewTransaction(3, to2, big.NewInt(3), 3, big.NewInt(3), []byte{})
	t4 := types.NewTransaction(4, to1, big.NewInt(4), 4, big.NewInt(4), []byte{})
	transactions := types.Transactions([]*types.Transaction{t1, t2, t3, t4})
	tr := deriveTrie(transactions)
	// for verifying the proof
	root := types.DeriveSha(transactions, trie.NewStackTrie(nil))
	for i := 0; i < transactions.Len(); i++ {
		var proof proofPairList
		keybuf := new(bytes.Buffer)
		rlp.Encode(keybuf, uint(i))
		if err := tr.Prove(keybuf.Bytes(), 0, &proof); err != nil {
			t.Fatal("Prove err:", err)
		}
		// verify the proof
		value, err := trie.VerifyProof(root, keybuf.Bytes(), &proof)
		if err != nil {
			t.Fatal("verify proof error")
		}
		encodedTransaction, err := rlp.EncodeToBytes(transactions[i])
		if err != nil {
			t.Fatal("encode receipt error")
		}
		if !reflect.DeepEqual(encodedTransaction, value) {
			t.Fatal("verify does not return the receipt we want")
		}
	}
}

func TestReceiptProof(t *testing.T) {
	root1 := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	root2 := []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	r1 := types.NewReceipt(root1, false, 1)
	r2 := types.NewReceipt(root2, true, 2)
	r3 := types.NewReceipt(root2, false, 3)
	r4 := types.NewReceipt(root1, true, 4)
	receipts := types.Receipts([]*types.Receipt{r1, r2, r3, r4})
	tr := deriveTrie(receipts)
	// for verifying the proof
	root := types.DeriveSha(receipts, trie.NewStackTrie(nil))
	for i := 0; i < receipts.Len(); i++ {
		var proof proofPairList
		keybuf := new(bytes.Buffer)
		rlp.Encode(keybuf, uint(i))
		if err := tr.Prove(keybuf.Bytes(), 0, &proof); err != nil {
			t.Fatal("Prove err:", err)
		}
		// verify the proof
		value, err := trie.VerifyProof(root, keybuf.Bytes(), &proof)
		if err != nil {
			t.Fatal("verify proof error")
		}
		encodedReceipt, err := rlp.EncodeToBytes(receipts[i])
		if err != nil {
			t.Fatal("encode receipt error")
		}
		if !reflect.DeepEqual(encodedReceipt, value) {
			t.Fatal("verify does not return the receipt we want")
		}
	}
}
