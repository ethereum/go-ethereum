package eth

import (
	"net"
	"sync"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/event"
	ethlogger "github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/pow/ezp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/whisper"
)

const (
	seedNodeAddress = "poc-7.ethdev.com:30300"
)

var logger = ethlogger.NewLogger("SERV")

type Ethereum struct {
	// Channel for shutting down the ethereum
	shutdownChan chan bool
	quit         chan bool

	// DB interface
	db        ethutil.Database
	blacklist p2p.Blacklist

	//*** SERVICES ***
	// State manager for processing new blocks and managing the over all states
	blockManager *core.BlockManager
	txPool       *core.TxPool
	chainManager *core.ChainManager
	blockPool    *BlockPool
	whisper      *whisper.Whisper

	server   *p2p.Server
	eventMux *event.TypeMux
	txSub    event.Subscription
	blockSub event.Subscription

	RpcServer  *rpc.JsonRpcServer
	keyManager *crypto.KeyManager

	clientIdentity p2p.ClientIdentity

	synclock  sync.Mutex
	syncGroup sync.WaitGroup

	Mining bool
}

func New(db ethutil.Database, identity p2p.ClientIdentity, keyManager *crypto.KeyManager, nat p2p.NAT, port string, maxPeers int) (*Ethereum, error) {

	saveProtocolVersion(db)
	ethutil.Config.Db = db

	eth := &Ethereum{
		shutdownChan:   make(chan bool),
		quit:           make(chan bool),
		db:             db,
		keyManager:     keyManager,
		clientIdentity: identity,
		blacklist:      p2p.NewBlacklist(),
		eventMux:       &event.TypeMux{},
	}

	eth.txPool = core.NewTxPool(eth)
	eth.chainManager = core.NewChainManager(eth.EventMux())
	eth.blockManager = core.NewBlockManager(eth)
	eth.chainManager.SetProcessor(eth.blockManager)
	eth.whisper = whisper.New()

	hasBlock := eth.chainManager.HasBlock
	insertChain := eth.chainManager.InsertChain
	eth.blockPool = NewBlockPool(hasBlock, insertChain, ezp.Verify)

	// Start services
	eth.txPool.Start()

	ethProto := EthProtocol(eth.txPool, eth.chainManager, eth.blockPool)
	protocols := []p2p.Protocol{ethProto, eth.whisper.Protocol()}

	server := &p2p.Server{
		Identity:   identity,
		MaxPeers:   maxPeers,
		Protocols:  protocols,
		ListenAddr: ":" + port,
		Blacklist:  eth.blacklist,
		NAT:        nat,
	}

	eth.server = server

	return eth, nil
}

func (s *Ethereum) KeyManager() *crypto.KeyManager {
	return s.keyManager
}

func (s *Ethereum) ClientIdentity() p2p.ClientIdentity {
	return s.clientIdentity
}

func (s *Ethereum) ChainManager() *core.ChainManager {
	return s.chainManager
}

func (s *Ethereum) BlockManager() *core.BlockManager {
	return s.blockManager
}

func (s *Ethereum) TxPool() *core.TxPool {
	return s.txPool
}

func (s *Ethereum) BlockPool() *BlockPool {
	return s.blockPool
}

func (s *Ethereum) EventMux() *event.TypeMux {
	return s.eventMux
}
func (self *Ethereum) Db() ethutil.Database {
	return self.db
}

func (s *Ethereum) IsMining() bool {
	return s.Mining
}

func (s *Ethereum) IsListening() bool {
	// XXX TODO
	return false
}

func (s *Ethereum) PeerCount() int {
	return s.server.PeerCount()
}

func (s *Ethereum) Peers() []*p2p.Peer {
	return s.server.Peers()
}

// Start the ethereum
func (s *Ethereum) Start(seed bool) error {
	err := s.server.Start()
	if err != nil {
		return err
	}
	s.blockPool.Start()
	s.whisper.Start()

	// broadcast transactions
	s.txSub = s.eventMux.Subscribe(core.TxPreEvent{})
	go s.txBroadcastLoop()

	// broadcast mined blocks
	s.blockSub = s.eventMux.Subscribe(core.NewMinedBlockEvent{})
	go s.blockBroadcastLoop()

	// TODO: read peers here
	if seed {
		logger.Infof("Connect to seed node %v", seedNodeAddress)
		if err := s.SuggestPeer(seedNodeAddress); err != nil {
			return err
		}
	}

	logger.Infoln("Server started")
	return nil
}

func (self *Ethereum) SuggestPeer(addr string) error {
	netaddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		logger.Errorf("couldn't resolve %s:", addr, err)
		return err
	}

	self.server.SuggestPeer(netaddr.IP, netaddr.Port, nil)
	return nil
}

func (s *Ethereum) Stop() {
	// Close the database
	defer s.db.Close()

	close(s.quit)

	s.txSub.Unsubscribe()    // quits txBroadcastLoop
	s.blockSub.Unsubscribe() // quits blockBroadcastLoop

	if s.RpcServer != nil {
		s.RpcServer.Stop()
	}
	s.txPool.Stop()
	s.eventMux.Stop()
	s.blockPool.Stop()
	s.whisper.Stop()

	logger.Infoln("Server stopped")
	close(s.shutdownChan)
}

// This function will wait for a shutdown and resumes main thread execution
func (s *Ethereum) WaitForShutdown() {
	<-s.shutdownChan
}

// now tx broadcasting is taken out of txPool
// handled here via subscription, efficiency?
func (self *Ethereum) txBroadcastLoop() {
	// automatically stops if unsubscribe
	for obj := range self.txSub.Chan() {
		event := obj.(core.TxPreEvent)
		self.server.Broadcast("eth", TxMsg, []interface{}{event.Tx.RlpData()})
	}
}

func (self *Ethereum) blockBroadcastLoop() {
	// automatically stops if unsubscribe
	for obj := range self.txSub.Chan() {
		event := obj.(core.NewMinedBlockEvent)
		self.server.Broadcast("eth", NewBlockMsg, event.Block.Value().Val)
	}
}

func saveProtocolVersion(db ethutil.Database) {
	d, _ := db.Get([]byte("ProtocolVersion"))
	protocolVersion := ethutil.NewValue(d).Uint()

	if protocolVersion == 0 {
		db.Put([]byte("ProtocolVersion"), ethutil.NewValue(ProtocolVersion).Bytes())
	}
}
