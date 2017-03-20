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
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/expanse-org/ethash"
	"github.com/expanse-org/go-expanse/accounts"
	"github.com/expanse-org/go-expanse/common"
	"github.com/expanse-org/go-expanse/common/compiler"
	"github.com/expanse-org/go-expanse/common/httpclient"
	"github.com/expanse-org/go-expanse/common/registrar/ethreg"
	"github.com/expanse-org/go-expanse/core"
	"github.com/expanse-org/go-expanse/core/types"
	"github.com/expanse-org/go-expanse/core/vm"
	"github.com/expanse-org/go-expanse/exp/downloader"
	"github.com/expanse-org/go-expanse/exp/filters"
	"github.com/expanse-org/go-expanse/ethdb"
	"github.com/expanse-org/go-expanse/event"
	"github.com/expanse-org/go-expanse/logger"
	"github.com/expanse-org/go-expanse/logger/glog"
	"github.com/expanse-org/go-expanse/miner"
	"github.com/expanse-org/go-expanse/node"
	"github.com/expanse-org/go-expanse/p2p"
	"github.com/expanse-org/go-expanse/rlp"
	"github.com/expanse-org/go-expanse/rpc"
)

const (
	epochLength    = 30000
	ethashRevision = 23

	autoDAGcheckInterval = 10 * time.Hour
	autoDAGepochHeight   = epochLength / 2
)

var (
	datadirInUseErrnos = map[uint]bool{11: true, 32: true, 35: true}
	portInUseErrRE     = regexp.MustCompile("address already in use")
)

type Config struct {
	ChainConfig *core.ChainConfig // chain configuration

	NetworkId int    // Network ID to use for selecting peers to connect to
	Genesis   string // Genesis JSON to seed the chain database with
	FastSync  bool   // Enables the state download based fast synchronisation algorithm

	BlockChainVersion  int
	SkipBcVersionCheck bool // e.g. blockchain export
	DatabaseCache      int
	DatabaseHandles    int

	NatSpec   bool
	DocRoot   string
	AutoDAG   bool
	PowTest   bool
	PowShared bool
	ExtraData []byte

	AccountManager *accounts.Manager
	Etherbase      common.Address
	GasPrice       *big.Int
	MinerThreads   int
	SolcPath       string

	GpoMinGasPrice          *big.Int
	GpoMaxGasPrice          *big.Int
	GpoFullBlockRatio       int
	GpobaseStepDown         int
	GpobaseStepUp           int
	GpobaseCorrectionFactor int

	EnableJit bool
	ForceJit  bool

	TestGenesisBlock *types.Block   // Genesis block to seed the chain database with (testing only!)
	TestGenesisState ethdb.Database // Genesis state to seed the database with (testing only!)
}

type Expanse struct {
	chainConfig *core.ChainConfig
	// Channel for shutting down the expanse
	shutdownChan chan bool

	// DB interfaces
	chainDb ethdb.Database // Block chain database
	dappDb  ethdb.Database // Dapp database

	// Handlers
	txPool          *core.TxPool
	txMu            sync.Mutex
	blockchain      *core.BlockChain
	accountManager  *accounts.Manager
	pow             *ethash.Ethash
	protocolManager *ProtocolManager
	SolcPath        string
	solc            *compiler.Solidity
	gpo             *GasPriceOracle

	GpoMinGasPrice          *big.Int
	GpoMaxGasPrice          *big.Int
	GpoFullBlockRatio       int
	GpobaseStepDown         int
	GpobaseStepUp           int
	GpobaseCorrectionFactor int

	httpclient *httpclient.HTTPClient

	eventMux *event.TypeMux
	miner    *miner.Miner

	Mining        bool
	MinerThreads  int
	NatSpec       bool
	AutoDAG       bool
	PowTest       bool
	autodagquit   chan bool
	etherbase     common.Address
	netVersionId  int
	netRPCService *PublicNetAPI
}

