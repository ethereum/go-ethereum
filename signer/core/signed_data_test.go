// Copyright 2018 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.
//
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var typesStandard = Types{
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
        "chainId": 1,
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

var domainStandard = TypedDataDomain{
	"Ether Mail",
	"1",
	big.NewInt(1),
	"0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC",
	"",
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

var typedData = TypedData{
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
	signature, err := api.SignData(context.Background(), TextPlain.Mime, a, hexutil.Encode([]byte("EHLO world")))
	if signature != nil {
		t.Errorf("Expected nil-data, got %x", signature)
	}
	if err != keystore.ErrDecrypt {
		t.Errorf("Expected ErrLocked! '%v'", err)
	}
	control.approveCh <- "No way"
	signature, err = api.SignData(context.Background(), TextPlain.Mime, a, hexutil.Encode([]byte("EHLO world")))
	if signature != nil {
		t.Errorf("Expected nil-data, got %x", signature)
	}
	if err != ErrRequestDenied {
		t.Errorf("Expected ErrRequestDenied! '%v'", err)
	}
	// text/plain
	control.approveCh <- "Y"
	control.inputCh <- "a_long_password"
	signature, err = api.SignData(context.Background(), TextPlain.Mime, a, hexutil.Encode([]byte("EHLO world")))
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

func TestMalformedDomainkeys(t *testing.T) {
	// Verifies that malformed domain keys are properly caught:
	//{
	//	"name": "Ether Mail",
	//	"version": "1",
	//	"chainId": 1,
	//	"vxerifyingContract": "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"
	//}
	jsonTypedData := `
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
        "chainId": 1,
        "vxerifyingContract": "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"
      },
      "message": {
        "from": {
          "name": "Cow",
          "wallet": "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826"
        },
        "to": {
          "name": "Bob",
          "wallet": "0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB"
        },
        "contents": "Hello, Bob!"
      }
    }
`
	var malformedDomainTypedData TypedData
	err := json.Unmarshal([]byte(jsonTypedData), &malformedDomainTypedData)
	if err != nil {
		t.Fatalf("unmarshalling failed '%v'", err)
	}
	_, err = malformedDomainTypedData.HashStruct("EIP712Domain", malformedDomainTypedData.Domain.Map())
	if err == nil || err.Error() != "provided data '<nil>' doesn't match type 'address'" {
		t.Errorf("Expected `provided data '<nil>' doesn't match type 'address'`, got '%v'", err)
	}
}

func TestMalformedTypesAndExtradata(t *testing.T) {
	// Verifies several quirks
	// 1. Using dynamic types and only validating the prefix:
	//{
	//	"name": "chainId",
	//	"type": "uint256 ... and now for something completely different"
	//}
	// 2. Extra data in message:
	//{
	//  "blahonga": "zonk bonk"
	//}
	jsonTypedData := `
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
            "type": "uint256 ... and now for something completely different"
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
        "chainId": 1,
        "verifyingContract": "0xCCCcccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"
      },
      "message": {
        "from": {
          "name": "Cow",
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
	var malformedTypedData TypedData
	err := json.Unmarshal([]byte(jsonTypedData), &malformedTypedData)
	if err != nil {
		t.Fatalf("unmarshalling failed '%v'", err)
	}

	malformedTypedData.Types["EIP712Domain"][2].Type = "uint256"
	malformedTypedData.Message["blahonga"] = "zonk bonk"
	_, err = malformedTypedData.HashStruct(malformedTypedData.PrimaryType, malformedTypedData.Message)
	if err == nil || err.Error() != "there is extra data provided in the message" {
		t.Errorf("Expected `there is extra data provided in the message`, got '%v'", err)
	}
}

func TestTypeMismatch(t *testing.T) {
	// Verifies that:
	// 1. Mismatches between the given type and data, i.e. `Person` and
	// 		the data item is a string, are properly caught:
	//{
	//	"name": "contents",
	//	"type": "Person"
	//},
	//{
	//	"contents": "Hello, Bob!" <-- string not "Person"
	//}
	// 2. Nonexistent types are properly caught:
	//{
	//	"name": "contents",
	//	"type": "Blahonga"
	//}
	jsonTypedData := `
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
            "type": "Person"
          }
        ]
      },
      "primaryType": "Mail",
      "domain": {
        "name": "Ether Mail",
        "version": "1",
        "chainId": 1,
        "verifyingContract": "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"
      },
      "message": {
        "from": {
          "name": "Cow",
          "wallet": "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826"
        },
        "to": {
          "name": "Bob",
          "wallet": "0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB"
        },
        "contents": "Hello, Bob!"
      }
    }
`
	var mismatchTypedData TypedData
	err := json.Unmarshal([]byte(jsonTypedData), &mismatchTypedData)
	if err != nil {
		t.Fatalf("unmarshalling failed '%v'", err)
	}
	_, err = mismatchTypedData.HashStruct(mismatchTypedData.PrimaryType, mismatchTypedData.Message)
	if err.Error() != "provided data 'Hello, Bob!' doesn't match type 'Person'" {
		t.Errorf("Expected `provided data 'Hello, Bob!' doesn't match type 'Person'`, got '%v'", err)
	}

	mismatchTypedData.Types["Mail"][2].Type = "Blahonga"
	_, err = mismatchTypedData.HashStruct(mismatchTypedData.PrimaryType, mismatchTypedData.Message)
	if err == nil || err.Error() != "reference type 'Blahonga' is undefined" {
		t.Fatalf("Expected `reference type 'Blahonga' is undefined`, got '%v'", err)
	}
}

