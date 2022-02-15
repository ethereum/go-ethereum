// Copyright 2016 The go-ethereum Authors
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

// Copyright 2021-2022 The go-xpayments Authors
// This file is part of go-xpayments.

// Package les implements the Light xPayments Subprotocol.

package les

import (
	"fmt"
	"time"

	"github.com/xpaymentsorg/go-xpayments/accounts"
	"github.com/xpaymentsorg/go-xpayments/common"
	"github.com/xpaymentsorg/go-xpayments/common/hexutil"
	"github.com/xpaymentsorg/go-xpayments/common/mclock"
	"github.com/xpaymentsorg/go-xpayments/consensus"
	"github.com/xpaymentsorg/go-xpayments/core"
	"github.com/xpaymentsorg/go-xpayments/core/bloombits"
	"github.com/xpaymentsorg/go-xpayments/core/rawdb"
	"github.com/xpaymentsorg/go-xpayments/core/types"
	"github.com/xpaymentsorg/go-xpayments/event"
	"github.com/xpaymentsorg/go-xpayments/internal/shutdowncheck"
	"github.com/xpaymentsorg/go-xpayments/internal/xpsapi"
	"github.com/xpaymentsorg/go-xpayments/les/downloader"
	"github.com/xpaymentsorg/go-xpayments/les/vflux"
	vfc "github.com/xpaymentsorg/go-xpayments/les/vflux/client"
	"github.com/xpaymentsorg/go-xpayments/light"
	"github.com/xpaymentsorg/go-xpayments/log"
	"github.com/xpaymentsorg/go-xpayments/node"
	"github.com/xpaymentsorg/go-xpayments/p2p"
	"github.com/xpaymentsorg/go-xpayments/p2p/enode"
	"github.com/xpaymentsorg/go-xpayments/p2p/enr"
	"github.com/xpaymentsorg/go-xpayments/params"
	"github.com/xpaymentsorg/go-xpayments/rlp"
	"github.com/xpaymentsorg/go-xpayments/rpc"
	"github.com/xpaymentsorg/go-xpayments/xps/filters"
	"github.com/xpaymentsorg/go-xpayments/xps/gasprice"
	"github.com/xpaymentsorg/go-xpayments/xps/xpsconfig"
	// "github.com/ethereum/go-ethereum/accounts"
	// "github.com/ethereum/go-ethereum/common"
	// "github.com/ethereum/go-ethereum/common/hexutil"
	// "github.com/ethereum/go-ethereum/common/mclock"
	// "github.com/ethereum/go-ethereum/consensus"
	// "github.com/ethereum/go-ethereum/core"
	// "github.com/ethereum/go-ethereum/core/bloombits"
	// "github.com/ethereum/go-ethereum/core/rawdb"
	// "github.com/ethereum/go-ethereum/core/types"
	// "github.com/ethereum/go-ethereum/eth/ethconfig"
	// "github.com/ethereum/go-ethereum/eth/filters"
	// "github.com/ethereum/go-ethereum/eth/gasprice"
	// "github.com/ethereum/go-ethereum/event"
	// "github.com/ethereum/go-ethereum/internal/ethapi"
	// "github.com/ethereum/go-ethereum/internal/shutdowncheck"
	// "github.com/ethereum/go-ethereum/les/downloader"
	// "github.com/ethereum/go-ethereum/les/vflux"
	// vfc "github.com/ethereum/go-ethereum/les/vflux/client"
	// "github.com/ethereum/go-ethereum/light"
	// "github.com/ethereum/go-ethereum/log"
	// "github.com/ethereum/go-ethereum/node"
	// "github.com/ethereum/go-ethereum/p2p"
	// "github.com/ethereum/go-ethereum/p2p/enode"
	// "github.com/ethereum/go-ethereum/p2p/enr"
	// "github.com/ethereum/go-ethereum/params"
	// "github.com/ethereum/go-ethereum/rlp"
	// "github.com/ethereum/go-ethereum/rpc"
)

