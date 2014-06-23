package ethrpc

import (
	"fmt"
	"github.com/ethereum/eth-go/ethpub"
	"github.com/ethereum/eth-go/ethlog"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

var logger = ethlog.NewLogger("JSON")

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

	logger.Infoln("Shutdown JSON-RPC server")
}

func (s *JsonRpcServer) Stop() {
	close(s.quit)
}

func (s *JsonRpcServer) Start() {
	logger.Infoln("Starting JSON-RPC server")
	go s.exitHandler()
	rpc.Register(&EthereumApi{ethp: s.ethp})
	rpc.HandleHTTP()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			logger.Infoln("Error starting JSON-RPC:", err)
			break
		}
		logger.Debugln("Incoming request.")
		go jsonrpc.ServeConn(conn)
	}
}

func NewJsonRpcServer(ethp *ethpub.PEthereum, port int) (*JsonRpcServer, error) {
	sport := fmt.Sprintf(":%d", port)
	l, err := net.Listen("tcp", sport)
	if err != nil {
		return nil, err
	}

	return &JsonRpcServer{
		listener: l,
		quit:     make(chan bool),
		ethp:     ethp,
	}, nil
}
