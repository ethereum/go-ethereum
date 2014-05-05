package etherpc

import (
	"github.com/ethereum/eth-go/ethpub"
	"github.com/ethereum/eth-go/ethutil"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

type JsonRpcServer struct {
	quit     chan bool
	listener net.Listener
	ethp     *ethpub.PEthereum
}

func (s *JsonRpcServer) exitHandler() {
out:
	for {
		select {
		case <-s.quit:
			s.listener.Close()
			break out
		}
	}

	ethutil.Config.Log.Infoln("[JSON] Shutdown JSON-RPC server")
}

func (s *JsonRpcServer) Stop() {
	close(s.quit)
}

func (s *JsonRpcServer) Start() {
	ethutil.Config.Log.Infoln("[JSON] Starting JSON-RPC server")
	go s.exitHandler()
	rpc.Register(&EthereumApi{ethp: s.ethp})
	rpc.HandleHTTP()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			ethutil.Config.Log.Infoln("[JSON] Error starting JSON-RPC:", err)
			break
		}
		ethutil.Config.Log.Debugln("[JSON] Incoming request.")
		go jsonrpc.ServeConn(conn)
	}
}

func NewJsonRpcServer(ethp *ethpub.PEthereum) *JsonRpcServer {
	l, err := net.Listen("tcp", ":30304")
	if err != nil {
		ethutil.Config.Log.Infoln("Error starting JSON-RPC")
	}

	return &JsonRpcServer{
		listener: l,
		quit:     make(chan bool),
		ethp:     ethp,
	}
}
