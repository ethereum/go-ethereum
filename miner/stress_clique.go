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

// +build none

// This file contains a miner stress test based on the Clique consensus engine.
package main

import (
	"bytes"
	"crypto/ecdsa"
	"io/ioutil"
	"math/big"
	"math/rand"
	"os"
	"time"

	"github.com/maticnetwork/bor/accounts/keystore"
	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/common/fdlimit"
	"github.com/maticnetwork/bor/core"
	"github.com/maticnetwork/bor/core/types"
	"github.com/maticnetwork/bor/crypto"
	"github.com/maticnetwork/bor/eth"
	"github.com/maticnetwork/bor/eth/downloader"
	"github.com/maticnetwork/bor/log"
	"github.com/maticnetwork/bor/node"
	"github.com/maticnetwork/bor/p2p"
	"github.com/maticnetwork/bor/p2p/enode"
	"github.com/maticnetwork/bor/params"
)

func main() {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	fdlimit.Raise(2048)

	// Generate a batch of accounts to seal and fund with
	faucets := make([]*ecdsa.PrivateKey, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i], _ = crypto.GenerateKey()
	}
	sealers := make([]*ecdsa.PrivateKey, 4)
	for i := 0; i < len(sealers); i++ {
		sealers[i], _ = crypto.GenerateKey()
	}
	// Create a Clique network based off of the Rinkeby config
	genesis := makeGenesis(faucets, sealers)

	var (
		nodes  []*node.Node
		enodes []*enode.Node
	)
	for _, sealer := range sealers {
		// Start the node and wait until it's up
		node, err := makeSealer(genesis)
		if err != nil {
			panic(err)
		}
		defer node.Close()

		for node.Server().NodeInfo().Ports.Listener == 0 {
			time.Sleep(250 * time.Millisecond)
		}
		// Connect the node to al the previous ones
		for _, n := range enodes {
			node.Server().AddPeer(n)
		}
		// Start tracking the node and it's enode
		nodes = append(nodes, node)
		enodes = append(enodes, node.Server().Self())

		// Inject the signer key and start sealing with it
		store := node.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
		signer, err := store.ImportECDSA(sealer, "")
		if err != nil {
			panic(err)
		}
		if err := store.Unlock(signer, ""); err != nil {
			panic(err)
		}
	}
	// Iterate over all the nodes and start signing with them
	time.Sleep(3 * time.Second)

	for _, node := range nodes {
		var ethereum *eth.Ethereum
		if err := node.Service(&ethereum); err != nil {
			panic(err)
		}
		if err := ethereum.StartMining(1); err != nil {
			panic(err)
		}
	}
	time.Sleep(3 * time.Second)

	// Start injecting transactions from the faucet like crazy
	nonces := make([]uint64, len(faucets))
	for {
		index := rand.Intn(len(faucets))

		// Fetch the accessor for the relevant signer
		var ethereum *eth.Ethereum
		if err := nodes[index%len(nodes)].Service(&ethereum); err != nil {
			panic(err)
		}
		// Create a self transaction and inject into the pool
		tx, err := types.SignTx(types.NewTransaction(nonces[index], crypto.PubkeyToAddress(faucets[index].PublicKey), new(big.Int), 21000, big.NewInt(100000000000), nil), types.HomesteadSigner{}, faucets[index])
		if err != nil {
			panic(err)
		}
		if err := ethereum.TxPool().AddLocal(tx); err != nil {
			panic(err)
		}
		nonces[index]++

		// Wait if we're too saturated
		if pend, _ := ethereum.TxPool().Stats(); pend > 2048 {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// makeGenesis creates a custom Clique genesis block based on some pre-defined
// signer and faucet accounts.
func makeGenesis(faucets []*ecdsa.PrivateKey, sealers []*ecdsa.PrivateKey) *core.Genesis {
	// Create a Clique network based off of the Rinkeby config
	genesis := core.DefaultRinkebyGenesisBlock()
	genesis.GasLimit = 25000000

	genesis.Config.ChainID = big.NewInt(18)
	genesis.Config.Clique.Period = 1
	genesis.Config.EIP150Hash = common.Hash{}

	genesis.Alloc = core.GenesisAlloc{}
	for _, faucet := range faucets {
		genesis.Alloc[crypto.PubkeyToAddress(faucet.PublicKey)] = core.GenesisAccount{
			Balance: new(big.Int).Exp(big.NewInt(2), big.NewInt(128), nil),
		}
	}
	// Sort the signers and embed into the extra-data section
	signers := make([]common.Address, len(sealers))
	for i, sealer := range sealers {
		signers[i] = crypto.PubkeyToAddress(sealer.PublicKey)
	}
	for i := 0; i < len(signers); i++ {
		for j := i + 1; j < len(signers); j++ {
			if bytes.Compare(signers[i][:], signers[j][:]) > 0 {
				signers[i], signers[j] = signers[j], signers[i]
			}
		}
	}
	genesis.ExtraData = make([]byte, 32+len(signers)*common.AddressLength+65)
	for i, signer := range signers {
		copy(genesis.ExtraData[32+i*common.AddressLength:], signer[:])
	}
	// Return the genesis block for initialization
	return genesis
}

func makeSealer(genesis *core.Genesis) (*node.Node, error) {
	// Define the basic configurations for the Ethereum node
	datadir, _ := ioutil.TempDir("", "")

	config := &node.Config{
		Name:    "bor",
		Version: params.Version,
		DataDir: datadir,
		P2P: p2p.Config{
			ListenAddr:  "0.0.0.0:0",
			NoDiscovery: true,
			MaxPeers:    25,
		},
		NoUSB: true,
	}
	// Start the node and configure a full Ethereum node on it
	stack, err := node.New(config)
	if err != nil {
		return nil, err
	}
	if err := stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return eth.New(ctx, &eth.Config{
			Genesis:         genesis,
			NetworkId:       genesis.Config.ChainID.Uint64(),
			SyncMode:        downloader.FullSync,
			DatabaseCache:   256,
			DatabaseHandles: 256,
			TxPool:          core.DefaultTxPoolConfig,
			GPO:             eth.DefaultConfig.GPO,
			Miner: Config{
				GasFloor: genesis.GasLimit * 9 / 10,
				GasCeil:  genesis.GasLimit * 11 / 10,
				GasPrice: big.NewInt(1),
				Recommit: time.Second,
			},
		})
	}); err != nil {
		return nil, err
	}
	// Start the node and return if successful
	return stack, stack.Start()
}