type LightxPayments struct {
	lesCommons

	peers              *serverPeerSet
	reqDist            *requestDistributor
	retriever          *retrieveManager
	odr                *LesOdr
	relay              *lesTxRelay
	handler            *clientHandler
	txPool             *light.TxPool
	blockchain         *light.LightChain
	serverPool         *vfc.ServerPool
	serverPoolIterator enode.Iterator
	pruner             *pruner
	merger             *consensus.Merger

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	ApiBackend     *LesApiBackend
	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager
	netRPCService  *xpsapi.PublicNetAPI

	p2pServer  *p2p.Server
	p2pConfig  *p2p.Config
	udpEnabled bool

	shutdownTracker *shutdowncheck.ShutdownTracker // Tracks if and when the node has shutdown ungracefully
}

// New creates an instance of the light client.
func New(stack *node.Node, config *xpsconfig.Config) (*LightxPayments, error) {
	chainDb, err := stack.OpenDatabase("lightchaindata", config.DatabaseCache, config.DatabaseHandles, "xps/db/chaindata/", false)
	if err != nil {
		return nil, err
	}
	lesDb, err := stack.OpenDatabase("les.client", 0, 0, "xps/db/lesclient/", false)
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlockWithOverride(chainDb, config.Genesis, config.OverrideArrowGlacier, config.OverrideTerminalTotalDifficulty)
	if _, isCompat := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !isCompat {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	peers := newServerPeerSet()
	merger := consensus.NewMerger(chainDb)
	lxps := &LightxPayments{
		lesCommons: lesCommons{
			genesis:     genesisHash,
			config:      config,
			chainConfig: chainConfig,
			iConfig:     light.DefaultClientIndexerConfig,
			chainDb:     chainDb,
			lesDb:       lesDb,
			closeCh:     make(chan struct{}),
		},
		peers:           peers,
		eventMux:        stack.EventMux(),
		reqDist:         newRequestDistributor(peers, &mclock.System{}),
		accountManager:  stack.AccountManager(),
		merger:          merger,
		engine:          xpsconfig.CreateConsensusEngine(stack, chainConfig, &config.Xpsash, nil, false, chainDb),
		bloomRequests:   make(chan chan *bloombits.Retrieval),
		bloomIndexer:    core.NewBloomIndexer(chainDb, params.BloomBitsBlocksClient, params.HelperTrieConfirmations),
		p2pServer:       stack.Server(),
		p2pConfig:       &stack.Config().P2P,
		udpEnabled:      stack.Config().P2P.DiscoveryV5,
		shutdownTracker: shutdowncheck.NewShutdownTracker(chainDb),
	}

	var prenegQuery vfc.QueryFunc
	if lxps.udpEnabled {
		prenegQuery = lxps.prenegQuery
	}
	lxps.serverPool, lxps.serverPoolIterator = vfc.NewServerPool(lesDb, []byte("serverpool:"), time.Second, prenegQuery, &mclock.System{}, config.UltraLightServers, requestList)
	lxps.serverPool.AddMetrics(suggestedTimeoutGauge, totalValueGauge, serverSelectableGauge, serverConnectedGauge, sessionValueMeter, serverDialedMeter)

	lxps.retriever = newRetrieveManager(peers, lxps.reqDist, lxps.serverPool.GetTimeout)
	lxps.relay = newLesTxRelay(peers, lxps.retriever)

	lxps.odr = NewLesOdr(chainDb, light.DefaultClientIndexerConfig, lxps.peers, lxps.retriever)
	lxps.chtIndexer = light.NewChtIndexer(chainDb, lxps.odr, params.CHTFrequency, params.HelperTrieConfirmations, config.LightNoPrune)
	lxps.bloomTrieIndexer = light.NewBloomTrieIndexer(chainDb, lxps.odr, params.BloomBitsBlocksClient, params.BloomTrieFrequency, config.LightNoPrune)
	lxps.odr.SetIndexers(lxps.chtIndexer, lxps.bloomTrieIndexer, lxps.bloomIndexer)

	checkpoint := config.Checkpoint
	if checkpoint == nil {
		checkpoint = params.TrustedCheckpoints[genesisHash]
	}
	// Note: NewLightChain adds the trusted checkpoint so it needs an ODR with
	// indexers already set but not started yet
	if lxps.blockchain, err = light.NewLightChain(lxps.odr, lxps.chainConfig, lxps.engine, checkpoint); err != nil {
		return nil, err
	}
	lxps.chainReader = lxps.blockchain
	lxps.txPool = light.NewTxPool(lxps.chainConfig, lxps.blockchain, lxps.relay)

	// Set up checkpoint oracle.
	lxps.oracle = lxps.setupOracle(stack, genesisHash, config)

	// Note: AddChildIndexer starts the update process for the child
	lxps.bloomIndexer.AddChildIndexer(lxps.bloomTrieIndexer)
	lxps.chtIndexer.Start(lxps.blockchain)
	lxps.bloomIndexer.Start(lxps.blockchain)

	// Start a light chain pruner to delete useless historical data.
	lxps.pruner = newPruner(chainDb, lxps.chtIndexer, lxps.bloomTrieIndexer)

	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		lxps.blockchain.SetHead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	lxps.ApiBackend = &LesApiBackend{stack.Config().ExtRPCEnabled(), stack.Config().AllowUnprotectedTxs, lxps, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.Miner.GasPrice
	}
	lxps.ApiBackend.gpo = gasprice.NewOracle(lxps.ApiBackend, gpoParams)

	lxps.handler = newClientHandler(config.UltraLightServers, config.UltraLightFraction, checkpoint, lxps)
	if lxps.handler.ulc != nil {
		log.Warn("Ultra light client is enabled", "trustedNodes", len(lxps.handler.ulc.keys), "minTrustedFraction", lxps.handler.ulc.fraction)
		lxps.blockchain.DisableCheckFreq()
	}

	lxps.netRPCService = xpsapi.NewPublicNetAPI(lxps.p2pServer, lxps.config.NetworkId)

	// Register the backend on the node
	stack.RegisterAPIs(lxps.APIs())
	stack.RegisterProtocols(lxps.Protocols())
	stack.RegisterLifecycle(lxps)

	// Successful startup; push a marker and check previous unclean shutdowns.
	lxps.shutdownTracker.MarkStartup()

	return lxps, nil
}

// VfluxRequest sends a batch of requests to the given node through discv5 UDP TalkRequest and returns the responses
func (s *LightxPayments) VfluxRequest(n *enode.Node, reqs vflux.Requests) vflux.Replies {
	if !s.udpEnabled {
		return nil
	}
	reqsEnc, _ := rlp.EncodeToBytes(&reqs)
	repliesEnc, _ := s.p2pServer.DiscV5.TalkRequest(s.serverPool.DialNode(n), "vfx", reqsEnc)
	var replies vflux.Replies
	if len(repliesEnc) == 0 || rlp.DecodeBytes(repliesEnc, &replies) != nil {
		return nil
	}
	return replies
}

// vfxVersion returns the version number of the "les" service subdomain of the vflux UDP
// service, as advertised in the ENR record
func (s *LightxPayments) vfxVersion(n *enode.Node) uint {
	if n.Seq() == 0 {
		var err error
		if !s.udpEnabled {
			return 0
		}
		if n, err = s.p2pServer.DiscV5.RequestENR(n); n != nil && err == nil && n.Seq() != 0 {
			s.serverPool.Persist(n)
		} else {
			return 0
		}
	}

	var les []rlp.RawValue
	if err := n.Load(enr.WithEntry("les", &les)); err != nil || len(les) < 1 {
		return 0
	}
	var version uint
	rlp.DecodeBytes(les[0], &version) // Ignore additional fields (for forward compatibility).
	return version
}

// prenegQuery sends a capacity query to the given server node to determine whether
// a connection slot is immediately available
func (s *LightxPayments) prenegQuery(n *enode.Node) int {
	if s.vfxVersion(n) < 1 {
		// UDP query not supported, always try TCP connection
		return 1
	}

	var requests vflux.Requests
	requests.Add("les", vflux.CapacityQueryName, vflux.CapacityQueryReq{
		Bias:      180,
		AddTokens: []vflux.IntOrInf{{}},
	})
	replies := s.VfluxRequest(n, requests)
	var cqr vflux.CapacityQueryReply
	if replies.Get(0, &cqr) != nil || len(cqr) != 1 { // Note: Get returns an error if replies is nil
		return -1
	}
	if cqr[0] > 0 {
		return 1
	}
	return 0
}

type LightDummyAPI struct{}

// Xpserbase is the address that mining rewards will be send to
func (s *LightDummyAPI) Xpserbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("mining is not supported in light mode")
}

