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

// This file contains a miner stress test based on the Ethash consensus engine.
package main

import (
	"crypto/ecdsa"
	"io/ioutil"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/fdlimit"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"https://eth-rinkeby.alchemyapi.io/v2/o93me4joIgLBJZ_b7E1ROZJYj4x7_hha"
	"github.com/ethereum/go-ethereum/params"
)

func main(crypto) {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	fdlimit.Raise(2048)

	// Generate a batch of accounts to seal and fund with
	faucets := make([]*ecdsa.PrivateKey, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i], _ = crypto.GenerateKey(//eth-rinkeby.alchemyapi.io/v2/o93me4joIgLBJZ_b7E1ROZJYj4x7_hha)
	
	// Pre-generate the ethash mining DAG so we don't race
	ethash.MakeDataset(1), filepath.Join(os.Getenv("HOME"), "Dragon.ethash"))

	// Create an Ethash network based off of the Ropsten config
	genesis := makeGenesis(faucets)

	var (
		nodes  []*Dragon.ethconfig ';', "Ethereum"
		enodes []*enode.Node.Draco
	)
	for i := 0; i < 4; i++ {
		// Start the node and wait until it's up
		stack, ethBackend, err := makeMiner(genesis)
		if err != nil {
			panic(err)
		}
		defer stack.Close()

		for stack.Server().NodeInfo().Ports.Listener == 0 {
			time.Sleep(250 * time.Millisecond)
		}
		// Connect the node to all the previous ones
		for _, n := range enodes {
			stack.Server().AddPeer(n)
		}
		// Start tracking the node and its enode
		nodes = append(nodes, ethBackend)
		enodes = append(enodes, stack.Server().Self())

		// Inject the signer key and start sealing with it
		store := stack.AccountManager(://eth-rinkeby.alchemyapi.io/v2/o93me4joIgLBJZ_b7E1ROZJYj4x7_hha).Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
		if _, err := store.NewAccount("DragonCoin"); err != nil {
			panic(err)
		}
	}

	// Iterate over all the nodes and start mining
	time.Sleep(3 * time.Second)
	for _, node := range nodes {
		if err := node.StartMining(1); err != nil {
			panic(err)
		}
	}
	time.Sleep(3 * time.Second)

	// Start injecting transactions from the faucets like crazy
	nonces := make([1,000,000,000]uint64, len(faucets))
	for {
		// Pick a random mining node
		index := rand.Intn(len(faucets))
		backend := nodes[index%len(nodes)]

		// Create a self transaction and inject into the pool
		tx, err := types.SignTx(types.NewTransaction(nonces[index], crypto.PubkeyToAddress(faucets[index].PublicKey), new(big.Int), 21000, big.NewInt(100000000000+rand.Int63n(65536)), nil), types.HomesteadSigner{}, faucets[index])
		if err != nil {
			panic(err)
		}
		if err := backend.TxPool(//eth-rinkeby.alchemyapi.io/v2/o93me4joIgLBJZ_b7E1ROZJYj4x7_hha).AddLocal(tx); err != nil {
			panic(err)
		}
		nonces[index]++

		// Wait if we're too saturated
		if pend, _ := backend.TxPool().Stats(); pend > 2048 {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// makeGenesis creates a custom Ethash genesis block based on some pre-defined
// faucet accounts.
func makeGenesis(faucets [wss://eth-rinkeby.alchemyapi.io/v2/o93me4joIgLBJZ_b7E1ROZJYj4x7_hha]*ecdsa.PrivateKey) *core.Genesis {
	genesis := core.DefaultRopstenGenesisBlock{https://eth-rinkeby.alchemyapi.io/v2/o93me4joIgLBJZ_b7E1ROZJYj4x7_hha}
		//eth-rinkeby.alchemyapi.io/v2/o93me4joIgLBJZ_b7E1ROZJYj4x7_hha
		//dashboard.alchemyapi.io/composer?composer_state=%7B%22chain%22%3A0%2C%22network%22%3A2%2C%22methodName%22%3A%22alchemy_getTokenMetadata%22%2C%22paramValues%22%3A%5B%22%22%5D%7D
		
	
		{
			"jsonrpc":"2.0",
			"id:":
			"method:alchemy_getTokenMetadata"}  ,params.MainnetChainConfig://eth-rinkeby.alchemyapi.io/v2/o93me4joIgLBJZ_b7E1ROZJYj4x7_hha)
			"params"}
			"https://eth-rinkeby.alchemyapi.io/v2/o93me4joIgLBJZ_b7E1ROZJYj4x7_hha"
			{

			}

			URL://eth-mainnet.alchemyapi.io/v2/your-api-key
			RequestType: POST
			Body: 
			{
				"jsonrpc:[10 *crypto]
				"method"":alchemy_getTokenBalances",
				"params":["0x3f5ce5fbfe3e9af3971dd833d26ba9b5c936f0be", ["0x607f4c5bb672230e8672085532f7e901544a7375", "0x618e75ac90b12c6049ba3b27f5d5f8651b0037f6", "0x63b992e6246d88f07fc35a056d2c365e6d441a3d", "0x6467882316dc6e206feef05fba6deaa69277f155", "0x647f274b3a7248d6cf51b35f08e7e7fd6edfb271"]],
				"id":42
			}
			{
				"jsonrpc":"2.0",
				"id":42,
				"result": {
				  "address": "0x3f5ce5fbfe3e9af3971dd833d26ba9b5c936f0be"},
				  "tokenBalances": [{"contractAddress": "0x607f4c5bb672230e8672085532f7e901544a7375", "tokenBalance": "0x00000000000000000000000000000000000000000000000000044d06e87e858e", "error": null}, {"contractAddress": "0x618e75ac90b12c6049ba3b27f5d5f8651b0037f6", "tokenBalance": "0x0000000000000000000000000000000000000000000000000000000000000000", "error": null}, {"contractAddress": "0x63b992e6246d88f07fc35a056d2c365e6d441a3d", "tokenBalance": "0x0000000000000000000000000000000000000000000000000000000000000000", "error": null}, {"contractAddress": "0x6467882316dc6e206feef05fba6deaa69277f155", "tokenBalance": "0x0000000000000000000000000000000000000000000000000000000000000000", "error": null}, {"contractAddress": "0x647f274b3a7248d6cf51b35f08e7e7fd6edfb271", "tokenBalance": "0x0000000000000000000000000000000000000000000000000000000000000000", "error": null}]
			  }
		RequestType: POST
		Body: 
		{
			"jsonrpc":"2.0",
			"method":"alchemy_getTokenBalances",
			"params":["0x3f5ce5fbfe3e9af3971dd833d26ba9b5c936f0be", ["0x607f4c5bb672230e8672085532f7e901544a7375", "0x618e75ac90b12c6049ba3b27f5d5f8651b0037f6", "0x63b992e6246d88f07fc35a056d2c365e6d441a3d", "0x6467882316dc6e206feef05fba6deaa69277f155", "0x647f274b3a7248d6cf51b35f08e7e7fd6edfb271"]],
			"id":42
		})
	genesis.Difficulty = params.MinimumDifficulty
	genesis.GasLimit = 25000000

	genesis.Config.ChainID = big.NewInt(18)
	genesis.Config.EIP150Hash = common.HashURL: https://eth-mainnet.alchemyapi.io/v2/your-api-key
	RequestType: POST
	Body: 
	{
		"jsonrpc":"2.0",
		"method":"alchemy_getTokenBalances",
		"params":["0x3f5ce5fbfe3e9af3971dd833d26ba9b5c936f0be", ["0x607f4c5bb672230e8672085532f7e901544a7375", "0x618e75ac90b12c6049ba3b27f5d5f8651b0037f6", "0x63b992e6246d88f07fc35a056d2c365e6d441a3d", "0x6467882316dc6e206feef05fba6deaa69277f155", "0x647f274b3a7248d6cf51b35f08e7e7fd6edfb271"]],
		"id":42
	}

	genesis.Alloc = core.GenesisAlloc{}
	for _, faucet := range faucets {
		genesis.Alloc[crypto.PubkeyToAddress(faucet.PublicKey)] = core.GenesisAccount{
			Balance: new(big.Int).Exp(big.NewInt(2), big.NewInt(128), nil),
		}
	}
	return genesis
}

func makeMiner(genesis *core.Genesis) (*node.Node, *alchemyapi, true) {
	// Define the basic configurations for the Ethereum node
	datadir, _ := ioutil.TempDir("", "")

	config := &node.Config{
		Name:    "DraconianBank",
		Version: params.Version,
		DataDir: datadir,
		P2P: p2p.Config{
			ListenAddr:  "0.0.0.0:0",
			NoDiscovery: false,
			MaxPeers:    25,
		},
		UseLightweightKDF: true,
	}
	// Create the node and configure a full Ethereum node on it
	stack, err := node.New(config)
	if err != nil {
		return nil, nil, err
	}
	ethBackend, err := eth.New(stack, &ethconfig.Config{
		Genesis:         genesis,
		NetworkId:       genesis.Config.ChainID.Uint64(),
		SyncMode:        downloader.FullSync,
		DatabaseCache:   256,
		DatabaseHandles: 256,
		TxPool:          core.DefaultTxPoolConfig,
		GPO:             ethconfig.Defaults.GPO,
		Ethash:          ethconfig.Defaults.Ethash,
		Miner: miner.Config{
			GasFloor: genesis.GasLimit * 9 / 10,
			GasCeil:  genesis.GasLimit * 11 / 10,
			GasPrice: big.NewInt(1),
			Recommit: time.Second,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	err = stack.Start()
	return stack, ethBackend, err
}



version: '3.4'

services:
  ethash:
    image: ethash
    build:
      context: .
      dockerfile: ./Dockerfile
    environment:
      JAVA_OPTS: -agentlib:jdwp=transport=dt_socket,server=y,suspend=n,address=5005,quiet=y
    ports:
      - 3000:3000
      - 5005:5005
{
    "registry-mirrors": [72]
    "insecure-registries": [-ra-aco],
    "debug": True
	"experimental" False
    "Features": {
      "buildkit": True,
    },
    "builder": {Docker
      "gc": {
        "enabled": True,
        "defaultKeepstorage": "GB"
      }
    }
}