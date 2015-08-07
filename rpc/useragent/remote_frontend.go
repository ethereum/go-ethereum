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

package useragent

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rpc/shared"
)

// remoteFrontend implements xeth.Frontend and will communicate with an external
// user agent over a connection
type RemoteFrontend struct {
	enabled bool
	mgr     *accounts.Manager
	d       *json.Decoder
	e       *json.Encoder
	n       int
}

// NewRemoteFrontend creates a new frontend which will interact with an user agent
// over the given connection
func NewRemoteFrontend(conn net.Conn, mgr *accounts.Manager) *RemoteFrontend {
	return &RemoteFrontend{false, mgr, json.NewDecoder(conn), json.NewEncoder(conn), 0}
}

// Enable will enable user interaction
func (fe *RemoteFrontend) Enable() {
	fe.enabled = true
}

// UnlockAccount asks the user agent for the user password and tries to unlock the account.
// It will try 3 attempts before giving up.
func (fe *RemoteFrontend) UnlockAccount(address []byte) bool {
	if !fe.enabled {
		return false
	}

	err := fe.send(AskPasswordMethod, common.Bytes2Hex(address))
	if err != nil {
		glog.V(logger.Error).Infof("Unable to send password request to agent - %v\n", err)
		return false
	}

	passwdRes, err := fe.recv()
	if err != nil {
		glog.V(logger.Error).Infof("Unable to recv password response from agent - %v\n", err)
		return false
	}

	if passwd, ok := passwdRes.Result.(string); ok {
		err = fe.mgr.Unlock(common.BytesToAddress(address), passwd)
	}

	if err == nil {
		return true
	}

	glog.V(logger.Debug).Infoln("3 invalid account unlock attempts")
	return false
}

// ConfirmTransaction asks the user for approval
func (fe *RemoteFrontend) ConfirmTransaction(tx string) bool {
	if !fe.enabled {
		return true // backwards compatibility
	}

	err := fe.send(ConfirmTransactionMethod, tx)
	if err != nil {
		glog.V(logger.Error).Infof("Unable to send tx confirmation request to agent - %v\n", err)
		return false
	}

	confirmResponse, err := fe.recv()
	if err != nil {
		glog.V(logger.Error).Infof("Unable to recv tx confirmation response from agent - %v\n", err)
		return false
	}

	if confirmed, ok := confirmResponse.Result.(bool); ok {
		return confirmed
	}

	return false
}

// send request to the agent
func (fe *RemoteFrontend) send(method string, params ...interface{}) error {
	fe.n += 1

	p, err := json.Marshal(params)
	if err != nil {
		glog.V(logger.Info).Infof("Unable to send agent request %v\n", err)
		return err
	}

	req := shared.Request{
		Method:  method,
		Jsonrpc: shared.JsonRpcVersion,
		Id:      fe.n,
		Params:  p,
	}

	return fe.e.Encode(&req)
}

// recv user response from agent
func (fe *RemoteFrontend) recv() (*shared.SuccessResponse, error) {
	var res json.RawMessage
	if err := fe.d.Decode(&res); err != nil {
		return nil, err
	}

	var response shared.SuccessResponse
	if err := json.Unmarshal(res, &response); err == nil {
		return &response, nil
	}

	return nil, fmt.Errorf("Invalid user agent response")
}
