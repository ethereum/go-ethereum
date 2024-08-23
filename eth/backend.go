// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package eth implements the Ethereum protocol.
package eth

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/XinFinOrg/XDPoSChain/XDCx"
	"github.com/XinFinOrg/XDPoSChain/XDCxlending"
	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/common/hexutil"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/consensus/ethash"
	"github.com/XinFinOrg/XDPoSChain/contracts"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/bloombits"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/eth/downloader"
	"github.com/XinFinOrg/XDPoSChain/eth/ethconfig"
	"github.com/XinFinOrg/XDPoSChain/eth/filters"
	"github.com/XinFinOrg/XDPoSChain/eth/gasprice"
	"github.com/XinFinOrg/XDPoSChain/eth/hooks"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/event"
	"github.com/XinFinOrg/XDPoSChain/internal/ethapi"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/miner"
	"github.com/XinFinOrg/XDPoSChain/node"
	"github.com/XinFinOrg/XDPoSChain/p2p"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/rlp"
	"github.com/XinFinOrg/XDPoSChain/rpc"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
}

// Ethereum implements the Ethereum full node service.
type Ethereum struct {
	config      *ethconfig.Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan chan bool // Channel for shutting down the ethereum

	// Handlers
	txPool          *core.TxPool
	orderPool       *core.OrderPool
	lendingPool     *core.LendingPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer

	// DB interfaces
	chainDb ethdb.Database // Block chain database

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

	lock    sync.RWMutex // Protects the variadic fields (e.g. gas price and etherbase)
	XDCX    *XDCx.XDCX
	Lending *XDCxlending.Lending
}

func (s *Ethereum) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}

