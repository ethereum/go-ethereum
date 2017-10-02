// Copyright 2016 The go-burnout Authors
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

// Package les implements the Light Burnout Subprotocol.
package les

import (
	"fmt"
	"sync"
	"time"

	"github.com/burnout/go-burnout/accounts"
	"github.com/burnout/go-burnout/common"
	"github.com/burnout/go-burnout/common/hexutil"
	"github.com/burnout/go-burnout/consensus"
	"github.com/burnout/go-burnout/core"
	"github.com/burnout/go-burnout/core/types"
	"github.com/burnout/go-burnout/brn"
	"github.com/burnout/go-burnout/brn/downloader"
	"github.com/burnout/go-burnout/brn/filters"
	"github.com/burnout/go-burnout/brn/gasprice"
	"github.com/burnout/go-burnout/brndb"
	"github.com/burnout/go-burnout/event"
	"github.com/burnout/go-burnout/internal/ethapi"
	"github.com/burnout/go-burnout/light"
	"github.com/burnout/go-burnout/log"
	"github.com/burnout/go-burnout/node"
	"github.com/burnout/go-burnout/p2p"
	"github.com/burnout/go-burnout/p2p/discv5"
	"github.com/burnout/go-burnout/params"
	rpc "github.com/burnout/go-burnout/rpc"
)

type LightBurnout struct {
	odr         *LesOdr
	relay       *LesTxRelay
	chainConfig *params.ChainConfig
	// Channel for shutting down the service
	shutdownChan chan bool
	// Handlers
	peers           *peerSet
	txPool          *light.TxPool
	blockchain      *light.LightChain
	protocolManager *ProtocolManager
	serverPool      *serverPool
	reqDist         *requestDistributor
	retriever       *retrieveManager
	// DB interfaces
	chainDb brndb.Database // Block chain database

	ApiBackend *LesApiBackend

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	networkId     uint64
	netRPCService *ethapi.PublicNetAPI

	wg sync.WaitGroup
}

func New(ctx *node.ServiceContext, config *brn.Config) (*LightBurnout, error) {
	chainDb, err := brn.CreateDB(ctx, config, "lightchaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, isCompat := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !isCompat {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	peers := newPeerSet()
	quitSync := make(chan struct{})

	brn := &LightBurnout{
		chainConfig:    chainConfig,
		chainDb:        chainDb,
		eventMux:       ctx.EventMux,
		peers:          peers,
		reqDist:        newRequestDistributor(peers, quitSync),
		accountManager: ctx.AccountManager,
		engine:         brn.CreateConsensusEngine(ctx, config, chainConfig, chainDb),
		shutdownChan:   make(chan bool),
		networkId:      config.NetworkId,
	}

	brn.relay = NewLesTxRelay(peers, brn.reqDist)
	brn.serverPool = newServerPool(chainDb, quitSync, &brn.wg)
	brn.retriever = newRetrieveManager(peers, brn.reqDist, brn.serverPool)
	brn.odr = NewLesOdr(chainDb, brn.retriever)
	if brn.blockchain, err = light.NewLightChain(brn.odr, brn.chainConfig, brn.engine); err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		brn.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	brn.txPool = light.NewTxPool(brn.chainConfig, brn.blockchain, brn.relay)
	if brn.protocolManager, err = NewProtocolManager(brn.chainConfig, true, config.NetworkId, brn.eventMux, brn.engine, brn.peers, brn.blockchain, nil, chainDb, brn.odr, brn.relay, quitSync, &brn.wg); err != nil {
		return nil, err
	}
	brn.ApiBackend = &LesApiBackend{brn, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	brn.ApiBackend.gpo = gasprice.NewOracle(brn.ApiBackend, gpoParams)
	return brn, nil
}

func lesTopic(genesisHash common.Hash) discv5.Topic {
	return discv5.Topic("LES@" + common.Bytes2Hex(genesisHash.Bytes()[0:8]))
}

type LightDummyAPI struct{}

// Etherbase is the address that mining rewards will be send to
func (s *LightDummyAPI) Etherbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Coinbase is the address that mining rewards will be send to (alias for Etherbase)
func (s *LightDummyAPI) Coinbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

// Hashrate returns the POW hashrate
func (s *LightDummyAPI) Hashrate() hexutil.Uint {
	return 0
}

// Mining returns an indication if this node is currently mining.
func (s *LightDummyAPI) Mining() bool {
	return false
}

// APIs returns the collection of RPC services the burnout package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *LightBurnout) APIs() []rpc.API {
	return append(ethapi.GetAPIs(s.ApiBackend), []rpc.API{
		{
			Namespace: "brn",
			Version:   "1.0",
			Service:   &LightDummyAPI{},
			Public:    true,
		}, {
			Namespace: "brn",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "brn",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, true),
			Public:    true,
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *LightBurnout) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *LightBurnout) BlockChain() *light.LightChain      { return s.blockchain }
func (s *LightBurnout) TxPool() *light.TxPool              { return s.txPool }
func (s *LightBurnout) Engine() consensus.Engine           { return s.engine }
func (s *LightBurnout) LesVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *LightBurnout) Downloader() *downloader.Downloader { return s.protocolManager.downloader }
func (s *LightBurnout) EventMux() *event.TypeMux           { return s.eventMux }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *LightBurnout) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

// Start implements node.Service, starting all internal goroutines needed by the
// Burnout protocol implementation.
func (s *LightBurnout) Start(srvr *p2p.Server) error {
	log.Warn("Light client mode is an experimental feature")
	s.netRPCService = ethapi.NewPublicNetAPI(srvr, s.networkId)
	s.serverPool.start(srvr, lesTopic(s.blockchain.Genesis().Hash()))
	s.protocolManager.Start()
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Burnout protocol.
func (s *LightBurnout) Stop() error {
	s.odr.Stop()
	s.blockchain.Stop()
	s.protocolManager.Stop()
	s.txPool.Stop()

	s.eventMux.Stop()

	time.Sleep(time.Millisecond * 200)
	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
