// Copyright 2014 The go-expanse Authors
// This file is part of the go-expanse library.
//
// The go-expanse library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-expanse library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-expanse library. If not, see <http://www.gnu.org/licenses/>.

// Package exp implements the Expanse protocol.
package exp

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

	"github.com/expanse-project/ethash"
	"github.com/expanse-project/go-expanse/accounts"
	"github.com/expanse-project/go-expanse/common"
	"github.com/expanse-project/go-expanse/common/compiler"
	"github.com/expanse-project/go-expanse/core"
	"github.com/expanse-project/go-expanse/core/types"
	"github.com/expanse-project/go-expanse/core/vm"
	"github.com/expanse-project/go-expanse/crypto"
	"github.com/expanse-project/go-expanse/exp/downloader"
	"github.com/expanse-project/go-expanse/ethdb"
	"github.com/expanse-project/go-expanse/event"
	"github.com/expanse-project/go-expanse/logger"
	"github.com/expanse-project/go-expanse/logger/glog"
	"github.com/expanse-project/go-expanse/miner"
	"github.com/expanse-project/go-expanse/p2p"
	"github.com/expanse-project/go-expanse/p2p/discover"
	"github.com/expanse-project/go-expanse/p2p/nat"
	"github.com/expanse-project/go-expanse/whisper"
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
		// EXP/DEV Go Bootnodes
		discover.MustParseNode("enode://f944c6702a78a0cbcd6505b76daff069dad2e45ff88896c475da2bef47091c88e5b4042211233e397ad958be998003a2674151e60719c5fdeeff5f8cc2c231a1@74.196.59.103:42786"),
		discover.MustParseNode("enode://4055ec69e53df4bfecb95e3b65c28e4f2a1145a3bdc4d85d077b552248cf159951afd649f044783bebf48b902fbc0e96978c76236fd4ab3d5ef7d95d72b84ee5@45.55.217.136:42786"),
		discover.MustParseNode("enode://68c545b62f060dc78d4bbb9fe65cfd5979dede3c2f73fcbba9bac7fb9d1cff70e77b39cce4fa5dfdeb0d064c82db1ad8acee3915fb41f45d0a42000f92c1fd73@192.3.54.134:42786"),
        discover.MustParseNode("enode://753d7d97ffc944edf42b676731b28e059d669484eb16e4526778f83e72d35922dee01b2967722e640a4a210e923aecdb4e4962b2d70fd2ca5dc2d911590e0737@192.3.149.110:42786"),
	}

	staticNodes  = "static-nodes.json"  // Path within <datadir> to search for the static node list
	trustedNodes = "trusted-nodes.json" // Path within <datadir> to search for the trusted node list
)

