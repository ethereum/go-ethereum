package main

import (
	"container/list"
	"github.com/ethereum/ethdb-go"
	"github.com/ethereum/ethutil-go"
	"github.com/ethereum/ethwire-go"
	"log"
	"net"
	"sync/atomic"
	"time"
)

func eachPeer(peers *list.List, callback func(*Peer, *list.Element)) {
	// Loop thru the peers and close them (if we had them)
	for e := peers.Front(); e != nil; e = e.Next() {
		if peer, ok := e.Value.(*Peer); ok {
			callback(peer, e)
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

	return nil
}

func (s *Server) Broadcast(msgType ethwire.MsgType, data []byte) {
	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		p.QueueMessage(ethwire.NewMessage(msgType, data))
	})
}

const (
	processReapingTimeout = 1 // TODO increase
)

func (s *Server) ReapDeadPeers() {
	for {
		eachPeer(s.peers, func(p *Peer, e *list.Element) {
			if atomic.LoadInt32(&p.disconnect) == 1 || (p.inbound && (time.Now().Unix()-p.lastPong) > int64(5*time.Minute)) {
				log.Println("Dead peer found .. reaping")

				s.peers.Remove(e)
			}
		})

		time.Sleep(processReapingTimeout * time.Second)
	}
}

// Start the server
func (s *Server) Start() {
	// For now this function just blocks the main thread
	ln, err := net.Listen("tcp", ":12345")
	if err != nil {
		// This is mainly for testing to create a "network"
		if Debug {
			log.Println("Connection listening disabled. Acting as client")

			err = s.ConnectToPeer("localhost:12345")
			if err != nil {
				log.Println("Error starting server", err)

				s.Stop()
			}

			return
		} else {
			log.Fatal(err)
		}
	}

	// Start the reaping processes
	go s.ReapDeadPeers()

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
	/*
		go func() {
			for {
				s.Broadcast("block", s.blockManager.bc.GenesisBlock().RlpEncode())

				time.Sleep(1000 * time.Millisecond)
			}
		}()
	*/
}

func (s *Server) Stop() {
	// Close the database
	defer s.db.Close()

	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		p.Stop()
	})

	s.shutdownChan <- true
}

// This function will wait for a shutdown and resumes main thread execution
func (s *Server) WaitForShutdown() {
	<-s.shutdownChan
}
