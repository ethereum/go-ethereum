package types

import (
	"bytes"
	"encoding/hex"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func genTxs(num uint64) (Transactions, error) {
	key, err := crypto.HexToECDSA("deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	if err != nil {
		return nil, err
	}
	var addr = crypto.PubkeyToAddress(key.PublicKey)
	newTx := func(i uint64) (*Transaction, error) {
		signer := NewEIP155Signer(big.NewInt(18))
		tx, err := SignTx(NewTransaction(i, addr, new(big.Int), 0, new(big.Int).SetUint64(10000000), nil), signer, key)
		return tx, err
	}
	var txs Transactions
	for i := uint64(0); i < num; i++ {
		tx, err := newTx(i)
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

func TestDeriveSha(t *testing.T) {
	txs, err := genTxs(200)
	if err != nil {
		t.Fatal(err)
	}
	got := DeriveSha(txs)
	exp, _ := hex.DecodeString("d325882c79784f5751b8871ac628c2841154d5b739c5ad811bf0de74d927ad63")
	if !bytes.Equal(got[:], exp) {
		t.Errorf("got %x exp %x", got, exp)
	}
}

// BenchmarkDeriveSha-6   	    1746	    655499 ns/op	  307236 B/op	    4276 allocs/op
func BenchmarkDeriveSha(b *testing.B) {
	txs, err := genTxs(200)
	if err != nil {
		b.Fatal(err)
	}
	var got common.Hash
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		got = DeriveSha(txs)
	}

	exp, _ := hex.DecodeString("d325882c79784f5751b8871ac628c2841154d5b739c5ad811bf0de74d927ad63")
	if !bytes.Equal(got[:], exp) {
		b.Errorf("got %x exp %x", got, exp)
	}

}