type Config struct {
	Name         string
	NetworkId    int
	GenesisNonce int
	GenesisFile  string
	GenesisBlock *types.Block // used by block tests
	Olympic      bool

	BlockChainVersion  int
	SkipBcVersionCheck bool // e.g. blockchain export
	DatabaseCache      int

	DataDir   string
	LogFile   string
	Verbosity int
	LogJSON   string
	VmDebug   bool
	NatSpec   bool
	AutoDAG   bool
	PowTest   bool
	ExtraData []byte

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

	Etherbase      common.Address
	GasPrice       *big.Int
	MinerThreads   int
	AccountManager *accounts.Manager
	SolcPath       string

	GpoMinGasPrice          *big.Int
	GpoMaxGasPrice          *big.Int
	GpoFullBlockRatio       int
	GpobaseStepDown         int
	GpobaseStepUp           int
	GpobaseCorrectionFactor int

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

type Expanse struct {
	// Channel for shutting down the expanse
	shutdownChan chan bool

	// DB interfaces
	chainDb common.Database // Block chain databe
	dappDb  common.Database // Dapp database

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
	SolcPath        string
	solc            *compiler.Solidity

	GpoMinGasPrice          *big.Int
	GpoMaxGasPrice          *big.Int
	GpoFullBlockRatio       int
	GpobaseStepDown         int
	GpobaseStepUp           int
	GpobaseCorrectionFactor int

	net      *p2p.Server
	eventMux *event.TypeMux
	miner    *miner.Miner

	// logger logger.LogSystem

	Mining        bool
	MinerThreads  int
	NatSpec       bool
	DataDir       string
	AutoDAG       bool
	PowTest       bool
	autodagquit   chan bool
	etherbase     common.Address
	clientVersion string
	netVersionId  int
	shhVersionId  int
}

func New(config *Config) (*Expanse, error) {
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
		newdb = func(path string) (common.Database, error) { return ethdb.NewLDBDatabase(path, config.DatabaseCache) }
	}

	// attempt to merge database together, upgrading from an old version
	if err := mergeDatabases(config.DataDir, newdb); err != nil {
		return nil, err
	}

	chainDb, err := newdb(filepath.Join(config.DataDir, "chaindata"))
	if err != nil {
		return nil, fmt.Errorf("blockchain db err: %v", err)
	}
	if db, ok := chainDb.(*ethdb.LDBDatabase); ok {
		db.Meter("eth/db/chaindata/")
	}
	dappDb, err := newdb(filepath.Join(config.DataDir, "dapp"))
	if err != nil {
		return nil, fmt.Errorf("dapp db err: %v", err)
	}
	if db, ok := dappDb.(*ethdb.LDBDatabase); ok {
		db.Meter("eth/db/dapp/")
	}

	nodeDb := filepath.Join(config.DataDir, "nodes")
	glog.V(logger.Info).Infof("Protocol Versions: %v, Network Id: %v", ProtocolVersions, config.NetworkId)

	if len(config.GenesisFile) > 0 {
		fr, err := os.Open(config.GenesisFile)
		if err != nil {
			return nil, err
		}

		block, err := core.WriteGenesisBlock(chainDb, fr)
		if err != nil {
			return nil, err
		}
		glog.V(logger.Info).Infof("Successfully wrote genesis block. New genesis hash = %x\n", block.Hash())
	}

	if config.Olympic {
		_, err := core.WriteTestNetGenesisBlock(chainDb, 42)
		if err != nil {
			return nil, err
		}
		glog.V(logger.Error).Infoln("Starting Olympic network")
	}

	// This is for testing only.
	if config.GenesisBlock != nil {
		core.WriteBlock(chainDb, config.GenesisBlock)
		core.WriteHead(chainDb, config.GenesisBlock)
	}

	if !config.SkipBcVersionCheck {
		b, _ := chainDb.Get([]byte("BlockchainVersion"))
		bcVersion := int(common.NewValue(b).Uint())
		if bcVersion != config.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run gexp upgradedb.\n", bcVersion, config.BlockChainVersion)
		}
		saveBlockchainVersion(chainDb, config.BlockChainVersion)
	}
	glog.V(logger.Info).Infof("Blockchain DB Version: %d", config.BlockChainVersion)

	exp := &Expanse{
		shutdownChan:            make(chan bool),
		databasesClosed:         make(chan bool),
		chainDb:                 chainDb,
		dappDb:                  dappDb,
		eventMux:                &event.TypeMux{},
		accountManager:          config.AccountManager,
		DataDir:                 config.DataDir,
		etherbase:               config.Etherbase,
		clientVersion:           config.Name, // TODO should separate from Name
		netVersionId:            config.NetworkId,
		NatSpec:                 config.NatSpec,
		MinerThreads:            config.MinerThreads,
		SolcPath:                config.SolcPath,
		AutoDAG:                 config.AutoDAG,
		PowTest:                 config.PowTest,
		GpoMinGasPrice:          config.GpoMinGasPrice,
		GpoMaxGasPrice:          config.GpoMaxGasPrice,
		GpoFullBlockRatio:       config.GpoFullBlockRatio,
		GpobaseStepDown:         config.GpobaseStepDown,
		GpobaseStepUp:           config.GpobaseStepUp,
		GpobaseCorrectionFactor: config.GpobaseCorrectionFactor,
	}

	if config.PowTest {
		glog.V(logger.Info).Infof("ethash used in test mode")
		exp.pow, err = ethash.NewForTesting()
		if err != nil {
			return nil, err
		}
	} else {
		exp.pow = ethash.New()
	}
	//genesis := core.GenesisBlock(uint64(config.GenesisNonce), stateDb)
	exp.chainManager, err = core.NewChainManager(chainDb, exp.pow, exp.EventMux())
	if err != nil {
		if err == core.ErrNoGenesis {
			return nil, fmt.Errorf(`Genesis block not found. Please supply a genesis block with the "--genesis /path/to/file" argument`)
		}

		return nil, err
	}
	exp.txPool = core.NewTxPool(exp.EventMux(), exp.chainManager.State, exp.chainManager.GasLimit)

	exp.blockProcessor = core.NewBlockProcessor(chainDb, exp.pow, exp.chainManager, exp.EventMux())
	exp.chainManager.SetProcessor(exp.blockProcessor)
	exp.protocolManager = NewProtocolManager(config.NetworkId, exp.eventMux, exp.txPool, exp.pow, exp.chainManager)

	exp.miner = miner.New(exp, exp.EventMux(), exp.pow)
	exp.miner.SetGasPrice(config.GasPrice)
	exp.miner.SetExtra(config.ExtraData)

	if config.Shh {
		exp.whisper = whisper.New()
		exp.shhVersionId = int(exp.whisper.Version())
	}

	netprv, err := config.nodeKey()
	if err != nil {
		return nil, err
	}
	protocols := append([]p2p.Protocol{}, exp.protocolManager.SubProtocols...)
	if config.Shh {
		protocols = append(protocols, exp.whisper.Protocol())
	}
	exp.net = &p2p.Server{
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
		exp.net.ListenAddr = ":" + config.Port
	}

	vm.Debug = config.VmDebug

	return exp, nil
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

