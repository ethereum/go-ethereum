package eth

import (
	"container/list"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethdb"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethwire"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
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
	quit         chan bool
	// DB interface
	//db *ethdb.LDBDatabase
	db ethutil.Database
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

	peerMut sync.Mutex

	// Capabilities for outgoing peers
	serverCaps Caps

	nat NAT

	// Specifies the desired amount of maximum peers
	MaxPeers int
}

func New(caps Caps, usePnp bool) (*Ethereum, error) {
	db, err := ethdb.NewLDBDatabase()
	//db, err := ethdb.NewMemDatabase()
	if err != nil {
		return nil, err
	}

	var nat NAT
	if usePnp {
		nat, err = Discover()
		if err != nil {
			log.Println("UPnP failed", err)
		}
	}

	ethutil.Config.Db = db

	nonce, _ := ethutil.RandomUint64()
	ethereum := &Ethereum{
		shutdownChan: make(chan bool),
		quit:         make(chan bool),
		db:           db,
		peers:        list.New(),
		Nonce:        nonce,
		serverCaps:   caps,
		nat:          nat,
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

	if peer != nil && s.peers.Len() < s.MaxPeers {
		s.peers.PushBack(peer)
		peer.Start()
	}
}

func (s *Ethereum) ProcessPeerList(addrs []string) {
	for _, addr := range addrs {
		// TODO Probably requires some sanity checks
		s.ConnectToPeer(addr)
	}
}

func (s *Ethereum) ConnectToPeer(addr string) error {
	if s.peers.Len() < s.MaxPeers {
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

		peer := NewOutboundPeer(addr, s, s.serverCaps)

		s.peers.PushBack(peer)

		log.Printf("[SERV] Adding peer %d / %d\n", s.peers.Len(), s.MaxPeers)
	}

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
	// Reap the dead peers first
	s.reapPeers()

	// Create a new peer slice with at least the length of the total peers
	inboundPeers := make([]*Peer, s.peers.Len())
	length := 0
	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		// Only return peers with an actual ip
		if len(p.host) > 0 {
			inboundPeers[length] = p
			length++
		}
	})

	return inboundPeers[:length]
}

func (s *Ethereum) Broadcast(msgType ethwire.MsgType, data []interface{}) {
	msg := ethwire.NewMessage(msgType, data)
	s.BroadcastMsg(msg)
}

func (s *Ethereum) BroadcastMsg(msg *ethwire.Msg) {
	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		p.QueueMessage(msg)
	})
}

func (s *Ethereum) Peers() *list.List {
	return s.peers
}

func (s *Ethereum) reapPeers() {
	s.peerMut.Lock()
	defer s.peerMut.Unlock()

	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		if atomic.LoadInt32(&p.disconnect) == 1 || (p.inbound && (time.Now().Unix()-p.lastPong) > int64(5*time.Minute)) {
			s.peers.Remove(e)
		}
	})
}

func (s *Ethereum) ReapDeadPeerHandler() {
	reapTimer := time.NewTicker(processReapingTimeout * time.Second)

	for {
		select {
		case <-reapTimer.C:
			s.reapPeers()
		}
	}
}

// Start the ethereum
func (s *Ethereum) Start() {
	// Bind to addr and port
	ln, err := net.Listen("tcp", ":30303")
	if err != nil {
		log.Println("Connection listening disabled. Acting as client")
	} else {
		// Starting accepting connections
		log.Println("Ready and accepting connections")
		// Start the peer handler
		go s.peerHandler(ln)
	}

	if s.nat != nil {
		go s.upnpUpdateThread()
	}

	// Start the reaping processes
	go s.ReapDeadPeerHandler()

	// Start the tx pool
	s.TxPool.Start()

	if ethutil.Config.Seed {
		log.Println("Seeding")
		// Testnet seed bootstrapping
		resp, err := http.Get("http://www.ethereum.org/servers.poc2.txt")
		if err != nil {
			log.Println("Fetching seed failed:", err)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("Reading seed failed:", err)
			return
		}

		s.ConnectToPeer(string(body))
	}
}

func (s *Ethereum) peerHandler(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)

			continue
		}

		go s.AddPeer(conn)
	}
}

func (s *Ethereum) Stop() {
	// Close the database
	defer s.db.Close()

	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		p.Stop()
	})

	close(s.quit)

	s.TxPool.Stop()
	s.BlockManager.Stop()

	s.shutdownChan <- true
}

// This function will wait for a shutdown and resumes main thread execution
func (s *Ethereum) WaitForShutdown() {
	<-s.shutdownChan
}

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
			var err error
			_, err = s.nat.AddPortMapping("TCP", int(lport), int(lport), "eth listen port", 20*60)
			if err != nil {
				log.Println("can't add UPnP port mapping:", err)
				break out
			}
			if first && err == nil {
				_, err = s.nat.GetExternalAddress()
				if err != nil {
					log.Println("UPnP can't get external address:", err)
					continue out
				}
				first = false
			}
			timer.Reset(time.Minute * 15)
		case <-s.quit:
			break out
		}
	}

	timer.Stop()

	if err := s.nat.DeletePortMapping("TCP", int(lport), int(lport)); err != nil {
		log.Println("unable to remove UPnP port mapping:", err)
	} else {
		log.Println("succesfully disestablished UPnP port mapping")
	}
}
