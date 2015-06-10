package tests

import (
	"path/filepath"
	"testing"
)

func TestTransactions(t *testing.T) {
	err := RunTransactionTests(filepath.Join(transactionTestDir, "ttTransactionTest.json"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestWrongRLPTransactions(t *testing.T) {
	err := RunTransactionTests(filepath.Join(transactionTestDir, "ttWrongRLPTransaction.json"))
	if err != nil {
		t.Fatal(err)
	}
}

func Test10MBtx(t *testing.T) {
	err := RunTransactionTests(filepath.Join(transactionTestDir, "tt10mbDataField.json"))
	if err != nil {
		t.Fatal(err)
	}
}
