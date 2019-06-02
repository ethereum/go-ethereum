package endpoints

import (
	"net"

	"github.com/0xProject/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/log"
)

// StartIPCEndpoint starts an IPC endpoint.
func StartIPCEndpoint(ipcEndpoint string, apis []rpc.API) (net.Listener, *rpc.Server, error) {
	// Register all the APIs exposed by the services.
	handler := rpc.NewServer()
	for _, api := range apis {
		if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
			return nil, nil, err
		}
		log.Debug("IPC registered", "namespace", api.Namespace)
	}
	// All APIs registered, start the IPC listener.
	listener, err := ipcListen(ipcEndpoint)
	if err != nil {
		return nil, nil, err
	}
	go handler.ServeListener(listener)
	return listener, handler, nil
}
