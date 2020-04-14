// Copyright 2020 The go-ethereum Authors
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

// catalyst is the a prototype command-line client for the eth1-engine
package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth2"
	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	cli "gopkg.in/urfave/cli.v1"
)

const (
	clientIdentifier = "catalyst" // Client identifier to advertise over the network
)

var (
	// Git SHA1 commit hash of the release (set via linker flags)
	gitCommit = ""
	gitDate   = ""
	// The app that holds all commands and flags.
	app = utils.NewApp(gitCommit, gitDate, "the catalyst command line interface")
	// flags that configure the node
	nodeFlags = []cli.Flag{
		utils.BootnodesFlag,
		utils.BootnodesV4Flag,
		utils.BootnodesV5Flag,
		utils.DataDirFlag,
		utils.TxPoolLocalsFlag,
		utils.TxPoolNoLocalsFlag,
		utils.TxPoolJournalFlag,
		utils.TxPoolRejournalFlag,
		utils.TxPoolPriceLimitFlag,
		utils.TxPoolPriceBumpFlag,
		utils.TxPoolAccountSlotsFlag,
		utils.TxPoolGlobalSlotsFlag,
		utils.TxPoolAccountQueueFlag,
		utils.TxPoolGlobalQueueFlag,
		utils.TxPoolLifetimeFlag,
		utils.SyncModeFlag,
		utils.ExitWhenSyncedFlag,
		utils.GCModeFlag,
		utils.SnapshotFlag,
		utils.WhitelistFlag,
		utils.CacheFlag,
		utils.CacheDatabaseFlag,
		utils.CacheTrieFlag,
		utils.CacheGCFlag,
		utils.CacheSnapshotFlag,
		utils.CacheNoPrefetchFlag,
		utils.ListenPortFlag,
		utils.MaxPeersFlag,
		utils.MaxPendingPeersFlag,
		utils.MiningEnabledFlag,
		utils.NATFlag,
		utils.NoDiscoverFlag,
		utils.DiscoveryV5Flag,
		utils.NetrestrictFlag,
		utils.NodeKeyFileFlag,
		utils.NodeKeyHexFlag,
		utils.DNSDiscoveryFlag,
		utils.RopstenFlag,
		utils.RinkebyFlag,
		utils.GoerliFlag,
		utils.VMEnableDebugFlag,
		utils.NetworkIdFlag,
	}

	rpcFlags = []cli.Flag{
		utils.RPCEnabledFlag,
		utils.RPCListenAddrFlag,
		utils.RPCPortFlag,
		utils.RPCCORSDomainFlag,
		utils.RPCVirtualHostsFlag,
		utils.RPCApiFlag,
	}
)

func makeConfigNode(ctx *cli.Context) *node.Node {
	cfg := node.DefaultConfig
	cfg.Name = clientIdentifier
	//cfg.Version = params.VersionWithCommit(gitCommit, gitDate)
	//cfg.HTTPModules = append(cfg.HTTPModules, "eth")
	//cfg.WSModules = append(cfg.WSModules, "eth")
	//cfg.IPCPath = "geth.ipc"

	// Load config file.
	//if file := ctx.GlobalString(configFileFlag.Name); file != "" {
	//if err := loadConfig(file, &cfg); err != nil {
	//utils.Fatalf("%v", err)
	//}
	//}

	// Apply flags.
	utils.SetNodeConfig(ctx, &cfg)
	stack, err := node.New(&cfg)
	if err != nil {
		utils.Fatalf("Failed to create the protocol stack: %v", err)
	}

	return stack
}

// RegisterEthService adds an Ethereum client to the stack.
func RegisterEth2Service(stack *node.Node, cfg *eth.Config) {
	var err error
	err = stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return eth.New(ctx, cfg)
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to register the Ethereum service: %v", err))
	}
}

func makeValidatorNode(ctx *cli.Context) *node.Node {
	n := makeConfigNode(ctx)

	rpcSrv := rpc.NewServer()
	rpcSrv.RegisterName("eth2", new(eth2.Eth2RPC))

	RegisterEth2Service(n, &cfg.Eth)
	return n
}

func startNode(ctx *cli.Context, stack *node.Node) {
	utils.StartNode(stack)
	/*rpcClient*/ _, err := stack.Attach()
	if err != nil {
		utils.Fatalf("Failed to attach to self: %v", err)
	}
	//ethClient := ethclient.NewClient(rpcClient)
}

func catalyst(ctx *cli.Context) error {
	node := makeValidatorNode(ctx)
	defer node.Close()
	startNode(ctx, node)
	node.Wait()
	return nil
}

func init() {
	// Initialize the CLI app and start Geth
	app.Action = catalyst
	app.HideVersion = true // we have a command to print the version
	app.Copyright = "Copyright 2013-2020 The go-ethereum Authors"
	app.Commands = []cli.Command{}
	sort.Sort(cli.CommandsByName(app.Commands))

	app.Flags = append(app.Flags, nodeFlags...)
	app.Flags = append(app.Flags, rpcFlags...)

	app.Before = func(ctx *cli.Context) error {
		return debug.Setup(ctx)
	}
	app.After = func(ctx *cli.Context) error {
		debug.Exit()
		//console.Stdin.Close() // Resets terminal mode.
		return nil
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
