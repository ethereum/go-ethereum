// Copyright 2018 The go-ethereum Authors
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

package swarm

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"fmt"
	"io"
	"math/big"
	"net"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/chequebook"
	"github.com/ethereum/go-ethereum/contracts/ens"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/api"
	httpapi "github.com/ethereum/go-ethereum/swarm/api/http"
	"github.com/ethereum/go-ethereum/swarm/fuse"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/stream"
	"github.com/ethereum/go-ethereum/swarm/pss"
	"github.com/ethereum/go-ethereum/swarm/state"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/storage/feed"
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
	"github.com/ethereum/go-ethereum/swarm/swap"
	"github.com/ethereum/go-ethereum/swarm/tracing"
)

var (
	updateGaugesPeriod = 5 * time.Second
	startCounter       = metrics.NewRegisteredCounter("stack,start", nil)
	stopCounter        = metrics.NewRegisteredCounter("stack,stop", nil)
	uptimeGauge        = metrics.NewRegisteredGauge("stack.uptime", nil)
	requestsCacheGauge = metrics.NewRegisteredGauge("storage.cache.requests.size", nil)
)

// the swarm stack
type Swarm struct {
	config            *api.Config        // swarm configuration
	api               *api.API           // high level api layer (fs/manifest)
	dns               api.Resolver       // DNS registrar
	fileStore         *storage.FileStore // distributed preimage archive, the local API to the storage with document level storage/retrieval support
	streamer          *stream.Registry
	bzz               *network.Bzz       // the logistic manager
	backend           chequebook.Backend // simple blockchain Backend
	privateKey        *ecdsa.PrivateKey
	netStore          *storage.NetStore
	sfs               *fuse.SwarmFS // need this to cleanup all the active mounts on node exit
	ps                *pss.Pss
	swap              *swap.Swap
	stateStore        *state.DBStore
	accountingMetrics *protocols.AccountingMetrics
	cleanupFuncs      []func() error

	tracerClose io.Closer
}

