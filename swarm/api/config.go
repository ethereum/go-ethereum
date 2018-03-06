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

package api

import (
	"crypto/ecdsa"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/ens"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/services/swap"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const (
	DefaultHTTPListenAddr = "127.0.0.1"
	DefaultHTTPPort       = "8500"
)

// separate bzz directories
// allow several bzz nodes running in parallel
type Config struct {
	// serialised/persisted fields
	*storage.StoreParams
	*storage.ChunkerParams
	*network.HiveParams
	Swap *swap.SwapParams
	//*network.SyncParams
	Contract        common.Address
	EnsRoot         common.Address
	EnsAPIs         []string
	Path            string
	ListenAddr      string
	Port            string
	PublicKey       string
	BzzKey          string
	NetworkId       uint64
	SwapEnabled     bool
	SyncEnabled     bool
	PssEnabled      bool
	ResourceEnabled bool
	SwapApi         string
	Cors            string
	BzzAccount      string
	BootNodes       string
	privateKey      *ecdsa.PrivateKey
}

//create a default config with all parameters to set to defaults
func NewConfig() (self *Config) {

	self = &Config{
		StoreParams:   storage.NewDefaultStoreParams(),
		ChunkerParams: storage.NewChunkerParams(),
		HiveParams:    network.NewHiveParams(),
		//SyncParams:    network.NewDefaultSyncParams(),
		Swap:            swap.NewDefaultSwapParams(),
		ListenAddr:      DefaultHTTPListenAddr,
		Port:            DefaultHTTPPort,
		Path:            node.DefaultDataDir(),
		EnsAPIs:         nil,
		EnsRoot:         ens.TestNetAddress,
		NetworkId:       network.NetworkID,
		SwapEnabled:     false,
		SyncEnabled:     true,
		PssEnabled:      true,
		ResourceEnabled: true,
		SwapApi:         "",
		BootNodes:       "",
	}

	return
}

//some config params need to be initialized after the complete
//config building phase is completed (e.g. due to overriding flags)
func (self *Config) Init(prvKey *ecdsa.PrivateKey) {

	address := crypto.PubkeyToAddress(prvKey.PublicKey)
	self.Path = filepath.Join(self.Path, "bzz-"+common.Bytes2Hex(address.Bytes()))
	err := os.MkdirAll(self.Path, os.ModePerm)
	if err != nil {
		log.Error(fmt.Sprintf("Error creating root swarm data directory: %v", err))
		return
	}

	pubkey := crypto.FromECDSAPub(&prvKey.PublicKey)
	pubkeyhex := common.ToHex(pubkey)
	keyhex := crypto.Keccak256Hash(pubkey).Hex()

	self.PublicKey = pubkeyhex
	self.BzzKey = keyhex

	if self.SwapEnabled {
		self.Swap.Init(self.Contract, prvKey)
	}
	self.privateKey = prvKey
	self.StoreParams.Init(self.Path)
}

func (self *Config) ShiftPrivateKey() (privKey *ecdsa.PrivateKey) {
	if self.privateKey != nil {
		privKey = self.privateKey
		self.privateKey = nil
	}
	return privKey
}
