package eth

import (
	"container/list"
	"github.com/ethereum/ethchain-go"
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

const (
	processReapingTimeout = 60 // TODO increase
)

type Ethereum struct {
	// Channel for shutting down the ethereum
	shutdownChan chan bool
	// DB interface
	//db *ethdb.LDBDatabase
	db *ethdb.MemDatabase
	// Block manager for processing new blocks and managing the block chain
	BlockManager *ethchain.BlockManager
	// The transaction pool. Transaction can be pushed on this pool
	// for later including in the blocks
	TxPool *ethchain.TxPool
	// Peers (NYI)
	peers *list.List
	// Nonce
	Nonce uint64
}

func New() (*Ethereum, error) {
	//db, err := ethdb.NewLDBDatabase()
	db, err := ethdb.NewMemDatabase()
	if err != nil {
		return nil, err
	}

	ethutil.Config.Db = db

	nonce, _ := ethutil.RandomUint64()
	ethereum := &Ethereum{
		shutdownChan: make(chan bool),
		db:           db,
		peers:        list.New(),
		Nonce:        nonce,
	}
	ethereum.TxPool = ethchain.NewTxPool()
	ethereum.TxPool.Speaker = ethereum
	ethereum.BlockManager = ethchain.NewBlockManager(ethereum)

	ethereum.TxPool.BlockManager = ethereum.BlockManager
	ethereum.BlockManager.TransactionPool = ethereum.TxPool

	return ethereum, nil
}

func (s *Ethereum) AddPeer(conn net.Conn) {
	peer := NewPeer(conn, s, true)

	if peer != nil {
		if s.peers.Len() > 25 {
			log.Println("SEED")
			peer.Start(true)
		} else {
			s.peers.PushBack(peer)
			peer.Start(false)
		}
	}
}

func (s *Ethereum) ProcessPeerList(addrs []string) {
	for _, addr := range addrs {
		// TODO Probably requires some sanity checks
		s.ConnectToPeer(addr)
	}
}

func (s *Ethereum) ConnectToPeer(addr string) error {
	peer := NewOutboundPeer(addr, s)

	s.peers.PushBack(peer)

	return nil
}

func (s *Ethereum) OutboundPeers() []*Peer {
	// Create a new peer slice with at least the length of the total peers
	outboundPeers := make([]*Peer, s.peers.Len())
	length := 0
	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		if !p.inbound {
			outboundPeers[length] = p
			length++
		}
	})

	return outboundPeers[:length]
}

func (s *Ethereum) InboundPeers() []*Peer {
	// Create a new peer slice with at least the length of the total peers
	inboundPeers := make([]*Peer, s.peers.Len())
	length := 0
	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		if p.inbound {
			inboundPeers[length] = p
			length++
		}
	})

	return inboundPeers[:length]
}

func (s *Ethereum) Broadcast(msgType ethwire.MsgType, data interface{}) {
	msg := ethwire.NewMessage(msgType, data)
	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		p.QueueMessage(msg)
	})
}

func (s *Ethereum) ReapDeadPeers() {
	for {
		eachPeer(s.peers, func(p *Peer, e *list.Element) {
			if atomic.LoadInt32(&p.disconnect) == 1 || (p.inbound && (time.Now().Unix()-p.lastPong) > int64(5*time.Minute)) {
				s.peers.Remove(e)
			}
		})

		time.Sleep(processReapingTimeout * time.Second)
	}
}

// Start the ethereum
func (s *Ethereum) Start() {
	// Bind to addr and port
	ln, err := net.Listen("tcp", ":30303")
	if err != nil {
		// This is mainly for testing to create a "network"
		if ethutil.Config.Debug {
			log.Println("Connection listening disabled. Acting as client")

			/*
				err = s.ConnectToPeer("localhost:12345")
				if err != nil {
					log.Println("Error starting ethereum", err)

					s.Stop()
				}
			*/
		} else {
			log.Fatal(err)
		}
	} else {
		// Starting accepting connections
		go func() {
			log.Println("Ready and accepting connections")

			for {
				conn, err := ln.Accept()
				if err != nil {
					log.Println(err)

					continue
				}

				go s.AddPeer(conn)
			}
		}()
	}

	// Start the reaping processes
	go s.ReapDeadPeers()

	// Start the tx pool
	s.TxPool.Start()
}

func (s *Ethereum) Stop() {
	// Close the database
	defer s.db.Close()

	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		p.Stop()
	})

	s.shutdownChan <- true

	s.TxPool.Stop()
}

// This function will wait for a shutdown and resumes main thread execution
func (s *Ethereum) WaitForShutdown() {
	<-s.shutdownChan
}