// NewSwarm creates a new swarm service instance
// implements node.Service
// If mockStore is not nil, it will be used as the storage for chunk data.
// MockStore should be used only for testing.
func NewSwarm(config *api.Config, mockStore *mock.NodeStore) (self *Swarm, err error) {
	if bytes.Equal(common.FromHex(config.PublicKey), storage.ZeroAddr) {
		return nil, fmt.Errorf("empty public key")
	}
	if bytes.Equal(common.FromHex(config.BzzKey), storage.ZeroAddr) {
		return nil, fmt.Errorf("empty bzz key")
	}

	var backend chequebook.Backend
	if config.SwapAPI != "" && config.SwapEnabled {
		log.Info("connecting to SWAP API", "url", config.SwapAPI)
		backend, err = ethclient.Dial(config.SwapAPI)
		if err != nil {
			return nil, fmt.Errorf("error connecting to SWAP API %s: %s", config.SwapAPI, err)
		}
	}

	self = &Swarm{
		config:       config,
		backend:      backend,
		privateKey:   config.ShiftPrivateKey(),
		cleanupFuncs: []func() error{},
	}
	log.Debug("Setting up Swarm service components")

	config.HiveParams.Discovery = true

	bzzconfig := &network.BzzConfig{
		NetworkID:    config.NetworkID,
		OverlayAddr:  common.FromHex(config.BzzKey),
		HiveParams:   config.HiveParams,
		LightNode:    config.LightNodeEnabled,
		BootnodeMode: config.BootnodeMode,
	}

	self.stateStore, err = state.NewDBStore(filepath.Join(config.Path, "state-store.db"))
	if err != nil {
		return
	}

	// set up high level api
	var resolver *api.MultiResolver
	if len(config.EnsAPIs) > 0 {
		opts := []api.MultiResolverOption{}
		for _, c := range config.EnsAPIs {
			tld, endpoint, addr := parseEnsAPIAddress(c)
			r, err := newEnsClient(endpoint, addr, config, self.privateKey)
			if err != nil {
				return nil, err
			}
			opts = append(opts, api.MultiResolverOptionWithResolver(r, tld))

		}
		resolver = api.NewMultiResolver(opts...)
		self.dns = resolver
	}

	lstore, err := storage.NewLocalStore(config.LocalStoreParams, mockStore)
	if err != nil {
		return nil, err
	}

	self.netStore, err = storage.NewNetStore(lstore, nil)
	if err != nil {
		return nil, err
	}

	to := network.NewKademlia(
		common.FromHex(config.BzzKey),
		network.NewKadParams(),
	)
	delivery := stream.NewDelivery(to, self.netStore)
	self.netStore.NewNetFetcherFunc = network.NewFetcherFactory(delivery.RequestFromPeers, config.DeliverySkipCheck).New

	if config.SwapEnabled {
		balancesStore, err := state.NewDBStore(filepath.Join(config.Path, "balances.db"))
		if err != nil {
			return nil, err
		}
		self.swap = swap.New(balancesStore)
		self.accountingMetrics = protocols.SetupAccountingMetrics(10*time.Second, filepath.Join(config.Path, "metrics.db"))
	}

	var nodeID enode.ID
	if err := nodeID.UnmarshalText([]byte(config.NodeID)); err != nil {
		return nil, err
	}

	syncing := stream.SyncingAutoSubscribe
	if !config.SyncEnabled || config.LightNodeEnabled {
		syncing = stream.SyncingDisabled
	}

	retrieval := stream.RetrievalEnabled
	if config.LightNodeEnabled {
		retrieval = stream.RetrievalClientOnly
	}

	registryOptions := &stream.RegistryOptions{
		SkipCheck:       config.DeliverySkipCheck,
		Syncing:         syncing,
		Retrieval:       retrieval,
		SyncUpdateDelay: config.SyncUpdateDelay,
		MaxPeerServers:  config.MaxStreamPeerServers,
	}
	self.streamer = stream.NewRegistry(nodeID, delivery, self.netStore, self.stateStore, registryOptions, self.swap)

	// Swarm Hash Merklised Chunking for Arbitrary-length Document/File storage
	self.fileStore = storage.NewFileStore(self.netStore, self.config.FileStoreParams)

	var feedsHandler *feed.Handler
	fhParams := &feed.HandlerParams{}

	feedsHandler = feed.NewHandler(fhParams)
	feedsHandler.SetStore(self.netStore)

	lstore.Validators = []storage.ChunkValidator{
		storage.NewContentAddressValidator(storage.MakeHashFunc(storage.DefaultHash)),
		feedsHandler,
	}

	err = lstore.Migrate()
	if err != nil {
		return nil, err
	}

	log.Debug("Setup local storage")

	self.bzz = network.NewBzz(bzzconfig, to, self.stateStore, self.streamer.GetSpec(), self.streamer.Run)

	// Pss = postal service over swarm (devp2p over bzz)
	self.ps, err = pss.NewPss(to, config.Pss)
	if err != nil {
		return nil, err
	}
	if pss.IsActiveHandshake {
		pss.SetHandshakeController(self.ps, pss.NewHandshakeParams())
	}

	self.api = api.NewAPI(self.fileStore, self.dns, feedsHandler, self.privateKey)

	self.sfs = fuse.NewSwarmFS(self.api)
	log.Debug("Initialized FUSE filesystem")

	return self, nil
}

// parseEnsAPIAddress parses string according to format
// [tld:][contract-addr@]url and returns ENSClientConfig structure
// with endpoint, contract address and TLD.
func parseEnsAPIAddress(s string) (tld, endpoint string, addr common.Address) {
	isAllLetterString := func(s string) bool {
		for _, r := range s {
			if !unicode.IsLetter(r) {
				return false
			}
		}
		return true
	}
	endpoint = s
	if i := strings.Index(endpoint, ":"); i > 0 {
		if isAllLetterString(endpoint[:i]) && len(endpoint) > i+2 && endpoint[i+1:i+3] != "//" {
			tld = endpoint[:i]
			endpoint = endpoint[i+1:]
		}
	}
	if i := strings.Index(endpoint, "@"); i > 0 {
		addr = common.HexToAddress(endpoint[:i])
		endpoint = endpoint[i+1:]
	}
	return
}

// ensClient provides functionality for api.ResolveValidator
type ensClient struct {
	*ens.ENS
	*ethclient.Client
}

