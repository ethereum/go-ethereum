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
	"strings"
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
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"github.com/XinFinOrg/XDPoSChain/core/txpool"
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
	txPool          *txpool.TxPool
	orderPool       *txpool.OrderPool
	lendingPool     *txpool.LendingPool
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

func (e *Ethereum) AddLesServer(ls LesServer) {
	e.lesServer = ls
	ls.SetBloomBitsIndexer(e.bloomIndexer)
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

	// Assemble the Ethereum object
	chainDb, err := ctx.OpenDatabase("chaindata", config.DatabaseCache, config.DatabaseHandles, "eth/db/chaindata/", false)
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}

	networkID := config.NetworkId
	if networkID == 0 {
		networkID = chainConfig.ChainId.Uint64()
	}
	common.CopyConstans(networkID)

	log.Info(strings.Repeat("-", 153))
	for _, line := range strings.Split(chainConfig.Description(), "\n") {
		log.Info(line)
	}
	log.Info(strings.Repeat("-", 153))

	eth := &Ethereum{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, &config.Ethash, chainConfig, chainDb),
		shutdownChan:   make(chan bool),
		networkId:      networkID,
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

	bcVersion := rawdb.ReadDatabaseVersion(chainDb)
	var dbVer = "<nil>"
	if bcVersion != nil {
		dbVer = fmt.Sprintf("%d", *bcVersion)
	}
	log.Info("Initialising Ethereum protocol", "versions", ProtocolVersions, "network", networkID, "dbversion", dbVer)

	if !config.SkipBcVersionCheck {
		if bcVersion != nil && *bcVersion > core.BlockChainVersion {
			return nil, fmt.Errorf("database version is v%d, not supports v%d", *bcVersion, core.BlockChainVersion)
		} else if bcVersion == nil || *bcVersion < core.BlockChainVersion {
			if bcVersion != nil { // only print warning on upgrade, not on init
				log.Warn("Upgrade blockchain database version", "from", dbVer, "to", core.BlockChainVersion)
			}
			rawdb.WriteDatabaseVersion(chainDb, core.BlockChainVersion)
		}
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

	// Rollback according to SetHeadFlag
	if common.RollbackNumber != 0 {
		target := common.RollbackNumber
		common.RollbackNumber = 0
		currentBlock := eth.blockchain.CurrentBlock()
		if currentBlock == nil {
			return nil, fmt.Errorf("not find current block when rollback to %d", common.RollbackNumber)
		}
		currentNumber := currentBlock.NumberU64()
		if target > currentNumber {
			return nil, fmt.Errorf("can't rollback to %d which is greater than current %d", target, currentNumber)
		}
		log.Warn("Start rollback", "target", target, "current", currentNumber)
		err := eth.blockchain.SetHead(target)
		if err != nil {
			return nil, fmt.Errorf("fail to rollback: target=%d, current=%d, err: %w", target, currentNumber, err)
		}
		log.Warn("Rollback completed", "target", target)
	}

	if engine, ok := eth.blockchain.Engine().(*XDPoS.XDPoS); ok {
		err := engine.Initial(eth.blockchain, eth.blockchain.CurrentHeader())
		if err != nil {
			return nil, err
		}
	}

	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		eth.blockchain.SetHead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	eth.bloomIndexer.Start(eth.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	eth.txPool = txpool.NewTxPool(config.TxPool, eth.chainConfig, eth.blockchain)
	eth.orderPool = txpool.NewOrderPool(eth.chainConfig, eth.blockchain)
	eth.lendingPool = txpool.NewLendingPool(eth.chainConfig, eth.blockchain)

	if eth.protocolManager, err = NewProtocolManagerEx(eth.chainConfig, config.SyncMode, networkID, eth.eventMux, eth.txPool, eth.orderPool, eth.lendingPool, eth.engine, eth.blockchain, chainDb); err != nil {
		return nil, err
	}
	eth.miner = miner.New(eth, eth.chainConfig, eth.EventMux(), eth.engine, ctx.GetConfig().AnnounceTxs)
	eth.miner.SetExtra(makeExtraData(config.ExtraData))

	if eth.chainConfig.XDPoS != nil {
		eth.ApiBackend = &EthApiBackend{eth, nil, eth.engine.(*XDPoS.XDPoS)}
	} else {
		eth.ApiBackend = &EthApiBackend{eth, nil, nil}
	}
	eth.ApiBackend.gpo = gasprice.NewOracle(eth.ApiBackend, config.GPO, config.GasPrice)

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
					return fmt.Errorf("fail to create tx sign for importing block: %v", err)
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
func (e *Ethereum) APIs() []rpc.API {
	apis := ethapi.GetAPIs(e.ApiBackend, e.BlockChain())

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, e.engine.APIs(e.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicEthereumAPI(e),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(e),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(e.protocolManager.downloader, e.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(e),
			Public:    false,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   filters.NewFilterAPI(filters.NewFilterSystem(e.ApiBackend, filters.Config{LogCacheSize: e.config.FilterLogCacheSize}), false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(e),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(e),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(e.chainConfig, e),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   e.netRPCService,
			Public:    true,
		},
	}...)
}

func (e *Ethereum) ResetWithGenesisBlock(gb *types.Block) {
	e.blockchain.ResetWithGenesisBlock(gb)
}

func (e *Ethereum) Etherbase() (eb common.Address, err error) {
	e.lock.RLock()
	etherbase := e.etherbase
	e.lock.RUnlock()

	if etherbase != (common.Address{}) {
		return etherbase, nil
	}
	if wallets := e.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			etherbase := accounts[0].Address

			e.lock.Lock()
			e.etherbase = etherbase
			e.lock.Unlock()

			log.Info("Etherbase automatically configured", "address", etherbase)
			return etherbase, nil
		}
	}
	return common.Address{}, errors.New("etherbase must be explicitly specified")
}

