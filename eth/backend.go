package eth

import (
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/blockpool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/vm"
	"github.com/ethereum/go-ethereum/whisper"
)

var (
	servlogger = logger.NewLogger("SERV")
	jsonlogger = logger.NewJsonLogger()

	defaultBootNodes = []*discover.Node{
		// ETH/DEV cmd/bootnode
		discover.MustParseNode("enode://6cdd090303f394a1cac34ecc9f7cda18127eafa2a3a06de39f6d920b0e583e062a7362097c7c65ee490a758b442acd5c80c6fce4b148c6a391e946b45131365b@54.169.166.226:30303"),
		// ETH/DEV cpp-ethereum (poc-8.ethdev.com)
		discover.MustParseNode("enode://4a44599974518ea5b0f14c31c4463692ac0329cb84851f3435e6d1b18ee4eae4aa495f846a0fa1219bd58035671881d44423876e57db2abd57254d0197da0ebe@5.1.83.226:30303"),
	}
)

type Config struct {
	Name            string
	ProtocolVersion int
	NetworkId       int

	DataDir   string
	LogFile   string
	LogLevel  int
	LogFormat string
	VmDebug   bool

	MaxPeers int
	Port     string

	// This should be a space-separated list of
	// discovery node URLs.
	BootNodes string

	// This key is used to identify the node on the network.
	// If nil, an ephemeral key is used.
	NodeKey *ecdsa.PrivateKey

	NAT  nat.Interface
	Shh  bool
	Dial bool

	MinerThreads   int
	AccountManager *accounts.Manager

	// NewDB is used to create databases.
	// If nil, the default is to create leveldb databases on disk.
	NewDB func(path string) (common.Database, error)
}

func (cfg *Config) parseBootNodes() []*discover.Node {
	if cfg.BootNodes == "" {
		return defaultBootNodes
	}
	var ns []*discover.Node
	for _, url := range strings.Split(cfg.BootNodes, " ") {
		if url == "" {
			continue
		}
		n, err := discover.ParseNode(url)
		if err != nil {
			servlogger.Errorf("Bootstrap URL %s: %v\n", url, err)
			continue
		}
		ns = append(ns, n)
	}
	return ns
}

func (cfg *Config) nodeKey() (*ecdsa.PrivateKey, error) {
	// use explicit key from command line args if set
	if cfg.NodeKey != nil {
		return cfg.NodeKey, nil
	}
	// use persistent key if present
	keyfile := path.Join(cfg.DataDir, "nodekey")
	key, err := crypto.LoadECDSA(keyfile)
	if err == nil {
		return key, nil
	}
	// no persistent key, generate and store a new one
	if key, err = crypto.GenerateKey(); err != nil {
		return nil, fmt.Errorf("could not generate server key: %v", err)
	}
	if err := ioutil.WriteFile(keyfile, crypto.FromECDSA(key), 0600); err != nil {
		servlogger.Errorln("could not persist nodekey: ", err)
	}
	return key, nil
}

type Ethereum struct {
	// Channel for shutting down the ethereum
	shutdownChan chan bool

	// DB interfaces
	blockDb common.Database // Block chain database
	stateDb common.Database // State changes database
	extraDb common.Database // Extra database (txs, etc)

	//*** SERVICES ***
	// State manager for processing new blocks and managing the over all states
	blockProcessor *core.BlockProcessor
	txPool         *core.TxPool
	chainManager   *core.ChainManager
	blockPool      *blockpool.BlockPool
	accountManager *accounts.Manager
	whisper        *whisper.Whisper
	pow            *ethash.Ethash

	net      *p2p.Server
	eventMux *event.TypeMux
	txSub    event.Subscription
	blockSub event.Subscription
	miner    *miner.Miner

	logger logger.LogSystem

	Mining          bool
	DataDir         string
	version         string
	ProtocolVersion int
	NetworkId       int
}

