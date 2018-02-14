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

package swarm

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"fmt"
	"net"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/chequebook"
	"github.com/ethereum/go-ethereum/contracts/ens"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/api"
	httpapi "github.com/ethereum/go-ethereum/swarm/api/http"
	"github.com/ethereum/go-ethereum/swarm/fuse"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/stream"
	"github.com/ethereum/go-ethereum/swarm/pss"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
)

// the swarm stack
type Swarm struct {
	config *api.Config  // swarm configuration
	api    *api.Api     // high level api layer (fs/manifest)
	dns    api.Resolver // DNS registrar
	//dbAccess    *network.DbAccess      // access to local chunk db iterator and storage counter
	//storage storage.ChunkStore // internal access to storage, common interface to cloud storage backends
	dpa *storage.DPA // distributed preimage archive, the local API to the storage with document level storage/retrieval support
	//depo        network.StorageHandler // remote request handler, interface between bzz protocol and the storage
	streamer *stream.Registry
	//cloud       storage.CloudStore // procurement, cloud storage backend (can multi-cloud)
	bzz         *network.Bzz       // the logistic manager
	backend     chequebook.Backend // simple blockchain Backend
	privateKey  *ecdsa.PrivateKey
	corsString  string
	swapEnabled bool
	lstore      *storage.LocalStore // local store, needs to store for releasing resources after node stopped
	sfs         *fuse.SwarmFS       // need this to cleanup all the active mounts on node exit
	ps          *pss.Pss
}

type SwarmAPI struct {
	Api     *api.Api
	Backend chequebook.Backend
	PrvKey  *ecdsa.PrivateKey
}

func (self *Swarm) API() *SwarmAPI {
	return &SwarmAPI{
		Api:     self.api,
		Backend: self.backend,
		PrvKey:  self.privateKey,
	}
}

