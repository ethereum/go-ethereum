package core

import (
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"testing"
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
	"Val": {
		{
			Name: "field",
			Type: "bytes[][]",
		},
	},
}

var messageStandard = map[string]interface{}{
	"field": [][][]byte{{{1}, {2}}, {{3}, {4}}},
}

var domainStandard = apitypes.TypedDataDomain{
	Name:              "Ether Mail",
	Version:           "1",
	ChainId:           math.NewHexOrDecimal256(1),
	VerifyingContract: "0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC",
	Salt:              "",
}

var typedData = apitypes.TypedData{
	Types:       typesStandard,
	PrimaryType: "Val",
	Domain:      domainStandard,
	Message:     messageStandard,
}

func TestEncodeDataRecursiveBytes(t *testing.T) {
	_, err := typedData.EncodeData(typedData.PrimaryType, typedData.Message, 0)
	if err != nil {
		t.Fatalf("got err %v", err)
	}
}
