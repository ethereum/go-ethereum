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

package comms

import (
	"fmt"
	"math/rand"
	"net"

	"encoding/json"

	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
)

type IpcConfig struct {
	Endpoint string
}

type ipcClient struct {
	endpoint string
	c        net.Conn
	codec    codec.Codec
	coder    codec.ApiCoder
}

func (self *ipcClient) Close() {
	self.coder.Close()
}

func (self *ipcClient) Send(req interface{}) error {
	var err error
	if r, ok := req.(*shared.Request); ok {
		if err = self.coder.WriteResponse(r); err != nil {
			if _, ok := err.(*net.OpError); ok { // connection lost, retry once
				if err = self.reconnect(); err == nil {
					err = self.coder.WriteResponse(r)
				}
			}
		}
		return err
	}

	return fmt.Errorf("Invalid request (%T)", req)
}

func (self *ipcClient) Recv() (interface{}, error) {
	res, err := self.coder.ReadResponse()
	if err != nil {
		return nil, err
	}

	if r, ok := res.(shared.SuccessResponse); ok {
		return r.Result, nil
	}

	if r, ok := res.(shared.ErrorResponse); ok {
		return r.Error, nil
	}

	return res, err
}

func (self *ipcClient) SupportedModules() (map[string]string, error) {
	req := shared.Request{
		Id:      1,
		Jsonrpc: "2.0",
		Method:  "modules",
	}

	if err := self.coder.WriteResponse(req); err != nil {
		return nil, err
	}

	res, err := self.coder.ReadResponse()
	if err != nil {
		return nil, err
	}

	if sucRes, ok := res.(shared.SuccessResponse); ok {
		data, _ := json.Marshal(sucRes.Result)
		modules := make(map[string]string)
		err = json.Unmarshal(data, &modules)
		if err == nil {
			return modules, nil
		}
	}

	return nil, fmt.Errorf("Invalid response")
}

// Create a new IPC client, UNIX domain socket on posix, named pipe on Windows
func NewIpcClient(cfg IpcConfig, codec codec.Codec) (*ipcClient, error) {
	return newIpcClient(cfg, codec)
}

// Start IPC server
func StartIpc(cfg IpcConfig, codec codec.Codec, offeredApi shared.EthereumApi) error {
	return startIpc(cfg, codec, offeredApi)
}

func newIpcConnId() int {
	return rand.Int() % 1000000
}