func New(ctx *node.ServiceContext, config *Config) (*Expanse, error) {
	// Open the chain database and perform any upgrades needed
	chainDb, err := ctx.OpenDatabase("chaindata", config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := chainDb.(*ethdb.LDBDatabase); ok {
		db.Meter("exp/db/chaindata/")
	}
	if err := upgradeChainDatabase(chainDb); err != nil {
		return nil, err
	}
	if err := addMipmapBloomBins(chainDb); err != nil {
		return nil, err
	}

	dappDb, err := ctx.OpenDatabase("dapp", config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := dappDb.(*ethdb.LDBDatabase); ok {
		db.Meter("exp/db/dapp/")
	}
	glog.V(logger.Info).Infof("Protocol Versions: %v, Network Id: %v", ProtocolVersions, config.NetworkId)

	// Load up any custom genesis block if requested
	if len(config.Genesis) > 0 {
		block, err := core.WriteGenesisBlock(chainDb, strings.NewReader(config.Genesis))
		if err != nil {
			return nil, err
		}
		glog.V(logger.Info).Infof("Successfully wrote custom genesis block: %x", block.Hash())
	}

	// Load up a test setup if directly injected
	if config.TestGenesisState != nil {
		chainDb = config.TestGenesisState
	}
	if config.TestGenesisBlock != nil {
		core.WriteTd(chainDb, config.TestGenesisBlock.Hash(), config.TestGenesisBlock.Difficulty())
		core.WriteBlock(chainDb, config.TestGenesisBlock)
		core.WriteCanonicalHash(chainDb, config.TestGenesisBlock.Hash(), config.TestGenesisBlock.NumberU64())
		core.WriteHeadBlockHash(chainDb, config.TestGenesisBlock.Hash())
	}

	if !config.SkipBcVersionCheck {
		bcVersion := core.GetBlockChainVersion(chainDb)
		if bcVersion != config.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run gexp upgradedb.\n", bcVersion, config.BlockChainVersion)
		}
		core.WriteBlockChainVersion(chainDb, config.BlockChainVersion)
	}
	glog.V(logger.Info).Infof("Blockchain DB Version: %d", config.BlockChainVersion)

	exp := &Expanse{
		shutdownChan:            make(chan bool),
		chainDb:                 chainDb,
		dappDb:                  dappDb,
		eventMux:                ctx.EventMux,
		accountManager:          config.AccountManager,
		etherbase:               config.Etherbase,
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
	switch {
	case config.PowTest:
		glog.V(logger.Info).Infof("ethash used in test mode")
		exp.pow, err = ethash.NewForTesting()
		if err != nil {
			return nil, err
		}
	case config.PowShared:
		glog.V(logger.Info).Infof("ethash used in shared mode")
		exp.pow = ethash.NewShared()

	default:
		exp.pow = ethash.New()
	}

	// load the genesis block or write a new one if no genesis
	// block is prenent in the database.
	genesis := core.GetBlock(chainDb, core.GetCanonicalHash(chainDb, 0))
	if genesis == nil {
		genesis, err = core.WriteDefaultGenesisBlock(chainDb)
		if err != nil {
			return nil, err
		}
		glog.V(logger.Info).Infoln("WARNING: Wrote default expanse genesis block")
	}

	if config.ChainConfig == nil {
		return nil, errors.New("missing chain config")
	}
	core.WriteChainConfig(chainDb, genesis.Hash(), config.ChainConfig)

	exp.chainConfig = config.ChainConfig
	exp.chainConfig.VmConfig = vm.Config{
		EnableJit: config.EnableJit,
		ForceJit:  config.ForceJit,
	}

	exp.blockchain, err = core.NewBlockChain(chainDb, exp.chainConfig, exp.pow, exp.EventMux())
	if err != nil {
		if err == core.ErrNoGenesis {
			return nil, fmt.Errorf(`No chain found. Please initialise a new chain using the "init" subcommand.`)
		}
		return nil, err
	}
	exp.gpo = NewGasPriceOracle(exp)

	newPool := core.NewTxPool(exp.chainConfig, exp.EventMux(), exp.blockchain.State, exp.blockchain.GasLimit)
	exp.txPool = newPool

	if exp.protocolManager, err = NewProtocolManager(exp.chainConfig, config.FastSync, config.NetworkId, exp.eventMux, exp.txPool, exp.pow, exp.blockchain, chainDb); err != nil {
		return nil, err
	}
	exp.miner = miner.New(exp, exp.chainConfig, exp.EventMux(), exp.pow)
	exp.miner.SetGasPrice(config.GasPrice)
	exp.miner.SetExtra(config.ExtraData)

	return exp, nil
}

// APIs returns the collection of RPC services the expanse package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *Expanse) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "exp",
			Version:   "1.0",
			Service:   NewPublicEthereumAPI(s),
			Public:    true,
		}, {
			Namespace: "exp",
			Version:   "1.0",
			Service:   NewPublicAccountAPI(s.accountManager),
			Public:    true,
		}, {
			Namespace: "personal",
			Version:   "1.0",
			Service:   NewPrivateAccountAPI(s),
			Public:    false,
		}, {
			Namespace: "exp",
			Version:   "1.0",
			Service:   NewPublicBlockChainAPI(s.chainConfig, s.blockchain, s.miner, s.chainDb, s.gpo, s.eventMux, s.accountManager),
			Public:    true,
		}, {
			Namespace: "exp",
			Version:   "1.0",
			Service:   NewPublicTransactionPoolAPI(s),
			Public:    true,
		}, {
			Namespace: "exp",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "exp",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "txpool",
			Version:   "1.0",
			Service:   NewPublicTxPoolAPI(s),
			Public:    true,
		}, {
			Namespace: "exp",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.chainDb, s.eventMux),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   ethreg.NewPrivateRegistarAPI(s.chainConfig, s.blockchain, s.chainDb, s.txPool, s.accountManager),
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicEthereumAPI(s),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicAccountAPI(s.accountManager),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicBlockChainAPI(s.chainConfig, s.blockchain, s.miner, s.chainDb, s.gpo, s.eventMux, s.accountManager),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicTransactionPoolAPI(s),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.chainDb, s.eventMux),
			Public:    true,
		},
	}
}