// set in js console via admin interface or wrapper from cli flags
func (e *Ethereum) SetEtherbase(etherbase common.Address) {
	e.lock.Lock()
	e.etherbase = etherbase
	e.lock.Unlock()

	e.miner.SetEtherbase(etherbase)
}

// ValidateMasternode checks if node's address is in set of masternodes
func (e *Ethereum) ValidateMasternode() (bool, error) {
	eb, err := e.Etherbase()
	if err != nil {
		return false, err
	}
	if e.chainConfig.XDPoS != nil {
		//check if miner's wallet is in set of validators
		c := e.engine.(*XDPoS.XDPoS)

		authorized := c.IsAuthorisedAddress(e.blockchain, e.blockchain.CurrentHeader(), eb)
		if !authorized {
			//This miner doesn't belong to set of validators
			return false, nil
		}
	} else {
		return false, errors.New("only verify masternode permission in XDPoS protocol")
	}
	return true, nil
}

func (e *Ethereum) StartStaking(local bool) error {
	eb, err := e.Etherbase()
	if err != nil {
		log.Error("Cannot start mining without etherbase", "err", err)
		return fmt.Errorf("etherbase missing: %v", err)
	}
	if XDPoS, ok := e.engine.(*XDPoS.XDPoS); ok {
		wallet, err := e.accountManager.Find(accounts.Account{Address: eb})
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
		atomic.StoreUint32(&e.protocolManager.acceptTxs, 1)
	}
	go e.miner.Start(eb)
	return nil
}

func (e *Ethereum) StopStaking() {
	e.miner.Stop()
}

func (e *Ethereum) IsStaking() bool     { return e.miner.Mining() }
func (e *Ethereum) Miner() *miner.Miner { return e.miner }

func (e *Ethereum) AccountManager() *accounts.Manager  { return e.accountManager }
func (e *Ethereum) BlockChain() *core.BlockChain       { return e.blockchain }
func (e *Ethereum) TxPool() *txpool.TxPool             { return e.txPool }
func (e *Ethereum) EventMux() *event.TypeMux           { return e.eventMux }
func (e *Ethereum) Engine() consensus.Engine           { return e.engine }
func (e *Ethereum) ChainDb() ethdb.Database            { return e.chainDb }
func (e *Ethereum) IsListening() bool                  { return true } // Always listening
func (e *Ethereum) EthVersion() int                    { return int(e.protocolManager.SubProtocols[0].Version) }
func (e *Ethereum) NetVersion() uint64                 { return e.networkId }
func (e *Ethereum) Downloader() *downloader.Downloader { return e.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (e *Ethereum) Protocols() []p2p.Protocol {
	if e.lesServer == nil {
		return e.protocolManager.SubProtocols
	}
	return append(e.protocolManager.SubProtocols, e.lesServer.Protocols()...)
}

// Start implements node.Service, starting all internal goroutines needed by the
// Ethereum protocol implementation.
func (e *Ethereum) Start(srvr *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	e.startBloomHandlers()

	// Start the RPC service
	e.netRPCService = ethapi.NewPublicNetAPI(srvr, e.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if e.config.LightServ > 0 {
		if e.config.LightPeers >= srvr.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", e.config.LightPeers, srvr.MaxPeers)
		}
		maxPeers -= e.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	e.protocolManager.Start(maxPeers)
	if e.lesServer != nil {
		e.lesServer.Start(srvr)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Ethereum protocol.
func (e *Ethereum) Stop() error {
	e.bloomIndexer.Close()
	e.blockchain.Stop()
	e.protocolManager.Stop()
	if e.lesServer != nil {
		e.lesServer.Stop()
	}
	e.txPool.Stop()
	e.miner.Stop()
	e.eventMux.Stop()

	e.chainDb.Close()
	close(e.shutdownChan)

	return nil
}

func (e *Ethereum) GetPeer() int {
	return len(e.protocolManager.peers.peers)
}

func (e *Ethereum) GetXDCX() *XDCx.XDCX {
	return e.XDCX
}

func (e *Ethereum) OrderPool() *txpool.OrderPool {
	return e.orderPool
}

func (e *Ethereum) GetXDCXLending() *XDCxlending.Lending {
	return e.Lending
}

// LendingPool geth eth lending pool
func (e *Ethereum) LendingPool() *txpool.LendingPool {
	return e.lendingPool
}
