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
	"encoding/json"
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
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"github.com/XinFinOrg/XDPoSChain/core/txpool"
	"github.com/XinFinOrg/XDPoSChain/core/txpool/legacypool"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/eth/downloader"
	"github.com/XinFinOrg/XDPoSChain/eth/ethconfig"
	"github.com/XinFinOrg/XDPoSChain/eth/filters"
	"github.com/XinFinOrg/XDPoSChain/eth/gasprice"
	"github.com/XinFinOrg/XDPoSChain/eth/hooks"
	"github.com/XinFinOrg/XDPoSChain/eth/tracers"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/event"
	"github.com/XinFinOrg/XDPoSChain/internal/ethapi"
	"github.com/XinFinOrg/XDPoSChain/internal/version"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/miner"
	"github.com/XinFinOrg/XDPoSChain/node"
	"github.com/XinFinOrg/XDPoSChain/p2p"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/rlp"
	"github.com/XinFinOrg/XDPoSChain/rpc"
	ver "github.com/XinFinOrg/XDPoSChain/version"
)

// Ethereum implements the Ethereum full node service.
type Ethereum struct {
	config *ethconfig.Config

	// Channel for shutting down the service
	shutdownChan chan bool // Channel for shutting down the ethereum

	// Handlers
	txPool *txpool.TxPool

	orderPool       *legacypool.OrderPool
	lendingPool     *legacypool.LendingPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager

	// DB interfaces
	chainDb ethdb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	APIBackend *EthAPIBackend

	miner     *miner.Miner
	gasPrice  *big.Int
	etherbase common.Address

	networkId     uint64
	netRPCService *ethapi.NetAPI

	p2pServer *p2p.Server

	lock    sync.RWMutex // Protects the variadic fields (e.g. gas price and etherbase)
	XDCX    *XDCx.XDCX
	Lending *XDCxlending.Lending
}

