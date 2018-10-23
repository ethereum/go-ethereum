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
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

var typesStandard = EIP712Types{
	"EIP712Domain": {
		{
			"name": "name",
			"type": "string",
		},
		{
			"name": "version",
			"type": "string",
		},
		{
			"name": "chainId",
			"type": "uint256",
		},
		{
			"name": "verifyingContract",
			"type": "address",
		},
	},
	"Person": {
		{
			"name": "name",
			"type": "string",
		},
		{
			"name": "wallet",
			"type": "address",
		},
	},
	"Mail": {
		{
			"name": "from",
			"type": "Person",
		},
		{
			"name": "to",
			"type": "Person",
		},
		{
			"name": "contents",
			"type": "string",
		},
	},
}

const primaryType = "Mail"

var domainStandard = EIP712Domain{
	"Ether Mail",
	"1",
	big.NewInt(1),
	"0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC",
	nil,
}

var dataStandard = map[string]interface{}{
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
	typesStandard,
	primaryType,
	domainStandard,
	dataStandard,
}

func TestSignData(t *testing.T) {
	api, control := setup(t)
	//Create two accounts
	createAccount(control, api, t)
	createAccount(control, api, t)
	control <- "1"
	list, err := api.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	a := common.NewMixedcaseAddress(list[0])

	control <- "Y"
	control <- "wrongpassword"
	signature, err := api.SignData(context.Background(), TextPlain.Mime, a, []byte("EHLO world"))
	if signature != nil {
		t.Errorf("Expected nil-data, got %x", signature)
	}
	if err != keystore.ErrDecrypt {
		t.Errorf("Expected ErrLocked! %v", err)
	}
	control <- "No way"
	signature, err = api.SignData(context.Background(), TextPlain.Mime, a, []byte("EHLO world"))
	if signature != nil {
		t.Errorf("Expected nil-data, got %x", signature)
	}
	if err != ErrRequestDenied {
		t.Errorf("Expected ErrRequestDenied! %v", err)
	}
	// text/plain
	control <- "Y"
	control <- "a_long_password"
	signature, err = api.SignData(context.Background(), TextPlain.Mime, a, []byte("EHLO world"))
	if err != nil {
		t.Fatal(err)
	}
	if signature == nil || len(signature) != 65 {
		t.Errorf("Expected 65 byte signature (got %d bytes)", len(signature))
	}
	// data/typed
	control <- "Y"
	control <- "a_long_password"
	signature, err = api.SignTypedData(context.Background(), a, typedData)
	if err != nil {
		t.Fatal(err)
	}
	if signature == nil || len(signature) != 65 {
		t.Errorf("Expected 65 byte signature (got %d bytes)", len(signature))
	}
	// TODO: test signature r,s,v values
}

func TestHashStruct(t *testing.T) {
	mainHash := fmt.Sprintf("0x%s", common.Bytes2Hex(typedData.HashStruct(typedData.PrimaryType, typedData.Message)))
	if mainHash != "0xc52c0ee5d84264471806290a3f2c4cecfc5490626bf912d01f240d7a274b371e" {
		t.Fatal(fmt.Errorf("hashStruct result %s is incorrect", mainHash))
	}

	domainHash := fmt.Sprintf("0x%s", common.Bytes2Hex(typedData.HashStruct("EIP712Domain", typedData.Domain.Map())))
	if domainHash != "0xf2cee375fa42b42143804025fc449deafd50cc031ca257e0b194a650a912090f" {
		t.Fatal(fmt.Errorf("hashStruct result %s is incorrect", domainHash))
	}
}

func TestEncodeType(t *testing.T) {
	domainTypeEncoding := string(typedData.EncodeType("EIP712Domain"))
	if domainTypeEncoding != "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)" {
		t.Fatal(fmt.Errorf("encodeType result %s is incorrect", domainTypeEncoding))
	}

	mailTypeEncoding := string(typedData.EncodeType(typedData.PrimaryType))
	if mailTypeEncoding != "Mail(Person from,Person to,string contents)Person(string name,address wallet)" {
		t.Fatal(fmt.Errorf("encodeType result %s is incorrect", mailTypeEncoding))
	}
}

func TestTypeHash(t *testing.T) {
	mailTypeHash := fmt.Sprintf("0x%s", common.Bytes2Hex(typedData.TypeHash(typedData.PrimaryType)))
	if mailTypeHash != "0xa0cedeb2dc280ba39b857546d74f5549c3a1d7bdc2dd96bf881f76108e23dac2" {
		t.Fatal(fmt.Errorf("typeHash result %s is incorrect", mailTypeHash))
	}
}

func TestEncodeData(t *testing.T) {
	dataEncoding := fmt.Sprintf("0x%s", common.Bytes2Hex(typedData.EncodeData(typedData.PrimaryType, typedData.Message)))
	if dataEncoding != "0xa0cedeb2dc280ba39b857546d74f5549c3a1d7bdc2dd96bf881f76108e23dac2fc71e5fa27ff56c350aa531bc129ebdf613b772b6604664f5d8dbe21b85eb0c8cd54f074a4af31b4411ff6a60c9719dbd559c221c8ac3492d9d872b041d703d1b5aadf3154a261abdd9086fc627b61efca26ae5702701d05cd2305f7c52a2fc8" {
		t.Fatal(fmt.Errorf("encodeData result %s is incorrect", dataEncoding))
	}
}