func (s *Expanse) NodeInfo() *NodeInfo {
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
func (s *Expanse) PeersInfo() (peersinfo []*PeerInfo) {
	for _, peer := range s.net.Peers() {
		if peer != nil {
			peersinfo = append(peersinfo, newPeerInfo(peer))
		}
	}
	return
}

func (s *Expanse) ResetWithGenesisBlock(gb *types.Block) {
	s.chainManager.ResetWithGenesisBlock(gb)
}

func (s *Expanse) StartMining(threads int) error {
	eb, err := s.Etherbase()
	if err != nil {
		err = fmt.Errorf("Cannot start mining without etherbase address: %v", err)
		glog.V(logger.Error).Infoln(err)
		return err
	}

	go s.miner.Start(eb, threads)
	return nil
}

func (s *Expanse) Etherbase() (eb common.Address, err error) {
	eb = s.etherbase
	if (eb == common.Address{}) {
		addr, e := s.AccountManager().AddressByIndex(0)
		if e != nil {
			err = fmt.Errorf("etherbase address must be explicitly specified")
		}
		eb = common.HexToAddress(addr)
	}
	return
}

// set in js console via admin interface or wrapper from cli flags
func (self *Expanse) SetEtherbase(etherbase common.Address) {
	self.etherbase = etherbase
	self.miner.SetEtherbase(etherbase)
}

func (s *Expanse) StopMining()         { s.miner.Stop() }
func (s *Expanse) IsMining() bool      { return s.miner.Mining() }
func (s *Expanse) Miner() *miner.Miner { return s.miner }

// func (s *Expanse) Logger() logger.LogSystem             { return s.logger }
func (s *Expanse) Name() string                         { return s.net.Name }
func (s *Expanse) AccountManager() *accounts.Manager    { return s.accountManager }
func (s *Expanse) ChainManager() *core.ChainManager     { return s.chainManager }
func (s *Expanse) BlockProcessor() *core.BlockProcessor { return s.blockProcessor }
func (s *Expanse) TxPool() *core.TxPool                 { return s.txPool }
func (s *Expanse) Whisper() *whisper.Whisper            { return s.whisper }
func (s *Expanse) EventMux() *event.TypeMux             { return s.eventMux }
func (s *Expanse) ChainDb() common.Database             { return s.chainDb }
func (s *Expanse) DappDb() common.Database              { return s.dappDb }
func (s *Expanse) IsListening() bool                    { return true } // Always listening
func (s *Expanse) PeerCount() int                       { return s.net.PeerCount() }
func (s *Expanse) Peers() []*p2p.Peer                   { return s.net.Peers() }
func (s *Expanse) MaxPeers() int                        { return s.net.MaxPeers }
func (s *Expanse) ClientVersion() string                { return s.clientVersion }
func (s *Expanse) EthVersion() int                      { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *Expanse) NetVersion() int                      { return s.netVersionId }
func (s *Expanse) ShhVersion() int                      { return s.shhVersionId }
func (s *Expanse) Downloader() *downloader.Downloader   { return s.protocolManager.downloader }

// Start the expanse
func (s *Expanse) Start() error {
	jsonlogger.LogJson(&logger.LogStarting{
		ClientString:    s.net.Name,
		ProtocolVersion: s.EthVersion(),
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
func (s *Expanse) syncDatabases() {
	ticker := time.NewTicker(1 * time.Minute)
done:
	for {
		select {
		case <-ticker.C:
			// don't change the order of database flushes
			if err := s.dappDb.Flush(); err != nil {
				glog.Fatalf("fatal error: flush dappDb: %v (Restart your node. We are aware of this issue)\n", err)
			}
			if err := s.chainDb.Flush(); err != nil {
				glog.Fatalf("fatal error: flush chainDb: %v (Restart your node. We are aware of this issue)\n", err)
			}
		case <-s.shutdownChan:
			break done
		}
	}

	s.chainDb.Close()
	s.dappDb.Close()

	close(s.databasesClosed)
}

func (s *Expanse) StartForTest() {
	jsonlogger.LogJson(&logger.LogStarting{
		ClientString:    s.net.Name,
		ProtocolVersion: s.EthVersion(),
	})
}

// AddPeer connects to the given node and maintains the connection until the
// server is shut down. If the connection fails for any reason, the server will
// attempt to reconnect the peer.
func (self *Expanse) AddPeer(nodeURL string) error {
	n, err := discover.ParseNode(nodeURL)
	if err != nil {
		return fmt.Errorf("invalid node URL: %v", err)
	}
	self.net.AddPeer(n)
	return nil
}

func (s *Expanse) Stop() {
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
func (s *Expanse) WaitForShutdown() {
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
func (self *Expanse) StartAutoDAG() {
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

// stopAutoDAG stops automatic DAG pregeneration by quitting the loop
func (self *Expanse) StopAutoDAG() {
	if self.autodagquit != nil {
		close(self.autodagquit)
		self.autodagquit = nil
	}
	glog.V(logger.Info).Infof("Automatic pregeneration of ethash DAG OFF (ethash dir: %s)", ethash.DefaultDir)
}

func (self *Expanse) Solc() (*compiler.Solidity, error) {
	var err error
	if self.solc == nil {
		self.solc, err = compiler.New(self.SolcPath)
	}
	return self.solc, err
}

// set in js console via admin interface or wrapper from cli flags
func (self *Expanse) SetSolc(solcPath string) (*compiler.Solidity, error) {
	self.SolcPath = solcPath
	self.solc = nil
	return self.Solc()
}

// dagFiles(epoch) returns the two alternative DAG filenames (not a path)
// 1) <revision>-<hex(seedhash[8])> 2) full-R<revision>-<hex(seedhash[8])>
func dagFiles(epoch uint64) (string, string) {
	seedHash, _ := ethash.GetSeedHash(epoch * epochLength)
	dag := fmt.Sprintf("full-R%d-%x", ethashRevision, seedHash[:8])
	return dag, "full-R" + dag
}

func saveBlockchainVersion(db common.Database, bcVersion int) {
	d, _ := db.Get([]byte("BlockchainVersion"))
	blockchainVersion := common.NewValue(d).Uint()

	if blockchainVersion == 0 {
		db.Put([]byte("BlockchainVersion"), common.NewValue(bcVersion).Bytes())
	}
}

// mergeDatabases when required merge old database layout to one single database
func mergeDatabases(datadir string, newdb func(path string) (common.Database, error)) error {
	// Check if already upgraded
	data := filepath.Join(datadir, "chaindata")
	if _, err := os.Stat(data); !os.IsNotExist(err) {
		return nil
	}
	// make sure it's not just a clean path
	chainPath := filepath.Join(datadir, "blockchain")
	if _, err := os.Stat(chainPath); os.IsNotExist(err) {
		return nil
	}
	glog.Infoln("Database upgrade required. Upgrading...")

	database, err := newdb(data)
	if err != nil {
		return fmt.Errorf("creating data db err: %v", err)
	}
	defer database.Close()

	// Migrate blocks
	chainDb, err := newdb(chainPath)
	if err != nil {
		return fmt.Errorf("state db err: %v", err)
	}
	defer chainDb.Close()

	if chain, ok := chainDb.(*ethdb.LDBDatabase); ok {
		glog.Infoln("Merging blockchain database...")
		it := chain.NewIterator()
		for it.Next() {
			database.Put(it.Key(), it.Value())
		}
		it.Release()
	}

	// Migrate state
	stateDb, err := newdb(filepath.Join(datadir, "state"))
	if err != nil {
		return fmt.Errorf("state db err: %v", err)
	}
	defer stateDb.Close()

	if state, ok := stateDb.(*ethdb.LDBDatabase); ok {
		glog.Infoln("Merging state database...")
		it := state.NewIterator()
		for it.Next() {
			database.Put(it.Key(), it.Value())
		}
		it.Release()
	}

	// Migrate transaction / receipts
	extraDb, err := newdb(filepath.Join(datadir, "extra"))
	if err != nil {
		return fmt.Errorf("state db err: %v", err)
	}
	defer extraDb.Close()

	if extra, ok := extraDb.(*ethdb.LDBDatabase); ok {
		glog.Infoln("Merging transaction database...")

		it := extra.NewIterator()
		for it.Next() {
			database.Put(it.Key(), it.Value())
		}
		it.Release()
	}

	return nil
}
