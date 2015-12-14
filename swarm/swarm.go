package swarm

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/httpclient"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/api"
	httpapi "github.com/ethereum/go-ethereum/swarm/api/http"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/services/chequebook"
	"github.com/ethereum/go-ethereum/swarm/services/ens"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const (
	Namespace = "bzz"
	Version   = "0.1" // versioning reflect POC and release versions
)

var ENSContractAddr = common.HexToAddress("0x504cdf3992d8f81a4182bd7b24e270d3a28711e3")

// the swarm stack
type Swarm struct {
	ethereum    *eth.Ethereum
	config      *api.Config            // swarm configuration
	api         *api.Api               // high level api layer (fs/manifest)
	dns         api.Resolver           // DNS registrar
	dbAccess    *network.DbAccess      // access to local chunk db iterator and storage counter
	storage     storage.ChunkStore     // internal access to storage, common interface to cloud storage backends
	dpa         *storage.DPA           // distributed preimage archive, the local API to the storage with document level storage/retrieval support
	depo        network.StorageHandler // remote request handler, interface between bzz protocol and the storage
	cloud       storage.CloudStore     // procurement, cloud storage backend (can multi-cloud)
	hive        *network.Hive          // the logistic manager
	client      *httpclient.HTTPClient // bzz capable light http client
	backend     bind.Backend           // simple blockchain Backend
	privateKey  *ecdsa.PrivateKey
	swapEnabled bool
}

type SwarmAPI struct {
	Api     *api.Api
	Backend bind.Backend
	PrvKey  *ecdsa.PrivateKey
}

func (self *Swarm) API() *SwarmAPI {
	return &SwarmAPI{
		Api:     self.api,
		Backend: self.backend,
		PrvKey:  self.privateKey,
	}
}

type Backend interface {
	GetTxReceipt(txhash common.Hash) *types.Receipt
	BalanceAt(address common.Address) *big.Int
}

