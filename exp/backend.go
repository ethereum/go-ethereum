// Copyright 2014 The go-ethereum Authors && Copyright 2015 go-expanse Authors
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
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
	
	"github.com/expanse-org/ethash"
	"github.com/expanse-project/go-expanse/accounts"
	"github.com/expanse-project/go-expanse/common"
	"github.com/expanse-project/go-expanse/common/compiler"
	"github.com/expanse-project/go-expanse/common/httpclient"
	"github.com/expanse-project/go-expanse/core"
	"github.com/expanse-project/go-expanse/core/state"
	"github.com/expanse-project/go-expanse/core/types"
	"github.com/expanse-project/go-expanse/core/vm"
	"github.com/expanse-project/go-expanse/crypto"
	"github.com/expanse-project/go-expanse/eth/downloader"
	"github.com/expanse-project/go-expanse/ethdb"
	"github.com/expanse-project/go-expanse/event"
	"github.com/expanse-project/go-expanse/logger"
	"github.com/expanse-project/go-expanse/logger/glog"
	"github.com/expanse-project/go-expanse/miner"
	"github.com/expanse-project/go-expanse/p2p"
	"github.com/expanse-project/go-expanse/p2p/discover"
	"github.com/expanse-project/go-expanse/p2p/nat"
	"github.com/expanse-project/go-expanse/rlp"
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

	datadirInUseErrNos = []uint{11, 32, 35}
	portInUseErrRE     = regexp.MustCompile("address already in use")

	defaultBootNodes = []*discover.Node{
		// EXP/DEV Go Bootnodes
		discover.MustParseNode("enode://f944c6702a78a0cbcd6505b76daff069dad2e45ff88896c475da2bef47091c88e5b4042211233e397ad958be998003a2674151e60719c5fdeeff5f8cc2c231a1@74.196.59.103:42786"),
		discover.MustParseNode("enode://4055ec69e53df4bfecb95e3b65c28e4f2a1145a3bdc4d85d077b552248cf159951afd649f044783bebf48b902fbc0e96978c76236fd4ab3d5ef7d95d72b84ee5@45.55.217.136:42786"),
		discover.MustParseNode("enode://68c545b62f060dc78d4bbb9fe65cfd5979dede3c2f73fcbba9bac7fb9d1cff70e77b39cce4fa5dfdeb0d064c82db1ad8acee3915fb41f45d0a42000f92c1fd73@192.3.54.134:42786"),
        discover.MustParseNode("enode://753d7d97ffc944edf42b676731b28e059d669484eb16e4526778f83e72d35922dee01b2967722e640a4a210e923aecdb4e4962b2d70fd2ca5dc2d911590e0737@192.3.149.110:42786"),
	}

	defaultTestNetBootNodes = []*discover.Node{
		discover.MustParseNode("enode://e4533109cc9bd7604e4ff6c095f7a1d807e15b38e9bfeb05d3b7c423ba86af0a9e89abbf40bd9dde4250fef114cd09270fa4e224cbeef8b7bf05a51e8260d6b8@94.242.229.4:40404"),
		discover.MustParseNode("enode://8c336ee6f03e99613ad21274f269479bf4413fb294d697ef15ab897598afb931f56beb8e97af530aee20ce2bcba5776f4a312bc168545de4d43736992c814592@94.242.229.203:30303"),
	}

	staticNodes  = "static-nodes.json"  // Path within <datadir> to search for the static node list
	trustedNodes = "trusted-nodes.json" // Path within <datadir> to search for the trusted node list
)

type Config struct {
	DevMode bool
	TestNet bool

	Name         string
	NetworkId    int
	GenesisFile  string
	GenesisBlock *types.Block // used by block tests
	FastSync     bool
	Olympic      bool

	BlockChainVersion  int
	SkipBcVersionCheck bool // e.g. blockchain export
	DatabaseCache      int

	DataDir   string
	LogFile   string
	Verbosity int
	VmDebug   bool
	NatSpec   bool
	DocRoot   string
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
	NewDB func(path string) (ethdb.Database, error)
}