// New creates a new Ethereum object (including the
// initialisation of the common Ethereum object)
func New(ctx *node.ServiceContext, config *ethconfig.Config, XDCXServ *XDCx.XDCX, lendingServ *XDCxlending.Lending) (*Ethereum, error) {
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run eth.Ethereum in light sync mode, use les.LightEthereum")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}

	log.Info("Initialised chain configuration", "config", chainConfig)

	eth := &Ethereum{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, &config.Ethash, chainConfig, chainDb),
		shutdownChan:   make(chan bool),
		networkId:      config.NetworkId,
		gasPrice:       config.GasPrice,
		etherbase:      config.Etherbase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks),
	}
	// Inject XDCX Service into main Eth Service.
	if XDCXServ != nil {
		eth.XDCX = XDCXServ
	}
	if lendingServ != nil {
		eth.Lending = lendingServ
	}
	log.Info("Initialising Ethereum protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	if !config.SkipBcVersionCheck {
		bcVersion := core.GetBlockChainVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run geth upgradedb.\n", bcVersion, core.BlockChainVersion)
		}
		core.WriteBlockChainVersion(chainDb, core.BlockChainVersion)
	}
	var (
		vmConfig    = vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
		cacheConfig = &core.CacheConfig{Disabled: config.NoPruning, TrieNodeLimit: config.TrieCache, TrieTimeLimit: config.TrieTimeout}
	)
	if eth.chainConfig.XDPoS != nil {
		c := eth.engine.(*XDPoS.XDPoS)
		c.GetXDCXService = func() utils.TradingService {
			return eth.XDCX
		}
		c.GetLendingService = func() utils.LendingService {
			return eth.Lending
		}
	}
	eth.blockchain, err = core.NewBlockChainEx(chainDb, XDCXServ.GetLevelDB(), cacheConfig, eth.chainConfig, eth.engine, vmConfig)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		eth.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	eth.bloomIndexer.Start(eth.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	eth.txPool = core.NewTxPool(config.TxPool, eth.chainConfig, eth.blockchain)
	eth.orderPool = core.NewOrderPool(eth.chainConfig, eth.blockchain)
	eth.lendingPool = core.NewLendingPool(eth.chainConfig, eth.blockchain)
	if common.RollbackHash != (common.Hash{}) {
		curBlock := eth.blockchain.CurrentBlock()
		if curBlock == nil {
			log.Warn("not find current block when rollback")
		}
		prevBlock := eth.blockchain.GetBlockByHash(common.RollbackHash)

		if curBlock != nil && prevBlock != nil && curBlock.NumberU64() > prevBlock.NumberU64() {
			for ; curBlock != nil && curBlock.NumberU64() != prevBlock.NumberU64(); curBlock = eth.blockchain.GetBlock(curBlock.ParentHash(), curBlock.NumberU64()-1) {
				eth.blockchain.Rollback([]common.Hash{curBlock.Hash()})
			}
		}

		if prevBlock != nil {
			err := eth.blockchain.SetHead(prevBlock.NumberU64())
			if err != nil {
				log.Crit("Err Rollback", "err", err)
				return nil, err
			}
		} else {
			log.Error("skip SetHead because target block is nil when rollback")
		}
	}

	if eth.protocolManager, err = NewProtocolManagerEx(eth.chainConfig, config.SyncMode, config.NetworkId, eth.eventMux, eth.txPool, eth.orderPool, eth.lendingPool, eth.engine, eth.blockchain, chainDb); err != nil {
		return nil, err
	}
	eth.miner = miner.New(eth, eth.chainConfig, eth.EventMux(), eth.engine, ctx.GetConfig().AnnounceTxs)
	eth.miner.SetExtra(makeExtraData(config.ExtraData))

	if eth.chainConfig.XDPoS != nil {
		eth.ApiBackend = &EthApiBackend{eth, nil, eth.engine.(*XDPoS.XDPoS)}
	} else {
		eth.ApiBackend = &EthApiBackend{eth, nil, nil}
	}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	eth.ApiBackend.gpo = gasprice.NewOracle(eth.ApiBackend, gpoParams)

	// Set global ipc endpoint.
	eth.blockchain.IPCEndpoint = ctx.GetConfig().IPCEndpoint()

	if eth.chainConfig.XDPoS != nil {
		c := eth.engine.(*XDPoS.XDPoS)
		signHook := func(block *types.Block) error {
			eb, err := eth.Etherbase()
			if err != nil {
				log.Error("Cannot get etherbase for append m2 header", "err", err)
				return fmt.Errorf("etherbase missing: %v", err)
			}
			ok := eth.txPool.IsSigner != nil && eth.txPool.IsSigner(eb)
			if !ok {
				return nil
			}
			if block.NumberU64()%common.MergeSignRange == 0 || !eth.chainConfig.IsTIP2019(block.Number()) {
				if err := contracts.CreateTransactionSign(chainConfig, eth.txPool, eth.accountManager, block, chainDb, eb); err != nil {
					return fmt.Errorf("Fail to create tx sign for importing block: %v", err)
				}
			}
			return nil
		}

		appendM2HeaderHook := func(block *types.Block) (*types.Block, bool, error) {
			eb, err := eth.Etherbase()
			if err != nil {
				log.Error("Cannot get etherbase for append m2 header", "err", err)
				return block, false, fmt.Errorf("etherbase missing: %v", err)
			}
			m1, err := c.RecoverSigner(block.Header())
			if err != nil {
				return block, false, fmt.Errorf("can't get block creator: %v", err)
			}
			m2, err := c.GetValidator(m1, eth.blockchain, block.Header())
			if err != nil {
				return block, false, fmt.Errorf("can't get block validator: %v", err)
			}
			if m2 == eb {
				wallet, err := eth.accountManager.Find(accounts.Account{Address: eb})
				if err != nil {
					log.Error("Can't find coinbase account wallet", "err", err)
					return block, false, err
				}
				header := block.Header()
				sighash, err := wallet.SignHash(accounts.Account{Address: eb}, c.SigHash(header).Bytes())
				if err != nil || sighash == nil {
					log.Error("Can't get signature hash of m2", "sighash", sighash, "err", err)
					return block, false, err
				}
				header.Validator = sighash
				return types.NewBlockWithHeader(header).WithBody(block.Transactions(), block.Uncles()), true, nil
			}
			return block, false, nil
		}

		eth.protocolManager.fetcher.SetSignHook(signHook)
		eth.protocolManager.fetcher.SetAppendM2HeaderHook(appendM2HeaderHook)

		/*
			XDPoS1.0 Specific hooks
		*/
		hooks.AttachConsensusV1Hooks(c, eth.blockchain, chainConfig)
		hooks.AttachConsensusV2Hooks(c, eth.blockchain, chainConfig)

		eth.txPool.IsSigner = func(address common.Address) bool {
			currentHeader := eth.blockchain.CurrentHeader()
			header := currentHeader
			// Sometimes, the latest block hasn't been inserted to chain yet
			// getSnapshot from parent block if it exists
			parentHeader := eth.blockchain.GetHeader(currentHeader.ParentHash, currentHeader.Number.Uint64()-1)
			if parentHeader != nil {
				// not genesis block
				header = parentHeader
			}
			return c.IsAuthorisedAddress(eth.blockchain, header, address)
		}

	}
	return eth, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"XDC",
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
func CreateDB(ctx *node.ServiceContext, config *ethconfig.Config, name string) (ethdb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// CreateConsensusEngine creates the required type of consensus engine instance for an Ethereum service
func CreateConsensusEngine(ctx *node.ServiceContext, config *ethash.Config, chainConfig *params.ChainConfig, db ethdb.Database) consensus.Engine {
	// If delegated-proof-of-stake is requested, set it up
	if chainConfig.XDPoS != nil {
		return XDPoS.New(chainConfig, db)
	}

	// Otherwise assume proof-of-work
	switch {
	case config.PowMode == ethash.ModeFake:
		log.Warn("Ethash used in fake mode")
		return ethash.NewFaker()
	case config.PowMode == ethash.ModeTest:
		log.Warn("Ethash used in test mode")
		return ethash.NewTester()
	case config.PowMode == ethash.ModeShared:
		log.Warn("Ethash used in shared mode")
		return ethash.NewShared()
	default:
		engine := ethash.New(ethash.Config{
			CacheDir:       ctx.ResolvePath(config.CacheDir),
			CachesInMem:    config.CachesInMem,
			CachesOnDisk:   config.CachesOnDisk,
			DatasetDir:     config.DatasetDir,
			DatasetsInMem:  config.DatasetsInMem,
			DatasetsOnDisk: config.DatasetsOnDisk,
		})
		engine.SetThreads(-1) // Disable CPU mining
		return engine
	}
}

// APIs returns the collection of RPC services the ethereum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *Ethereum) APIs() []rpc.API {
	apis := ethapi.GetAPIs(s.ApiBackend, s.BlockChain())

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicEthereumAPI(s),
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
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   filters.NewFilterAPI(filters.NewFilterSystem(s.ApiBackend, filters.Config{LogCacheSize: s.config.FilterLogCacheSize}), false),
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

func (s *Ethereum) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *Ethereum) Etherbase() (eb common.Address, err error) {
	s.lock.RLock()
	etherbase := s.etherbase
	s.lock.RUnlock()

	if etherbase != (common.Address{}) {
		return etherbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			etherbase := accounts[0].Address

			s.lock.Lock()
			s.etherbase = etherbase
			s.lock.Unlock()

			log.Info("Etherbase automatically configured", "address", etherbase)
			return etherbase, nil
		}
	}
	return common.Address{}, errors.New("etherbase must be explicitly specified")
}

