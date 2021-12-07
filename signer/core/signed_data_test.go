// Copyright 2019 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

var typesStandard = apitypes.Types{
	"EIP712Domain": {
		{
			Name: "name",
			Type: "string",
		},
		{
			Name: "version",
			Type: "string",
		},
		{
			Name: "chainId",
			Type: "uint256",
		},
		{
			Name: "verifyingContract",
			Type: "address",
		},
	},
	"Person": {
		{
			Name: "name",
			Type: "string",
		},
		{
			Name: "wallet",
			Type: "address",
		},
	},
	"Mail": {
		{
			Name: "from",
			Type: "Person",
		},
		{
			Name: "to",
			Type: "Person",
		},
		{
			Name: "contents",
			Type: "string",
		},
	},
}

var jsonTypedData = `
    {
      "types": {
        "EIP712Domain": [
          {
            "name": "name",
            "type": "string"
          },
          {
            "name": "version",
            "type": "string"
          },
          {
            "name": "chainId",
            "type": "uint256"
          },
          {
            "name": "verifyingContract",
            "type": "address"
          }
        ],
        "Person": [
          {
            "name": "name",
            "type": "string"
          },
          {
            "name": "test",
            "type": "uint8"
          },
          {
            "name": "wallet",
            "type": "address"
          }
        ],
        "Mail": [
          {
            "name": "from",
            "type": "Person"
          },
          {
            "name": "to",
            "type": "Person"
          },
          {
            "name": "contents",
            "type": "string"
          }
        ]
      },
      "primaryType": "Mail",
      "domain": {
        "name": "Ether Mail",
        "version": "1",
        "chainId": "1",
        "verifyingContract": "0xCCCcccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"
      },
      "message": {
        "from": {
          "name": "Cow",
		  "test": 3,
          "wallet": "0xcD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826"
        },
        "to": {
          "name": "Bob",
          "wallet": "0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB"
        },
        "contents": "Hello, Bob!"
      }
    }
`

const primaryType = "Mail"

var domainStandard = apitypes.TypedDataDomain{
	Name:              "Ether Mail",
	Version:           "1",
	ChainId:           math.NewHexOrDecimal256(1),
	VerifyingContract: "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC",
	Salt:              "",
}

var messageStandard = map[string]interface{}{
	"from": map[string]interface{}{
		"name":   "Cow",
		"wallet": "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826",
	},
	"to": map[string]interface{}{
		"name":   "Bob",
		"wallet": "0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB",
	},
	"contents": "Hello, Bob!",
}

var typedData = apitypes.TypedData{
	Types:       typesStandard,
	PrimaryType: primaryType,
	Domain:      domainStandard,
	Message:     messageStandard,
}

func TestSignData(t *testing.T) {
	api, control := setup(t)
	//Create two accounts
	createAccount(control, api, t)
	createAccount(control, api, t)
	control.approveCh <- "1"
	list, err := api.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	a := common.NewMixedcaseAddress(list[0])

	control.approveCh <- "Y"
	control.inputCh <- "wrongpassword"
	signature, err := api.SignData(context.Background(), apitypes.TextPlain.Mime, a, hexutil.Encode([]byte("EHLO world")))
	if signature != nil {
		t.Errorf("Expected nil-data, got %x", signature)
	}
	if err != keystore.ErrDecrypt {
		t.Errorf("Expected ErrLocked! '%v'", err)
	}
	control.approveCh <- "No way"
	signature, err = api.SignData(context.Background(), apitypes.TextPlain.Mime, a, hexutil.Encode([]byte("EHLO world")))
	if signature != nil {
		t.Errorf("Expected nil-data, got %x", signature)
	}
	if err != core.ErrRequestDenied {
		t.Errorf("Expected ErrRequestDenied! '%v'", err)
	}
	// text/plain
	control.approveCh <- "Y"
	control.inputCh <- "a_long_password"
	signature, err = api.SignData(context.Background(), apitypes.TextPlain.Mime, a, hexutil.Encode([]byte("EHLO world")))
	if err != nil {
		t.Fatal(err)
	}
	if signature == nil || len(signature) != 65 {
		t.Errorf("Expected 65 byte signature (got %d bytes)", len(signature))
	}
	// data/typed
	control.approveCh <- "Y"
	control.inputCh <- "a_long_password"
	signature, err = api.SignTypedData(context.Background(), a, typedData)
	if err != nil {
		t.Fatal(err)
	}
	if signature == nil || len(signature) != 65 {
		t.Errorf("Expected 65 byte signature (got %d bytes)", len(signature))
	}
}

