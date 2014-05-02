package main

import (
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

type JsonRpcServer struct {
	quit     chan bool
	listener net.Listener
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

	//	ethutil.Config.Log.Infoln("[JSON] Shutdown JSON-RPC server")
	log.Println("[JSON] Shutdown JSON-RPC server")
}

func (s *JsonRpcServer) Stop() {
	close(s.quit)
}

func (s *JsonRpcServer) Start() {
	//	ethutil.Config.Log.Infoln("[JSON] Starting JSON-RPC server")
	log.Println("[JSON] Starting JSON-RPC server")
	go s.exitHandler()
	rpc.Register(new(MainPackage))
	rpc.HandleHTTP()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			//			ethutil.Config.Log.Infoln("[JSON] Error starting JSON-RPC:", err)
			log.Println("[JSON] Error starting JSON-RPC:", err)
			continue
		}
		log.Println("Incoming request")
		go jsonrpc.ServeConn(conn)
	}
}

func NewJsonRpcServer() *JsonRpcServer {
	l, err := net.Listen("tcp", ":30304")
	if err != nil {
		//		ethutil.Config.Log.Infoln("Error starting JSON-RPC")
		log.Println("Error starting JSON-RPC")
	}

	return &JsonRpcServer{
		listener: l,
		quit:     make(chan bool),
	}
}

func main() {
	s := NewJsonRpcServer()
	s.Start()
	/*

		conn, err := net.Dial("tcp", "localhost:30304")

		if err != nil {
			panic(err)
		}
		defer conn.Close()
		c := jsonrpc.NewClient(conn)
		var reply int
		err = c.Call("MainPackage.Test", nil, &reply)
		log.Println("ERR:", err)
		log.Println("result:", reply)
	*/
}