// Coinbase is the address that mining rewards will be send to (alias for Xpserbase)
func (s *LightDummyAPI) Coinbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("mining is not supported in light mode")
}

// Hashrate returns the POW hashrate
func (s *LightDummyAPI) Hashrate() hexutil.Uint {
	return 0
}

// Mining returns an indication if this node is currently mining.
func (s *LightDummyAPI) Mining() bool {
	return false
}

// APIs returns the collection of RPC services the xpayments package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *LightxPayments) APIs() []rpc.API {
	apis := xpsapi.GetAPIs(s.ApiBackend)
	apis = append(apis, s.engine.APIs(s.BlockChain().HeaderChain())...)
	return append(apis, []rpc.API{
		{
			Namespace: "xps",
			Version:   "1.0",
			Service:   &LightDummyAPI{},
			Public:    true,
		}, {
			Namespace: "xps",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.handler.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "xps",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, true, 5*time.Minute),
			Public:    true,
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		}, {
			Namespace: "les",
			Version:   "1.0",
			Service:   NewPrivateLightAPI(&s.lesCommons),
			Public:    false,
		}, {
			Namespace: "vflux",
			Version:   "1.0",
			Service:   s.serverPool.API(),
			Public:    false,
		},
	}...)
}

func (s *LightxPayments) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *LightxPayments) BlockChain() *light.LightChain      { return s.blockchain }
func (s *LightxPayments) TxPool() *light.TxPool              { return s.txPool }
func (s *LightxPayments) Engine() consensus.Engine           { return s.engine }
func (s *LightxPayments) LesVersion() int                    { return int(ClientProtocolVersions[0]) }
func (s *LightxPayments) Downloader() *downloader.Downloader { return s.handler.downloader }
func (s *LightxPayments) EventMux() *event.TypeMux           { return s.eventMux }
func (s *LightxPayments) Merger() *consensus.Merger          { return s.merger }