func TestDomainChainId(t *testing.T) {
	withoutChainID := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
			},
		},
		Domain: apitypes.TypedDataDomain{
			Name: "test",
		},
	}

	if _, ok := withoutChainID.Domain.Map()["chainId"]; ok {
		t.Errorf("Expected the chainId key to not be present in the domain map")
	}
	// should encode successfully
	if _, err := withoutChainID.HashStruct("EIP712Domain", withoutChainID.Domain.Map()); err != nil {
		t.Errorf("Expected the typedData to encode the domain successfully, got %v", err)
	}
	withChainID := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "chainId", Type: "uint256"},
			},
		},
		Domain: apitypes.TypedDataDomain{
			Name:    "test",
			ChainId: math.NewHexOrDecimal256(1),
		},
	}

	if _, ok := withChainID.Domain.Map()["chainId"]; !ok {
		t.Errorf("Expected the chainId key be present in the domain map")
	}
	// should encode successfully
	if _, err := withChainID.HashStruct("EIP712Domain", withChainID.Domain.Map()); err != nil {
		t.Errorf("Expected the typedData to encode the domain successfully, got %v", err)
	}
}

func TestHashStruct(t *testing.T) {
	hash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		t.Fatal(err)
	}
	mainHash := fmt.Sprintf("0x%s", common.Bytes2Hex(hash))
	if mainHash != "0xc52c0ee5d84264471806290a3f2c4cecfc5490626bf912d01f240d7a274b371e" {
		t.Errorf("Expected different hashStruct result (got %s)", mainHash)
	}

	hash, err = typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		t.Error(err)
	}
	domainHash := fmt.Sprintf("0x%s", common.Bytes2Hex(hash))
	if domainHash != "0xf2cee375fa42b42143804025fc449deafd50cc031ca257e0b194a650a912090f" {
		t.Errorf("Expected different domain hashStruct result (got %s)", domainHash)
	}
}

func TestEncodeType(t *testing.T) {
	domainTypeEncoding := string(typedData.EncodeType("EIP712Domain"))
	if domainTypeEncoding != "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)" {
		t.Errorf("Expected different encodeType result (got %s)", domainTypeEncoding)
	}

	mailTypeEncoding := string(typedData.EncodeType(typedData.PrimaryType))
	if mailTypeEncoding != "Mail(Person from,Person to,string contents)Person(string name,address wallet)" {
		t.Errorf("Expected different encodeType result (got %s)", mailTypeEncoding)
	}
}

func TestTypeHash(t *testing.T) {
	mailTypeHash := fmt.Sprintf("0x%s", common.Bytes2Hex(typedData.TypeHash(typedData.PrimaryType)))
	if mailTypeHash != "0xa0cedeb2dc280ba39b857546d74f5549c3a1d7bdc2dd96bf881f76108e23dac2" {
		t.Errorf("Expected different typeHash result (got %s)", mailTypeHash)
	}
}

func TestEncodeData(t *testing.T) {
	hash, err := typedData.EncodeData(typedData.PrimaryType, typedData.Message, 0)
	if err != nil {
		t.Fatal(err)
	}
	dataEncoding := fmt.Sprintf("0x%s", common.Bytes2Hex(hash))
	if dataEncoding != "0xa0cedeb2dc280ba39b857546d74f5549c3a1d7bdc2dd96bf881f76108e23dac2fc71e5fa27ff56c350aa531bc129ebdf613b772b6604664f5d8dbe21b85eb0c8cd54f074a4af31b4411ff6a60c9719dbd559c221c8ac3492d9d872b041d703d1b5aadf3154a261abdd9086fc627b61efca26ae5702701d05cd2305f7c52a2fc8" {
		t.Errorf("Expected different encodeData result (got %s)", dataEncoding)
	}
}

func TestFormatter(t *testing.T) {
	var d apitypes.TypedData
	err := json.Unmarshal([]byte(jsonTypedData), &d)
	if err != nil {
		t.Fatalf("unmarshalling failed '%v'", err)
	}
	formatted, _ := d.Format()
	for _, item := range formatted {
		t.Logf("'%v'\n", item.Pprint(0))
	}

	j, _ := json.Marshal(formatted)
	t.Logf("'%v'\n", string(j))
}

func sign(typedData apitypes.TypedData) ([]byte, []byte, error) {
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return nil, nil, err
	}
	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return nil, nil, err
	}
	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	sighash := crypto.Keccak256(rawData)
	return typedDataHash, sighash, nil
}

func TestJsonFiles(t *testing.T) {
	testfiles, err := ioutil.ReadDir("testdata/")
	if err != nil {
		t.Fatalf("failed reading files: %v", err)
	}
	for i, fInfo := range testfiles {
		if !strings.HasSuffix(fInfo.Name(), "json") {
			continue
		}
		expectedFailure := strings.HasPrefix(fInfo.Name(), "expfail")
		data, err := ioutil.ReadFile(path.Join("testdata", fInfo.Name()))
		if err != nil {
			t.Errorf("Failed to read file %v: %v", fInfo.Name(), err)
			continue
		}
		var typedData apitypes.TypedData
		err = json.Unmarshal(data, &typedData)
		if err != nil {
			t.Errorf("Test %d, file %v, json unmarshalling failed: %v", i, fInfo.Name(), err)
			continue
		}
		_, _, err = sign(typedData)
		t.Logf("Error %v\n", err)
		if err != nil && !expectedFailure {
			t.Errorf("Test %d failed, file %v: %v", i, fInfo.Name(), err)
		}
		if expectedFailure && err == nil {
			t.Errorf("Test %d succeeded (expected failure), file %v: %v", i, fInfo.Name(), err)
		}
	}
}

