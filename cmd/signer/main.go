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
	"io"
	"os"
	"path/filepath"
	"fmt"
	"net"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"gopkg.in/urfave/cli.v1"
)

func main() {

	app := cli.NewApp()
	app.Name = "signer"
	app.Usage = "Manage ethereum account operations"
	app.Flags = []cli.Flag{
		cli.Int64Flag{
			Name:  "chainid",
			Value: params.MainnetChainConfig.ChainId.Int64(),
			Usage: "chain identifier",
		},
		cli.IntFlag{
			Name:  "loglevel",
			Value: 4,
			Usage: "log level to emit to the screen",
		},
		cli.StringFlag{
			Name:  "keystore",
			Value: filepath.Join(node.DefaultDataDir(), "keystore"),
			Usage: "Directory for the keystore",
		},
		utils.NetworkIdFlag,
		utils.LightKDFFlag,
		utils.NoUSBFlag,
		utils.RPCListenAddrFlag,
		cli.IntFlag{
			Name:  "rpcport",
			Usage: "HTTP-RPC server listening port",
			Value: node.DefaultHTTPPort+5,
		},
	}

	app.Action = func(c *cli.Context) error {
		// Set up the logger to print everything and the random generator
		log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(c.Int("loglevel")), log.StreamHandler(os.Stdout, log.TerminalFormat(true))))

		var (
			server   = rpc.NewServer()
			api      = NewSignerAPI(
						c.Int64(utils.NetworkIdFlag.Name),
						c.String("keystore"),
						c.Bool(utils.NoUSBFlag.Name),
						NewCommandlineUI())
			listener net.Listener
			err      error
		)

		// register signer API with server
		if err = server.RegisterName("account", api); err != nil {
			utils.Fatalf("Could not register signer API: %v", err)
		}

		// start http server
		endpoint := fmt.Sprintf("%s:%d", c.String(utils.RPCListenAddrFlag.Name), c.Int("rpcport"))
		if listener, err = net.Listen("tcp", endpoint); err != nil {
			utils.Fatalf("Could not start http listener: %v", err)
		}
		log.Info(fmt.Sprintf("HTTP endpoint opened: http://%s", endpoint))
		cors := []string{"*"}

		rpc.NewHTTPServer(cors, server).Serve(listener)
		return nil
	}
	app.Run(os.Args)

}

// Create account
// curl -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","method":"account_new","params":["test"],"id":67}' localhost:8550

// List accounts
// curl -i -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","method":"account_list","params":[""],"id":67}' http://localhost:8550/

// Make transaction
// send(0x12)
// a52c101e0000000000000000000000000000000000000000000000000000000000000012
// curl -i -H "Content-Type: application/json" -X POST --data '{"jsonrpc":"2.0","method":"account_signTransaction","params":["0x82A2A876D39022B3019932D30Cd9c97ad5616813","pw",{"gas":"0x333","gasPrice":"0x123","nonce":"0x0","to":"0x07a565b7ed7d7a678680a4c162885bedbb695fe0", "value":"0x10", "input":"0xa52c101e0000000000000000000000000000000000000000000000000000000000000012"}],"id":67}' http://localhost:8550/

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
