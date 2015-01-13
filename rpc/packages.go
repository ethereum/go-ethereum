/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/*

For each request type, define the following:

1. RpcRequest "To" method [message.go], which does basic validation and conversion to "Args" type via json.Decoder()
2. json.Decoder() calls "UnmarshalJSON" defined on each "Args" struct
3. EthereumApi method, taking the "Args" type and replying with an interface to be marshalled to JSON

*/
package rpc

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/xeth"
)

type EthereumApi struct {
	pipe *xeth.JSXEth
}

func (p *EthereumApi) GetBlock(args *GetBlockArgs, reply *interface{}) error {
	err := args.requirements()
	if err != nil {
		return err
	}

	if args.BlockNumber > 0 {
		*reply = p.pipe.BlockByNumber(args.BlockNumber)
	} else {
		*reply = p.pipe.BlockByHash(args.Hash)
	}
	return nil
}

func (p *EthereumApi) Transact(args *NewTxArgs, reply *interface{}) error {
	err := args.requirements()
	if err != nil {
		return err
	}
	result, _ := p.pipe.Transact(p.pipe.Key().PrivateKey, args.Recipient, args.Value, args.Gas, args.GasPrice, args.Body)
	*reply = result
	return nil
}

func (p *EthereumApi) Create(args *NewTxArgs, reply *interface{}) error {
	err := args.requirementsContract()
	if err != nil {
		return err
	}

	result, _ := p.pipe.Transact(p.pipe.Key().PrivateKey, "", args.Value, args.Gas, args.GasPrice, args.Body)
	*reply = result
	return nil
}

func (p *EthereumApi) PushTx(args *PushTxArgs, reply *interface{}) error {
	err := args.requirementsPushTx()
	if err != nil {
		return err
	}
	result, _ := p.pipe.PushTx(args.Tx)
	*reply = result
	return nil
}

func (p *EthereumApi) GetKey(args interface{}, reply *interface{}) error {
	*reply = p.pipe.Key()
	return nil
}

func (p *EthereumApi) GetStorageAt(args *GetStorageArgs, reply *interface{}) error {
	err := args.requirements()
	if err != nil {
		return err
	}

	state := p.pipe.World().SafeGet(ethutil.Hex2Bytes(args.Address))

	var hx string
	if strings.Index(args.Key, "0x") == 0 {
		hx = string([]byte(args.Key)[2:])
	} else {
		// Convert the incoming string (which is a bigint) into hex
		i, _ := new(big.Int).SetString(args.Key, 10)
		hx = ethutil.Bytes2Hex(i.Bytes())
	}
	jsonlogger.Debugf("GetStorageAt(%s, %s)\n", args.Address, hx)
	value := state.Storage(ethutil.Hex2Bytes(hx))
	*reply = GetStorageAtRes{Address: args.Address, Key: args.Key, Value: value.Str()}
	return nil
}

func (p *EthereumApi) GetPeerCount(reply *interface{}) error {
	*reply = p.pipe.PeerCount()
	return nil
}

func (p *EthereumApi) GetIsListening(reply *interface{}) error {
	*reply = p.pipe.IsListening()
	return nil
}

func (p *EthereumApi) GetCoinbase(reply *interface{}) error {
	*reply = p.pipe.CoinBase()
	return nil
}

func (p *EthereumApi) GetIsMining(reply *interface{}) error {
	*reply = p.pipe.IsMining()
	return nil
}

func (p *EthereumApi) GetTxCountAt(args *GetTxCountArgs, reply *interface{}) error {
	err := args.requirements()
	if err != nil {
		return err
	}
	*reply = p.pipe.TxCountAt(args.Address)
	return nil
}

func (p *EthereumApi) GetBalanceAt(args *GetBalanceArgs, reply *interface{}) error {
	err := args.requirements()
	if err != nil {
		return err
	}
	state := p.pipe.World().SafeGet(ethutil.Hex2Bytes(args.Address))
	*reply = BalanceRes{Balance: state.Balance().String(), Address: args.Address}
	return nil
}

func (p *EthereumApi) GetCodeAt(args *GetCodeAtArgs, reply *interface{}) error {
	err := args.requirements()
	if err != nil {
		return err
	}
	*reply = p.pipe.CodeAt(args.Address)
	return nil
}

type GetBlockArgs struct {
	BlockNumber int32
	Hash        string
}

func (obj *GetBlockArgs) UnmarshalJSON(b []byte) (err error) {
	argint, argstr := int32(0), ""
	if err = json.Unmarshal(b, &argint); err == nil {
		obj.BlockNumber = argint
		return
	}
	if err = json.Unmarshal(b, &argstr); err == nil {
		obj.Hash = argstr
		return
	}
	return NewErrorResponse(ErrorDecodeArgs)
}

func (obj *GetBlockArgs) requirements() error {
	if obj.BlockNumber == 0 && obj.Hash == "" {
		return NewErrorResponse("GetBlock requires either a block 'number' or a block 'hash' as argument")
	}
	return nil
}

