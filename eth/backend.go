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
	"time"

	"bytes"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/contracts"
	"github.com/ethereum/go-ethereum/contracts/validator/contract"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/bloombits"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/eth/gasprice"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
}

// Ethereum implements the Ethereum full node service.
type Ethereum struct {
	config      *Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan  chan bool    // Channel for shutting down the ethereum
	stopDbUpgrade func() error // stop chain db sequential key upgrade

	// Handlers
	txPool          *core.TxPool
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

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and etherbase)
}

func (s *Ethereum) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}

// New creates a new Ethereum object (including the
// initialisation of the common Ethereum object)
func New(ctx *node.ServiceContext, config *Config) (*Ethereum, error) {
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
	stopDbUpgrade := upgradeDeduplicateData(chainDb)
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
		stopDbUpgrade:  stopDbUpgrade,
		networkId:      config.NetworkId,
		gasPrice:       config.GasPrice,
		etherbase:      config.Etherbase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks),
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
	eth.blockchain, err = core.NewBlockChain(chainDb, cacheConfig, eth.chainConfig, eth.engine, vmConfig)
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

	if eth.protocolManager, err = NewProtocolManager(eth.chainConfig, config.SyncMode, config.NetworkId, eth.eventMux, eth.txPool, eth.engine, eth.blockchain, chainDb); err != nil {
		return nil, err
	}
	eth.miner = miner.New(eth, eth.chainConfig, eth.EventMux(), eth.engine, ctx.GetConfig().AnnounceTxs)
	eth.miner.SetExtra(makeExtraData(config.ExtraData))

	eth.ApiBackend = &EthApiBackend{eth, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	eth.ApiBackend.gpo = gasprice.NewOracle(eth.ApiBackend, gpoParams)

	// Set global ipc endpoint.
	eth.blockchain.IPCEndpoint = ctx.GetConfig().IPCEndpoint()

	if eth.chainConfig.Posv != nil {
		c := eth.engine.(*posv.Posv)
		signHook := func(block *types.Block) error {
			ok, err := eth.ValidateMasternode()
			if err != nil {
				return fmt.Errorf("Can't verify masternode permission: %v", err)
			}
			if !ok {
				// silently return as this node doesn't have masternode permission to sign block
				return nil
			}
			if err := contracts.CreateTransactionSign(chainConfig, eth.txPool, eth.accountManager, block, chainDb); err != nil {
				return fmt.Errorf("Fail to create tx sign for importing block: %v", err)
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
				wallet, _ := eth.accountManager.Find(accounts.Account{Address: eb})
				header := block.Header()
				sighash, _ := wallet.SignHash(accounts.Account{Address: eb}, posv.SigHash(header).Bytes())
				header.Validator = sighash
				return types.NewBlockWithHeader(header).WithBody(block.Transactions(), block.Uncles()), true, nil
			}
			return block, false, nil
		}

		eth.protocolManager.fetcher.SetSignHook(signHook)
		eth.protocolManager.fetcher.SetAppendM2HeaderHook(appendM2HeaderHook)

		// Hook prepares validators M2 for the current epoch at checkpoint block
		c.HookValidator = func(header *types.Header, signers []common.Address) ([]byte, error) {
			start := time.Now()
			validators, err := GetValidators(eth.blockchain, signers)
			if err != nil {
				return []byte{}, err
			}
			header.Validators = validators
			log.Debug("Time Calculated HookValidator ", "block", header.Number.Uint64(), "time", common.PrettyDuration(time.Since(start)))
			return validators, nil
		}

		// Hook scans for bad masternodes and decide to penalty them
		c.HookPenalty = func(chain consensus.ChainReader, blockNumberEpoc uint64) ([]common.Address, error) {
			client, err := eth.blockchain.GetClient()
			if err != nil {
				return nil, err
			}
			prevEpoc := blockNumberEpoc - chain.Config().Posv.Epoch
			if prevEpoc >= 0 {
				start := time.Now()
				prevHeader := chain.GetHeaderByNumber(prevEpoc)
				penSigners := c.GetMasternodes(chain, prevHeader)
				if len(penSigners) > 0 {
					blockSignerAddr := common.HexToAddress(common.BlockSigners)
					// Loop for each block to check missing sign.
					for i := prevEpoc; i < blockNumberEpoc; i++ {
						blockHeader := chain.GetHeaderByNumber(i)
						if len(penSigners) > 0 {
							signedMasternodes, err := contracts.GetSignersFromContract(c, blockSignerAddr, client, blockHeader.Hash())
							if err != nil {
								return nil, err
							}
							if len(signedMasternodes) > 0 {
								// Check signer signed?
								for _, signed := range signedMasternodes {
									for j, addr := range penSigners {
										if signed == addr {
											// Remove it from dupSigners.
											penSigners = append(penSigners[:j], penSigners[j+1:]...)
										}
									}
								}
							}
						} else {
							break
						}
					}
				}
				log.Debug("Time Calculated HookPenalty ", "block", blockNumberEpoc, "time", common.PrettyDuration(time.Since(start)))
				return penSigners, nil
			}
			return []common.Address{}, nil
		}

		// Hook calculates reward for masternodes
		c.HookReward = func(chain consensus.ChainReader, state *state.StateDB, header *types.Header) (error, map[string]interface{}) {
			client, err := eth.blockchain.GetClient()
			if err != nil {
				log.Crit("Fail to connect IPC client for blockSigner", "error", err)
			}
			number := header.Number.Uint64()
			rCheckpoint := chain.Config().Posv.RewardCheckpoint
			foudationWalletAddr := chain.Config().Posv.FoudationWalletAddr
			if foudationWalletAddr == (common.Address{}) {
				log.Error("Foundation Wallet Address is empty", "error", foudationWalletAddr)
			}
			rewards := make(map[string]interface{})
			if number > 0 && number-rCheckpoint > 0 && foudationWalletAddr != (common.Address{}) {
				start := time.Now()
				// Get signers in blockSigner smartcontract.
				addr := common.HexToAddress(common.BlockSigners)
				// Get reward inflation.
				chainReward := new(big.Int).Mul(new(big.Int).SetUint64(chain.Config().Posv.Reward), new(big.Int).SetUint64(params.Ether))
				chainReward = rewardInflation(chainReward, number, common.BlocksPerYear)

				totalSigner := new(uint64)
				signers, err := contracts.GetRewardForCheckpoint(c, chain, addr, number, rCheckpoint, client, totalSigner)
				log.Debug("Time Get Signers", "block", header.Number.Uint64(), "time", common.PrettyDuration(time.Since(start)))
				if err != nil {
					log.Crit("Fail to get signers for reward checkpoint", "error", err)
				}
				rewards["signers"] = signers
				rewardSigners, err := contracts.CalculateRewardForSigner(chainReward, signers, *totalSigner)
				if err != nil {
					log.Crit("Fail to calculate reward for signers", "error", err)
				}
				// Get validator.
				validator, err := contract.NewTomoValidator(common.HexToAddress(common.MasternodeVotingSMC), client)
				if err != nil {
					log.Crit("Fail get instance of Tomo Validator", "error", err)
				}
				// Add reward for coin holders.
				voterResults := make(map[common.Address]interface{})
				if len(signers) > 0 {
					for signer, calcReward := range rewardSigners {
						err, rewards := contracts.CalculateRewardForHolders(c, foudationWalletAddr, validator, state, signer, calcReward)
						if err != nil {
							log.Crit("Fail to calculate reward for holders.", "error", err)
						}
						voterResults[signer] = rewards
					}
				}
				rewards["rewards"] = voterResults
				log.Debug("Time Calculated HookReward ", "block", header.Number.Uint64(), "time", common.PrettyDuration(time.Since(start)))
			}
			return nil, rewards
		}

		// Hook verifies masternodes set
		c.HookVerifyMNs = func(header *types.Header, signers []common.Address) error {
			number := header.Number.Int64()
			if number > 0 && number%common.EpocBlockRandomize == 0 {
				start := time.Now()
				validators, err := GetValidators(eth.blockchain, signers)
				log.Debug("Time Calculated HookVerifyMNs ", "block", header.Number.Uint64(), "time", common.PrettyDuration(time.Since(start)))
				if err != nil {
					return err
				}
				if !bytes.Equal(header.Validators, validators) {
					return posv.ErrInvalidCheckpointValidators
				}
			}
			return nil
		}
		eth.txPool.IsMasterNode = func(address common.Address) bool {
			currentHeader := eth.blockchain.CurrentHeader()
			snap, err := c.GetSnapshot(eth.blockchain, currentHeader)
			if err != nil {
				log.Error("Can't get snapshot with current header ", "number", currentHeader.Number, "hash", currentHeader.Hash().Hex(), "err", err)
				return false
			}
			if _, ok := snap.Signers[address]; ok {
				return true
			}
			return false
		}
	}
	return eth, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"tomo",
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
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (ethdb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*ethdb.LDBDatabase); ok {
		db.Meter("eth/db/chaindata/")
	}
	return db, nil
}

