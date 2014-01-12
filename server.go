package main

import (
	"container/list"
	"github.com/ethereum/ethdb-go"
	"github.com/ethereum/ethutil-go"
	"github.com/ethereum/ethwire-go"
	"log"
	"net"
	"time"
)

func eachPeer(peers *list.List, callback func(*Peer)) {
	// Loop thru the peers and close them (if we had them)
	for e := peers.Front(); e != nil; e = e.Next() {
		if peer, ok := e.Value.(*Peer); ok {
			callback(peer)
		}
	}
}


type Server struct {
	// Channel for shutting down the server
	shutdownChan chan bool
	// DB interface
	//db *ethdb.LDBDatabase
	db *ethdb.MemDatabase
	// Block manager for processing new blocks and managing the block chain
	blockManager *BlockManager
	// Peers (NYI)
	peers *list.List
	// Nonce
	Nonce uint64
}

func NewServer() (*Server, error) {
	//db, err := ethdb.NewLDBDatabase()
	db, err := ethdb.NewMemDatabase()
	if err != nil {
		return nil, err
	}

	ethutil.SetConfig(db)

	nonce, _ := ethutil.RandomUint64()
	server := &Server{
		shutdownChan: make(chan bool),
		blockManager: NewBlockManager(),
		db:           db,
		peers:        list.New(),
		Nonce:        nonce,
	}

	return server, nil
}

func (s *Server) AddPeer(conn net.Conn) {
	peer := NewPeer(conn, s, true)

	if peer != nil {
		s.peers.PushBack(peer)
		peer.Start()

		log.Println("Peer connected ::", conn.RemoteAddr())
	}
}

func (s *Server) ConnectToPeer(addr string) error {
	peer := NewOutboundPeer(addr, s)

	s.peers.PushBack(peer)

	peer.Start()


	return nil
}

func (s *Server) Broadcast(msgType string, data []byte) {
	eachPeer(s.peers, func(p *Peer) {
		p.QueueMessage(ethwire.NewMessage(msgType, 0, data))
	})
}

// Start the server
func (s *Server) Start() {
	// For now this function just blocks the main thread
	ln, err := net.Listen("tcp", ":12345")
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Println(err)

				continue
			}

			go s.AddPeer(conn)
		}
	}()

	// TMP
	go func() {
		for {
			s.Broadcast("block", s.blockManager.bc.GenesisBlock().MarshalRlp())

			time.Sleep(1000 * time.Millisecond)
		}
	}()
}

func (s *Server) Stop() {
	// Close the database
	defer s.db.Close()

	eachPeer(s.peers, func(p *Peer) {
			p.Stop()
	})

	s.shutdownChan <- true
}

// This function will wait for a shutdown and resumes main thread execution
func (s *Server) WaitForShutdown() {
	<-s.shutdownChan
}