// New creates a new Ethereum object (including the
// initialisation of the common Ethereum object)
func New(stack *node.Node, config *ethconfig.Config, XDCXServ *XDCx.XDCX, lendingServ *XDCxlending.Lending) (*Ethereum, error) {
	// Ensure configuration values are compatible and sane
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run eth.Ethereum in light sync mode, light mode has been deprecated")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	if config.Miner.GasCeil == 0 {
		log.Warn("Sanitizing invalid miner gas limit", "provided", config.Miner.GasCeil, "updated", ethconfig.Defaults.Miner.GasCeil)
		config.Miner.GasCeil = ethconfig.Defaults.Miner.GasCeil
	}
	if config.Miner.GasPrice == nil || config.Miner.GasPrice.Cmp(common.Big0) < 0 {
		log.Warn("Sanitizing invalid miner gas price", "provided", config.Miner.GasPrice, "updated", ethconfig.Defaults.Miner.GasPrice)
		config.Miner.GasPrice = new(big.Int).Set(ethconfig.Defaults.Miner.GasPrice)
	}

	chainDb, err := stack.OpenDatabase("chaindata", config.DatabaseCache, config.DatabaseHandles, "eth/db/chaindata/", false)
	if err != nil {
		return nil, err
	}
	// Here we determine genesis hash and active ChainConfig.
	// We need these to figure out the consensus parameters and to set up history pruning.
	chainConfig, _, err := core.LoadChainConfig(chainDb, config.Genesis)
	if err != nil {
		return nil, err
	}

	// Set networkID to chainID by default.
	networkID := config.NetworkId
	if networkID == 0 {
		networkID = chainConfig.ChainID.Uint64()
	}
	common.CopyConstants(networkID)

	// Assemble the Ethereum object.
	eth := &Ethereum{
		config:         config,
		chainDb:        chainDb,
		eventMux:       stack.EventMux(),
		accountManager: stack.AccountManager(),
		engine:         CreateConsensusEngine(stack, chainConfig, chainDb),
		shutdownChan:   make(chan bool),
		networkId:      networkID,
		gasPrice:       config.Miner.GasPrice,
		etherbase:      config.Miner.Etherbase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks, params.BloomConfirms),
		p2pServer:      stack.Server(),
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

	// Create BlockChain object.
	if !config.SkipBcVersionCheck {
		if bcVersion != nil && *bcVersion > core.BlockChainVersion {
			return nil, fmt.Errorf("database version is v%d, XDC %s only supports v%d", *bcVersion, version.WithMeta, core.BlockChainVersion)
		} else if bcVersion == nil || *bcVersion < core.BlockChainVersion {
			if bcVersion != nil { // only print warning on upgrade, not on init
				log.Warn("Upgrade blockchain database version", "from", dbVer, "to", core.BlockChainVersion)
			}
			rawdb.WriteDatabaseVersion(chainDb, core.BlockChainVersion)
		}
	}

	badBlocks := rawdb.ReadAllBadBlocks(chainDb)
	log.Info("Bad blocks in db", "count", len(badBlocks))
	for i, block := range badBlocks {
		log.Info("Bad block in db", "i", i, "number", block.Number(), "hash", block.Hash().Hex())
	}
	if config.DeleteAllBadBlocks {
		if len(badBlocks) == 0 {
			log.Warn("No bad blocks in db to delete")
		} else {
			rawdb.DeleteBadBlocks(chainDb)
			log.Info(fmt.Sprintf("Deleted %d bad blocks in db", len(badBlocks)))
		}
	}

	var (
		vmConfig    = vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
		cacheConfig = &core.CacheConfig{
			TrieCleanLimit:    config.TrieCleanCache,
			TrieCleanPrefetch: config.Prefetch,
			TrieDirtyLimit:    config.TrieDirtyCache,
			TrieDirtyDisabled: config.NoPruning,
			TrieTimeLimit:     config.TrieTimeout,
			Preimages:         config.Preimages,
		}
	)
	if config.VMTrace != "" {
		traceConfig := json.RawMessage("{}")
		if config.VMTraceJsonConfig != "" {
			traceConfig = json.RawMessage(config.VMTraceJsonConfig)
		}
		t, err := tracers.LiveDirectory.New(config.VMTrace, traceConfig)
		if err != nil {
			return nil, fmt.Errorf("Failed to create tracer %s: %v", config.VMTrace, err)
		}
		vmConfig.Tracer = t
	}
	if chainConfig.XDPoS != nil {
		c := eth.engine.(*XDPoS.XDPoS)
		c.GetXDCXService = func() utils.TradingService {
			return eth.XDCX
		}
		c.GetLendingService = func() utils.LendingService {
			return eth.Lending
		}
	}
	eth.blockchain, err = core.NewBlockChainEx(chainDb, XDCXServ.GetLevelDB(), cacheConfig, config.Genesis, eth.engine, vmConfig)
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
		currentNumber := currentBlock.Number.Uint64()
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

	eth.bloomIndexer.Start(eth.blockchain)

	// TxPool
	if config.TxPool.Journal != "" {
		config.TxPool.Journal = stack.ResolvePath(config.TxPool.Journal)
	}
	legacyPool := legacypool.New(config.TxPool, eth.blockchain)

	eth.txPool, err = txpool.New(config.TxPool.PriceLimit, eth.blockchain, []txpool.SubPool{legacyPool})
	if err != nil {
		return nil, err
	}

	eth.orderPool = legacypool.NewOrderPool(eth.blockchain.Config(), eth.blockchain)
	eth.lendingPool = legacypool.NewLendingPool(eth.blockchain.Config(), eth.blockchain)

	if eth.protocolManager, err = NewProtocolManagerEx(eth.blockchain.Config(), config.SyncMode, networkID, eth.eventMux, eth.txPool, eth.orderPool, eth.lendingPool, eth.engine, eth.blockchain, chainDb); err != nil {
		return nil, err
	}
	eth.miner = miner.New(eth, &config.Miner, eth.blockchain.Config(), eth.EventMux(), eth.engine, stack.Config().AnnounceTxs)
	eth.miner.SetExtra(makeExtraData(config.Miner.ExtraData))

	var xdPoS *XDPoS.XDPoS = nil
	if chainConfig.XDPoS != nil {
		xdPoS = eth.engine.(*XDPoS.XDPoS)
	}
	eth.APIBackend = &EthAPIBackend{
		allowUnprotectedTxs: stack.Config().AllowUnprotectedTxs,
		eth:                 eth,
		gpo:                 nil,
		XDPoS:               xdPoS,
	}

	if eth.APIBackend.allowUnprotectedTxs {
		log.Info("Unprotected transactions allowed")
	}
	eth.APIBackend.gpo = gasprice.NewOracle(eth.APIBackend, config.GPO, config.Miner.GasPrice)

	// Set global ipc endpoint.
	eth.blockchain.IPCEndpoint = stack.IPCEndpoint()

	if chainConfig.XDPoS != nil {
		c := eth.engine.(*XDPoS.XDPoS)
		signHook := func(block *types.Block) error {
			eb, err := eth.Etherbase()
			if err != nil {
				log.Error("Cannot get etherbase for append m2 header", "err", err)
				return fmt.Errorf("etherbase missing: %v", err)
			}
			ok := eth.txPool.IsSigner(eb)
			if !ok {
				return nil
			}
			if block.NumberU64()%common.MergeSignRange == 0 || !chainConfig.IsTIP2019(block.Number()) {
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
				return types.NewBlockWithHeader(header).WithBody(*block.Body()), true, nil
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

		isSigner := func(address common.Address) bool {
			return c.IsAuthorisedAddress(eth.blockchain, eth.blockchain.CurrentHeader(), address)
		}
		eth.txPool.SetSigner(isSigner)
	}
	// Start the RPC service
	eth.netRPCService = ethapi.NewNetAPI(eth.p2pServer, eth.NetVersion())

	// Register the backend on the node
	stack.RegisterAPIs(eth.APIs())
	stack.RegisterProtocols(eth.Protocols())
	stack.RegisterLifecycle(eth)
	return eth, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(ver.Major<<16 | ver.Minor<<8 | ver.Patch),
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
func CreateConsensusEngine(stack *node.Node, chainConfig *params.ChainConfig, db ethdb.Database) consensus.Engine {
	// If delegated-proof-of-stake is requested, set it up
	if chainConfig.XDPoS != nil {
		return XDPoS.New(chainConfig, db)
	}

	return ethash.NewFaker()
}

// APIs return the collection of RPC services the ethereum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (e *Ethereum) APIs() []rpc.API {
	apis := ethapi.GetAPIs(e.APIBackend, e.BlockChain())

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, e.engine.APIs(e.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "eth",
			Service:   NewEthereumAPI(e),
		}, {
			Namespace: "miner",
			Service:   NewMinerAPI(e),
		}, {
			Namespace: "eth",
			Service:   downloader.NewDownloaderAPI(e.protocolManager.downloader, e.eventMux),
		}, {
			Namespace: "eth",
			Service:   filters.NewFilterAPI(filters.NewFilterSystem(e.APIBackend, filters.Config{LogCacheSize: e.config.FilterLogCacheSize}), false),
		}, {
			Namespace: "admin",
			Service:   NewAdminAPI(e),
		}, {
			Namespace: "debug",
			Service:   NewDebugAPI(e),
		}, {
			Namespace: "net",
			Service:   e.netRPCService,
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
	if e.blockchain.Config().XDPoS != nil {
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
func (e *Ethereum) BloomIndexer() *core.ChainIndexer   { return e.bloomIndexer }

// Protocols returns all the currently configured
func (e *Ethereum) Protocols() []p2p.Protocol {
	return e.protocolManager.SubProtocols
}

// Start implements node.Lifecycle, starting all internal goroutines needed by the
// Ethereum protocol implementation.
func (e *Ethereum) Start() error {
	// Start the bloom bits servicing goroutines
	e.startBloomHandlers(params.BloomBitsBlocks)

	// Figure out a max peers count based on the server limits
	maxPeers := e.p2pServer.MaxPeers
	if e.config.LightServ > 0 {
		if e.config.LightPeers >= e.p2pServer.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", e.config.LightPeers, e.p2pServer.MaxPeers)
		}
		maxPeers -= e.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	e.protocolManager.Start(maxPeers)
	return nil
}

// Stop implements node.Lifecycle, terminating all internal goroutines used by the
// Ethereum protocol.
func (e *Ethereum) Stop() error {
	e.bloomIndexer.Close()
	e.blockchain.Stop()
	e.protocolManager.Stop()

	e.txPool.Close()
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

func (e *Ethereum) OrderPool() *legacypool.OrderPool {
	return e.orderPool
}

func (e *Ethereum) GetXDCXLending() *XDCxlending.Lending {
	return e.Lending
}

// LendingPool geth eth lending pool
func (e *Ethereum) LendingPool() *legacypool.LendingPool {
	return e.lendingPool
}
