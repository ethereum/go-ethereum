// Copyright 2014 The go-burnout Authors
// This file is part of the go-burnout library.
//
// The go-burnout library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-burnout library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-burnout library. If not, see <http://www.gnu.org/licenses/>.

// Package brn implements the Burnout protocol.
package brn

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/burnoutcoin/go-burnout/accounts"
	"github.com/burnoutcoin/go-burnout/common"
	"github.com/burnoutcoin/go-burnout/common/hexutil"
	"github.com/burnoutcoin/go-burnout/consensus"
	"github.com/burnoutcoin/go-burnout/consensus/clique"
	"github.com/burnoutcoin/go-burnout/consensus/ethash"
	"github.com/burnoutcoin/go-burnout/core"
	"github.com/burnoutcoin/go-burnout/core/bloombits"
	"github.com/burnoutcoin/go-burnout/core/types"
	"github.com/burnoutcoin/go-burnout/core/vm"
	"github.com/burnoutcoin/go-burnout/brn/downloader"
	"github.com/burnoutcoin/go-burnout/brn/filters"
	"github.com/burnoutcoin/go-burnout/brn/gasprice"
	"github.com/burnoutcoin/go-burnout/brndb"
	"github.com/burnoutcoin/go-burnout/event"
	"github.com/burnoutcoin/go-burnout/internal/ethapi"
	"github.com/burnoutcoin/go-burnout/log"
	"github.com/burnoutcoin/go-burnout/miner"
	"github.com/burnoutcoin/go-burnout/node"
	"github.com/burnoutcoin/go-burnout/p2p"
	"github.com/burnoutcoin/go-burnout/params"
	"github.com/burnoutcoin/go-burnout/rlp"
	"github.com/burnoutcoin/go-burnout/rpc"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
}

// Burnout implements the Burnout full node service.
type Burnout struct {
	config      *Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan  chan bool    // Channel for shutting down the burnout
	stopDbUpgrade func() error // stop chain db sequential key upgrade

	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer

	// DB interfaces
	chainDb brndb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	ApiBackend *EthApiBackend

	miner     *miner.Miner
	gasPrice  *big.Int
	etherbase common.Address

	networkId     uint64
	netRPCService *ethapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and etherbase)
}

func (s *Burnout) AddLesServer(ls LesServer) {
	s.lesServer = ls
}

// New creates a new Burnout object (including the
// initialisation of the common Burnout object)
func New(ctx *node.ServiceContext, config *Config) (*Burnout, error) {
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run brn.Burnout in light sync mode, use les.LightBurnout")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}
	stopDbUpgrade := upgradeDeduplicateData(chainDb)
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	brn := &Burnout{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, config, chainConfig, chainDb),
		shutdownChan:   make(chan bool),
		stopDbUpgrade:  stopDbUpgrade,
		networkId:      config.NetworkId,
		gasPrice:       config.GasPrice,
		etherbase:      config.Etherbase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks),
	}

	log.Info("Initialising Burnout protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	if !config.SkipBcVersionCheck {
		bcVersion := core.GetBlockChainVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run geth upgradedb.\n", bcVersion, core.BlockChainVersion)
		}
		core.WriteBlockChainVersion(chainDb, core.BlockChainVersion)
	}

	vmConfig := vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
	brn.blockchain, err = core.NewBlockChain(chainDb, brn.chainConfig, brn.engine, vmConfig)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		brn.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	brn.bloomIndexer.Start(brn.blockchain.CurrentHeader(), brn.blockchain.SubscribeChainEvent)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	brn.txPool = core.NewTxPool(config.TxPool, brn.chainConfig, brn.blockchain)

	if brn.protocolManager, err = NewProtocolManager(brn.chainConfig, config.SyncMode, config.NetworkId, brn.eventMux, brn.txPool, brn.engine, brn.blockchain, chainDb); err != nil {
		return nil, err
	}
	brn.miner = miner.New(brn, brn.chainConfig, brn.EventMux(), brn.engine)
	brn.miner.SetExtra(makeExtraData(config.ExtraData))

	brn.ApiBackend = &EthApiBackend{brn, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	brn.ApiBackend.gpo = gasprice.NewOracle(brn.ApiBackend, gpoParams)

	return brn, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"geth",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (brndb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*brndb.LDBDatabase); ok {
		db.Meter("brn/db/chaindata/")
	}
	return db, nil
}

