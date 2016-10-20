package types

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestChainId(t *testing.T) {
	key, _ := defaultTestKey()

	tx := NewTransaction(0, common.Address{}, new(big.Int), new(big.Int), new(big.Int), nil)
	tx.SetSigner(NewEIP155Signer(big.NewInt(1)))

	var err error
	tx, err = tx.SignECDSA(key)
	if err != nil {
		t.Fatal(err)
	}

	tx.SetSigner(NewEIP155Signer(big.NewInt(2)))
	_, err = tx.From()
	if err != ErrInvalidChainId {
		t.Error("expected error:", ErrInvalidChainId)
	}

	tx.SetSigner(NewEIP155Signer(big.NewInt(1)))
	_, err = tx.From()
	if err != nil {
		t.Error("expected no error")
	}
}
