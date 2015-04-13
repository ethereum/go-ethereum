package tests

import (
	"testing"
)

func TestTransactions(t *testing.T) {
	notWorking := make(map[string]bool, 100)
	// TODO: all commented out tests should work!

	snafus := []string{
		"EmptyTransaction",
		"TransactionWithHihghNonce",
		"TransactionWithRvalueWrongSize",
		"TransactionWithSvalueHigh",
		"TransactionWithSvalueTooHigh",
		"TransactionWithSvalueWrongSize",
		"ValuesAsDec",
		"ValuesAsHex",
		"libsecp256k1test",
		"unpadedRValue",
	}

	for _, name := range snafus {
		notWorking[name] = true
	}

	var err error
	err = RunTransactionTests("./files/TransactionTests/ttTransactionTest.json",
		notWorking)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWrongRLPTransactions(t *testing.T) {
	notWorking := make(map[string]bool, 100)
	var err error
	err = RunTransactionTests("./files/TransactionTests/ttWrongRLPTransaction.json",
		notWorking)
	if err != nil {
		t.Fatal(err)
	}
}