func TestTypeOverflow(t *testing.T) {
	// Verifies data that doesn't fit into it:
	//{
	//	"test": 65536 <-- test defined as uint8
	//}
	var overflowTypedData TypedData
	err := json.Unmarshal([]byte(jsonTypedData), &overflowTypedData)
	if err != nil {
		t.Fatalf("unmarshalling failed '%v'", err)
	}
	// Set test to something outside uint8
	(overflowTypedData.Message["from"]).(map[string]interface{})["test"] = big.NewInt(65536)

	_, err = overflowTypedData.HashStruct(overflowTypedData.PrimaryType, overflowTypedData.Message)
	if err == nil || err.Error() != "integer larger than 'uint8'" {
		t.Fatalf("Expected `integer larger than 'uint8'`, got '%v'", err)
	}

	(overflowTypedData.Message["from"]).(map[string]interface{})["test"] = big.NewInt(3)
	(overflowTypedData.Message["to"]).(map[string]interface{})["test"] = big.NewInt(4)

	_, err = overflowTypedData.HashStruct(overflowTypedData.PrimaryType, overflowTypedData.Message)
	if err != nil {
		t.Fatalf("Expected no err, got '%v'", err)
	}
}

func TestArray(t *testing.T) {
	// Makes sure that arrays work fine
	//{
	//	"type": "address[]"
	//},
	//{
	//	"type": "string[]"
	//},
	//{
	//	"type": "uint16[]",
	//}

	jsonTypedData := `
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
	        "Foo": [
	          {
	            "name": "bar",
	            "type": "address[]"
	          }
	        ]
	      },
	      "primaryType": "Foo",
	      "domain": {
	        "name": "Lorem",
	        "version": "1",
	        "chainId": 1,
	        "verifyingContract": "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"
	      },
	      "message": {
	        "bar": [
	        	"0x0000000000000000000000000000000000000001",
	        	"0x0000000000000000000000000000000000000002",
	        	"0x0000000000000000000000000000000000000003"
        	]
	      }
	    }
	`
	var arrayTypedData TypedData
	err := json.Unmarshal([]byte(jsonTypedData), &arrayTypedData)
	if err != nil {
		t.Fatalf("unmarshalling failed '%v'", err)
	}
	_, err = arrayTypedData.HashStruct(arrayTypedData.PrimaryType, arrayTypedData.Message)
	if err != nil {
		t.Fatalf("Expected no err, got '%v'", err)
	}

	// Change array to string
	arrayTypedData.Types["Foo"][0].Type = "string[]"
	arrayTypedData.Message["bar"] = []interface{}{
		"lorem",
		"ipsum",
		"dolores",
	}
	_, err = arrayTypedData.HashStruct(arrayTypedData.PrimaryType, arrayTypedData.Message)
	if err != nil {
		t.Fatalf("Expected no err, got '%v'", err)
	}

	// Change array to uint
	arrayTypedData.Types["Foo"][0].Type = "uint[]"
	arrayTypedData.Message["bar"] = []interface{}{
		big.NewInt(1955),
		big.NewInt(108),
		big.NewInt(44010),
	}
	_, err = arrayTypedData.HashStruct(arrayTypedData.PrimaryType, arrayTypedData.Message)
	if err != nil {
		t.Fatalf("Expected no err, got '%v'", err)
	}

	// Should not work with fixed-size arrays
	arrayTypedData.Types["Foo"][0].Type = "uint[3]"
	_, err = arrayTypedData.HashStruct(arrayTypedData.PrimaryType, arrayTypedData.Message)
	if err == nil || err.Error() != "unknown type 'uint[3]'" {
		t.Fatalf("Expected `unknown type 'uint[3]'`, got '%v'", err)
	}
}

func TestCustomTypeAsArray(t *testing.T) {
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
			"name": "wallet",
            "type": "address"
          }
        ],
        "Person[]": [
          {
			"name": "baz",
            "type": "string"
          }
		],
        "Mail": [
          {
			"name": "from",
            "type": "Person"
          },
          {
			"name": "to",
            "type": "Person[]"
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
        "chainId": 1,
        "verifyingContract": "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"
      },
      "message": {
        "from": {
			"name": "Cow",
			"wallet": "0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826"
        },
        "to": {"baz": "foo"},
        "contents": "Hello, Bob!"
      }
    }

`
	var malformedTypedData TypedData
	err := json.Unmarshal([]byte(jsonTypedData), &malformedTypedData)
	if err != nil {
		t.Fatalf("unmarshalling failed '%v'", err)
	}
	_, err = malformedTypedData.HashStruct("EIP712Domain", malformedTypedData.Domain.Map())
	if err != nil {
		t.Errorf("Expected no error, got '%v'", err)
	}
}

func TestFormatter(t *testing.T) {

	var d TypedData
	err := json.Unmarshal([]byte(jsonTypedData), &d)
	if err != nil {
		t.Fatalf("unmarshalling failed '%v'", err)
	}
	formatted := d.Format()
	for _, item := range formatted {
		fmt.Printf("'%v'\n", item.Pprint(0))
	}

	j, _ := json.Marshal(formatted)
	fmt.Printf("'%v'\n", string(j))
}
