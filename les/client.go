// Copyright 2019 The go-ethereum Authors
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

// Package les implements the Light Ethereum Subprotocol.
package les

import (
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/bloombits"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/gasprice"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/internal/shutdowncheck"
	"github.com/ethereum/go-ethereum/les/downloader"
	"github.com/ethereum/go-ethereum/les/vflux"
	vfc "github.com/ethereum/go-ethereum/les/vflux/client"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

type LightEthereum struct {
	lesCommons

	peers              *peerSet
	reqDist            *requestDistributor
	retriever          *retrieveManager
	odr                *LesOdr
	relay              *lesTxRelay
	handler            *handler
	txPool             *light.TxPool
	blockchain         *light.LightChain
	serverPool         *vfc.ServerPool
	serverPoolIterator enode.Iterator
	pruner             *pruner
	merger             *consensus.Merger

	fetcher    *lightFetcher
	downloader *downloader.Downloader
	// Hooks used in the testing
	syncStart func(header *types.Header) // Hook called when the syncing is started
	syncEnd   func(header *types.Header) // Hook called when the syncing is done

	checkpoint    *params.TrustedCheckpoint
	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	ApiBackend     *LesApiBackend
	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager
	netRPCService  *ethapi.NetAPI

	p2pServer  *p2p.Server
	p2pConfig  *p2p.Config
	udpEnabled bool

	shutdownTracker *shutdowncheck.ShutdownTracker // Tracks if and when the node has shutdown ungracefully
}

// New creates an instance of the light client.
func New(stack *node.Node, config *ethconfig.Config) (*LightEthereum, error) {
	chainDb, err := stack.OpenDatabase("lightchaindata", config.DatabaseCache, config.DatabaseHandles, "eth/db/chaindata/", false)
	if err != nil {
		return nil, err
	}
	lesDb, err := stack.OpenDatabase("les.client", 0, 0, "eth/db/lesclient/", false)
	if err != nil {
		return nil, err
	}
	var overrides core.ChainOverrides
	if config.OverrideTerminalTotalDifficulty != nil {
		overrides.OverrideTerminalTotalDifficulty = config.OverrideTerminalTotalDifficulty
	}
	if config.OverrideTerminalTotalDifficultyPassed != nil {
		overrides.OverrideTerminalTotalDifficultyPassed = config.OverrideTerminalTotalDifficultyPassed
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlockWithOverride(chainDb, config.Genesis, &overrides)
	if _, isCompat := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !isCompat {
		return nil, genesisErr
	}
	log.Info("")
	log.Info(strings.Repeat("-", 153))
	for _, line := range strings.Split(chainConfig.String(), "\n") {
		log.Info(line)
	}
	log.Info(strings.Repeat("-", 153))
	log.Info("")

	peers := newPeerSet()
	merger := consensus.NewMerger(chainDb)
	leth := &LightEthereum{
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
		engine:          ethconfig.CreateConsensusEngine(stack, &config.Ethash, chainConfig.Clique, nil, false, chainDb),
		bloomRequests:   make(chan chan *bloombits.Retrieval),
		bloomIndexer:    core.NewBloomIndexer(chainDb, params.BloomBitsBlocksClient, params.HelperTrieConfirmations),
		p2pServer:       stack.Server(),
		p2pConfig:       &stack.Config().P2P,
		udpEnabled:      stack.Config().P2P.DiscoveryV5,
		shutdownTracker: shutdowncheck.NewShutdownTracker(chainDb),
	}

	var prenegQuery vfc.QueryFunc
	if leth.udpEnabled {
		prenegQuery = leth.prenegQuery
	}
	leth.serverPool, leth.serverPoolIterator = vfc.NewServerPool(lesDb, []byte("serverpool:"), time.Second, prenegQuery, &mclock.System{}, []string{}, requestList) //TODO ul server list
	leth.serverPool.AddMetrics(suggestedTimeoutGauge, totalValueGauge, serverSelectableGauge, serverConnectedGauge, sessionValueMeter, serverDialedMeter)

	leth.retriever = newRetrieveManager(peers, leth.reqDist, leth.serverPool.GetTimeout)
	leth.relay = newLesTxRelay(peers, leth.retriever)
	leth.odr = NewLesOdr(chainDb, light.DefaultClientIndexerConfig, leth.peers, leth.retriever)

	leth.chtIndexer = light.NewChtIndexer(chainDb, leth.odr, params.CHTFrequency, params.HelperTrieConfirmations, config.LightNoPrune)
	leth.bloomTrieIndexer = light.NewBloomTrieIndexer(chainDb, leth.odr, params.BloomBitsBlocksClient, params.BloomTrieFrequency, config.LightNoPrune)
	leth.odr.SetIndexers(leth.chtIndexer, leth.bloomTrieIndexer, leth.bloomIndexer)

	leth.checkpoint = config.Checkpoint
	if leth.checkpoint == nil {
		leth.checkpoint = params.TrustedCheckpoints[genesisHash]
	}
	// Note: NewLightChain adds the trusted checkpoint so it needs an ODR with
	// indexers already set but not started yet
	if chain, err := light.NewLightChain(leth.odr, leth.chainConfig, leth.engine, leth.checkpoint); err == nil {
		leth.blockchain = chain
		leth.chainReader = chain
	} else {
		return nil, err
	}
	// Set up checkpoint oracle.
	leth.oracle = leth.setupOracle(stack, genesisHash, config)

	// Note: AddChildIndexer starts the update process for the child
	leth.bloomIndexer.AddChildIndexer(leth.bloomTrieIndexer)
	leth.chtIndexer.Start(leth.blockchain)
	leth.bloomIndexer.Start(leth.blockchain)

	// Start a light chain pruner to delete useless historical data.
	leth.pruner = newPruner(chainDb, leth.chtIndexer, leth.bloomTrieIndexer)

	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		leth.blockchain.SetHead(compat.RewindTo)
		rawdb.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	leth.txPool = light.NewTxPool(leth.chainConfig, leth.blockchain, leth.relay)

	leth.ApiBackend = &LesApiBackend{stack.Config().ExtRPCEnabled(), stack.Config().AllowUnprotectedTxs, leth, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.Miner.GasPrice
	}
	leth.ApiBackend.gpo = gasprice.NewOracle(leth.ApiBackend, gpoParams)

	leth.setupHandler(false)

	leth.netRPCService = ethapi.NewNetAPI(leth.p2pServer, leth.config.NetworkId)

	// Register the backend on the node
	stack.RegisterAPIs(leth.APIs())
	stack.RegisterProtocols(leth.Protocols())
	stack.RegisterLifecycle(leth)

	// Successful startup; push a marker and check previous unclean shutdowns.
	leth.shutdownTracker.MarkStartup()

	return leth, nil
}

func (leth *LightEthereum) setupHandler(noInitAnnounce bool) {
	leth.handler = newHandler(leth.peers, leth.config.NetworkId)
	clientHandler := &clientHandler{
		forkFilter:     forkid.NewFilter(leth.blockchain),
		blockchain:     leth.blockchain,
		peers:          leth.peers,
		retriever:      leth.retriever,
		noInitAnnounce: noInitAnnounce,
	}
	var height uint64
	if leth.checkpoint != nil {
		height = (leth.checkpoint.SectionIndex+1)*params.CHTFrequency - 1
	}
	leth.fetcher = newLightFetcher(leth.blockchain, leth.engine, leth.peers, leth.chainDb, leth.reqDist, leth.synchronise)
	leth.downloader = downloader.New(height, leth.chainDb, leth.eventMux, nil, leth.blockchain, leth.peers.unregisterStringId)
	leth.peers.subscribe(&downloaderPeerNotify{
		downloader: leth.downloader,
		retriever:  leth.retriever,
	})
	clientHandler.fetcher, clientHandler.downloader = leth.fetcher, leth.downloader
	leth.handler.registerModule(clientHandler)

	fcClientHandler := &fcClientHandler{
		retriever: leth.retriever,
	}
	leth.handler.registerModule(fcClientHandler)

	vfxClientHandler := &vfxClientHandler{
		serverPool: leth.serverPool,
	}
	leth.handler.registerModule(vfxClientHandler)
}

// VfluxRequest sends a batch of requests to the given node through discv5 UDP TalkRequest and returns the responses
func (s *LightEthereum) VfluxRequest(n *enode.Node, reqs vflux.Requests) vflux.Replies {
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
func (s *LightEthereum) vfxVersion(n *enode.Node) uint {
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
func (s *LightEthereum) prenegQuery(n *enode.Node) int {
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

// Etherbase is the address that mining rewards will be send to
func (s *LightDummyAPI) Etherbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("mining is not supported in light mode")
}

// Coinbase is the address that mining rewards will be send to (alias for Etherbase)
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

// APIs returns the collection of RPC services the ethereum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *LightEthereum) APIs() []rpc.API {
	apis := ethapi.GetAPIs(s.ApiBackend)
	apis = append(apis, s.engine.APIs(s.BlockChain().HeaderChain())...)
	return append(apis, []rpc.API{
		{
			Namespace: "eth",
			Service:   &LightDummyAPI{},
		}, {
			Namespace: "eth",
			Service:   downloader.NewDownloaderAPI(s.downloader, s.eventMux),
		}, {
			Namespace: "net",
			Service:   s.netRPCService,
		}, {
			Namespace: "les",
			Service:   NewLightAPI(&s.lesCommons),
		}, {
			Namespace: "vflux",
			Service:   s.serverPool.API(),
		},
	}...)
}

func (s *LightEthereum) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *LightEthereum) BlockChain() *light.LightChain      { return s.blockchain }
func (s *LightEthereum) TxPool() *light.TxPool              { return s.txPool }
func (s *LightEthereum) Engine() consensus.Engine           { return s.engine }
func (s *LightEthereum) LesVersion() int                    { return int(ClientProtocolVersions[0]) }
func (s *LightEthereum) Downloader() *downloader.Downloader { return s.downloader }
func (s *LightEthereum) EventMux() *event.TypeMux           { return s.eventMux }
func (s *LightEthereum) Merger() *consensus.Merger          { return s.merger }

// Protocols returns all the currently configured network protocols to start.
func (s *LightEthereum) Protocols() []p2p.Protocol {
	return s.makeProtocols(ClientProtocolVersions, s.handler.runPeer, func(id enode.ID) interface{} {
		if p := s.peers.peer(id); p != nil {
			return p.Info()
		}
		return nil
	}, s.serverPoolIterator)
}

// Start implements node.Lifecycle, starting all internal goroutines needed by the
// light ethereum protocol implementation.
func (s *LightEthereum) Start() error {
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
	s.handler.start()
	s.serverPool.AddSource(discovery)
	s.serverPool.Start()
	// Start bloom request workers.
	s.wg.Add(bloomServiceThreads)
	s.startBloomHandlers(params.BloomBitsBlocksClient)
	if s.fetcher != nil {
		s.fetcher.start()
	}
	return nil
}

// Stop implements node.Lifecycle, terminating all internal goroutines used by the
// Ethereum protocol.
func (s *LightEthereum) Stop() error {
	close(s.closeCh)
	s.serverPool.Stop()
	s.handler.stop()
	s.reqDist.close()
	s.odr.Stop()
	s.relay.Stop()
	if s.bloomIndexer != nil {
		s.bloomIndexer.Close()
	}
	if s.chtIndexer != nil {
		s.chtIndexer.Close()
	}
	s.blockchain.Stop()
	if s.fetcher != nil {
		s.fetcher.stop()
	}
	if s.downloader != nil {
		s.downloader.Terminate()
	}
	//s.handler.stop()
	s.txPool.Stop()
	s.engine.Close()
	if s.pruner != nil {
		s.pruner.close()
	}
	s.eventMux.Stop()
	// Clean shutdown marker as the last thing before closing db
	s.shutdownTracker.Stop()

	s.chainDb.Close()
	s.lesDb.Close()
	s.wg.Wait()
	log.Info("Light ethereum stopped")
	return nil
}
