package eth

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/whisper"
)

const (
	epochLength    = 30000
	ethashRevision = 23

	autoDAGcheckInterval = 10 * time.Hour
	autoDAGepochHeight   = epochLength / 2
)

var (
	jsonlogger = logger.NewJsonLogger()

	defaultBootNodes = []*discover.Node{
		// ETH/DEV Go Bootnodes
		discover.MustParseNode("enode://a979fb575495b8d6db44f750317d0f4622bf4c2aa3365d6af7c284339968eef29b69ad0dce72a4d8db5ebb4968de0e3bec910127f134779fbcb0cb6d3331163c@52.16.188.185:30303"),
		discover.MustParseNode("enode://de471bccee3d042261d52e9bff31458daecc406142b401d4cd848f677479f73104b9fdeb090af9583d3391b7f10cb2ba9e26865dd5fca4fcdc0fb1e3b723c786@54.94.239.50:30303"),
		// ETH/DEV cpp-ethereum (poc-9.ethdev.com)
		discover.MustParseNode("enode://487611428e6c99a11a9795a6abe7b529e81315ca6aad66e2a2fc76e3adf263faba0d35466c2f8f68d561dbefa8878d4df5f1f2ddb1fbeab7f42ffb8cd328bd4a@5.1.83.226:30303"),
	}

	staticNodes  = "static-nodes.json"  // Path within <datadir> to search for the static node list
	trustedNodes = "trusted-nodes.json" // Path within <datadir> to search for the trusted node list
)

type Config struct {
	Name            string
	ProtocolVersion int
	NetworkId       int
	GenesisNonce    int

	BlockChainVersion  int
	SkipBcVersionCheck bool // e.g. blockchain export

	DataDir   string
	LogFile   string
	Verbosity int
	LogJSON   string
	VmDebug   bool
	NatSpec   bool
	AutoDAG   bool

	MaxPeers        int
	MaxPendingPeers int
	Discovery       bool
	Port            string

	// Space-separated list of discovery node URLs
	BootNodes string

	// This key is used to identify the node on the network.
	// If nil, an ephemeral key is used.
	NodeKey *ecdsa.PrivateKey

	NAT  nat.Interface
	Shh  bool
	Dial bool

	Etherbase      string
	GasPrice       *big.Int
	MinerThreads   int
	AccountManager *accounts.Manager
	SolcPath       string

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
			glog.V(logger.Error).Infof("Bootstrap URL %s: %v\n", url, err)
			continue
		}
		ns = append(ns, n)
	}
	return ns
}

