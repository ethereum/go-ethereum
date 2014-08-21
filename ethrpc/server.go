package ethrpc

import (
	"fmt"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"

	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethpipe"
)

var logger = ethlog.NewLogger("JSON")

type JsonRpcServer struct {
	quit     chan bool
	listener net.Listener
	pipe     *ethpipe.JSPipe
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
	rpc.Register(&EthereumApi{pipe: s.pipe})
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

func NewJsonRpcServer(pipe *ethpipe.JSPipe, port int) (*JsonRpcServer, error) {
	sport := fmt.Sprintf(":%d", port)
	l, err := net.Listen("tcp", sport)
	if err != nil {
		return nil, err
	}

	return &JsonRpcServer{
		listener: l,
		quit:     make(chan bool),
		pipe:     pipe,
	}, nil
}
