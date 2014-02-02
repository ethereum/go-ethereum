package eth

import (
	"container/list"
	"github.com/ethereum/ethchain-go"
	"github.com/ethereum/ethdb-go"
	"github.com/ethereum/ethutil-go"
	"github.com/ethereum/ethwire-go"
	"log"
	"net"
	"strconv"
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

	Addr net.Addr

	nat NAT
}

func New() (*Ethereum, error) {
	//db, err := ethdb.NewLDBDatabase()
	db, err := ethdb.NewMemDatabase()
	if err != nil {
		return nil, err
	}

	ethutil.Config.Db = db

	/*
		nat, err := Discover()
		if err != nil {
			log.Printf("Can'them discover upnp: %v", err)
		}
	*/

	nonce, _ := ethutil.RandomUint64()
	ethereum := &Ethereum{
		shutdownChan: make(chan bool),
		db:           db,
		peers:        list.New(),
		Nonce:        nonce,
		//nat:          nat,
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
	var alreadyConnected bool

	eachPeer(s.peers, func(p *Peer, v *list.Element) {
		if p.conn == nil {
			return
		}
		phost, _, _ := net.SplitHostPort(p.conn.RemoteAddr().String())
		ahost, _, _ := net.SplitHostPort(addr)

		if phost == ahost {
			alreadyConnected = true
			return
		}
	})

	if alreadyConnected {
		return nil
	}

	peer := NewOutboundPeer(addr, s)

	s.peers.PushBack(peer)

	return nil
}

func (s *Ethereum) OutboundPeers() []*Peer {
	// Create a new peer slice with at least the length of the total peers
	outboundPeers := make([]*Peer, s.peers.Len())
	length := 0
	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		if !p.inbound && p.conn != nil {
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

func (s *Ethereum) InOutPeers() []*Peer {
	// Create a new peer slice with at least the length of the total peers
	inboundPeers := make([]*Peer, s.peers.Len())
	length := 0
	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		inboundPeers[length] = p
		length++
	})

	return inboundPeers[:length]
}

func (s *Ethereum) Broadcast(msgType ethwire.MsgType, data []interface{}) {
	msg := ethwire.NewMessage(msgType, data)
	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		p.QueueMessage(msg)
	})
}

func (s *Ethereum) Peers() *list.List {
	return s.peers
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

// FIXME
func (s *Ethereum) upnpUpdateThread() {
	// Go off immediately to prevent code duplication, thereafter we renew
	// lease every 15 minutes.
	timer := time.NewTimer(0 * time.Second)
	lport, _ := strconv.ParseInt("30303", 10, 16)
	first := true
out:
	for {
		select {
		case <-timer.C:
			listenPort, err := s.nat.AddPortMapping("TCP", int(lport), int(lport), "eth listen port", 20*60)
			if err != nil {
				log.Printf("can't add UPnP port mapping: %v\n", err)
			}
			if first && err == nil {
				externalip, err := s.nat.GetExternalAddress()
				if err != nil {
					log.Printf("UPnP can't get external address: %v\n", err)
					continue out
				}
				// externalip, listenport
				log.Println("Successfully bound via UPnP to", externalip, listenPort)
				first = false
			}
			timer.Reset(time.Minute * 15)
		case <-s.shutdownChan:
			break out
		}
	}

	timer.Stop()

	if err := s.nat.DeletePortMapping("tcp", int(lport), int(lport)); err != nil {
		log.Printf("unable to remove UPnP port mapping: %v\n", err)
	} else {
		log.Printf("succesfully disestablished UPnP port mapping\n")
	}
}

// Start the ethereum
func (s *Ethereum) Start() {
	// Bind to addr and port
	ln, err := net.Listen("tcp", ":30303")
	if err != nil {
		log.Println("Connection listening disabled. Acting as client")
	} else {
		s.Addr = ln.Addr()
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
