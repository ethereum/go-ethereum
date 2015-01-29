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
2. json.Decoder() calls "UnmarshalON" defined on each "Args" struct
3. EthereumApi method, taking the "Args" type and replying with an interface to be marshalled to ON

*/
package rpc

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/xeth"
)

func toHex(b []byte) string {
	return "0x" + ethutil.Bytes2Hex(b)
}
func fromHex(s string) []byte {
	if len(s) > 1 {
		if s[0:2] == "0x" {
			s = s[2:]
		}
		return ethutil.Hex2Bytes(s)
	}
	return nil
}

type RpcServer interface {
	Start()
	Stop()
}

func NewEthereumApi(xeth *xeth.XEth) *EthereumApi {
	return &EthereumApi{xeth: xeth}
}

type EthereumApi struct {
	xeth *xeth.XEth
}

func (p *EthereumApi) GetBlock(args *GetBlockArgs, reply *interface{}) error {
	err := args.requirements()
	if err != nil {
		return err
	}

	if args.BlockNumber > 0 {
		*reply = p.xeth.BlockByNumber(args.BlockNumber)
	} else {
		*reply = p.xeth.BlockByHash(args.Hash)
	}
	return nil
}

func (p *EthereumApi) Transact(args *NewTxArgs, reply *interface{}) error {
	err := args.requirements()
	if err != nil {
		return err
	}
	result, _ := p.xeth.Transact( /* TODO specify account */ args.To, args.Value, args.Gas, args.GasPrice, args.Data)
	*reply = result
	return nil
}

func (p *EthereumApi) Call(args *NewTxArgs, reply *interface{}) error {
	result, err := p.xeth.Call( /* TODO specify account */ args.To, args.Value, args.Gas, args.GasPrice, args.Data)
	if err != nil {
		return err
	}

	*reply = result
	return nil
}

func (p *EthereumApi) PushTx(args *PushTxArgs, reply *interface{}) error {
	err := args.requirementsPushTx()
	if err != nil {
		return err
	}
	result, _ := p.xeth.PushTx(args.Tx)
	*reply = result
	return nil
}

func (p *EthereumApi) GetStorageAt(args *GetStorageArgs, reply *interface{}) error {
	err := args.requirements()
	if err != nil {
		return err
	}

	state := p.xeth.State().SafeGet(args.Address)

	var hx string
	if strings.Index(args.Key, "0x") == 0 {
		hx = string([]byte(args.Key)[2:])
	} else {
		// Convert the incoming string (which is a bigint) into hex
		i, _ := new(big.Int).SetString(args.Key, 10)
		hx = ethutil.Bytes2Hex(i.Bytes())
	}
	rpclogger.Debugf("GetStorageAt(%s, %s)\n", args.Address, hx)
	value := state.Storage(ethutil.Hex2Bytes(hx))
	*reply = GetStorageAtRes{Address: args.Address, Key: args.Key, Value: value.Str()}
	return nil
}

func (p *EthereumApi) GetPeerCount(reply *interface{}) error {
	*reply = p.xeth.PeerCount()
	return nil
}

func (p *EthereumApi) GetIsListening(reply *interface{}) error {
	*reply = p.xeth.IsListening()
	return nil
}

func (p *EthereumApi) GetCoinbase(reply *interface{}) error {
	*reply = p.xeth.Coinbase()
	return nil
}

func (p *EthereumApi) GetIsMining(reply *interface{}) error {
	*reply = p.xeth.IsMining()
	return nil
}

func (p *EthereumApi) GetTxCountAt(args *GetTxCountArgs, reply *interface{}) error {
	err := args.requirements()
	if err != nil {
		return err
	}
	*reply = p.xeth.TxCountAt(args.Address)
	return nil
}

func (p *EthereumApi) GetBalanceAt(args *GetBalanceArgs, reply *interface{}) error {
	err := args.requirements()
	if err != nil {
		return err
	}
	state := p.xeth.State().SafeGet(args.Address)
	*reply = BalanceRes{Balance: state.Balance().String(), Address: args.Address}
	return nil
}

func (p *EthereumApi) GetCodeAt(args *GetCodeAtArgs, reply *interface{}) error {
	err := args.requirements()
	if err != nil {
		return err
	}
	*reply = p.xeth.CodeAt(args.Address)
	return nil
}

func (p *EthereumApi) Sha3(args *Sha3Args, reply *interface{}) error {
	*reply = toHex(crypto.Sha3(fromHex(args.Data)))
	return nil
}

func (p *EthereumApi) GetRequestReply(req *RpcRequest, reply *interface{}) error {
	// Spec at https://github.com/ethereum/wiki/wiki/Generic-ON-RPC
	rpclogger.DebugDetailf("%T %s", req.Params, req.Params)
	switch req.Method {
	case "eth_coinbase":
		return p.GetCoinbase(reply)
	case "eth_listening":
		return p.GetIsListening(reply)
	case "eth_mining":
		return p.GetIsMining(reply)
	case "eth_peerCount":
		return p.GetPeerCount(reply)
	case "eth_countAt":
		args, err := req.ToGetTxCountArgs()
		if err != nil {
			return err
		}
		return p.GetTxCountAt(args, reply)
	case "eth_codeAt":
		args, err := req.ToGetCodeAtArgs()
		if err != nil {
			return err
		}
		return p.GetCodeAt(args, reply)
	case "eth_balanceAt":
		args, err := req.ToGetBalanceArgs()
		if err != nil {
			return err
		}
		return p.GetBalanceAt(args, reply)
	case "eth_stateAt":
		args, err := req.ToGetStorageArgs()
		if err != nil {
			return err
		}
		return p.GetStorageAt(args, reply)
	case "eth_blockByNumber", "eth_blockByHash":
		args, err := req.ToGetBlockArgs()
		if err != nil {
			return err
		}
		return p.GetBlock(args, reply)
	case "eth_transact":
		args, err := req.ToNewTxArgs()
		if err != nil {
			return err
		}
		return p.Transact(args, reply)
	case "eth_call":
		args, err := req.ToNewTxArgs()
		if err != nil {
			return err
		}
		return p.Call(args, reply)
	case "web3_sha3":
		args, err := req.ToSha3Args()
		if err != nil {
			return err
		}
		return p.Sha3(args, reply)
	default:
		return NewErrorResponse(fmt.Sprintf("%v %s", ErrorNotImplemented, req.Method))
	}

	rpclogger.DebugDetailf("Reply: %T %s", reply, reply)
	return nil
}
