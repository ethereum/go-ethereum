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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/contracts/ens"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/pss"
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
	*storage.FileStoreParams
	*storage.LocalStoreParams
	*network.HiveParams
	Swap                 *swap.LocalProfile
	Pss                  *pss.PssParams
	Contract             common.Address
	EnsRoot              common.Address
	EnsAPIs              []string
	Path                 string
	ListenAddr           string
	Port                 string
	PublicKey            string
	BzzKey               string
	Enode                *enode.Node `toml:"-"`
	NetworkID            uint64
	SwapEnabled          bool
	SyncEnabled          bool
	SyncingSkipCheck     bool
	DeliverySkipCheck    bool
	MaxStreamPeerServers int
	LightNodeEnabled     bool
	BootnodeMode         bool
	SyncUpdateDelay      time.Duration
	SwapAPI              string
	Cors                 string
	BzzAccount           string
	GlobalStoreAPI       string
	privateKey           *ecdsa.PrivateKey
}

//create a default config with all parameters to set to defaults
func NewConfig() (c *Config) {

	c = &Config{
		LocalStoreParams:     storage.NewDefaultLocalStoreParams(),
		FileStoreParams:      storage.NewFileStoreParams(),
		HiveParams:           network.NewHiveParams(),
		Swap:                 swap.NewDefaultSwapParams(),
		Pss:                  pss.NewPssParams(),
		ListenAddr:           DefaultHTTPListenAddr,
		Port:                 DefaultHTTPPort,
		Path:                 node.DefaultDataDir(),
		EnsAPIs:              nil,
		EnsRoot:              ens.TestNetAddress,
		NetworkID:            network.DefaultNetworkID,
		SwapEnabled:          false,
		SyncEnabled:          true,
		SyncingSkipCheck:     false,
		MaxStreamPeerServers: 10000,
		DeliverySkipCheck:    true,
		SyncUpdateDelay:      15 * time.Second,
		SwapAPI:              "",
	}

	return
}

//some config params need to be initialized after the complete
//config building phase is completed (e.g. due to overriding flags)
func (c *Config) Init(prvKey *ecdsa.PrivateKey, nodeKey *ecdsa.PrivateKey) error {

	// create swarm dir and record key
	err := c.createAndSetPath(c.Path, prvKey)
	if err != nil {
		return fmt.Errorf("Error creating root swarm data directory: %v", err)
	}
	c.setKey(prvKey)

	// create the new enode record
	// signed with the ephemeral node key
	enodeParams := &network.EnodeParams{
		PrivateKey: prvKey,
		EnodeKey:   nodeKey,
		Lightnode:  c.LightNodeEnabled,
		Bootnode:   c.BootnodeMode,
	}
	c.Enode, err = network.NewEnode(enodeParams)
	if err != nil {
		return fmt.Errorf("Error creating enode: %v", err)
	}

	// initialize components that depend on the swarm instance's private key
	if c.SwapEnabled {
		c.Swap.Init(c.Contract, prvKey)
	}

	c.LocalStoreParams.Init(c.Path)
	c.LocalStoreParams.BaseKey = common.FromHex(c.BzzKey)

	c.Pss = c.Pss.WithPrivateKey(c.privateKey)
	return nil
}

func (c *Config) ShiftPrivateKey() (privKey *ecdsa.PrivateKey) {
	if c.privateKey != nil {
		privKey = c.privateKey
		c.privateKey = nil
	}
	return privKey
}

func (c *Config) setKey(prvKey *ecdsa.PrivateKey) {
	bzzkeybytes := network.PrivateKeyToBzzKey(prvKey)
	pubkey := crypto.FromECDSAPub(&prvKey.PublicKey)
	pubkeyhex := hexutil.Encode(pubkey)
	keyhex := hexutil.Encode(bzzkeybytes)

	c.privateKey = prvKey
	c.PublicKey = pubkeyhex
	c.BzzKey = keyhex
}

func (c *Config) createAndSetPath(datadirPath string, prvKey *ecdsa.PrivateKey) error {
	address := crypto.PubkeyToAddress(prvKey.PublicKey)
	bzzdirPath := filepath.Join(datadirPath, "bzz-"+common.Bytes2Hex(address.Bytes()))
	err := os.MkdirAll(bzzdirPath, os.ModePerm)
	if err != nil {
		return err
	}
	c.Path = bzzdirPath
	return nil
}