// creates a new swarm service instance
// implements node.Service
func NewSwarm(ctx *node.ServiceContext, config *api.Config, swapEnabled, syncEnabled bool) (self *Swarm, err error) {

	if bytes.Equal(common.FromHex(config.PublicKey), storage.ZeroKey) {
		return nil, fmt.Errorf("empty public key")
	}
	if bytes.Equal(common.FromHex(config.BzzKey), storage.ZeroKey) {
		return nil, fmt.Errorf("empty bzz key")
	}

	var ethereum *eth.Ethereum
	if err := ctx.Service(&ethereum); err != nil {
		return nil, fmt.Errorf("unable to find Ethereum service: %v", err)
	}
	self = &Swarm{
		config:      config,
		ethereum:    ethereum,
		swapEnabled: swapEnabled,
		client:      ethereum.HTTPClient(),
		privateKey:  config.Swap.PrivateKey(),
	}
	glog.V(logger.Debug).Infof("[BZZ] Setting up Swarm service components")

	hash := storage.MakeHashFunc(config.ChunkerParams.Hash)
	lstore, err := storage.NewLocalStore(hash, config.StoreParams)
	if err != nil {
		return
	}

	// setup local store
	glog.V(logger.Debug).Infof("[BZZ] Set up local storage")

	self.dbAccess = network.NewDbAccess(lstore)
	glog.V(logger.Debug).Infof("[BZZ] Set up local db access (iterator/counter)")

	// set up the kademlia hive
	self.hive = network.NewHive(
		common.HexToHash(self.config.BzzKey), // key to hive (kademlia base address)
		config.HiveParams,                    // configuration parameters
		swapEnabled,                          // SWAP enabled
		syncEnabled,                          // syncronisation enabled
	)
	glog.V(logger.Debug).Infof("[BZZ] Set up swarm network with Kademlia hive")

	// setup cloud storage backend
	cloud := network.NewForwarder(self.hive)
	glog.V(logger.Debug).Infof("[BZZ] -> set swarm forwarder as cloud storage backend")
	// setup cloud storage internal access layer

	self.storage = storage.NewNetStore(hash, lstore, cloud, config.StoreParams)
	glog.V(logger.Debug).Infof("[BZZ] -> swarm net store shared access layer to Swarm Chunk Store")

	// set up Depo (storage handler = cloud storage access layer for incoming remote requests)
	self.depo = network.NewDepo(hash, lstore, self.storage)
	glog.V(logger.Debug).Infof("[BZZ] -> REmote Access to CHunks")

	// set up DPA, the cloud storage local access layer
	dpaChunkStore := storage.NewDpaChunkStore(lstore, self.storage)
	glog.V(logger.Debug).Infof("[BZZ] -> Local Access to Swarm")
	// Swarm Hash Merklised Chunking for Arbitrary-length Document/File storage
	self.dpa = storage.NewDPA(dpaChunkStore, self.config.ChunkerParams)
	glog.V(logger.Debug).Infof("[BZZ] -> Content Store API")

	// set up blockchain and contract interface bindings
	glog.V(logger.Debug).Infof("[BZZ] -> native backend for abigen contract bindings")
	self.backend = eth.NewContractBackend(ethereum)

	// set up high level api
	transactOpts := bind.NewKeyedTransactor(self.privateKey)
	// backend := ethereum.ContractBackend()
	self.dns = ens.NewENS(transactOpts, ENSContractAddr, self.backend)
	glog.V(logger.Debug).Infof("[BZZ] -> Swarm Domain Name Registrar")

	self.api = api.NewApi(self.dpa, self.dns)
	// Manifests for Smart Hosting
	glog.V(logger.Debug).Infof("[BZZ] -> Web3 virtual server API")

	return self, nil
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
func (self *Swarm) Start(net *p2p.Server) error {
	connectPeer := func(url string) error {
		node, err := discover.ParseNode(url)
		if err != nil {
			return fmt.Errorf("invalid node URL: %v", err)
		}
		net.AddPeer(node)
		return nil
	}
	// set chequebook
	if self.swapEnabled {
		err := self.SetChequebook()
		if err != nil {
			return fmt.Errorf("Unable to set chequebook for SWAP: %v", err)
		}
		glog.V(logger.Debug).Infof("[BZZ] -> cheque book for SWAP: %v", self.config.Swap.Chequebook())
	} else {
		glog.V(logger.Debug).Infof("[BZZ] SWAP disabled: no cheque book set")
	}

	glog.V(logger.Warn).Infof("[BZZ] Starting Swarm service")
	self.hive.Start(
		discover.PubkeyID(&net.PrivateKey.PublicKey),
		func() string { return net.ListenAddr },
		connectPeer,
	)
	glog.V(logger.Info).Infof("[BZZ] Swarm network started on bzz address: %v", self.hive.Addr())

	self.dpa.Start()
	glog.V(logger.Debug).Infof("[BZZ] Swarm DPA started")

	// start swarm http proxy server
	if self.config.Port != "" {
		go httpapi.StartHttpServer(self.api, self.config.Port)
	}
	glog.V(logger.Debug).Infof("[BZZ] Swarm http proxy started on port: %v", self.config.Port)

	// register roundtripper (using proxy) as bzz scheme handler
	// for the ethereum http client
	// this is a place holder until schemes and ports are properly mapped in config
	schemes := map[string]string{
		"bzz": self.config.Port,
	}
	for scheme, port := range schemes {
		self.client.RegisterScheme(scheme, &httpapi.RoundTripper{Port: port})
	}
	glog.V(logger.Debug).Infof("[BZZ] Swarm protocol handlers registered for url schemes: %v", schemes)

	return nil
}

// implements the node.Service interface
// stops all component services.
func (self *Swarm) Stop() error {
	self.dpa.Stop()
	self.hive.Stop()
	if ch := self.config.Swap.Chequebook(); ch != nil {
		ch.Stop()
		ch.Save()
	}
	return self.config.Save()
}

// implements the node.Service interface
func (self *Swarm) Protocols() []p2p.Protocol {
	proto, err := network.Bzz(self.depo, self.backend, self.hive, self.dbAccess, self.config.Swap, self.config.SyncParams)
	if err != nil {
		return nil
	}
	return []p2p.Protocol{proto}
}

// implements node.Service
// Apis returns the RPC Api descriptors the Swarm implementation offers
func (self *Swarm) APIs() []rpc.API {
	return []rpc.API{
		// public APIs.
		rpc.API{Namespace, Version, api.NewStorage(self.api), true},
		rpc.API{"ens", Version, self.dns, true},
		rpc.API{Namespace, Version, &Info{self.config, chequebook.ContractParams}, true},
		// admin APIs
		rpc.API{Namespace, Version, api.NewFileSystem(self.api), false},
		rpc.API{Namespace, Version, api.NewControl(self.api, self.hive), false},
		// rpc.API{Namespace, Version, api.NewAdmin(self), false},
		// TODO: external apis exposed
		rpc.API{"chequebook", chequebook.Version, chequebook.NewApi(self.config.Swap.Chequebook), true},
	}
}

func (self *Swarm) Api() *api.Api {
	return self.api
}

//
func (self *Swarm) SetChequebook() (err error) {
	done, err := self.config.Swap.SetChequebook(self.config.Path, self.backend)
	if err != nil {
		return err
	}
	go func() {
		ok := <-done
		if ok {
			glog.V(logger.Info).Infof("[BZZ] Swarm: new chequebook set (%v): saving config file, resetting all connections in the hive", self.config.Swap.Contract.Hex())
			self.config.Save()
			self.hive.DropAll()
		}
	}()
	return nil
}

// Local swarm without netStore
func NewLocalSwarm(datadir, port string) (self *Swarm, err error) {

	prvKey, err := crypto.GenerateKey()
	if err != nil {
		return
	}

	config, err := api.NewConfig(datadir, common.Address{}, prvKey)
	if err != nil {
		return
	}
	config.Port = port

	dpa, err := storage.NewLocalDPA(datadir)
	if err != nil {
		return
	}

	self = &Swarm{
		api:    api.NewApi(dpa, nil),
		config: config,
	}

	return
}

// serialisable info about swarm
type Info struct {
	*api.Config
	*chequebook.Params
}

func (self *Info) Info() *Info {
	return self
}