// set in js console via admin interface or wrapper from cli flags
func (self *Ethereum) SetEtherbase(etherbase common.Address) {
	self.lock.Lock()
	self.etherbase = etherbase
	self.lock.Unlock()

	self.miner.SetEtherbase(etherbase)
}

// ValidateMasternode checks if node's address is in set of masternodes
func (s *Ethereum) ValidateMasternode() (bool, error) {
	eb, err := s.Etherbase()
	if err != nil {
		return false, err
	}
	if s.chainConfig.XDPoS != nil {
		//check if miner's wallet is in set of validators
		c := s.engine.(*XDPoS.XDPoS)

		authorized := c.IsAuthorisedAddress(s.blockchain, s.blockchain.CurrentHeader(), eb)
		if !authorized {
			//This miner doesn't belong to set of validators
			return false, nil
		}
	} else {
		return false, errors.New("Only verify masternode permission in XDPoS protocol")
	}
	return true, nil
}

func (s *Ethereum) StartStaking(local bool) error {
	eb, err := s.Etherbase()
	if err != nil {
		log.Error("Cannot start mining without etherbase", "err", err)
		return fmt.Errorf("etherbase missing: %v", err)
	}
	if XDPoS, ok := s.engine.(*XDPoS.XDPoS); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("Etherbase account unavailable locally", "address", eb, "err", err)
			return fmt.Errorf("signer missing: %v", err)
		}
		XDPoS.Authorize(eb, wallet.SignHash)
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

func (s *Ethereum) StopStaking() {
	s.miner.Stop()
}
func (s *Ethereum) IsStaking() bool     { return s.miner.Mining() }
func (s *Ethereum) Miner() *miner.Miner { return s.miner }

func (s *Ethereum) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *Ethereum) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *Ethereum) TxPool() *core.TxPool               { return s.txPool }
func (s *Ethereum) EventMux() *event.TypeMux           { return s.eventMux }
func (s *Ethereum) Engine() consensus.Engine           { return s.engine }
func (s *Ethereum) ChainDb() ethdb.Database            { return s.chainDb }
func (s *Ethereum) IsListening() bool                  { return true } // Always listening
func (s *Ethereum) EthVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *Ethereum) NetVersion() uint64                 { return s.networkId }
func (s *Ethereum) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *Ethereum) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	}
	return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
}

// Start implements node.Service, starting all internal goroutines needed by the
// Ethereum protocol implementation.
func (s *Ethereum) Start(srvr *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	s.startBloomHandlers()

	// Start the RPC service
	s.netRPCService = ethapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		if s.config.LightPeers >= srvr.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", s.config.LightPeers, srvr.MaxPeers)
		}
		maxPeers -= s.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}
func (s *Ethereum) SaveData() {
	s.blockchain.SaveData()
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Ethereum protocol.
func (s *Ethereum) Stop() error {
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

func (s *Ethereum) GetPeer() int {
	return len(s.protocolManager.peers.peers)
}

func (s *Ethereum) GetXDCX() *XDCx.XDCX {
	return s.XDCX
}

func (s *Ethereum) OrderPool() *core.OrderPool {
	return s.orderPool
}

func (s *Ethereum) GetXDCXLending() *XDCxlending.Lending {
	return s.Lending
}

// LendingPool geth eth lending pool
func (s *Ethereum) LendingPool() *core.LendingPool {
	return s.lendingPool
}
