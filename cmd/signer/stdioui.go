// Copyright 2017 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.
//

package main

import (
	"bufio"
	"github.com/ethereum/go-ethereum/log"
	"github.com/powerman/rpc-codec/jsonrpc2"
	"io"
	"os"
	"sync"
)

type StdIOUI struct {
	client *jsonrpc2.Client
	//	codec  rpc.ClientCodec
	mu sync.Mutex
}

func NewStdIOUI() *StdIOUI {
	in, out := bufio.NewReader(os.Stdin), os.Stdout

	//codec := rpc2.NewJSONCodec()
	return &StdIOUI{client: jsonrpc2.NewClient(&rwc{in, out})}
	//return &StdIOUI{}
}

func (ui StdIOUI) dispatch(serviceMethod string, args interface{}, reply interface{}) error {

	//	ui.mu.Lock()
	//	defer ui.mu.Unlock()
	//This is not synchronized, which should not be necssary. Ideally, the UI should be able
	// to get requests and send responses out-of-order -- thus the rpc has an ID.

	//	in, out := bufio.NewReader(os.Stdin), os.Stdout
	//	codec := jsonrpc.NewClientCodec(&rwc{in, out})

	//	c := rpc.NewClientWithCodec(codec)
	//	return c.Call(serviceMethod, args, &reply)

	err := ui.client.Call(serviceMethod, args, &reply)
	if err != nil {
		log.Info("Error", "exc", err.Error())
	}
	return err
}

func (ui StdIOUI) ApproveTx(request *SignTxRequest) (SignTxResponse, error) {
	result := SignTxResponse{}
	if err := ui.dispatch("ApproveTx", request, &result); err != nil {
		return result, err
	}
	return result, nil
}

func (ui StdIOUI) ApproveSignData(request *SignDataRequest) (SignDataResponse, error) {
	var result SignDataResponse
	if err := ui.dispatch("ApproveSignData", request, &result); err != nil {
		return result, err
	}
	return result, nil
}

func (ui StdIOUI) ApproveExport(request *ExportRequest) (ExportResponse, error) {
	var result ExportResponse
	if err := ui.dispatch("ApproveExport", request, &result); err != nil {
		return result, err
	}
	return result, nil
}

func (ui StdIOUI) ApproveImport(request *ImportRequest) (ImportResponse, error) {
	var result ImportResponse
	if err := ui.dispatch("ApproveImport", request, &result); err != nil {
		return result, err
	}
	return result, nil
}

func (ui StdIOUI) ApproveListing(request *ListRequest) (ListResponse, error) {
	var result ListResponse
	if err := ui.dispatch("ApproveListing", request, &result); err != nil {
		return result, err
	}
	return result, nil
}

func (ui StdIOUI) ApproveNewAccount(request *NewAccountRequest) (NewAccountResponse, error) {
	var result NewAccountResponse
	if err := ui.dispatch("ApproveNewAccount", request, &result); err != nil {
		return result, err
	}
	return result, nil
}

func (ui StdIOUI) ShowError(message string) {
	err := ui.dispatch("ShowError", &Message{message}, nil)
	if err != nil {
		log.Info("Error calling 'ShowError'", "exc", err.Error(), "msg", message)
	}
}

func (ui StdIOUI) ShowInfo(message string) {
	err := ui.dispatch("ShowInfo", Message{message}, nil)
	if err != nil {
		log.Info("Error calling 'ShowInfo'", "exc", err.Error(), "msg", message)
	}
}

type rwc struct {
	io.Reader
	io.Writer
}

func (r *rwc) Close() error {
	if err := os.Stdin.Close(); err != nil {
		return err
	}
	return os.Stdout.Close()
	//return nil
}
