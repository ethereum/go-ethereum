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

// startHTTP starts a global store with HTTP RPC server.
// It is used for "http" cli command.
func startHTTP(ctx *cli.Context) (err error) {
	server, cleanup, err := newServer(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	listener, err := net.Listen("tcp", ctx.String("addr"))
	if err != nil {
		return err
	}
	log.Info("http", "address", listener.Addr().String())

	return http.Serve(listener, server)
}

// startWS starts a global store with WebSocket RPC server.
// It is used for "websocket" cli command.
func startWS(ctx *cli.Context) (err error) {
	server, cleanup, err := newServer(ctx)
	if err != nil {
		return err
	}
	defer cleanup()

	listener, err := net.Listen("tcp", ctx.String("addr"))
	if err != nil {
		return err
	}
	origins := ctx.StringSlice("origins")
	log.Info("websocket", "address", listener.Addr().String(), "origins", origins)

	return http.Serve(listener, server.WebsocketHandler(origins))
}

// newServer creates a global store and returns its RPC server.
// Returned cleanup function should be called only if err is nil.
func newServer(ctx *cli.Context) (server *rpc.Server, cleanup func(), err error) {
	log.PrintOrigins(true)
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(ctx.Int("verbosity")), log.StreamHandler(os.Stdout, log.TerminalFormat(false))))

	cleanup = func() {}
	var globalStore mock.GlobalStorer
	dir := ctx.String("dir")
	if dir != "" {
		dbStore, err := db.NewGlobalStore(dir)
		if err != nil {
			return nil, nil, err
		}
		cleanup = func() {
			dbStore.Close()
		}
		globalStore = dbStore
		log.Info("database global store", "dir", dir)
	} else {
		globalStore = mem.NewGlobalStore()
		log.Info("in-memory global store")
	}

	server = rpc.NewServer()
	if err := server.RegisterName("mockStore", globalStore); err != nil {
		cleanup()
		return nil, nil, err
	}

	return server, cleanup, nil
}
