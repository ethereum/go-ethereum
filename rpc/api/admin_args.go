// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// go-ethereum is free software: you can redistribute it and/or modify
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

package api

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/rpc/shared"
)

type AddPeerArgs struct {
	Url string
}

func (args *AddPeerArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) != 1 {
		return shared.NewDecodeParamError("Expected enode as argument")
	}

	urlstr, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("url", "not a string")
	}
	args.Url = urlstr

	return nil
}

type ImportExportChainArgs struct {
	Filename string
}

func (args *ImportExportChainArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) != 1 {
		return shared.NewDecodeParamError("Expected filename as argument")
	}

	filename, ok := obj[0].(string)
	if !ok {
		return shared.NewInvalidTypeError("filename", "not a string")
	}
	args.Filename = filename

	return nil
}

type VerbosityArgs struct {
	Level int
}

func (args *VerbosityArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) != 1 {
		return shared.NewDecodeParamError("Expected enode as argument")
	}

	level, err := numString(obj[0])
	if err == nil {
		args.Level = int(level.Int64())
	}

	return nil
}

type SetSolcArgs struct {
	Path string
}

func (args *SetSolcArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) != 1 {
		return shared.NewDecodeParamError("Expected path as argument")
	}

	if pathstr, ok := obj[0].(string); ok {
		args.Path = pathstr
		return nil
	}

	return shared.NewInvalidTypeError("path", "not a string")
}

type StartRPCArgs struct {
	ListenAddress string
	ListenPort    uint
	CorsDomain    string
	Apis          string
}

func (args *StartRPCArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	args.ListenAddress = "127.0.0.1"
	args.ListenPort = 8545
	args.Apis = "net,eth,web3"

	if len(obj) >= 1 && obj[0] != nil {
		if addr, ok := obj[0].(string); ok {
			args.ListenAddress = addr
		} else {
			return shared.NewInvalidTypeError("listenAddress", "not a string")
		}
	}

	if len(obj) >= 2 && obj[1] != nil {
		if port, ok := obj[1].(float64); ok && port >= 0 && port <= 64*1024 {
			args.ListenPort = uint(port)
		} else {
			return shared.NewInvalidTypeError("listenPort", "not a valid port number")
		}
	}

	if len(obj) >= 3 && obj[2] != nil {
		if corsDomain, ok := obj[2].(string); ok {
			args.CorsDomain = corsDomain
		} else {
			return shared.NewInvalidTypeError("corsDomain", "not a string")
		}
	}

	if len(obj) >= 4 && obj[3] != nil {
		if apis, ok := obj[3].(string); ok {
			args.Apis = apis
		} else {
			return shared.NewInvalidTypeError("apis", "not a string")
		}
	}

	return nil
}

type SleepArgs struct {
	S int
}

func (args *SleepArgs) UnmarshalJSON(b []byte) (err error) {

	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}
	if len(obj) >= 1 {
		if obj[0] != nil {
			if n, err := numString(obj[0]); err == nil {
				args.S = int(n.Int64())
			} else {
				return shared.NewInvalidTypeError("N", "not an integer: "+err.Error())
			}
		} else {
			return shared.NewInsufficientParamsError(0, 1)
		}
	}
	return nil
}

type SleepBlocksArgs struct {
	N       int64
	Timeout int64
}

func (args *SleepBlocksArgs) UnmarshalJSON(b []byte) (err error) {

	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	args.N = 1
	args.Timeout = 0
	if len(obj) >= 1 && obj[0] != nil {
		if n, err := numString(obj[0]); err == nil {
			args.N = n.Int64()
		} else {
			return shared.NewInvalidTypeError("N", "not an integer: "+err.Error())
		}
	}

	if len(obj) >= 2 && obj[1] != nil {
		if n, err := numString(obj[1]); err == nil {
			args.Timeout = n.Int64()
		} else {
			return shared.NewInvalidTypeError("Timeout", "not an integer: "+err.Error())
		}
	}

	return nil
}

type SetGlobalRegistrarArgs struct {
	NameReg         string
	ContractAddress string
}

func (args *SetGlobalRegistrarArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) == 0 {
		return shared.NewDecodeParamError("Expected namereg address")
	}

	if len(obj) >= 1 {
		if namereg, ok := obj[0].(string); ok {
			args.NameReg = namereg
		} else {
			return shared.NewInvalidTypeError("NameReg", "not a string")
		}
	}

	if len(obj) >= 2 && obj[1] != nil {
		if addr, ok := obj[1].(string); ok {
			args.ContractAddress = addr
		} else {
			return shared.NewInvalidTypeError("ContractAddress", "not a string")
		}
	}

	return nil
}