// CreateConsensusEngine creates the required type of consensus engine instance for an Burnout service
func CreateConsensusEngine(ctx *node.ServiceContext, config *Config, chainConfig *params.ChainConfig, db brndb.Database) consensus.Engine {
	// If proof-of-authority is requested, set it up
	if chainConfig.Clique != nil {
		return clique.New(chainConfig.Clique, db)
	}
	// Otherwise assume proof-of-work
	switch {
	case config.PowFake:
		log.Warn("Ethash used in fake mode")
		return ethash.NewFaker()
	case config.PowTest:
		log.Warn("Ethash used in test mode")
		return ethash.NewTester()
	case config.PowShared:
		log.Warn("Ethash used in shared mode")
		return ethash.NewShared()
	default:
		engine := ethash.New(ctx.ResolvePath(config.EthashCacheDir), config.EthashCachesInMem, config.EthashCachesOnDisk,
			config.EthashDatasetDir, config.EthashDatasetsInMem, config.EthashDatasetsOnDisk)
		engine.SetThreads(-1) // Disable CPU mining
		return engine
	}
}

// APIs returns the collection of RPC services the burnout package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *Burnout) APIs() []rpc.API {
	apis := ethapi.GetAPIs(s.ApiBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "brn",
			Version:   "1.0",
			Service:   NewPublicBurnoutAPI(s),
			Public:    true,
		}, {
			Namespace: "brn",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "brn",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "brn",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, false),
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
		},
	}...)
}

func (s *Burnout) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *Burnout) Etherbase() (eb common.Address, err error) {
	s.lock.RLock()
	etherbase := s.etherbase
	s.lock.RUnlock()

	if etherbase != (common.Address{}) {
		return etherbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			return accounts[0].Address, nil
		}
	}
	return common.Address{}, fmt.Errorf("etherbase address must be explicitly specified")
}

// set in js console via admin interface or wrapper from cli flags
func (self *Burnout) SetEtherbase(etherbase common.Address) {
	self.lock.Lock()
	self.etherbase = etherbase
	self.lock.Unlock()

	self.miner.SetEtherbase(etherbase)
}

func (s *Burnout) StartMining(local bool) error {
	eb, err := s.Etherbase()
	if err != nil {
		log.Error("Cannot start mining without etherbase", "err", err)
		return fmt.Errorf("etherbase missing: %v", err)
	}
	if clique, ok := s.engine.(*clique.Clique); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("Etherbase account unavailable locally", "err", err)
			return fmt.Errorf("singer missing: %v", err)
		}
		clique.Authorize(eb, wallet.SignHash)
	}
	if local {
		// If local (CPU) mining is started, we can disable the transaction rejection
		// mechanism introduced to speed sync times. CPU mining on mainnet is ludicrous
		// so noone will ever hit this path, whereas marking sync done on CPU mining
		// will ensure that private networks work in single miner mode too.
		atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)
	}
	go s.miner.Start(eb)
	return nil
}

func (s *Burnout) StopMining()         { s.miner.Stop() }
func (s *Burnout) IsMining() bool      { return s.miner.Mining() }
func (s *Burnout) Miner() *miner.Miner { return s.miner }

func (s *Burnout) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *Burnout) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *Burnout) TxPool() *core.TxPool               { return s.txPool }
func (s *Burnout) EventMux() *event.TypeMux           { return s.eventMux }
func (s *Burnout) Engine() consensus.Engine           { return s.engine }
func (s *Burnout) ChainDb() brndb.Database            { return s.chainDb }
func (s *Burnout) IsListening() bool                  { return true } // Always listening
func (s *Burnout) EthVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *Burnout) NetVersion() uint64                 { return s.networkId }
func (s *Burnout) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *Burnout) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	}
	return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
}

// Start implements node.Service, starting all internal goroutines needed by the
// Burnout protocol implementation.
func (s *Burnout) Start(srvr *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	s.startBloomHandlers()

	// Start the RPC service
	s.netRPCService = ethapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		maxPeers -= s.config.LightPeers
		if maxPeers < srvr.MaxPeers/2 {
			maxPeers = srvr.MaxPeers / 2
		}
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Burnout protocol.
func (s *Burnout) Stop() error {
	if s.stopDbUpgrade != nil {
		s.stopDbUpgrade()
	}
	s.bloomIndexer.Close()
	s.blockchain.Stop()
	s.protocolManager.Stop()
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
