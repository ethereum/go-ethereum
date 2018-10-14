package core

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
)

type TypedData struct {
	Types     		map[string] interface{}       	`json:"types"`
	PrimaryType 	string                			`json:"primaryType"`
	Domain 			EIP712Domain       				`json:"domain"`
	Message 		map[string]    interface{} 		`json:"message"`
}

type EIP712Domain struct {
	Name 				string           	`json:"name"`
	Version 			string           	`json:"version"`
	ChainId 			big.Int          	`json:"chainId"`
	VerifyingContract 	common.Address 		`json:"verifyingContract"`
	Salt 				hexutil.Bytes  		`json:"salt"`
}

// Typed data according to EIP712
//
// hash = keccak256("\x19${byteVersion}${domainSeparator}${hashStruct(message)}")
func (api *SignerAPI) SignTypedData(ctx context.Context, addr common.MixedcaseAddress, data TypedData) (hexutil.Bytes, error) {
	fmt.Println("addr", addr)
	//fmt.Println("data", data)
	fmt.Println("data.Domain", data.Domain)
	return common.Hex2Bytes("0xdeadbeef"), nil
}