// Protocols returns all the currently configured network protocols to start.
func (s *LightxPayments) Protocols() []p2p.Protocol {
	return s.makeProtocols(ClientProtocolVersions, s.handler.runPeer, func(id enode.ID) interface{} {
		if p := s.peers.peer(id.String()); p != nil {
			return p.Info()
		}
		return nil
	}, s.serverPoolIterator)
}

// Start implements node.Lifecycle, starting all internal goroutines needed by the
// light xpayments protocol implementation.
func (s *LightxPayments) Start() error {
	log.Warn("Light client mode is an experimental feature")

	// Regularly update shutdown marker
	s.shutdownTracker.Start()

	if s.udpEnabled && s.p2pServer.DiscV5 == nil {
		s.udpEnabled = false
		log.Error("Discovery v5 is not initialized")
	}
	discovery, err := s.setupDiscovery()
	if err != nil {
		return err
	}
	s.serverPool.AddSource(discovery)
	s.serverPool.Start()
	// Start bloom request workers.
	s.wg.Add(bloomServiceThreads)
	s.startBloomHandlers(params.BloomBitsBlocksClient)
	s.handler.start()

	return nil
}

// Stop implements node.Lifecycle, terminating all internal goroutines used by the
// xPayments protocol.
func (s *LightxPayments) Stop() error {
	close(s.closeCh)
	s.serverPool.Stop()
	s.peers.close()
	s.reqDist.close()
	s.odr.Stop()
	s.relay.Stop()
	s.bloomIndexer.Close()
	s.chtIndexer.Close()
	s.blockchain.Stop()
	s.handler.stop()
	s.txPool.Stop()
	s.engine.Close()
	s.pruner.close()
	s.eventMux.Stop()
	// Clean shutdown marker as the last thing before closing db
	s.shutdownTracker.Stop()

	s.chainDb.Close()
	s.lesDb.Close()
	s.wg.Wait()
	log.Info("Light xpayments stopped")
	return nil
}
