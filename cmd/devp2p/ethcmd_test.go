// Copyright 2022 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/docker/docker/pkg/reexec"
	"github.com/ethereum/go-ethereum/cmd/devp2p/internal/ethtest"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/internal/cmdtest"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
)

var (
	genesisPath   = "./internal/ethtest/testdata/genesis.json"
	halfchainFile = "./internal/ethtest/testdata/halfchain.rlp"
)

type testEth struct {
	*cmdtest.TestCmd
}

func TestMain(m *testing.M) {
	// Run the app if we've been exec'd as "ethkey-test" in runEthkey.
	reexec.Register("devp2p-test", func() {
		if err := app.Run(os.Args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	})
	// check if we have been reexec'd
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}

func TestNewBlockAnnouncement(t *testing.T) {
	geth, err := runGeth()
	if err != nil {
		t.Fatalf("could not run geth: %v", err)
	}
	defer geth.Close()

	tt := new(testEth)
	tt.TestCmd = cmdtest.NewTestCmd(t, tt)

	// NewBlock msg with block 1000 from fullchain.rlp.
	msg := "f90206f901fef901f9a00e70e01064023f70f047dbf0b97e7109e4aa5df3d643d45f8e91e47d3d67a424a01dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347940000000000000000000000000000000000000000a04327919f498aefdfe87f9c5a83cfd4a0bb444973d4a99d44fa9f4728497c3608a056e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421a056e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421b9010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000830200008203e884800000008083e51c5880a02807078eecff82eef7ad486076c4989e331a8a4c54592a690f083ef7b24820ec886cad2133e8e98457c0c08407d20000"
	hash := common.HexToHash("0x8c795a2497f393359fd66bfd5696442d12a81ccb1110dffd636604bfc9af4df3")

	rpc, err := geth.Attach()
	if err != nil {
		tt.Fatalf("unable to attach client: %v", err)
	}
	ctx := context.Background()
	client := ethclient.NewClient(rpc)

	// Before the announcement, the propogated block should not be retrievable.
	_, err = client.BlockByHash(ctx, hash)
	if err == nil || (err != nil && err.Error() != "not found") {
		tt.Fatalf("should fail to retrieve block before announcement: %v", err)
	}

	// Run devp2p eth new-block against node.
	args := []string{"--verbosity=5", "eth", fmt.Sprintf("--genesis=%s", genesisPath), fmt.Sprintf("--chain=%s", halfchainFile), "new-block", geth.Server().Self().String(), msg}
	tt.Run("devp2p-test", args...)
	tt.WaitExit()

	// Give geth a moment to ingest the new block.
	time.Sleep(time.Second)

	// After the announcement, the block should be readily available.
	got, err := client.BlockByHash(ctx, hash)
	if err != nil {
		tt.Fatalf("unable to get announced block: %v", err)
	}
	if hash != got.Hash() {
		tt.Fatalf("block mismatch (got: %s, want: %s)", hash, got.Hash())
	}
}

// runGeth creates and starts a geth node
func runGeth() (*node.Node, error) {
	stack, err := node.New(&node.Config{
		P2P: p2p.Config{
			ListenAddr:  "127.0.0.1:0",
			NoDiscovery: true,
			MaxPeers:    1,
			NoDial:      true,
		},
	})
	if err != nil {
		return nil, err
	}

	err = setupGeth(stack)
	if err != nil {
		stack.Close()
		return nil, err
	}
	if err = stack.Start(); err != nil {
		stack.Close()
		return nil, err
	}
	return stack, nil
}

func setupGeth(stack *node.Node) error {
	chain, err := ethtest.LoadChain(halfchainFile, genesisPath)
	if err != nil {
		return err
	}

	backend, err := eth.New(stack, &ethconfig.Config{
		Genesis:                 chain.Genesis(),
		NetworkId:               chain.Config().ChainID.Uint64(), // 19763
		DatabaseCache:           10,
		TrieCleanCache:          10,
		TrieCleanCacheJournal:   "",
		TrieCleanCacheRejournal: 60 * time.Minute,
		TrieDirtyCache:          16,
		TrieTimeout:             60 * time.Minute,
		SnapshotCache:           10,
	})
	if err != nil {
		return err
	}

	_, err = backend.BlockChain().InsertChain(chain.Blocks()[1:])
	return err
}
