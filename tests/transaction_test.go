package tests

import (
	"path/filepath"
	"testing"
)

var transactionTestDir = filepath.Join(baseDir, "TransactionTests")

func TestTransactions(t *testing.T) {
	notWorking := make(map[string]bool, 100)

	// TODO: all these tests should work! remove them from the array when they work
	snafus := []string{
		"TransactionWithHihghNonce256", // fails due to testing upper bound of 256 bit nonce
	}

	for _, name := range snafus {
		notWorking[name] = true
	}

	var err error
	err = RunTransactionTests(filepath.Join(transactionTestDir, "ttTransactionTest.json"),
		notWorking)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWrongRLPTransactions(t *testing.T) {
	notWorking := make(map[string]bool, 100)
	var err error
	err = RunTransactionTests(filepath.Join(transactionTestDir, "ttWrongRLPTransaction.json"),
		notWorking)
	if err != nil {
		t.Fatal(err)
	}
}

func Test10MBtx(t *testing.T) {
	notWorking := make(map[string]bool, 100)
	var err error
	err = RunTransactionTests(filepath.Join(transactionTestDir, "tt10mbDataField.json"),
		notWorking)
	if err != nil {
		t.Fatal(err)
	}
}