// TestFuzzerFiles tests some files that have been found by fuzzing to cause
// crashes or hangs.
func TestFuzzerFiles(t *testing.T) {
	corpusdir := path.Join("testdata", "fuzzing")
	testfiles, err := ioutil.ReadDir(corpusdir)
	if err != nil {
		t.Fatalf("failed reading files: %v", err)
	}
	verbose := false
	for i, fInfo := range testfiles {
		data, err := ioutil.ReadFile(path.Join(corpusdir, fInfo.Name()))
		if err != nil {
			t.Errorf("Failed to read file %v: %v", fInfo.Name(), err)
			continue
		}
		var typedData apitypes.TypedData
		err = json.Unmarshal(data, &typedData)
		if err != nil {
			t.Errorf("Test %d, file %v, json unmarshalling failed: %v", i, fInfo.Name(), err)
			continue
		}
		_, err = typedData.EncodeData("EIP712Domain", typedData.Domain.Map(), 1)
		if verbose && err != nil {
			t.Logf("%d, EncodeData[1] err: %v\n", i, err)
		}
		_, err = typedData.EncodeData(typedData.PrimaryType, typedData.Message, 1)
		if verbose && err != nil {
			t.Logf("%d, EncodeData[2] err: %v\n", i, err)
		}
		typedData.Format()
	}
}

var gnosisTypedData = `
{
	"types": {
		"EIP712Domain": [
			{ "type": "address", "name": "verifyingContract" }
		],
		"SafeTx": [
			{ "type": "address", "name": "to" },
			{ "type": "uint256", "name": "value" },
			{ "type": "bytes", "name": "data" },
			{ "type": "uint8", "name": "operation" },
			{ "type": "uint256", "name": "safeTxGas" },
			{ "type": "uint256", "name": "baseGas" },
			{ "type": "uint256", "name": "gasPrice" },
			{ "type": "address", "name": "gasToken" },
			{ "type": "address", "name": "refundReceiver" },
			{ "type": "uint256", "name": "nonce" }
		]
	},
	"domain": {
		"verifyingContract": "0x25a6c4BBd32B2424A9c99aEB0584Ad12045382B3"
	},
	"primaryType": "SafeTx",
	"message": {
		"to": "0x9eE457023bB3De16D51A003a247BaEaD7fce313D",
		"value": "20000000000000000",
		"data": "0x",
		"operation": 0,
		"safeTxGas": 27845,
		"baseGas": 0,
		"gasPrice": "0",
		"gasToken": "0x0000000000000000000000000000000000000000",
		"refundReceiver": "0x0000000000000000000000000000000000000000",
		"nonce": 3
	}
}`

var gnosisTx = `
{
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
    }
`

// TestGnosisTypedData tests the scenario where a user submits a full EIP-712
// struct without using the gnosis-specific endpoint
func TestGnosisTypedData(t *testing.T) {
	var td apitypes.TypedData
	err := json.Unmarshal([]byte(gnosisTypedData), &td)
	if err != nil {
		t.Fatalf("unmarshalling failed '%v'", err)
	}
	_, sighash, err := sign(td)
	if err != nil {
		t.Fatal(err)
	}
	expSigHash := common.FromHex("0x28bae2bd58d894a1d9b69e5e9fde3570c4b98a6fc5499aefb54fb830137e831f")
	if !bytes.Equal(expSigHash, sighash) {
		t.Fatalf("Error, got %x, wanted %x", sighash, expSigHash)
	}
}

// TestGnosisCustomData tests the scenario where a user submits only the gnosis-safe
// specific data, and we fill the TypedData struct on our side
func TestGnosisCustomData(t *testing.T) {
	var tx core.GnosisSafeTx
	err := json.Unmarshal([]byte(gnosisTx), &tx)
	if err != nil {
		t.Fatal(err)
	}
	var td = tx.ToTypedData()
	_, sighash, err := sign(td)
	if err != nil {
		t.Fatal(err)
	}
	expSigHash := common.FromHex("0x28bae2bd58d894a1d9b69e5e9fde3570c4b98a6fc5499aefb54fb830137e831f")
	if !bytes.Equal(expSigHash, sighash) {
		t.Fatalf("Error, got %x, wanted %x", sighash, expSigHash)
	}
}
