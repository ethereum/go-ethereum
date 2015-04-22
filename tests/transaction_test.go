package tests

import (
	"testing"
)

func TestTransactions(t *testing.T) {
	notWorking := make(map[string]bool, 100)

	// TODO: all these tests should work! remove them from the array when they work
	snafus := []string{
		"TransactionWithHihghNonce", // fails due to testing upper bound of 256 bit nonce
		"TransactionWithSvalueHigh", // fails due to C++ wrong ECDSA r,s ranges. see https://github.com/ethereum/yellowpaper/pull/112
		"TransactionWithSvalue0",    // Invalid, 0 < s according to YP eq 205. probably typo/copy-paste error
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

//Not working until it's fields are in HEX
func Test10MBtx(t *testing.T) {
	t.Skip("Skipped in lieu of HEX encoding fix in this file.")
	notWorking := make(map[string]bool, 100)
	var err error
	err = RunTransactionTests("./files/TransactionTests/tt10mbDataField.json",
		notWorking)
	if err != nil {
		t.Fatal(err)
	}
}