func (s *Expanse) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *Expanse) Etherbase() (eb common.Address, err error) {
	eb = s.etherbase
	if (eb == common.Address{}) {
		firstAccount, err := s.AccountManager().AccountByIndex(0)
		eb = firstAccount.Address
		if err != nil {
			return eb, fmt.Errorf("etherbase address must be explicitly specified")
		}
	}
	return eb, nil
}

// set in js console via admin interface or wrapper from cli flags
func (self *Expanse) SetEtherbase(etherbase common.Address) {
	self.etherbase = etherbase
	self.miner.SetEtherbase(etherbase)
}

func (s *Expanse) StopMining()         { s.miner.Stop() }
func (s *Expanse) IsMining() bool      { return s.miner.Mining() }
func (s *Expanse) Miner() *miner.Miner { return s.miner }

func (s *Expanse) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *Expanse) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *Expanse) TxPool() *core.TxPool               { return s.txPool }
func (s *Expanse) EventMux() *event.TypeMux           { return s.eventMux }
func (s *Expanse) ChainDb() ethdb.Database            { return s.chainDb }
func (s *Expanse) DappDb() ethdb.Database             { return s.dappDb }
func (s *Expanse) IsListening() bool                  { return true } // Always listening
func (s *Expanse) EthVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *Expanse) NetVersion() int                    { return s.netVersionId }
func (s *Expanse) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *Expanse) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

// Start implements node.Service, starting all internal goroutines needed by the
// Expanse protocol implementation.
func (s *Expanse) Start(srvr *p2p.Server) error {
	if s.AutoDAG {
		s.StartAutoDAG()
	}
	s.protocolManager.Start()
	s.netRPCService = NewPublicNetAPI(srvr, s.NetVersion())
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Expanse protocol.
func (s *Expanse) Stop() error {
	s.blockchain.Stop()
	s.protocolManager.Stop()
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.StopAutoDAG()

	s.chainDb.Close()
	s.dappDb.Close()
	close(s.shutdownChan)

	return nil
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
			if err := core.WriteBody(db, block.Hash(), block.Body()); err != nil {
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
		if err := core.WriteBody(db, current.Hash(), current.Body()); err != nil {
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
