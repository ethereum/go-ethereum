package ws

import (
	"net"
	
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rpc"
)

// Server is the basic configuration needs for the HTTP server and also
// includes CORS settings.
type Server struct {
	//Addr       string
	CorsString string
	Endpoint	string
}

// startWS initializes and starts the websocket RPC endpoint.
func StartWSServer(apis []rpc.API, server *Server) error {
	
	// Generate the whitelist based on the allowed modules
	/*whitelist := make(map[string]bool)
	for _, module := range modules {
		whitelist[module] = true
	}*/
	// Register all the APIs exposed by the services
	handler := rpc.NewServer()
	for _, api := range apis {
		//if whitelist[api.Namespace] || (len(whitelist) == 0 && api.Public) {
			if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
				return err
			}
			glog.V(logger.Debug).Infof("WebSocket registered %T under '%s'", api.Service, api.Namespace)
		//}
	}
	// All APIs registered, start the HTTP listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", server.Endpoint); err != nil {
		return err
	}
	rpc.NewWSServer(server.CorsString, handler).Serve(listener)
	glog.V(logger.Info).Infof("WebSocket endpoint opened: ws://%s", server.Endpoint)

	// All listeners booted successfully
	//n.wsEndpoint = endpoint
	//n.wsListener = listener
	//n.wsHandler = handler

	return nil
}
