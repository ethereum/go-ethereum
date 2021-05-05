/*
* Developed by: Md. Muhaimin Shah Pahalovi
* Generated: 5/5/21
* This file is generated to support Lukso pandora module.
* Purpose:
 */
package core

import (
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"testing"
)

func TestInsertHeaderChainWithPendingHeaders(t *testing.T)  {
	// prepare a dummy bc
	engine := ethash.NewFullFaker()
	_, blockchain, err := newCanonical(engine, 0, false)
	if err != nil {
		t.Fatalf("failed to create pristine chain: %v", err)
	}
	defer blockchain.Stop()

	// prepare header chain
	headers := makeHeaderChain(blockchain.CurrentHeader(), 5, engine, blockchain.db, 0)
	if _, err := blockchain.InsertHeaderChain(headers, 1); err != nil {
		t.Fatalf("insert header chain failed due to %v", err)
	}

	tempHeaders := blockchain.GetTempHeadersSince(headers[0].Hash())
	for index, tHeader := range tempHeaders {
		if tHeader.Hash() != headers[index].Hash() {
			// dumped and received temporary headers are not equal then raise an error
			t.Fatalf("header missmatched. Temp stored header and inserted headers are not same. tempHash %v headerHash %v", tHeader.Hash(), headers[index].Hash())
		}
	}
}