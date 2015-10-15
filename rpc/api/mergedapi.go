// Copyright 2015 The go-ethereum Authors
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

package api

import (
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rpc/shared"
)

const (
	MergedApiVersion = "1.0"
)

// combines multiple API's
type MergedApi struct {
	apis    map[string]string
	methods map[string]shared.EthereumApi
}

// create new merged api instance
func newMergedApi(apis ...shared.EthereumApi) *MergedApi {
	mergedApi := new(MergedApi)
	mergedApi.apis = make(map[string]string, len(apis))
	mergedApi.methods = make(map[string]shared.EthereumApi)

	for _, api := range apis {
		if api != nil {
			mergedApi.apis[api.Name()] = api.ApiVersion()
			for _, method := range api.Methods() {
				mergedApi.methods[method] = api
			}
		}
	}
	return mergedApi
}

// Supported RPC methods
func (self *MergedApi) Methods() []string {
	all := make([]string, len(self.methods))
	for method, _ := range self.methods {
		all = append(all, method)
	}
	return all
}

// Call the correct API's Execute method for the given request
func (self *MergedApi) Execute(req *shared.Request) (interface{}, error) {
	glog.V(logger.Detail).Infof("%s %s", req.Method, req.Params)

	if res, _ := self.handle(req); res != nil {
		return res, nil
	}
	if api, found := self.methods[req.Method]; found {
		return api.Execute(req)
	}
	return nil, shared.NewNotImplementedError(req.Method)
}

func (self *MergedApi) Name() string {
	return shared.MergedApiName
}

func (self *MergedApi) ApiVersion() string {
	return MergedApiVersion
}

func (self *MergedApi) handle(req *shared.Request) (interface{}, error) {
	if req.Method == "modules" { // provided API's
		return self.apis, nil
	}

	return nil, nil
}