type NewTxArgs struct {
	Sec       string `json:"sec"`
	Recipient string `json:"recipient"`
	Value     string `json:"value"`
	Gas       string `json:"gas"`
	GasPrice  string `json:"gasprice"`
	Init      string `json:"init"`
	Body      string `json:"body"`
}

// type TxResponse struct {
// 	Hash string
// }

func (obj *NewTxArgs) UnmarshalJSON(b []byte) (err error) {
	if err = json.Unmarshal(b, obj); err == nil {
		return
	}
	return NewErrorResponse(ErrorDecodeArgs)
}

func (a *NewTxArgs) requirements() error {
	if a.Recipient == "" {
		return NewErrorResponse("Transact requires a 'recipient' address as argument")
	}
	if a.Value == "" {
		return NewErrorResponse("Transact requires a 'value' as argument")
	}
	if a.Gas == "" {
		return NewErrorResponse("Transact requires a 'gas' value as argument")
	}
	if a.GasPrice == "" {
		return NewErrorResponse("Transact requires a 'gasprice' value as argument")
	}
	return nil
}

func (a *NewTxArgs) requirementsContract() error {
	if a.Value == "" {
		return NewErrorResponse("Create requires a 'value' as argument")
	}
	if a.Gas == "" {
		return NewErrorResponse("Create requires a 'gas' value as argument")
	}
	if a.GasPrice == "" {
		return NewErrorResponse("Create requires a 'gasprice' value as argument")
	}
	if a.Body == "" {
		return NewErrorResponse("Create requires a 'body' value as argument")
	}
	return nil
}

type PushTxArgs struct {
	Tx string `json:"tx"`
}

func (obj *PushTxArgs) UnmarshalJSON(b []byte) (err error) {
	arg0 := ""
	if err = json.Unmarshal(b, arg0); err == nil {
		obj.Tx = arg0
		return
	}
	return NewErrorResponse(ErrorDecodeArgs)
}

func (a *PushTxArgs) requirementsPushTx() error {
	if a.Tx == "" {
		return NewErrorResponse("PushTx requires a 'tx' as argument")
	}
	return nil
}

type GetStorageArgs struct {
	Address string
	Key     string
}

func (obj *GetStorageArgs) UnmarshalJSON(b []byte) (err error) {
	arg0 := ""
	if err = json.Unmarshal(b, arg0); err == nil {
		obj.Address = arg0
		return
	}
	return NewErrorResponse(ErrorDecodeArgs)
}

func (a *GetStorageArgs) requirements() error {
	if a.Address == "" {
		return NewErrorResponse("GetStorageAt requires an 'address' value as argument")
	}
	if a.Key == "" {
		return NewErrorResponse("GetStorageAt requires an 'key' value as argument")
	}
	return nil
}

type GetStorageAtRes struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Address string `json:"address"`
}

type GetTxCountArgs struct {
	Address string `json:"address"`
}

// type GetTxCountRes struct {
// 	Nonce int `json:"nonce"`
// }

func (obj *GetTxCountArgs) UnmarshalJSON(b []byte) (err error) {
	arg0 := ""
	if err = json.Unmarshal(b, arg0); err == nil {
		obj.Address = arg0
		return
	}
	return NewErrorResponse("Could not determine JSON parameters")
}

func (a *GetTxCountArgs) requirements() error {
	if a.Address == "" {
		return NewErrorResponse("GetTxCountAt requires an 'address' value as argument")
	}
	return nil
}

// type GetPeerCountRes struct {
// 	PeerCount int `json:"peerCount"`
// }

// type GetListeningRes struct {
// 	IsListening bool `json:"isListening"`
// }

// type GetCoinbaseRes struct {
// 	Coinbase string `json:"coinbase"`
// }

// type GetMiningRes struct {
// 	IsMining bool `json:"isMining"`
// }

type GetBalanceArgs struct {
	Address string
}

func (obj *GetBalanceArgs) UnmarshalJSON(b []byte) (err error) {
	arg0 := ""
	if err = json.Unmarshal(b, &arg0); err == nil {
		obj.Address = arg0
		return
	}
	return NewErrorResponse("Could not determine JSON parameters")
}

func (a *GetBalanceArgs) requirements() error {
	if a.Address == "" {
		return NewErrorResponse("GetBalanceAt requires an 'address' value as argument")
	}
	return nil
}

type BalanceRes struct {
	Balance string `json:"balance"`
	Address string `json:"address"`
}

type GetCodeAtArgs struct {
	Address string
}

func (obj *GetCodeAtArgs) UnmarshalJSON(b []byte) (err error) {
	arg0 := ""
	if err = json.Unmarshal(b, &arg0); err == nil {
		obj.Address = arg0
		return
	}
	return NewErrorResponse(ErrorDecodeArgs)
}

func (a *GetCodeAtArgs) requirements() error {
	if a.Address == "" {
		return NewErrorResponse("GetCodeAt requires an 'address' value as argument")
	}
	return nil
}
