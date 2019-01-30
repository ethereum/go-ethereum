// Copyright 2019 The go-ethereum Authors
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

package main

import (
	"net"
	"net/http"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
	"github.com/ethereum/go-ethereum/swarm/storage/mock/db"
	"github.com/ethereum/go-ethereum/swarm/storage/mock/mem"
	cli "gopkg.in/urfave/cli.v1"
)

func startWS(ctx *cli.Context) (err error) {
	log.PrintOrigins(true)
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(ctx.Int("verbosity")), log.StreamHandler(os.Stdout, log.TerminalFormat(true))))

	var globalStore mock.GlobalStorer
	dir := ctx.String("dir")
	if dir != "" {
		dbStore, err := db.NewGlobalStore(dir)
		if err != nil {
			return err
		}
		defer dbStore.Close()
		globalStore = dbStore
		log.Info("database global store", "dir", dir)
	} else {
		globalStore = mem.NewGlobalStore()
		log.Info("in-memory global store")
	}

	server := rpc.NewServer()
	if err := server.RegisterName("mockStore", globalStore); err != nil {
		return err
	}

	endpoint := ctx.String("endpoint")
	listener, err := net.Listen("tcp", endpoint)
	if err != nil {
		return err
	}
	wsAddress := listener.Addr().String()
	origins := ctx.StringSlice("origins")
	log.Info("websocket", "address", wsAddress, "origins", origins)

	return http.Serve(listener, server.WebsocketHandler(origins))
}
