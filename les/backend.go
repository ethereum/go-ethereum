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

// Package les implements the Light Ethereum Subprotocol.
package les

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/eth/gasprice"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/pow"
	rpc "github.com/ethereum/go-ethereum/rpc"
)

type LightEthereum struct {
	odr         *LesOdr
	relay       *LesTxRelay
	chainConfig *params.ChainConfig
	// Channel for shutting down the service
	shutdownChan chan bool
	// Handlers
	txPool          *light.TxPool
	blockchain      *light.LightChain
	protocolManager *ProtocolManager
	// DB interfaces
	chainDb ethdb.Database // Block chain database

	ApiBackend *LesApiBackend

	eventMux       *event.TypeMux
	pow            pow.PoW
	accountManager *accounts.Manager
	solcPath       string
	solc           *compiler.Solidity

	netVersionId  int
	netRPCService *ethapi.PublicNetAPI
}

func New(ctx *node.ServiceContext, config *eth.Config) (*LightEthereum, error) {
	chainDb, err := eth.CreateDB(ctx, config, "lightchaindata")
	if err != nil {
		return nil, err
	}
	if err := eth.SetupGenesisBlock(&chainDb, config); err != nil {
		return nil, err
	}
	odr := NewLesOdr(chainDb)
	relay := NewLesTxRelay()
	eth := &LightEthereum{
		odr:            odr,
		relay:          relay,
		chainDb:        chainDb,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		pow:            eth.CreatePoW(ctx, config),
		shutdownChan:   make(chan bool),
		netVersionId:   config.NetworkId,
		solcPath:       config.SolcPath,
	}

	if config.ChainConfig == nil {
		return nil, errors.New("missing chain config")
	}
	eth.chainConfig = config.ChainConfig
	eth.blockchain, err = light.NewLightChain(odr, eth.chainConfig, eth.pow, eth.eventMux)
	if err != nil {
		if err == core.ErrNoGenesis {
			return nil, fmt.Errorf(`Genesis block not found. Please supply a genesis block with the "--genesis /path/to/file" argument`)
		}
		return nil, err
	}

	eth.txPool = light.NewTxPool(eth.chainConfig, eth.eventMux, eth.blockchain, eth.relay)
	if eth.protocolManager, err = NewProtocolManager(eth.chainConfig, config.LightMode, config.NetworkId, eth.eventMux, eth.pow, eth.blockchain, nil, chainDb, odr, relay); err != nil {
		return nil, err
	}

	eth.ApiBackend = &LesApiBackend{eth, nil}
	eth.ApiBackend.gpo = gasprice.NewLightPriceOracle(eth.ApiBackend)
	return eth, nil
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

// APIs returns the collection of RPC services the ethereum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *LightEthereum) APIs() []rpc.API {
	return append(ethapi.GetAPIs(s.ApiBackend, s.solcPath), []rpc.API{
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   &LightDummyAPI{},
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "eth",
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

func (s *LightEthereum) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *LightEthereum) BlockChain() *light.LightChain      { return s.blockchain }
func (s *LightEthereum) TxPool() *light.TxPool              { return s.txPool }
func (s *LightEthereum) LesVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *LightEthereum) Downloader() *downloader.Downloader { return s.protocolManager.downloader }
func (s *LightEthereum) EventMux() *event.TypeMux           { return s.eventMux }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *LightEthereum) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

// Start implements node.Service, starting all internal goroutines needed by the
// Ethereum protocol implementation.
func (s *LightEthereum) Start(srvr *p2p.Server) error {
	log.Warn("Light client mode is an experimental feature")
	s.netRPCService = ethapi.NewPublicNetAPI(srvr, s.netVersionId)
	s.protocolManager.Start(srvr)
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Ethereum protocol.
func (s *LightEthereum) Stop() error {
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