// creates a new swarm service instance
// implements node.Service
// If mockStore is not nil, it will be used as the storage for chunk data.
// MockStore should be used only for testing.
func NewSwarm(ctx *node.ServiceContext, backend chequebook.Backend, ensClient *ethclient.Client, config *api.Config, mockStore *mock.NodeStore) (self *Swarm, err error) {

	if bytes.Equal(common.FromHex(config.PublicKey), storage.ZeroKey) {
		return nil, fmt.Errorf("empty public key")
	}
	if bytes.Equal(common.FromHex(config.BzzKey), storage.ZeroKey) {
		return nil, fmt.Errorf("empty bzz key")
	}

	self = &Swarm{
		config:     config,
		backend:    backend,
		privateKey: config.ShiftPrivateKey(),
	}
	log.Debug(fmt.Sprintf("Setting up Swarm service components"))

	hash := storage.MakeHashFunc(config.ChunkerParams.Hash)
	self.lstore, err = storage.NewLocalStore(hash, config.StoreParams, common.Hex2Bytes(config.BzzKey), mockStore)
	if err != nil {
		return
	}

	// setup local store
	log.Debug(fmt.Sprintf("Set up local storage"))

	kp := network.NewKadParams()
	to := network.NewKademlia(
		common.FromHex(config.BzzKey),
		kp,
	)

	config.HiveParams.Discovery = true

	// setup cloud storage internal access layer
	//self.cloud = &storage.Forwarder{}
	//self.storage = storage.NewNetStore(hash, self.lstore, self.cloud, config.StoreParams)
	log.Debug(fmt.Sprintf("-> swarm net store shared access layer to Swarm Chunk Store"))
	nodeid := discover.PubkeyID(crypto.ToECDSAPub(common.FromHex(config.PublicKey)))
	addr := network.NewAddrFromNodeID(nodeid)
	bzzconfig := &network.BzzConfig{
		OverlayAddr:  common.FromHex(config.BzzKey),
		UnderlayAddr: addr.UAddr,
		HiveParams:   config.HiveParams,
	}

	db := storage.NewDBAPI(self.lstore)
	delivery := stream.NewDelivery(to, db)
	self.streamer = stream.NewRegistry(addr, delivery, self.lstore, false)
	stream.RegisterSwarmSyncerServer(self.streamer, db)
	stream.RegisterSwarmSyncerClient(self.streamer, db)

	self.bzz = network.NewBzz(bzzconfig, to, nil)

	// set up DPA, the cloud storage local access layer
	dpaChunkStore := storage.NewNetStore(self.lstore, self.streamer.Retrieve)
	log.Debug(fmt.Sprintf("-> Local Access to Swarm"))
	// Swarm Hash Merklised Chunking for Arbitrary-length Document/File storage
	self.dpa = storage.NewDPA(dpaChunkStore, self.config.ChunkerParams)
	log.Debug(fmt.Sprintf("-> Content Store API"))

	// Pss = postal service over swarm (devp2p over bzz)
	if self.config.PssEnabled {
		pssparams := pss.NewPssParams(self.privateKey)
		self.ps = pss.NewPss(to, self.dpa, pssparams)
		if pss.IsActiveHandshake {
			pss.SetHandshakeController(self.ps, pss.NewHandshakeParams())
		}
	}

	// set up high level api
	transactOpts := bind.NewKeyedTransactor(self.privateKey)

	if ensClient == nil {
		log.Warn("No ENS, please specify non-empty --ens-api to use domain name resolution")
	} else {
		self.dns, err = ens.NewENS(transactOpts, config.EnsRoot, ensClient)
		if err != nil {
			return nil, err
		}
	}
	log.Debug(fmt.Sprintf("-> Swarm Domain Name Registrar @ address %v", config.EnsRoot.Hex()))

	var resourceHandler *storage.ResourceHandler
	// if use resource updates
	if self.config.ResourceEnabled {
		var resourceValidator storage.ResourceValidator
		if self.dns != nil {
			resourceValidator, err = storage.NewENSValidator(config.EnsRoot, ensClient, transactOpts, storage.NewGenericResourceSigner(self.privateKey))
			if err != nil {
				return nil, err
			}
		}
		hashfunc := storage.MakeHashFunc(storage.SHA3Hash)
		chunkStore := storage.NewResourceChunkStore(self.lstore, func(*storage.Chunk) error { return nil })
		resourceHandler, err = storage.NewResourceHandler(hashfunc, chunkStore, ensClient, resourceValidator)
		if err != nil {
			return nil, err
		}
	}

	self.api = api.NewApi(self.dpa, self.dns, resourceHandler)
	// Manifests for Smart Hosting
	log.Debug(fmt.Sprintf("-> Web3 virtual server API"))

	self.sfs = fuse.NewSwarmFS(self.api)
	log.Debug("-> Initializing Fuse file system")

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
func (self *Swarm) Start(srv *p2p.Server) error {

	// update uaddr to correct enode
	newaddr := self.bzz.UpdateLocalAddr([]byte(srv.Self().String()))
	log.Warn("Updated bzz local addr", "oaddr", fmt.Sprintf("%x", newaddr.OAddr), "uaddr", fmt.Sprintf("%s", newaddr.UAddr))

	// set chequebook
	if self.config.SwapEnabled {
		ctx := context.Background() // The initial setup has no deadline.
		err := self.SetChequebook(ctx)
		if err != nil {
			return fmt.Errorf("Unable to set chequebook for SWAP: %v", err)
		}
		log.Debug(fmt.Sprintf("-> cheque book for SWAP: %v", self.config.Swap.Chequebook()))
	} else {
		log.Debug(fmt.Sprintf("SWAP disabled: no cheque book set"))
	}

	log.Warn(fmt.Sprintf("Starting Swarm service"))

	err := self.bzz.Start(srv)
	if err != nil {
		log.Error("bzz failed", "err", err)
		return err
	}
	log.Info(fmt.Sprintf("Swarm network started on bzz address: %x", self.bzz.Hive.Overlay.BaseAddr()))

	if self.ps != nil {
		self.ps.Start(srv)
		log.Info("Pss started")
	}

	self.dpa.Start()
	log.Debug(fmt.Sprintf("Swarm DPA started"))

	// start swarm http proxy server
	if self.config.Port != "" {
		addr := net.JoinHostPort(self.config.ListenAddr, self.config.Port)
		go httpapi.StartHttpServer(self.api, &httpapi.ServerConfig{
			Addr:       addr,
			CorsString: self.config.Cors,
		})
	}

	log.Debug(fmt.Sprintf("Swarm http proxy started on port: %v", self.config.Port))

	if self.config.Cors != "" {
		log.Debug(fmt.Sprintf("Swarm http proxy started with corsdomain: %v", self.config.Cors))
	}

	return nil
}

// implements the node.Service interface
// stops all component services.
func (self *Swarm) Stop() error {
	self.dpa.Stop()
	if self.ps != nil {
		self.ps.Stop()
	}
	if ch := self.config.Swap.Chequebook(); ch != nil {
		ch.Stop()
		ch.Save()
	}

	if self.lstore != nil {
		self.lstore.DbStore.Close()
	}
	self.sfs.Stop()
	return self.bzz.Stop()
}

// implements the node.Service interface
func (self *Swarm) Protocols() (protos []p2p.Protocol) {
	protos = append(protos, self.bzz.Protocols()...)

	if self.ps != nil {
		protos = append(protos, self.ps.Protocols()...)
	}
	if self.streamer != nil {
		protos = append(protos, self.streamer.Protocols()...)
	}
	return
}

func (self *Swarm) RegisterPssProtocol(spec *protocols.Spec, targetprotocol *p2p.Protocol, options *pss.ProtocolParams) (*pss.Protocol, error) {
	if !pss.IsActiveProtocol {
		return nil, fmt.Errorf("Pss protocols not available (built with !nopssprotocol tag)")
	}
	topic := pss.ProtocolTopic(spec)
	return pss.RegisterProtocol(self.ps, &topic, spec, targetprotocol, options)
}

// implements node.Service
// APIs returns the RPC Api descriptors the Swarm implementation offers
func (self *Swarm) APIs() []rpc.API {

	apis := []rpc.API{
		// public APIs
		{
			Namespace: "bzz",
			Version:   "0.1",
			Service:   &Info{self.config, chequebook.ContractParams},
			Public:    true,
		},
		// admin APIs
		{
			Namespace: "bzz",
			Version:   "0.1",
			Service:   api.NewControl(self.api, self.bzz.Hive),
			Public:    false,
		},
		{
			Namespace: "chequebook",
			Version:   chequebook.Version,
			Service:   chequebook.NewApi(self.config.Swap.Chequebook),
			Public:    false,
		},
		{
			Namespace: "swarmfs",
			Version:   fuse.Swarmfs_Version,
			Service:   self.sfs,
			Public:    false,
		},
		// storage APIs
		// DEPRECATED: Use the HTTP API instead
		{
			Namespace: "bzz",
			Version:   "0.1",
			Service:   api.NewStorage(self.api),
			Public:    true,
		},
		{
			Namespace: "bzz",
			Version:   "0.1",
			Service:   api.NewFileSystem(self.api),
			Public:    false,
		},
		// {Namespace, Version, api.NewAdmin(self), false},
	}

	apis = append(apis, self.bzz.APIs()...)

	if self.ps != nil {
		apis = append(apis, self.ps.APIs()...)
	}

	return apis
}

func (self *Swarm) Api() *api.Api {
	return self.api
}

// SetChequebook ensures that the local checquebook is set up on chain.
func (self *Swarm) SetChequebook(ctx context.Context) error {
	err := self.config.Swap.SetChequebook(ctx, self.backend, self.config.Path)
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("new chequebook set (%v): saving config file, resetting all connections in the hive", self.config.Swap.Contract.Hex()))
	return nil
}

// serialisable info about swarm
type Info struct {
	*api.Config
	*chequebook.Params
}

func (self *Info) Info() *Info {
	return self
}