// CreateConsensusEngine creates the required type of consensus engine instance for an Ethereum service
func CreateConsensusEngine(ctx *node.ServiceContext, config *ethash.Config, chainConfig *params.ChainConfig, db ethdb.Database) consensus.Engine {
	// If proof-of-stake-voting is requested, set it up
	if chainConfig.Posv != nil {
		return posv.New(chainConfig.Posv, db)
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
	apis := ethapi.GetAPIs(s.ApiBackend)

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
	return common.Address{}, fmt.Errorf("etherbase must be explicitly specified")
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
	if s.chainConfig.Posv != nil {
		//check if miner's wallet is in set of validators
		c := s.engine.(*posv.Posv)
		snap, err := c.GetSnapshot(s.blockchain, s.blockchain.CurrentHeader())
		if err != nil {
			return false, fmt.Errorf("Can't verify masternode permission: %v", err)
		}
		if _, authorized := snap.Signers[eb]; !authorized {
			//This miner doesn't belong to set of validators
			return false, nil
		}
	} else {
		return false, fmt.Errorf("Only verify masternode permission in PoSV protocol")
	}
	return true, nil
}

func (s *Ethereum) StartStaking(local bool) error {
	eb, err := s.Etherbase()
	if err != nil {
		log.Error("Cannot start mining without etherbase", "err", err)
		return fmt.Errorf("etherbase missing: %v", err)
	}
	if posv, ok := s.engine.(*posv.Posv); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("Etherbase account unavailable locally", "err", err)
			return fmt.Errorf("signer missing: %v", err)
		}
		posv.Authorize(eb, wallet.SignHash)
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

func (s *Ethereum) StopStaking()        { s.miner.Stop() }
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

// Stop implements node.Service, terminating all internal goroutines used by the
// Ethereum protocol.
func (s *Ethereum) Stop() error {
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

func GetValidators(bc *core.BlockChain, masternodes []common.Address) ([]byte, error) {
	if bc.Config().Posv == nil {
		return nil, core.ErrNotPoSV
	}
	client, err := bc.GetClient()
	if err != nil {
		return nil, err
	}
	// Check m2 exists on chaindb.
	// Get secrets and opening at epoc block checkpoint.

	var candidates []int64
	if err != nil {
		return nil, err
	}
	lenSigners := int64(len(masternodes))
	if lenSigners > 0 {
		for _, addr := range masternodes {
			random, err := contracts.GetRandomizeFromContract(client, addr)
			if err != nil {
				return nil, err
			}
			candidates = append(candidates, random)
		}
		// Get randomize m2 list.
		m2, err := contracts.GenM2FromRandomize(candidates, lenSigners)
		if err != nil {
			return nil, err
		}
		return contracts.BuildValidatorFromM2(m2), nil
	}
	return nil, core.ErrNotFoundM1
}

func rewardInflation(chainReward *big.Int, number uint64, blockPerYear uint64) *big.Int {
	if blockPerYear*2 <= number && number < blockPerYear*6 {
		chainReward.Div(chainReward, new(big.Int).SetUint64(2))
	}
	if blockPerYear*6 <= number {
		chainReward.Div(chainReward, new(big.Int).SetUint64(4))
	}

	return chainReward
}
