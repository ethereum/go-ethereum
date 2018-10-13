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
// If the format "\x19\x46" ‖ domainSeparator ‖ hashStruct(message)` is not respected,
// an error is returned
func (api *SignerAPI) SignStructuredData(ctx context.Context, data TypedData) (hexutil.Bytes, error) {
	fmt.Println("data", data)
	fmt.Println("data.PrimaryType", data.PrimaryType)
	return common.Hex2Bytes("0xdeadbeef"), nil
}

// TypedData represents a request to create a new filter.
// Same as ethereum.FilterQuery but with UnmarshalJSON() method.
//type TypedData ethereum.TypedData

// UnmarshalJSON sets *args fields with given data.
//func (args *TypedData) UnmarshalJSON(data []byte) error {
//	type input struct {
//		Hash *common.Hash		`json:"hash"`
//	}
//
//	var raw input
//	if err := json.Unmarshal(data, &raw); err != nil {
//		return err
//	}
//
//	if raw.Hash != nil {
//		args.Hash = raw.Hash
//	}
//
//	return nil
//}