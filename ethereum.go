package eth

import (
	"container/list"
	"fmt"
	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethdb"
	"github.com/ethereum/eth-go/ethrpc"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethwire"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
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
	// State manager for processing new blocks and managing the over all states
	stateManager *ethchain.StateManager
	// The transaction pool. Transaction can be pushed on this pool
	// for later including in the blocks
	txPool *ethchain.TxPool
	// The canonical chain
	blockChain *ethchain.BlockChain
	// Peers (NYI)
	peers *list.List
	// Nonce
	Nonce uint64

	Addr net.Addr
	Port string

	peerMut sync.Mutex

	// Capabilities for outgoing peers
	serverCaps Caps

	nat NAT

	// Specifies the desired amount of maximum peers
	MaxPeers int

	Mining bool

	listening bool

	reactor *ethutil.ReactorEngine

	RpcServer *ethrpc.JsonRpcServer
}

func New(caps Caps, usePnp bool) (*Ethereum, error) {
	db, err := ethdb.NewLDBDatabase("database")
	//db, err := ethdb.NewMemDatabase()
	if err != nil {
		return nil, err
	}

	var nat NAT
	if usePnp {
		nat, err = Discover()
		if err != nil {
			ethutil.Config.Log.Debugln("UPnP failed", err)
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
	ethereum.reactor = ethutil.NewReactorEngine()

	ethereum.txPool = ethchain.NewTxPool(ethereum)
	ethereum.blockChain = ethchain.NewBlockChain(ethereum)
	ethereum.stateManager = ethchain.NewStateManager(ethereum)

	// Start the tx pool
	ethereum.txPool.Start()

	return ethereum, nil
}

// Replay block
func (self *Ethereum) BlockDo(hash []byte) error {
	block := self.blockChain.GetBlock(hash)
	if block == nil {
		return fmt.Errorf("unknown block %x", hash)
	}

	parent := self.blockChain.GetBlock(block.PrevHash)

	_, err := self.stateManager.ApplyDiff(parent.State(), parent, block)
	if err != nil {
		return err
	}

	return nil

}

func (s *Ethereum) Reactor() *ethutil.ReactorEngine {
	return s.reactor
}

func (s *Ethereum) BlockChain() *ethchain.BlockChain {
	return s.blockChain
}

func (s *Ethereum) StateManager() *ethchain.StateManager {
	return s.stateManager
}

func (s *Ethereum) TxPool() *ethchain.TxPool {
	return s.txPool
}

func (s *Ethereum) ServerCaps() Caps {
	return s.serverCaps
}
func (s *Ethereum) IsMining() bool {
	return s.Mining
}
func (s *Ethereum) PeerCount() int {
	return s.peers.Len()
}
func (s *Ethereum) IsUpToDate() bool {
	upToDate := true
	eachPeer(s.peers, func(peer *Peer, e *list.Element) {
		if atomic.LoadInt32(&peer.connected) == 1 {
			if peer.catchingUp == true {
				upToDate = false
			}
		}
	})
	return upToDate
}
func (s *Ethereum) PushPeer(peer *Peer) {
	s.peers.PushBack(peer)
}
func (s *Ethereum) IsListening() bool {
	return s.listening
}

func (s *Ethereum) AddPeer(conn net.Conn) {
	peer := NewPeer(conn, s, true)

	if peer != nil {
		if s.peers.Len() < s.MaxPeers {
			peer.Start()
		} else {
			ethutil.Config.Log.Debugf("[SERV] Max connected peers reached. Not adding incoming peer.")
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
	if s.peers.Len() < s.MaxPeers {
		var alreadyConnected bool

		ahost, _, _ := net.SplitHostPort(addr)
		var chost string

		ips, err := net.LookupIP(ahost)

		if err != nil {
			return err
		} else {
			// If more then one ip is available try stripping away the ipv6 ones
			if len(ips) > 1 {
				var ipsv4 []net.IP
				// For now remove the ipv6 addresses
				for _, ip := range ips {
					if strings.Contains(ip.String(), "::") {
						continue
					} else {
						ipsv4 = append(ipsv4, ip)
					}
				}
				if len(ipsv4) == 0 {
					return fmt.Errorf("[SERV] No IPV4 addresses available for hostname")
				}

				// Pick a random ipv4 address, simulating round-robin DNS.
				rand.Seed(time.Now().UTC().UnixNano())
				i := rand.Intn(len(ipsv4))
				chost = ipsv4[i].String()
			} else {
				if len(ips) == 0 {
					return fmt.Errorf("[SERV] No IPs resolved for the given hostname")
					return nil
				}
				chost = ips[0].String()
			}
		}

		eachPeer(s.peers, func(p *Peer, v *list.Element) {
			if p.conn == nil {
				return
			}
			phost, _, _ := net.SplitHostPort(p.conn.RemoteAddr().String())

			if phost == chost {
				alreadyConnected = true
				//ethutil.Config.Log.Debugf("[SERV] Peer %s already added.\n", chost)
				return
			}
		})

		if alreadyConnected {
			return nil
		}

		NewOutboundPeer(addr, s, s.serverCaps)
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
	eachPeer(s.peers, func(p *Peer, e *list.Element) {
		if atomic.LoadInt32(&p.disconnect) == 1 || (p.inbound && (time.Now().Unix()-p.lastPong) > int64(5*time.Minute)) {
			s.removePeerElement(e)
		}
	})
}

func (s *Ethereum) removePeerElement(e *list.Element) {
	s.peerMut.Lock()
	defer s.peerMut.Unlock()

	s.peers.Remove(e)

	s.reactor.Post("peerList", s.peers)
}

func (s *Ethereum) RemovePeer(p *Peer) {
	eachPeer(s.peers, func(peer *Peer, e *list.Element) {
		if peer == p {
			s.removePeerElement(e)
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
func (s *Ethereum) Start(seed bool) {
	// Bind to addr and port
	ln, err := net.Listen("tcp", ":"+s.Port)
	if err != nil {
		log.Println("Connection listening disabled. Acting as client")
		s.listening = false
	} else {
		s.listening = true
		// Starting accepting connections
		ethutil.Config.Log.Infoln("Ready and accepting connections")
		// Start the peer handler
		go s.peerHandler(ln)
	}

	if s.nat != nil {
		go s.upnpUpdateThread()
	}

	// Start the reaping processes
	go s.ReapDeadPeerHandler()

	if seed {
		s.Seed()
	}
}

func (s *Ethereum) Seed() {
	ethutil.Config.Log.Debugln("[SERV] Retrieving seed nodes")

	// Eth-Go Bootstrapping
	ips, er := net.LookupIP("seed.bysh.me")
	if er == nil {
		peers := []string{}
		for _, ip := range ips {
			node := fmt.Sprintf("%s:%d", ip.String(), 30303)
			ethutil.Config.Log.Debugln("[SERV] Found DNS Go Peer:", node)
			peers = append(peers, node)
		}
		s.ProcessPeerList(peers)
	}

	// Official DNS Bootstrapping
	_, nodes, err := net.LookupSRV("eth", "tcp", "ethereum.org")
	if err == nil {
		peers := []string{}
		// Iterate SRV nodes
		for _, n := range nodes {
			target := n.Target
			port := strconv.Itoa(int(n.Port))
			// Resolve target to ip (Go returns list, so may resolve to multiple ips?)
			addr, err := net.LookupHost(target)
			if err == nil {
				for _, a := range addr {
					// Build string out of SRV port and Resolved IP
					peer := net.JoinHostPort(a, port)
					ethutil.Config.Log.Debugln("[SERV] Found DNS Bootstrap Peer:", peer)
					peers = append(peers, peer)
				}
			} else {
				ethutil.Config.Log.Debugln("[SERV} Couldn't resolve :", target)
			}
		}
		// Connect to Peer list
		s.ProcessPeerList(peers)
	} else {
		// Fallback to servers.poc3.txt
		resp, err := http.Get("http://www.ethereum.org/servers.poc3.txt")
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
			ethutil.Config.Log.Debugln(err)

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

	if s.RpcServer != nil {
		s.RpcServer.Stop()
	}
	s.txPool.Stop()
	s.stateManager.Stop()

	close(s.shutdownChan)
}

// This function will wait for a shutdown and resumes main thread execution
func (s *Ethereum) WaitForShutdown() {
	<-s.shutdownChan
}

func (s *Ethereum) upnpUpdateThread() {
	// Go off immediately to prevent code duplication, thereafter we renew
	// lease every 15 minutes.
	timer := time.NewTimer(5 * time.Minute)
	lport, _ := strconv.ParseInt(s.Port, 10, 16)
	first := true
out:
	for {
		select {
		case <-timer.C:
			var err error
			_, err = s.nat.AddPortMapping("TCP", int(lport), int(lport), "eth listen port", 20*60)
			if err != nil {
				ethutil.Config.Log.Debugln("can't add UPnP port mapping:", err)
				break out
			}
			if first && err == nil {
				_, err = s.nat.GetExternalAddress()
				if err != nil {
					ethutil.Config.Log.Debugln("UPnP can't get external address:", err)
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
		ethutil.Config.Log.Debugln("unable to remove UPnP port mapping:", err)
	} else {
		ethutil.Config.Log.Debugln("succesfully disestablished UPnP port mapping")
	}
}