func (cfg *Config) parseBootNodes() []*discover.Node {
	if cfg.BootNodes == "" {
		if cfg.TestNet {
			return defaultTestNetBootNodes
		}

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
	chainDb ethdb.Database // Block chain database
	dappDb  ethdb.Database // Dapp database

	//*** SERVICES ***
	// State manager for processing new blocks and managing the over all states
	blockProcessor  *core.BlockProcessor
	txPool          *core.TxPool
	blockchain      *core.BlockChain
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

	httpclient *httpclient.HTTPClient

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
	logger.New(config.DataDir, config.LogFile, config.Verbosity)

	// Let the database take 3/4 of the max open files (TODO figure out a way to get the actual limit of the open files)
	const dbCount = 3
	ethdb.OpenFileLimit = 128 / (dbCount + 1)

	newdb := config.NewDB
	if newdb == nil {
		newdb = func(path string) (ethdb.Database, error) { return ethdb.NewLDBDatabase(path, config.DatabaseCache) }
	}

	// Open the chain database and perform any upgrades needed
	chainDb, err := newdb(filepath.Join(config.DataDir, "chaindata"))
	if err != nil {
		var ok bool
		errno := uint(err.(syscall.Errno))
		for _, no := range datadirInUseErrNos {
			if errno == no {
				ok = true
				break
			}
		}
		if ok {
			err = fmt.Errorf("%v (check if another instance of geth is already running with the same data directory '%s')", err, config.DataDir)
		}
		return nil, fmt.Errorf("blockchain db err: %v", err)
	}
	if db, ok := chainDb.(*ethdb.LDBDatabase); ok {
		db.Meter("eth/db/chaindata/")
	}
	if err := upgradeChainDatabase(chainDb); err != nil {
		return nil, err
	}
	if err := addMipmapBloomBins(chainDb); err != nil {
		return nil, err
	}

	dappDb, err := newdb(filepath.Join(config.DataDir, "dapp"))
	if err != nil {
		var ok bool
		for _, no := range datadirInUseErrNos {
			if uint(err.(syscall.Errno)) == no {
				ok = true
				break
			}
		}
		if ok {
			err = fmt.Errorf("%v (check if another instance of geth is already running with the same data directory '%s')", err, config.DataDir)
		}
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

	// different modes
	switch {
	case config.Olympic:
		glog.V(logger.Error).Infoln("Starting Olympic network")
		fallthrough
	case config.DevMode:
		_, err := core.WriteOlympicGenesisBlock(chainDb, 42)
		if err != nil {
			return nil, err
		}
	case config.TestNet:
		state.StartingNonce = 1048576 // (2**20)
		_, err := core.WriteTestNetGenesisBlock(chainDb, 0x6d6f7264656e)
		if err != nil {
			return nil, err
		}
	}
	// This is for testing only.
	if config.GenesisBlock != nil {
		core.WriteTd(chainDb, config.GenesisBlock.Hash(), config.GenesisBlock.Difficulty())
		core.WriteBlock(chainDb, config.GenesisBlock)
		core.WriteCanonicalHash(chainDb, config.GenesisBlock.Hash(), config.GenesisBlock.NumberU64())
		core.WriteHeadBlockHash(chainDb, config.GenesisBlock.Hash())
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
		httpclient:              httpclient.New(config.DocRoot),
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
	exp.blockchain, err = core.NewBlockChain(chainDb, exp.pow, exp.EventMux())
	if err != nil {
		if err == core.ErrNoGenesis {
			return nil, fmt.Errorf(`Genesis block not found. Please supply a genesis block with the "--genesis /path/to/file" argument`)
		}
		return nil, err
	}
	newPool := core.NewTxPool(exp.EventMux(), exp.blockchain.State, exp.blockchain.GasLimit)
	exp.txPool = newPool

	exp.blockProcessor = core.NewBlockProcessor(chainDb, exp.pow, exp.blockchain, exp.EventMux())
	exp.blockchain.SetProcessor(exp.blockProcessor)
	if exp.protocolManager, err = NewProtocolManager(config.FastSync, config.NetworkId, exp.eventMux, exp.txPool, exp.pow, exp.blockchain, chainDb); err != nil {
		return nil, err
	}
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
		Td:         s.BlockChain().GetTd(s.BlockChain().CurrentBlock().Hash()).String(),
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
	s.blockchain.ResetWithGenesisBlock(gb)
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
func (s *Expanse) BlockChain() *core.BlockChain         { return s.blockchain }
func (s *Expanse) BlockProcessor() *core.BlockProcessor { return s.blockProcessor }
func (s *Expanse) TxPool() *core.TxPool                 { return s.txPool }
func (s *Expanse) Whisper() *whisper.Whisper            { return s.whisper }
func (s *Expanse) EventMux() *event.TypeMux             { return s.eventMux }
func (s *Expanse) ChainDb() ethdb.Database              { return s.chainDb }
func (s *Expanse) DappDb() ethdb.Database               { return s.dappDb }
func (s *Expanse) IsListening() bool                    { return true } // Always listening
func (s *Expanse) PeerCount() int                       { return s.net.PeerCount() }
func (s *Expanse) Peers() []*p2p.Peer                   { return s.net.Peers() }
func (s *Expanse) MaxPeers() int                        { return s.net.MaxPeers }
func (s *Expanse) ClientVersion() string                { return s.clientVersion }
func (s *Expanse) EthVersion() int                      { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *Expanse) NetVersion() int                      { return s.netVersionId }
func (s *Expanse) ShhVersion() int                      { return s.shhVersionId }
func (s *Expanse) Downloader() *downloader.Downloader   { return s.protocolManager.downloader }

// Start the ethereum
func (s *Expanse) Start() error {
	jsonlogger.LogJson(&logger.LogStarting{
		ClientString:    s.net.Name,
		ProtocolVersion: s.EthVersion(),
	})
	err := s.net.Start()
	if err != nil {
		if portInUseErrRE.MatchString(err.Error()) {
			err = fmt.Errorf("%v (possibly another instance of geth is using the same port)", err)
		}
		return err
	}

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
	s.blockchain.Stop()
	s.protocolManager.Stop()
	s.txPool.Stop()
	s.eventMux.Stop()
	if s.whisper != nil {
		s.whisper.Stop()
	}
	s.StopAutoDAG()

	s.chainDb.Close()
	s.dappDb.Close()
	close(s.shutdownChan)
}

// This function will wait for a shutdown and resumes main thread execution
func (s *Expanse) WaitForShutdown() {
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
				currentBlock := self.BlockChain().CurrentBlock().NumberU64()
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


// HTTPClient returns the light http client used for fetching offchain docs
// (natspec, source for verification)
func (self *Expanse) HTTPClient() *httpclient.HTTPClient {
	return self.httpclient
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

func saveBlockchainVersion(db ethdb.Database, bcVersion int) {
	d, _ := db.Get([]byte("BlockchainVersion"))
	blockchainVersion := common.NewValue(d).Uint()

	if blockchainVersion == 0 {
		db.Put([]byte("BlockchainVersion"), common.NewValue(bcVersion).Bytes())
	}
}

// upgradeChainDatabase ensures that the chain database stores block split into
// separate header and body entries.
func upgradeChainDatabase(db ethdb.Database) error {
	// Short circuit if the head block is stored already as separate header and body
	data, err := db.Get([]byte("LastBlock"))
	if err != nil {
		return nil
	}
	head := common.BytesToHash(data)

	if block := core.GetBlockByHashOld(db, head); block == nil {
		return nil
	}
	// At least some of the database is still the old format, upgrade (skip the head block!)
	glog.V(logger.Info).Info("Old database detected, upgrading...")

	if db, ok := db.(*ethdb.LDBDatabase); ok {
		blockPrefix := []byte("block-hash-")
		for it := db.NewIterator(); it.Next(); {
			// Skip anything other than a combined block
			if !bytes.HasPrefix(it.Key(), blockPrefix) {
				continue
			}
			// Skip the head block (merge last to signal upgrade completion)
			if bytes.HasSuffix(it.Key(), head.Bytes()) {
				continue
			}
			// Load the block, split and serialize (order!)
			block := core.GetBlockByHashOld(db, common.BytesToHash(bytes.TrimPrefix(it.Key(), blockPrefix)))

			if err := core.WriteTd(db, block.Hash(), block.DeprecatedTd()); err != nil {
				return err
			}
			if err := core.WriteBody(db, block.Hash(), &types.Body{block.Transactions(), block.Uncles()}); err != nil {
				return err
			}
			if err := core.WriteHeader(db, block.Header()); err != nil {
				return err
			}
			if err := db.Delete(it.Key()); err != nil {
				return err
			}
		}
		// Lastly, upgrade the head block, disabling the upgrade mechanism
		current := core.GetBlockByHashOld(db, head)

		if err := core.WriteTd(db, current.Hash(), current.DeprecatedTd()); err != nil {
			return err
		}
		if err := core.WriteBody(db, current.Hash(), &types.Body{current.Transactions(), current.Uncles()}); err != nil {
			return err
		}
		if err := core.WriteHeader(db, current.Header()); err != nil {
			return err
		}
	}
	return nil
}

func addMipmapBloomBins(db ethdb.Database) (err error) {
	const mipmapVersion uint = 2

	// check if the version is set. We ignore data for now since there's
	// only one version so we can easily ignore it for now
	var data []byte
	data, _ = db.Get([]byte("setting-mipmap-version"))
	if len(data) > 0 {
		var version uint
		if err := rlp.DecodeBytes(data, &version); err == nil && version == mipmapVersion {
			return nil
		}
	}

	defer func() {
		if err == nil {
			var val []byte
			val, err = rlp.EncodeToBytes(mipmapVersion)
			if err == nil {
				err = db.Put([]byte("setting-mipmap-version"), val)
			}
			return
		}
	}()
	latestBlock := core.GetBlock(db, core.GetHeadBlockHash(db))
	if latestBlock == nil { // clean database
		return
	}

	tstart := time.Now()
	glog.V(logger.Info).Infoln("upgrading db log bloom bins")
	for i := uint64(0); i <= latestBlock.NumberU64(); i++ {
		hash := core.GetCanonicalHash(db, i)
		if (hash == common.Hash{}) {
			return fmt.Errorf("chain db corrupted. Could not find block %d.", i)
		}
		core.WriteMipmapBloom(db, i, core.GetBlockReceipts(db, hash))
	}
	glog.V(logger.Info).Infoln("upgrade completed in", time.Since(tstart))
	return nil
}
