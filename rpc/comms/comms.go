package comms

import (
	"io"
	"net"

	"fmt"
	"strings"

	"strconv"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
)

const (
	maxHttpSizeReqLength = 1024 * 1024 // 1MB
)

var (
	// List with all API's which are offered over the in proc interface by default
	DefaultInProcApis = shared.AllApis

	// List with all API's which are offered over the IPC interface by default
	DefaultIpcApis = shared.AllApis

	// List with API's which are offered over thr HTTP/RPC interface by default
	DefaultHttpRpcApis = strings.Join([]string{
		shared.DbApiName, shared.EthApiName, shared.NetApiName, shared.Web3ApiName,
	}, ",")
)

type EthereumClient interface {
	// Close underlaying connection
	Close()
	// Send request
	Send(interface{}) error
	// Receive response
	Recv() (interface{}, error)
	// List with modules this client supports
	SupportedModules() (map[string]string, error)
}

func handle(conn net.Conn, api shared.EthereumApi, c codec.Codec) {
	codec := c.New(conn)

	for {
		req, err := codec.ReadRequest()
		if err == io.EOF {
			codec.Close()
			return
		} else if err != nil {
			glog.V(logger.Error).Infof("comms recv err - %v\n", err)
			codec.Close()
			return
		}

		var rpcResponse interface{}
		res, err := api.Execute(req)

		rpcResponse = shared.NewRpcResponse(req.Id, req.Jsonrpc, res, err)
		err = codec.WriteResponse(rpcResponse)
		if err != nil {
			glog.V(logger.Error).Infof("comms send err - %v\n", err)
			codec.Close()
			return
		}
	}
}

// Endpoint must be in the form of:
// ${protocol}:${path}
// e.g. ipc:/tmp/geth.ipc
//      rpc:localhost:8545
func ClientFromEndpoint(endpoint string, c codec.Codec) (EthereumClient, error) {
	if strings.HasPrefix(endpoint, "ipc:") {
		cfg := IpcConfig{
			Endpoint: endpoint[4:],
		}
		return NewIpcClient(cfg, codec.JSON)
	}

	if strings.HasPrefix(endpoint, "rpc:") {
		parts := strings.Split(endpoint, ":")
		addr := "http://localhost"
		port := uint(8545)
		if len(parts) >= 3 {
			addr = parts[1] + ":" + parts[2]
		}

		if len(parts) >= 4 {
			p, err := strconv.Atoi(parts[3])

			if err != nil {
				return nil, err
			}
			port = uint(p)
		}

		cfg := HttpConfig{
			ListenAddress: addr,
			ListenPort:    port,
		}

		return NewHttpClient(cfg, codec.JSON), nil
	}

	return nil, fmt.Errorf("Invalid endpoint")
}
