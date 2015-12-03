// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package api

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"github.com/ethereum/go-ethereum/swarm"
)

const (
	BzzApiVersion = "1.0"
)

// eth api provider
// See https://github.com/ethereum/wiki/wiki/JSON-RPC
type bzzApi struct {
	swarm   *swarm.Swarm
	methods map[string]bzzhandler
	codec   codec.ApiCoder
}

// eth callback handler
type bzzhandler func(*bzzApi, *shared.Request) (interface{}, error)

var (
	bzzMapping = map[string]bzzhandler{
		"bzz_info":     (*bzzApi).Info,
		"bzz_issue":    (*bzzApi).Issue,
		"bzz_cash":     (*bzzApi).Cash,
		"bzz_deposit":  (*bzzApi).Deposit,
		"bzz_register": (*bzzApi).Register,
		"bzz_resolve":  (*bzzApi).Resolve,
		"bzz_download": (*bzzApi).Download,
		"bzz_upload":   (*bzzApi).Upload,
		"bzz_get":      (*bzzApi).Get,
		"bzz_put":      (*bzzApi).Put,
		"bzz_modify":   (*bzzApi).Modify,
	}
)

func newSwarmOfflineError(method string) error {
	return shared.NewNotAvailableError(method, "swarm offline")
}

// create new bzzApi instance
func NewBzzApi(stack *node.Node, codec codec.Codec) *bzzApi {
	var swarm *swarm.Swarm
	stack.Service(&swarm)
	return &bzzApi{swarm, bzzMapping, codec.New(nil)}
}

// collection with supported methods
func (self *bzzApi) Methods() []string {
	methods := make([]string, len(self.methods))
	i := 0
	for k := range self.methods {
		methods[i] = k
		i++
	}
	return methods
}

// Execute given request
func (self *bzzApi) Execute(req *shared.Request) (interface{}, error) {
	if callback, ok := self.methods[req.Method]; ok {
		return callback(self, req)
	}

	return nil, shared.NewNotImplementedError(req.Method)
}

func (self *bzzApi) Name() string {
	return shared.BzzApiName
}

func (self *bzzApi) ApiVersion() string {
	return BzzApiVersion
}

func (self *bzzApi) Info(req *shared.Request) (interface{}, error) {
	s := self.swarm
	if s == nil {
		return nil, newSwarmOfflineError(req.Method)
	}
	return s.Api().Info(), nil
}

func (self *bzzApi) Issue(req *shared.Request) (interface{}, error) {
	s := self.swarm
	if s == nil {
		return nil, newSwarmOfflineError(req.Method)
	}

	args := new(BzzIssueArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	cheque, err := s.Api().Issue(common.HexToAddress(args.Beneficiary), args.Amount)
	if err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	out, err := json.MarshalIndent(cheque, "   ", "")
	if err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	return string(out), nil
}

func (self *bzzApi) Cash(req *shared.Request) (interface{}, error) {
	s := self.swarm
	if s == nil {
		return nil, newSwarmOfflineError(req.Method)
	}

	args := new(BzzCashArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	return s.Api().Cash(args.Cheque)

}

func (self *bzzApi) Deposit(req *shared.Request) (interface{}, error) {
	s := self.swarm
	if s == nil {
		return nil, newSwarmOfflineError(req.Method)
	}

	args := new(BzzDepositArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	return s.Api().Deposit(args.Amount)
}

func (self *bzzApi) Register(req *shared.Request) (interface{}, error) {
	s := self.swarm
	if s == nil {
		return nil, newSwarmOfflineError(req.Method)
	}

	args := new(BzzRegisterArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	err := s.Api().Register(common.HexToAddress(args.Address), args.Domain, common.HexToHash(args.ContentHash))
	return err == nil, err
}

func (self *bzzApi) Resolve(req *shared.Request) (interface{}, error) {
	s := self.swarm
	if s == nil {
		return nil, newSwarmOfflineError(req.Method)
	}

	args := new(BzzResolveArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	key, err := s.Api().Resolve(args.Domain)
	return key.Hex(), err
}

func (self *bzzApi) Download(req *shared.Request) (interface{}, error) {

	s := self.swarm
	if s == nil {
		return nil, newSwarmOfflineError(req.Method)
	}

	args := new(BzzDownloadArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	err := s.Api().Download(args.BzzPath, args.LocalPath)
	return err == nil, err
}

func (self *bzzApi) Upload(req *shared.Request) (interface{}, error) {

	s := self.swarm
	if s == nil {
		return nil, newSwarmOfflineError(req.Method)
	}

	args := new(BzzUploadArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	return s.Api().Upload(args.LocalPath, args.Index)
}

func (self *bzzApi) Get(req *shared.Request) (interface{}, error) {

	s := self.swarm
	if s == nil {
		return nil, newSwarmOfflineError(req.Method)
	}

	args := new(BzzGetArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	var content []byte
	var mimeType string
	var status, size int
	var err error
	content, mimeType, status, size, err = s.Api().Get(args.Path)

	obj := map[string]string{
		"content":     string(content),
		"contentType": mimeType,
		"status":      fmt.Sprintf("%v", status),
		"size":        fmt.Sprintf("%v", size),
	}

	return obj, err
}

func (self *bzzApi) Put(req *shared.Request) (interface{}, error) {

	s := self.swarm
	if s == nil {
		return nil, newSwarmOfflineError(req.Method)
	}

	args := new(BzzPutArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	return s.Api().Put(args.Content, args.ContenType)
}

func (self *bzzApi) Modify(req *shared.Request) (interface{}, error) {

	s := self.swarm
	if s == nil {
		return nil, newSwarmOfflineError(req.Method)
	}

	args := new(BzzModifyArgs)
	if err := self.codec.Decode(req.Params, &args); err != nil {
		return nil, shared.NewDecodeParamError(err.Error())
	}

	return s.Api().Modify(args.RootHash, args.Path, args.ContentHash, args.ContentType)
}
