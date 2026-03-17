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

// This file contains a miner stress test based on the Engine API flow.
package main

import (
	"crypto/ecdsa"
	"math/big"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/fdlimit"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
)

var refundContract = common.HexToAddress("0x1000000000000000000000000000000000000001")

func main() {
	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelInfo, true)))
	fdlimit.Raise(2048)

	// Generate a batch of accounts to seal and fund with
	faucets := make([]*ecdsa.PrivateKey, 128)
	for i := 0; i < len(faucets); i++ {
		faucets[i], _ = crypto.GenerateKey()
	}
	// Create a post-merge network where blocks are built/inserted through
	// engine API calls driven by a simulated beacon client.
	genesis := makeGenesis(faucets)

	// Handle interrupts.
	interruptCh := make(chan os.Signal, 5)
	signal.Notify(interruptCh, os.Interrupt)

	// Start one node that accepts transactions and builds/inserts blocks via
	// Engine API (through the simulated beacon driver).
	stack, backend, beacon, err := makeNode(genesis)
	if err != nil {
		panic(err)
	}
	defer stack.Close()
	defer beacon.Stop()

	// Start injecting transactions from the faucet like crazy
	var (
		sent      uint64
		nonces    = make([]uint64, len(faucets))
		signer    = types.LatestSigner(genesis.Config)
		refundSet = true // slot 0 starts as non-zero in genesis
	)
	for {
		// Stop when interrupted.
		select {
		case <-interruptCh:
			return
		default:
		}

		var (
			tx  *types.Transaction
			err error
		)
		// Every third tx targets a contract path that alternates set/clear.
		// Clearing a previously non-zero slot triggers gas refund.
		if sent%3 == 0 {
			var data []byte
			if refundSet {
				data = nil // empty calldata => clear slot to zero (refund path)
			} else {
				data = []byte{0x01} // non-empty calldata => set slot to one
			}
			tx, err = types.SignTx(types.NewTransaction(nonces[0], refundContract, new(big.Int), 50000, big.NewInt(100000000000), data), signer, faucets[0])
			if err != nil {
				panic(err)
			}
			nonces[0]++
			refundSet = !refundSet
		} else {
			index := 1 + rand.Intn(len(faucets)-1)
			tx, err = types.SignTx(types.NewTransaction(nonces[index], crypto.PubkeyToAddress(faucets[index].PublicKey), new(big.Int), 21000, big.NewInt(100000000000), nil), signer, faucets[index])
			if err != nil {
				panic(err)
			}
			nonces[index]++
		}
		errs := backend.TxPool().Add([]*types.Transaction{tx}, true)
		for _, err := range errs {
			if err != nil {
				panic(err)
			}
		}
		sent++

		// Create and import blocks through the engine API path.
		if sent%256 == 0 {
			beacon.Commit()
		}

		// Wait if we're too saturated
		if pend, _ := backend.TxPool().Stats(); pend > 4096 {
			beacon.Commit()
			time.Sleep(50 * time.Millisecond)
		}
	}
}

// makeGenesis creates a post-merge genesis block.
func makeGenesis(faucets []*ecdsa.PrivateKey) *core.Genesis {
	config := *params.AllDevChainProtocolChanges
	config.ChainID = big.NewInt(18)

	blockZero := uint64(0)
	config.AmsterdamTime = &blockZero
	config.BlobScheduleConfig.Amsterdam = &params.BlobConfig{
		Target:         14,
		Max:            21,
		UpdateFraction: 13739630,
	}

	genesis := &core.Genesis{
		Config:   &config,
		GasLimit: 25000000,
		Alloc:    types.GenesisAlloc{},
	}
	for _, faucet := range faucets {
		genesis.Alloc[crypto.PubkeyToAddress(faucet.PublicKey)] = types.Account{
			Balance: new(big.Int).Exp(big.NewInt(2), big.NewInt(128), nil),
		}
	}
	// Runtime code:
	// - empty calldata: SSTORE(0,0)
	// - non-empty calldata: SSTORE(0,1)
	// Slot 0 is initialized to 1 so the first clear includes gas refund.
	genesis.Alloc[refundContract] = types.Account{
		Code: common.FromHex("0x3615600b576001600055005b600060005500"),
		Storage: map[common.Hash]common.Hash{
			common.Hash{}: common.BigToHash(big.NewInt(1)),
		},
	}
	return genesis
}

func makeNode(genesis *core.Genesis) (*node.Node, *eth.Ethereum, *catalyst.SimulatedBeacon, error) {
	// Define the basic configurations for the Ethereum node
	datadir, _ := os.MkdirTemp("", "")

	config := &node.Config{
		Name:    "geth",
		DataDir: datadir,
	}
	// Start the node and configure a full Ethereum node on it
	stack, err := node.New(config)
	if err != nil {
		return nil, nil, nil, err
	}
	// Create and register the backend
	ethBackend, err := eth.New(stack, &ethconfig.Config{
		Genesis:            genesis,
		NetworkId:          genesis.Config.ChainID.Uint64(),
		SyncMode:           downloader.FullSync,
		DatabaseCache:      256,
		DatabaseHandles:    256,
		TxPool:             legacypool.DefaultConfig,
		GPO:                ethconfig.Defaults.GPO,
		Miner:              ethconfig.Defaults.Miner,
		SlowBlockThreshold: time.Second,
	})
	if err != nil {
		return nil, nil, nil, err
	}

	if err := stack.Start(); err != nil {
		return nil, nil, nil, err
	}
	driver, err := catalyst.NewSimulatedBeacon(0, common.Address{}, ethBackend)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := driver.Start(); err != nil {
		return nil, nil, nil, err
	}
	return stack, ethBackend, driver, nil
}
