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

// signer is a utility that can be used so sign transactions and
// arbitrary data.
package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/log"
	"net"
	"fmt"
)

var (
	ksLocation = flag.String("keystore", filepath.Join(node.DefaultDataDir(), "keystore"), "Directory for the keystore")
	chainID    = flag.Int64("chainid", params.MainnetChainConfig.ChainId.Int64(), "chain identifier")
)

func main() {
	flag.Parse()

	var (
		server = rpc.NewServer()
		api    = NewSignerAPI(*chainID, *ksLocation, true, NewCommandlineUI())
		listener net.Listener
		err error
	)

	// register signer API with server
	if err = server.RegisterName("account", api); err != nil {
		utils.Fatalf("Could not register signer API: %v", err)
	}

	endpoint := "localhost:8550"

	if listener, err = net.Listen("tcp", endpoint); err != nil {
		utils.Fatalf("Could not start http listener: %v", err)
	}
	log.Info(fmt.Sprintf("HTTP endpoint opened: http://%s", endpoint))
	fmt.Printf("HTTP endpoint opened: http://%s\n", endpoint)
	cors := []string{"*"}

	rpc.NewHTTPServer(cors, server).Serve(listener)
}
// Create account
// #curl -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","method":"account_new","params":["test"],"id":67}' localhost:8550

type rwc struct {
	io.Reader
	io.Writer
}

func (r *rwc) Close() error {
	if err := os.Stdin.Close(); err != nil {
		return err
	}
	return os.Stdout.Close()
}