func New(config *Config) (*Ethereum, error) {
	// Boostrap database
	servlogsystem := logger.New(config.DataDir, config.LogFile, config.LogLevel, config.LogFormat)

	newdb := config.NewDB
	if newdb == nil {
		newdb = func(path string) (common.Database, error) { return ethdb.NewLDBDatabase(path) }
	}
	blockDb, err := newdb(path.Join(config.DataDir, "blockchain"))
	if err != nil {
		return nil, err
	}
	stateDb, err := newdb(path.Join(config.DataDir, "state"))
	if err != nil {
		return nil, err
	}
	extraDb, err := ethdb.NewLDBDatabase(path.Join(config.DataDir, "extra"))

	// Perform database sanity checks
	d, _ := extraDb.Get([]byte("ProtocolVersion"))
	protov := int(common.NewValue(d).Uint())
	if protov != config.ProtocolVersion && protov != 0 {
		path := path.Join(config.DataDir, "blockchain")
		return nil, fmt.Errorf("Database version mismatch. Protocol(%d / %d). `rm -rf %s`", protov, config.ProtocolVersion, path)
	}
	saveProtocolVersion(extraDb, config.ProtocolVersion)
	servlogger.Infof("Protocol Version: %v, Network Id: %v", config.ProtocolVersion, config.NetworkId)

	eth := &Ethereum{
		shutdownChan:   make(chan bool),
		blockDb:        blockDb,
		stateDb:        stateDb,
		extraDb:        extraDb,
		eventMux:       &event.TypeMux{},
		logger:         servlogsystem,
		accountManager: config.AccountManager,
		DataDir:        config.DataDir,
		version:        config.Name, // TODO should separate from Name
	}

	eth.chainManager = core.NewChainManager(blockDb, stateDb, eth.EventMux())
	eth.pow = ethash.New(eth.chainManager)
	eth.txPool = core.NewTxPool(eth.EventMux())
	eth.blockProcessor = core.NewBlockProcessor(stateDb, extraDb, eth.pow, eth.txPool, eth.chainManager, eth.EventMux())
	eth.chainManager.SetProcessor(eth.blockProcessor)
	eth.whisper = whisper.New()
	eth.miner = miner.New(eth, eth.pow, config.MinerThreads)

	hasBlock := eth.chainManager.HasBlock
	insertChain := eth.chainManager.InsertChain
	eth.blockPool = blockpool.New(hasBlock, insertChain, eth.pow.Verify)

	netprv, err := config.nodeKey()
	if err != nil {
		return nil, err
	}

	ethProto := EthProtocol(config.ProtocolVersion, config.NetworkId, eth.txPool, eth.chainManager, eth.blockPool)
	protocols := []p2p.Protocol{ethProto}
	if config.Shh {
		protocols = append(protocols, eth.whisper.Protocol())
	}

	eth.net = &p2p.Server{
		PrivateKey:     netprv,
		Name:           config.Name,
		MaxPeers:       config.MaxPeers,
		Protocols:      protocols,
		NAT:            config.NAT,
		NoDial:         !config.Dial,
		BootstrapNodes: config.parseBootNodes(),
	}
	if len(config.Port) > 0 {
		eth.net.ListenAddr = ":" + config.Port
	}

	vm.Debug = config.VmDebug

	return eth, nil
}

type NodeInfo struct {
	Name       string
	NodeUrl    string
	NodeID     string
	IP         string
	DiscPort   int // UDP listening port for discovery protocol
	TCPPort    int // TCP listening port for RLPx
	Td         string
	ListenAddr string
}

func (s *Ethereum) NodeInfo() *NodeInfo {
	node := s.net.Self()

	return &NodeInfo{
		Name:       s.Name(),
		NodeUrl:    node.String(),
		NodeID:     node.ID.String(),
		IP:         node.IP.String(),
		DiscPort:   node.DiscPort,
		TCPPort:    node.TCPPort,
		ListenAddr: s.net.ListenAddr,
		Td:         s.ChainManager().Td().String(),
	}
}

type PeerInfo struct {
	ID            string
	Name          string
	Caps          string
	RemoteAddress string
	LocalAddress  string
}

func newPeerInfo(peer *p2p.Peer) *PeerInfo {
	var caps []string
	for _, cap := range peer.Caps() {
		caps = append(caps, cap.String())
	}
	return &PeerInfo{
		ID:            peer.ID().String(),
		Name:          peer.Name(),
		Caps:          strings.Join(caps, ", "),
		RemoteAddress: peer.RemoteAddr().String(),
		LocalAddress:  peer.LocalAddr().String(),
	}
}

// PeersInfo returns an array of PeerInfo objects describing connected peers
func (s *Ethereum) PeersInfo() (peersinfo []*PeerInfo) {
	for _, peer := range s.net.Peers() {
		if peer != nil {
			peersinfo = append(peersinfo, newPeerInfo(peer))
		}
	}
	return
}

func (s *Ethereum) ResetWithGenesisBlock(gb *types.Block) {
	s.chainManager.ResetWithGenesisBlock(gb)
	s.pow.UpdateCache(true)
}