type SetHashRegArgs struct {
	HashReg string
	Sender  string
}

func (args *SetHashRegArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) >= 1 && obj[0] != nil {
		if hashreg, ok := obj[0].(string); ok {
			args.HashReg = hashreg
		} else {
			return shared.NewInvalidTypeError("HashReg", "not a string")
		}
	}

	if len(obj) >= 2 && obj[1] != nil {
		if sender, ok := obj[1].(string); ok {
			args.Sender = sender
		} else {
			return shared.NewInvalidTypeError("Sender", "not a string")
		}
	}

	return nil
}

type SetUrlHintArgs struct {
	UrlHint string
	Sender  string
}

func (args *SetUrlHintArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) >= 1 && obj[0] != nil {
		if urlhint, ok := obj[0].(string); ok {
			args.UrlHint = urlhint
		} else {
			return shared.NewInvalidTypeError("UrlHint", "not a string")
		}
	}

	if len(obj) >= 2 && obj[1] != nil {
		if sender, ok := obj[1].(string); ok {
			args.Sender = sender
		} else {
			return shared.NewInvalidTypeError("Sender", "not a string")
		}
	}

	return nil
}

type SaveInfoArgs struct {
	ContractInfo compiler.ContractInfo
	Filename     string
}

func (args *SaveInfoArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 2 {
		return shared.NewInsufficientParamsError(len(obj), 2)
	}

	if jsonraw, err := json.Marshal(obj[0]); err == nil {
		if err = json.Unmarshal(jsonraw, &args.ContractInfo); err != nil {
			return err
		}
	} else {
		return err
	}

	if filename, ok := obj[1].(string); ok {
		args.Filename = filename
	} else {
		return shared.NewInvalidTypeError("Filename", "not a string")
	}

	return nil
}

type RegisterArgs struct {
	Sender         string
	Address        string
	ContentHashHex string
}

func (args *RegisterArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 3 {
		return shared.NewInsufficientParamsError(len(obj), 3)
	}

	if len(obj) >= 1 {
		if sender, ok := obj[0].(string); ok {
			args.Sender = sender
		} else {
			return shared.NewInvalidTypeError("Sender", "not a string")
		}
	}

	if len(obj) >= 2 {
		if address, ok := obj[1].(string); ok {
			args.Address = address
		} else {
			return shared.NewInvalidTypeError("Address", "not a string")
		}
	}

	if len(obj) >= 3 {
		if hex, ok := obj[2].(string); ok {
			args.ContentHashHex = hex
		} else {
			return shared.NewInvalidTypeError("ContentHashHex", "not a string")
		}
	}

	return nil
}

type RegisterUrlArgs struct {
	Sender      string
	ContentHash string
	Url         string
}

func (args *RegisterUrlArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) >= 1 {
		if sender, ok := obj[0].(string); ok {
			args.Sender = sender
		} else {
			return shared.NewInvalidTypeError("Sender", "not a string")
		}
	}

	if len(obj) >= 2 {
		if sender, ok := obj[1].(string); ok {
			args.ContentHash = sender
		} else {
			return shared.NewInvalidTypeError("ContentHash", "not a string")
		}
	}

	if len(obj) >= 3 {
		if sender, ok := obj[2].(string); ok {
			args.Url = sender
		} else {
			return shared.NewInvalidTypeError("Url", "not a string")
		}
	}

	return nil
}

type GetContractInfoArgs struct {
	Contract string
}

func (args *GetContractInfoArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	if len(obj) >= 1 {
		if contract, ok := obj[0].(string); ok {
			args.Contract = contract
		} else {
			return shared.NewInvalidTypeError("Contract", "not a string")
		}
	}

	return nil
}

type HttpGetArgs struct {
	Uri  string
	Path string
}

func (args *HttpGetArgs) UnmarshalJSON(b []byte) (err error) {
	var obj []interface{}
	if err := json.Unmarshal(b, &obj); err != nil {
		return shared.NewDecodeParamError(err.Error())
	}

	if len(obj) < 1 {
		return shared.NewInsufficientParamsError(len(obj), 1)
	}

	if len(obj) >= 1 {
		if uri, ok := obj[0].(string); ok {
			args.Uri = uri
		} else {
			return shared.NewInvalidTypeError("Uri", "not a string")
		}
	}

	if len(obj) >= 2 && obj[1] != nil {
		if path, ok := obj[1].(string); ok {
			args.Path = path
		} else {
			return shared.NewInvalidTypeError("Path", "not a string")
		}
	}

	return nil
}