// newEnsClient creates a new ENS client for that is a consumer of
// a ENS API on a specific endpoint. It is used as a helper function
// for creating multiple resolvers in NewSwarm function.
func newEnsClient(endpoint string, addr common.Address, config *api.Config, privkey *ecdsa.PrivateKey) (*ensClient, error) {
	log.Info("connecting to ENS API", "url", endpoint)
	client, err := rpc.Dial(endpoint)
	if err != nil {
		return nil, fmt.Errorf("error connecting to ENS API %s: %s", endpoint, err)
	}
	ethClient := ethclient.NewClient(client)

	ensRoot := config.EnsRoot
	if addr != (common.Address{}) {
		ensRoot = addr
	} else {
		a, err := detectEnsAddr(client)
		if err == nil {
			ensRoot = a
		} else {
			log.Warn(fmt.Sprintf("could not determine ENS contract address, using default %s", ensRoot), "err", err)
		}
	}
	transactOpts := bind.NewKeyedTransactor(privkey)
	dns, err := ens.NewENS(transactOpts, ensRoot, ethClient)
	if err != nil {
		return nil, err
	}
	log.Debug(fmt.Sprintf("-> Swarm Domain Name Registrar %v @ address %v", endpoint, ensRoot.Hex()))
	return &ensClient{
		ENS:    dns,
		Client: ethClient,
	}, err
}

// detectEnsAddr determines the ENS contract address by getting both the
// version and genesis hash using the client and matching them to either
// mainnet or testnet addresses
func detectEnsAddr(client *rpc.Client) (common.Address, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var version string
	if err := client.CallContext(ctx, &version, "net_version"); err != nil {
		return common.Address{}, err
	}

	block, err := ethclient.NewClient(client).BlockByNumber(ctx, big.NewInt(0))
	if err != nil {
		return common.Address{}, err
	}

	switch {

	case version == "1" && block.Hash() == params.MainnetGenesisHash:
		log.Info("using Mainnet ENS contract address", "addr", ens.MainNetAddress)
		return ens.MainNetAddress, nil

	case version == "3" && block.Hash() == params.TestnetGenesisHash:
		log.Info("using Testnet ENS contract address", "addr", ens.TestNetAddress)
		return ens.TestNetAddress, nil

	default:
		return common.Address{}, fmt.Errorf("unknown version and genesis hash: %s %s", version, block.Hash())
	}
}

/*
Start is called when the stack is started
* starts the network kademlia hive peer management
* (starts netStore level 0 api)
* starts DPA level 1 api (chunking -> store/retrieve requests)
* (starts level 2 api)
* starts http proxy server
* registers url scheme handlers for bzz, etc
* TODO: start subservices like sword, swear, swarmdns
*/
// implements the node.Service interface
func (s *Swarm) Start(srv *p2p.Server) error {
	startTime := time.Now()

	s.tracerClose = tracing.Closer

	// update uaddr to correct enode
	newaddr := s.bzz.UpdateLocalAddr([]byte(srv.Self().String()))
	log.Info("Updated bzz local addr", "oaddr", fmt.Sprintf("%x", newaddr.OAddr), "uaddr", fmt.Sprintf("%s", newaddr.UAddr))
	// set chequebook
	//TODO: Currently if swap is enabled and no chequebook (or inexistent) contract is provided, the node would crash.
	//Once we integrate back the contracts, this check MUST be revisited
	if s.config.SwapEnabled && s.config.SwapAPI != "" {
		ctx := context.Background() // The initial setup has no deadline.
		err := s.SetChequebook(ctx)
		if err != nil {
			return fmt.Errorf("Unable to set chequebook for SWAP: %v", err)
		}
		log.Debug(fmt.Sprintf("-> cheque book for SWAP: %v", s.config.Swap.Chequebook()))
	} else {
		log.Debug(fmt.Sprintf("SWAP disabled: no cheque book set"))
	}

	log.Info("Starting bzz service")

	err := s.bzz.Start(srv)
	if err != nil {
		log.Error("bzz failed", "err", err)
		return err
	}
	log.Info("Swarm network started", "bzzaddr", fmt.Sprintf("%x", s.bzz.Hive.BaseAddr()))

	if s.ps != nil {
		s.ps.Start(srv)
	}

	// start swarm http proxy server
	if s.config.Port != "" {
		addr := net.JoinHostPort(s.config.ListenAddr, s.config.Port)
		server := httpapi.NewServer(s.api, s.config.Cors)

		if s.config.Cors != "" {
			log.Debug("Swarm HTTP proxy CORS headers", "allowedOrigins", s.config.Cors)
		}

		log.Debug("Starting Swarm HTTP proxy", "port", s.config.Port)
		go func() {
			err := server.ListenAndServe(addr)
			if err != nil {
				log.Error("Could not start Swarm HTTP proxy", "err", err.Error())
			}
		}()
	}

	doneC := make(chan struct{})

	s.cleanupFuncs = append(s.cleanupFuncs, func() error {
		close(doneC)
		return nil
	})

	go func(time.Time) {
		for {
			select {
			case <-time.After(updateGaugesPeriod):
				uptimeGauge.Update(time.Since(startTime).Nanoseconds())
				requestsCacheGauge.Update(int64(s.netStore.RequestsCacheLen()))
			case <-doneC:
				return
			}
		}
	}(startTime)

	startCounter.Inc(1)
	s.streamer.Start(srv)
	return nil
}