// parseNodes parses a list of discovery node URLs loaded from a .json file.
func (cfg *Config) parseNodes(file string) []*discover.Node {
	// Short circuit if no node config is present
	path := filepath.Join(cfg.DataDir, file)
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	// Load the nodes from the config file
	blob, err := ioutil.ReadFile(path)
	if err != nil {
		glog.V(logger.Error).Infof("Failed to access nodes: %v", err)
		return nil
	}
	nodelist := []string{}
	if err := json.Unmarshal(blob, &nodelist); err != nil {
		glog.V(logger.Error).Infof("Failed to load nodes: %v", err)
		return nil
	}
	// Interpret the list as a discovery node array
	var nodes []*discover.Node
	for _, url := range nodelist {
		if url == "" {
			continue
		}
		node, err := discover.ParseNode(url)
		if err != nil {
			glog.V(logger.Error).Infof("Node URL %s: %v\n", url, err)
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes
}

func (cfg *Config) nodeKey() (*ecdsa.PrivateKey, error) {
	// use explicit key from command line args if set
	if cfg.NodeKey != nil {
		return cfg.NodeKey, nil
	}
	// use persistent key if present
	keyfile := filepath.Join(cfg.DataDir, "nodekey")
	key, err := crypto.LoadECDSA(keyfile)
	if err == nil {
		return key, nil
	}
	// no persistent key, generate and store a new one
	if key, err = crypto.GenerateKey(); err != nil {
		return nil, fmt.Errorf("could not generate server key: %v", err)
	}
	if err := crypto.SaveECDSA(keyfile, key); err != nil {
		glog.V(logger.Error).Infoln("could not persist nodekey: ", err)
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

	// Closed when databases are flushed and closed
	databasesClosed chan bool

	//*** SERVICES ***
	// State manager for processing new blocks and managing the over all states
	blockProcessor  *core.BlockProcessor
	txPool          *core.TxPool
	chainManager    *core.ChainManager
	accountManager  *accounts.Manager
	whisper         *whisper.Whisper
	pow             *ethash.Ethash
	protocolManager *ProtocolManager
	downloader      *downloader.Downloader
	SolcPath        string
	solc            *compiler.Solidity

	net      *p2p.Server
	eventMux *event.TypeMux
	miner    *miner.Miner

	// logger logger.LogSystem

	Mining        bool
	MinerThreads  int
	NatSpec       bool
	DataDir       string
	AutoDAG       bool
	autodagquit   chan bool
	etherbase     common.Address
	clientVersion string
	ethVersionId  int
	netVersionId  int
	shhVersionId  int
}

func New(config *Config) (*Ethereum, error) {
	// Bootstrap database
	logger.New(config.DataDir, config.LogFile, config.Verbosity)
	if len(config.LogJSON) > 0 {
		logger.NewJSONsystem(config.DataDir, config.LogJSON)
	}

	// Let the database take 3/4 of the max open files (TODO figure out a way to get the actual limit of the open files)
	const dbCount = 3
	ethdb.OpenFileLimit = 128 / (dbCount + 1)

	newdb := config.NewDB
	if newdb == nil {
		newdb = func(path string) (common.Database, error) { return ethdb.NewLDBDatabase(path) }
	}
	blockDb, err := newdb(filepath.Join(config.DataDir, "blockchain"))
	if err != nil {
		return nil, fmt.Errorf("blockchain db err: %v", err)
	}
	stateDb, err := newdb(filepath.Join(config.DataDir, "state"))
	if err != nil {
		return nil, fmt.Errorf("state db err: %v", err)
	}
	extraDb, err := newdb(filepath.Join(config.DataDir, "extra"))
	if err != nil {
		return nil, fmt.Errorf("extra db err: %v", err)
	}
	nodeDb := filepath.Join(config.DataDir, "nodes")

	// Perform database sanity checks
	d, _ := blockDb.Get([]byte("ProtocolVersion"))
	protov := int(common.NewValue(d).Uint())
	if protov != config.ProtocolVersion && protov != 0 {
		path := filepath.Join(config.DataDir, "blockchain")
		return nil, fmt.Errorf("Database version mismatch. Protocol(%d / %d). `rm -rf %s`", protov, config.ProtocolVersion, path)
	}
	saveProtocolVersion(blockDb, config.ProtocolVersion)
	glog.V(logger.Info).Infof("Protocol Version: %v, Network Id: %v", config.ProtocolVersion, config.NetworkId)

	if !config.SkipBcVersionCheck {
		b, _ := blockDb.Get([]byte("BlockchainVersion"))
		bcVersion := int(common.NewValue(b).Uint())
		if bcVersion != config.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run geth upgradedb.\n", bcVersion, config.BlockChainVersion)
		}
		saveBlockchainVersion(blockDb, config.BlockChainVersion)
	}
	glog.V(logger.Info).Infof("Blockchain DB Version: %d", config.BlockChainVersion)

	eth := &Ethereum{
		shutdownChan:    make(chan bool),
		databasesClosed: make(chan bool),
		blockDb:         blockDb,
		stateDb:         stateDb,
		extraDb:         extraDb,
		eventMux:        &event.TypeMux{},
		accountManager:  config.AccountManager,
		DataDir:         config.DataDir,
		etherbase:       common.HexToAddress(config.Etherbase),
		clientVersion:   config.Name, // TODO should separate from Name
		ethVersionId:    config.ProtocolVersion,
		netVersionId:    config.NetworkId,
		NatSpec:         config.NatSpec,
		MinerThreads:    config.MinerThreads,
		SolcPath:        config.SolcPath,
		AutoDAG:         config.AutoDAG,
	}

	eth.pow = ethash.New()
	genesis := core.GenesisBlock(uint64(config.GenesisNonce), stateDb)
	eth.chainManager, err = core.NewChainManager(genesis, blockDb, stateDb, eth.pow, eth.EventMux())
	if err != nil {
		return nil, err
	}
	eth.downloader = downloader.New(eth.EventMux(), eth.chainManager.HasBlock, eth.chainManager.GetBlock)
	eth.txPool = core.NewTxPool(eth.EventMux(), eth.chainManager.State, eth.chainManager.GasLimit)

	eth.blockProcessor = core.NewBlockProcessor(stateDb, extraDb, eth.pow, eth.chainManager, eth.EventMux())
	eth.chainManager.SetProcessor(eth.blockProcessor)
	eth.miner = miner.New(eth, eth.EventMux(), eth.pow)
	eth.miner.SetGasPrice(config.GasPrice)

	eth.protocolManager = NewProtocolManager(config.ProtocolVersion, config.NetworkId, eth.eventMux, eth.txPool, eth.chainManager, eth.downloader)
	if config.Shh {
		eth.whisper = whisper.New()
		eth.shhVersionId = int(eth.whisper.Version())
	}

	netprv, err := config.nodeKey()
	if err != nil {
		return nil, err
	}
	protocols := []p2p.Protocol{eth.protocolManager.SubProtocol}
	if config.Shh {
		protocols = append(protocols, eth.whisper.Protocol())
	}
	eth.net = &p2p.Server{
		PrivateKey:      netprv,
		Name:            config.Name,
		MaxPeers:        config.MaxPeers,
		MaxPendingPeers: config.MaxPendingPeers,
		Discovery:       config.Discovery,
		Protocols:       protocols,
		NAT:             config.NAT,
		NoDial:          !config.Dial,
		BootstrapNodes:  config.parseBootNodes(),
		StaticNodes:     config.parseNodes(staticNodes),
		TrustedNodes:    config.parseNodes(trustedNodes),
		NodeDatabase:    nodeDb,
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
		DiscPort:   int(node.UDP),
		TCPPort:    int(node.TCP),
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
}

func (s *Ethereum) StartMining(threads int) error {
	eb, err := s.Etherbase()
	if err != nil {
		err = fmt.Errorf("Cannot start mining without etherbase address: %v", err)
		glog.V(logger.Error).Infoln(err)
		return err
	}

	go s.miner.Start(eb, threads)
	return nil
}

func (s *Ethereum) Etherbase() (eb common.Address, err error) {
	eb = s.etherbase
	if (eb == common.Address{}) {
		primary, err := s.accountManager.Primary()
		if err != nil {
			return eb, err
		}
		if (primary == common.Address{}) {
			err = fmt.Errorf("no accounts found")
			return eb, err
		}
		eb = primary
	}
	return eb, nil
}

func (s *Ethereum) StopMining()         { s.miner.Stop() }
func (s *Ethereum) IsMining() bool      { return s.miner.Mining() }
func (s *Ethereum) Miner() *miner.Miner { return s.miner }

// func (s *Ethereum) Logger() logger.LogSystem             { return s.logger }
func (s *Ethereum) Name() string                         { return s.net.Name }
func (s *Ethereum) AccountManager() *accounts.Manager    { return s.accountManager }
func (s *Ethereum) ChainManager() *core.ChainManager     { return s.chainManager }
func (s *Ethereum) BlockProcessor() *core.BlockProcessor { return s.blockProcessor }
func (s *Ethereum) TxPool() *core.TxPool                 { return s.txPool }
func (s *Ethereum) Whisper() *whisper.Whisper            { return s.whisper }
func (s *Ethereum) EventMux() *event.TypeMux             { return s.eventMux }
func (s *Ethereum) BlockDb() common.Database             { return s.blockDb }
func (s *Ethereum) StateDb() common.Database             { return s.stateDb }
func (s *Ethereum) ExtraDb() common.Database             { return s.extraDb }
func (s *Ethereum) IsListening() bool                    { return true } // Always listening
func (s *Ethereum) PeerCount() int                       { return s.net.PeerCount() }
func (s *Ethereum) Peers() []*p2p.Peer                   { return s.net.Peers() }
func (s *Ethereum) MaxPeers() int                        { return s.net.MaxPeers }
func (s *Ethereum) ClientVersion() string                { return s.clientVersion }
func (s *Ethereum) EthVersion() int                      { return s.ethVersionId }
func (s *Ethereum) NetVersion() int                      { return s.netVersionId }
func (s *Ethereum) ShhVersion() int                      { return s.shhVersionId }
func (s *Ethereum) Downloader() *downloader.Downloader   { return s.downloader }

// Start the ethereum
func (s *Ethereum) Start() error {
	jsonlogger.LogJson(&logger.LogStarting{
		ClientString:    s.net.Name,
		ProtocolVersion: ProtocolVersion,
	})
	err := s.net.Start()
	if err != nil {
		return err
	}
	// periodically flush databases
	go s.syncDatabases()

	if s.AutoDAG {
		s.StartAutoDAG()
	}

	s.protocolManager.Start()

	if s.whisper != nil {
		s.whisper.Start()
	}

	glog.V(logger.Info).Infoln("Server started")
	return nil
}

// sync databases every minute. If flushing fails we exit immediatly. The system
// may not continue under any circumstances.
func (s *Ethereum) syncDatabases() {
	ticker := time.NewTicker(1 * time.Minute)
done:
	for {
		select {
		case <-ticker.C:
			// don't change the order of database flushes
			if err := s.extraDb.Flush(); err != nil {
				glog.Fatalf("fatal error: flush extraDb: %v (Restart your node. We are aware of this issue)\n", err)
			}
			if err := s.stateDb.Flush(); err != nil {
				glog.Fatalf("fatal error: flush stateDb: %v (Restart your node. We are aware of this issue)\n", err)
			}
			if err := s.blockDb.Flush(); err != nil {
				glog.Fatalf("fatal error: flush blockDb: %v (Restart your node. We are aware of this issue)\n", err)
			}
		case <-s.shutdownChan:
			break done
		}
	}

	s.blockDb.Close()
	s.stateDb.Close()
	s.extraDb.Close()

	close(s.databasesClosed)
}

func (s *Ethereum) StartForTest() {
	jsonlogger.LogJson(&logger.LogStarting{
		ClientString:    s.net.Name,
		ProtocolVersion: ProtocolVersion,
	})
}

// AddPeer connects to the given node and maintains the connection until the
// server is shut down. If the connection fails for any reason, the server will
// attempt to reconnect the peer.
func (self *Ethereum) AddPeer(nodeURL string) error {
	n, err := discover.ParseNode(nodeURL)
	if err != nil {
		return fmt.Errorf("invalid node URL: %v", err)
	}
	self.net.AddPeer(n)
	return nil
}

func (s *Ethereum) Stop() {
	s.net.Stop()
	s.chainManager.Stop()
	s.protocolManager.Stop()
	s.txPool.Stop()
	s.eventMux.Stop()
	if s.whisper != nil {
		s.whisper.Stop()
	}
	s.StopAutoDAG()

	close(s.shutdownChan)
}

// This function will wait for a shutdown and resumes main thread execution
func (s *Ethereum) WaitForShutdown() {
	<-s.databasesClosed
	<-s.shutdownChan
}

// StartAutoDAG() spawns a go routine that checks the DAG every autoDAGcheckInterval
// by default that is 10 times per epoch
// in epoch n, if we past autoDAGepochHeight within-epoch blocks,
// it calls ethash.MakeDAG  to pregenerate the DAG for the next epoch n+1
// if it does not exist yet as well as remove the DAG for epoch n-1
// the loop quits if autodagquit channel is closed, it can safely restart and
// stop any number of times.
// For any more sophisticated pattern of DAG generation, use CLI subcommand
// makedag
func (self *Ethereum) StartAutoDAG() {
	if self.autodagquit != nil {
		return // already started
	}
	go func() {
		glog.V(logger.Info).Infof("Automatic pregeneration of ethash DAG ON (ethash dir: %s)", ethash.DefaultDir)
		var nextEpoch uint64
		timer := time.After(0)
		self.autodagquit = make(chan bool)
		for {
			select {
			case <-timer:
				glog.V(logger.Info).Infof("checking DAG (ethash dir: %s)", ethash.DefaultDir)
				currentBlock := self.ChainManager().CurrentBlock().NumberU64()
				thisEpoch := currentBlock / epochLength
				if nextEpoch <= thisEpoch {
					if currentBlock%epochLength > autoDAGepochHeight {
						if thisEpoch > 0 {
							previousDag, previousDagFull := dagFiles(thisEpoch - 1)
							os.Remove(filepath.Join(ethash.DefaultDir, previousDag))
							os.Remove(filepath.Join(ethash.DefaultDir, previousDagFull))
							glog.V(logger.Info).Infof("removed DAG for epoch %d (%s)", thisEpoch-1, previousDag)
						}
						nextEpoch = thisEpoch + 1
						dag, _ := dagFiles(nextEpoch)
						if _, err := os.Stat(dag); os.IsNotExist(err) {
							glog.V(logger.Info).Infof("Pregenerating DAG for epoch %d (%s)", nextEpoch, dag)
							err := ethash.MakeDAG(nextEpoch*epochLength, "") // "" -> ethash.DefaultDir
							if err != nil {
								glog.V(logger.Error).Infof("Error generating DAG for epoch %d (%s)", nextEpoch, dag)
								return
							}
						} else {
							glog.V(logger.Error).Infof("DAG for epoch %d (%s)", nextEpoch, dag)
						}
					}
				}
				timer = time.After(autoDAGcheckInterval)
			case <-self.autodagquit:
				return
			}
		}
	}()
}

// dagFiles(epoch) returns the two alternative DAG filenames (not a path)
// 1) <revision>-<hex(seedhash[8])> 2) full-R<revision>-<hex(seedhash[8])>
func dagFiles(epoch uint64) (string, string) {
	seedHash, _ := ethash.GetSeedHash(epoch * epochLength)
	dag := fmt.Sprintf("full-R%d-%x", ethashRevision, seedHash[:8])
	return dag, "full-R" + dag
}

// stopAutoDAG stops automatic DAG pregeneration by quitting the loop
func (self *Ethereum) StopAutoDAG() {
	if self.autodagquit != nil {
		close(self.autodagquit)
		self.autodagquit = nil
	}
	glog.V(logger.Info).Infof("Automatic pregeneration of ethash DAG OFF (ethash dir: %s)", ethash.DefaultDir)
}

func saveProtocolVersion(db common.Database, protov int) {
	d, _ := db.Get([]byte("ProtocolVersion"))
	protocolVersion := common.NewValue(d).Uint()

	if protocolVersion == 0 {
		db.Put([]byte("ProtocolVersion"), common.NewValue(protov).Bytes())
	}
}

func saveBlockchainVersion(db common.Database, bcVersion int) {
	d, _ := db.Get([]byte("BlockchainVersion"))
	blockchainVersion := common.NewValue(d).Uint()

	if blockchainVersion == 0 {
		db.Put([]byte("BlockchainVersion"), common.NewValue(bcVersion).Bytes())
	}
}

func (self *Ethereum) Solc() (*compiler.Solidity, error) {
	var err error
	if self.solc == nil {
		self.solc, err = compiler.New(self.SolcPath)
	}
	return self.solc, err
}

// set in js console via admin interface or wrapper from cli flags
func (self *Ethereum) SetSolc(solcPath string) (*compiler.Solidity, error) {
	self.SolcPath = solcPath
	self.solc = nil
	return self.Solc()
}
