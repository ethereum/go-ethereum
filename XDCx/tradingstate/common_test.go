package tradingstate

import (
	"reflect"
	"testing"
)

// Testing scenario:
// encode originalTxMatchesBatch -> byteData
// decode byteData -> txMatchesBatch
// compare originalTxMatchesBatch and txMatchesBatch
func TestTxMatchesBatch(t *testing.T) {
	originalTxMatchesBatch := []TxDataMatch{
		{
			Order: []byte("order1"),
		},
		{
			Order: []byte("order2"),
		},
		{
			Order: []byte("order3"),
		},
	}

	encodedData, err := EncodeTxMatchesBatch(TxMatchBatch{
		Data: originalTxMatchesBatch,
	})
	if err != nil {
		t.Error("Failed to encode", err.Error())
	}

	txMatchesBatch, err := DecodeTxMatchesBatch(encodedData)
	if err != nil {
		t.Error("Failed to decode", err.Error())
	}

	eq := reflect.DeepEqual(originalTxMatchesBatch, txMatchesBatch.Data)
	if eq {
		t.Log("Awesome, encode and decode txMatchesBatch are correct")
	} else {
		t.Error("txMatchesBatch is different from originalTxMatchesBatch", "txMatchesBatch", txMatchesBatch, "originalTxMatchesBatch", originalTxMatchesBatch)
	}
}