// implements the node.Service interface
// stops all component services.
func (s *Swarm) Stop() error {
	if s.tracerClose != nil {
		err := s.tracerClose.Close()
		tracing.FinishSpans()
		if err != nil {
			return err
		}
	}

	if s.ps != nil {
		s.ps.Stop()
	}
	if ch := s.config.Swap.Chequebook(); ch != nil {
		ch.Stop()
		ch.Save()
	}
	if s.swap != nil {
		s.swap.Close()
	}
	if s.accountingMetrics != nil {
		s.accountingMetrics.Close()
	}
	if s.netStore != nil {
		s.netStore.Close()
	}
	s.sfs.Stop()
	stopCounter.Inc(1)
	s.streamer.Stop()

	err := s.bzz.Stop()
	if s.stateStore != nil {
		s.stateStore.Close()
	}

	for _, cleanF := range s.cleanupFuncs {
		err = cleanF()
		if err != nil {
			log.Error("encountered an error while running cleanup function", "err", err)
			break
		}
	}
	return err
}

// Protocols implements the node.Service interface
func (s *Swarm) Protocols() (protos []p2p.Protocol) {
	if s.config.BootnodeMode {
		protos = append(protos, s.bzz.Protocols()...)
	} else {
		protos = append(protos, s.bzz.Protocols()...)

		if s.ps != nil {
			protos = append(protos, s.ps.Protocols()...)
		}
	}
	return
}

// implements node.Service
// APIs returns the RPC API descriptors the Swarm implementation offers
func (s *Swarm) APIs() []rpc.API {

	apis := []rpc.API{
		// public APIs
		{
			Namespace: "bzz",
			Version:   "3.0",
			Service:   &Info{s.config, chequebook.ContractParams},
			Public:    true,
		},
		// admin APIs
		{
			Namespace: "bzz",
			Version:   "3.0",
			Service:   api.NewInspector(s.api, s.bzz.Hive, s.netStore),
			Public:    false,
		},
		{
			Namespace: "chequebook",
			Version:   chequebook.Version,
			Service:   chequebook.NewAPI(s.config.Swap.Chequebook),
			Public:    false,
		},
		{
			Namespace: "swarmfs",
			Version:   fuse.SwarmFSVersion,
			Service:   s.sfs,
			Public:    false,
		},
		{
			Namespace: "accounting",
			Version:   protocols.AccountingVersion,
			Service:   protocols.NewAccountingApi(s.accountingMetrics),
			Public:    false,
		},
	}

	apis = append(apis, s.bzz.APIs()...)

	if s.ps != nil {
		apis = append(apis, s.ps.APIs()...)
	}

	return apis
}

// SetChequebook ensures that the local checquebook is set up on chain.
func (s *Swarm) SetChequebook(ctx context.Context) error {
	err := s.config.Swap.SetChequebook(ctx, s.backend, s.config.Path)
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("new chequebook set (%v): saving config file, resetting all connections in the hive", s.config.Swap.Contract.Hex()))
	return nil
}

// RegisterPssProtocol adds a devp2p protocol to the swarm node's Pss instance
func (s *Swarm) RegisterPssProtocol(topic *pss.Topic, spec *protocols.Spec, targetprotocol *p2p.Protocol, options *pss.ProtocolParams) (*pss.Protocol, error) {
	return pss.RegisterProtocol(s.ps, topic, spec, targetprotocol, options)
}

// serialisable info about swarm
type Info struct {
	*api.Config
	*chequebook.Params
}

func (s *Info) Info() *Info {
	return s
}
