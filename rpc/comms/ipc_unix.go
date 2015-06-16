// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

package comms

import (
	"net"
	"os"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rpc/api"
	"github.com/ethereum/go-ethereum/rpc/codec"
)

func newIpcClient(cfg IpcConfig, codec codec.Codec) (*ipcClient, error) {
	c, err := net.DialUnix("unix", nil, &net.UnixAddr{cfg.Endpoint, "unix"})
	if err != nil {
		return nil, err
	}

	return &ipcClient{codec.New(c)}, nil
}

func startIpc(cfg IpcConfig, codec codec.Codec, api api.EthereumApi) error {
	os.Remove(cfg.Endpoint) // in case it still exists from a previous run

	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: cfg.Endpoint, Net: "unix"})
	if err != nil {
		return err
	}
	os.Chmod(cfg.Endpoint, 0600)

	go func() {
		for {
			conn, err := l.AcceptUnix()
			if err != nil {
				glog.V(logger.Error).Infof("Error accepting ipc connection - %v\n", err)
				continue
			}

			go handle(conn, api, codec)
		}

		os.Remove(cfg.Endpoint)
	}()

	glog.V(logger.Info).Infof("IPC service started (%s)\n", cfg.Endpoint)

	return nil
}
