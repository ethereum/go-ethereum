// Copyright 2015 The go-expanse Authors
// This file is part of the go-expanse library.
//
// The go-expanse library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-expanse library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-expanse library. If not, see <http://www.gnu.org/licenses/>.

// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

package comms

import (
	"net"
	"os"

	"github.com/expanse-project/go-expanse/fdtrack"
	"github.com/expanse-project/go-expanse/logger"
	"github.com/expanse-project/go-expanse/logger/glog"
	"github.com/expanse-project/go-expanse/rpc/codec"
	"github.com/expanse-project/go-expanse/rpc/shared"
)

func newIpcClient(cfg IpcConfig, codec codec.Codec) (*ipcClient, error) {
	c, err := net.DialUnix("unix", nil, &net.UnixAddr{cfg.Endpoint, "unix"})
	if err != nil {
		return nil, err
	}

	return &ipcClient{cfg.Endpoint, c, codec, codec.New(c)}, nil
}

func (self *ipcClient) reconnect() error {
	self.coder.Close()
	c, err := net.DialUnix("unix", nil, &net.UnixAddr{self.endpoint, "unix"})
	if err == nil {
		self.coder = self.codec.New(c)
	}

	return err
}

func startIpc(cfg IpcConfig, codec codec.Codec, api shared.ExpanseApi) error {
	os.Remove(cfg.Endpoint) // in case it still exists from a previous run

	l, err := net.Listen("unix", cfg.Endpoint)
	if err != nil {
		return err
	}
	l = fdtrack.WrapListener("ipc", l)
	os.Chmod(cfg.Endpoint, 0600)

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				glog.V(logger.Error).Infof("Error accepting ipc connection - %v\n", err)
				continue
			}

			id := newIpcConnId()
			glog.V(logger.Debug).Infof("New IPC connection with id %06d started\n", id)

			go handle(id, conn, api, codec)
		}

		os.Remove(cfg.Endpoint)
	}()

	glog.V(logger.Info).Infof("IPC service started (%s)\n", cfg.Endpoint)

	return nil
}