func (s *Ethereum) StartMining() error {
	cb, err := s.accountManager.Coinbase()
	if err != nil {
		servlogger.Errorf("Cannot start mining without coinbase: %v\n", err)
		return fmt.Errorf("no coinbase: %v", err)
	}
	s.miner.Start(cb)
	return nil
}

func (s *Ethereum) StopMining()    { s.miner.Stop() }
func (s *Ethereum) IsMining() bool { return s.miner.Mining() }

func (s *Ethereum) Logger() logger.LogSystem             { return s.logger }
func (s *Ethereum) Name() string                         { return s.net.Name }
func (s *Ethereum) AccountManager() *accounts.Manager    { return s.accountManager }
func (s *Ethereum) ChainManager() *core.ChainManager     { return s.chainManager }
func (s *Ethereum) BlockProcessor() *core.BlockProcessor { return s.blockProcessor }
func (s *Ethereum) TxPool() *core.TxPool                 { return s.txPool }
func (s *Ethereum) BlockPool() *blockpool.BlockPool      { return s.blockPool }
func (s *Ethereum) Whisper() *whisper.Whisper            { return s.whisper }
func (s *Ethereum) EventMux() *event.TypeMux             { return s.eventMux }
func (s *Ethereum) BlockDb() common.Database             { return s.blockDb }
func (s *Ethereum) StateDb() common.Database             { return s.stateDb }
func (s *Ethereum) ExtraDb() common.Database             { return s.extraDb }
func (s *Ethereum) IsListening() bool                    { return true } // Always listening
func (s *Ethereum) PeerCount() int                       { return s.net.PeerCount() }
func (s *Ethereum) Peers() []*p2p.Peer                   { return s.net.Peers() }
func (s *Ethereum) MaxPeers() int                        { return s.net.MaxPeers }
func (s *Ethereum) Version() string                      { return s.version }

// Start the ethereum
func (s *Ethereum) Start() error {
	jsonlogger.LogJson(&logger.LogStarting{
		ClientString:    s.net.Name,
		ProtocolVersion: ProtocolVersion,
	})

	if s.net.MaxPeers > 0 {
		err := s.net.Start()
		if err != nil {
			return err
		}
	}

	// Start services
	s.txPool.Start()
	s.blockPool.Start()

	if s.whisper != nil {
		s.whisper.Start()
	}

	// broadcast transactions
	s.txSub = s.eventMux.Subscribe(core.TxPreEvent{})
	go s.txBroadcastLoop()

	// broadcast mined blocks
	s.blockSub = s.eventMux.Subscribe(core.NewMinedBlockEvent{})
	go s.blockBroadcastLoop()

	servlogger.Infoln("Server started")
	return nil
}

func (s *Ethereum) StartForTest() {
	jsonlogger.LogJson(&logger.LogStarting{
		ClientString:    s.net.Name,
		ProtocolVersion: ProtocolVersion,
	})

	// Start services
	s.txPool.Start()
	s.blockPool.Start()
}

func (self *Ethereum) SuggestPeer(nodeURL string) error {
	n, err := discover.ParseNode(nodeURL)
	if err != nil {
		return fmt.Errorf("invalid node URL: %v", err)
	}
	self.net.SuggestPeer(n)
	return nil
}

func (s *Ethereum) Stop() {
	// Close the database
	defer s.blockDb.Close()
	defer s.stateDb.Close()
	defer s.extraDb.Close()

	s.txSub.Unsubscribe()    // quits txBroadcastLoop
	s.blockSub.Unsubscribe() // quits blockBroadcastLoop

	s.txPool.Stop()
	s.eventMux.Stop()
	s.blockPool.Stop()
	if s.whisper != nil {
		s.whisper.Stop()
	}

	servlogger.Infoln("Server stopped")
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
		self.net.Broadcast("eth", TxMsg, event.Tx.RlpData())
	}
}

func (self *Ethereum) blockBroadcastLoop() {
	// automatically stops if unsubscribe
	for obj := range self.blockSub.Chan() {
		switch ev := obj.(type) {
		case core.NewMinedBlockEvent:
			self.net.Broadcast("eth", NewBlockMsg, ev.Block.RlpData(), ev.Block.Td)
		}
	}
}

func saveProtocolVersion(db common.Database, protov int) {
	d, _ := db.Get([]byte("ProtocolVersion"))
	protocolVersion := common.NewValue(d).Uint()

	if protocolVersion == 0 {
		db.Put([]byte("ProtocolVersion"), common.NewValue(protov).Bytes())
	}
}
