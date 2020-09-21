package accounts

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestGnosisSafe(t *testing.T) {
	var txjson = `{
      "safe": "0x25a6c4BBd32B2424A9c99aEB0584Ad12045382B3",
      "to": "0x9eE457023bB3De16D51A003a247BaEaD7fce313D",
      "value": "20000000000000000",
      "data": null,
      "operation": 0,
      "gasToken": "0x0000000000000000000000000000000000000000",
      "safeTxGas": 27845,
      "baseGas": 0,
      "gasPrice": "0",
      "refundReceiver": "0x0000000000000000000000000000000000000000",
      "nonce": 3,
      "executionDate": null,
      "submissionDate": "2020-09-15T21:59:23.815748Z",
      "modified": "2020-09-15T21:59:23.815748Z",
      "blockNumber": null,
      "transactionHash": null,
      "safeTxHash": "0x28bae2bd58d894a1d9b69e5e9fde3570c4b98a6fc5499aefb54fb830137e831f",
      "executor": null,
      "isExecuted": false,
      "isSuccessful": null,
      "ethGasPrice": null,
      "gasUsed": null,
      "fee": null,
      "origin": null,
      "dataDecoded": null,
      "confirmationsRequired": null,
      "confirmations": [
        {
          "owner": "0xAd2e180019FCa9e55CADe76E4487F126Fd08DA34",
          "submissionDate": "2020-09-15T21:59:28.281243Z",
          "transactionHash": null,
          "confirmationType": "CONFIRMATION",
          "signature": "0x5e562065a0cb15d766dac0cd49eb6d196a41183af302c4ecad45f1a81958d7797753f04424a9b0aa1cb0448e4ec8e189540fbcdda7530ef9b9d95dfc2d36cb521b",
          "signatureType": "EOA"
        }
      ],
      "signatures": null
    }`
	var tx GnosisSafeTx
	if err := json.Unmarshal([]byte(txjson), &tx); err != nil {
		t.Fatal(err)
	}
	_, signingHash, err := GnosisSafeSigningHash(&tx)
	if err != nil {
		t.Fatal(err)
	}
	expHash := common.HexToHash("0x28bae2bd58d894a1d9b69e5e9fde3570c4b98a6fc5499aefb54fb830137e831f")
	if expHash != signingHash {
		t.Fatalf("Got %x, exp %x", signingHash, expHash)
	}
}
