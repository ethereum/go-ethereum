package swarm

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/chequebook"
	"github.com/ethereum/go-ethereum/common/httpclient"
	"github.com/ethereum/go-ethereum/common/registrar/ethreg"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/swarm/api"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

// the swarm stack
type Swarm struct {
	config   *api.Config            // swarm configuration
	api      *api.Api               // high level api layer (fs/manifest)
	dbAccess *network.DbAccess      // access to local chunk db iterator and storage counter
	storage  storage.ChunkStore     // internal access to storage, common interface to cloud storage backends
	dpa      *storage.DPA           // distributed preimage archive, the local API to the storage with document level storage/retrieval support
	depo     network.StorageHandler // remote request handler, interface between bzz protocol and the storage
	cloud    storage.CloudStore     // procurement, cloud storage backend (can multi-cloud)
	hive     *network.Hive          // the logistic manager
	client   *httpclient.HTTPClient // bzz capable light http client
}

// creates a new swarm service instance
// implements node.Service
func NewSwarm(stack *node.ServiceContext, config *api.Config, swapEnabled bool) (self *Swarm, err error) {

	if bytes.Equal(common.FromHex(config.PublicKey), storage.ZeroKey) {
		return nil, fmt.Errorf("empty public key")
	}
	if bytes.Equal(common.FromHex(config.BzzKey), storage.ZeroKey) {
		return nil, fmt.Errorf("empty bzz key")
	}

	var ethereum *eth.Ethereum
	if err := stack.Service(&ethereum); err != nil {
		return nil, fmt.Errorf("unable to find Ethereum service: %v", err)
	}
	self = &Swarm{
		config: config,
		client: ethereum.HTTPClient(),
	}
	glog.V(logger.Debug).Infof("[BZZ] Setting up Swarm service components")

	// setup local store
	hash := storage.MakeHashFunc(config.ChunkerParams.Hash)
	lstore, err := storage.NewLocalStore(hash, config.StoreParams)
	if err != nil {
		return
	}
	glog.V(logger.Debug).Infof("[BZZ] Set up local storage")

	self.dbAccess = network.NewDbAccess(lstore)
	glog.V(logger.Debug).Infof("[BZZ] Set up local db access (iterator/counter)")

	// set up the kademlia hive
	self.hive = network.NewHive(
		common.HexToHash(self.config.BzzKey), // key to hive (kademlia base address)
		config.HiveParams,                    // configuration parameters
	)
	glog.V(logger.Debug).Infof("[BZZ] Set up swarm network with Kademlia hive")

	// setup cloud storage backend
	cloud := network.NewForwarder(self.hive)
	glog.V(logger.Debug).Infof("[BZZ] -> set swarm forwarder as cloud storage backend")
	// setup cloud storage internal access layer

	self.storage = storage.NewNetStore(hash, lstore, cloud, config.StoreParams)
	glog.V(logger.Debug).Infof("[BZZ] -> Level 0: swarm net store shared access layer to Swarm Chunk Store")

	// set up Depo (storage handler = remote cloud storage access layer)
	self.depo = network.NewDepo(hash, lstore, self.storage)
	glog.V(logger.Debug).Infof("[BZZ] -> REmote Access to CHunks")

	// set up DPA, the cloud storage local access layer
	dpaChunkStore := storage.NewDpaChunkStore(lstore, self.storage)
	glog.V(logger.Debug).Infof("[BZZ] -> Local Access to Swarm")
	// Swarm Hash Merklised Chunking for Arbitrary-length Document/File storage
	self.dpa = storage.NewDPA(dpaChunkStore, self.config.ChunkerParams)
	glog.V(logger.Debug).Infof("[BZZ] -> Level 1: Document/File API")

	// set up high level api
	backend := api.NewEthApi(ethereum)
	backend.UpdateState()
	self.api = api.NewApi(self.dpa, ethreg.New(backend), self.config)
	// Manifests for Smart Hosting
	glog.V(logger.Debug).Infof("[BZZ] -> Level 2: Collection/Directory API")

	// set chequebook
	if swapEnabled {
		err = self.SetChequebook(backend)
		if err != nil {
			return nil, fmt.Errorf("Unable to set chequebook for SWAP: %v", err)
		}
		glog.V(logger.Debug).Infof("[BZZ] -> cheque book for SWAP")
	}
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
		go api.StartHttpServer(self.api, self.config.Port)
	}
	glog.V(logger.Debug).Infof("[BZZ] Swarm http proxy started on port: %v", self.config.Port)

	// register roundtripper (using proxy) as bzz scheme handler
	// for the ethereum http client
	// this is a place holder until schemes and ports are properly mapped in config
	schemes := map[string]string{
		"bzz": self.config.Port,
	}
	for scheme, port := range schemes {
		self.client.RegisterScheme(scheme, &api.RoundTripper{Port: port})
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
	proto, err := network.Bzz(self.depo, self.hive, self.dbAccess, self.config.Swap, self.config.SyncParams)
	if err != nil {
		return nil
	}
	return []p2p.Protocol{proto}
}

func (self *Swarm) Api() *api.Api {
	return self.api
}

// Backend interface implemented by eth or JSON-IPC client
func (self *Swarm) SetChequebook(backend chequebook.Backend) (err error) {
	done, err := self.config.Swap.SetChequebook(self.config.Path, backend)
	if err != nil {
		return err
	}
	go func() {
		ok := <-done
		if ok {
			glog.V(logger.Info).Infof("[BZZ] Swarm: new chequebook set (%v): saving config file, resetting all connections in the hive", self.config.Swap.Contract)
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
		api:    api.NewApi(dpa, nil, config),
		config: config,
	}

	return
}
