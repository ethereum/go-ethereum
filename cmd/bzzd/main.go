// Copyright 2016 The go-ethereum Authors
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
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/swarm"
	bzzapi "github.com/ethereum/go-ethereum/swarm/api"
	"github.com/ethereum/go-ethereum/swarm/network"
	"gopkg.in/urfave/cli.v1"
)

const clientIdentifier = "bzzd"

var (
	gitCommit string // Git SHA1 commit hash of the release (set via linker flags)
	app       = utils.NewApp(gitCommit, "Ethereum Swarm server daemon")
)

var (
	ChequebookAddrFlag = cli.StringFlag{
		Name:  "chequebook",
		Usage: "chequebook contract address",
	}
	SwarmAccountFlag = cli.StringFlag{
		Name:  "bzzaccount",
		Usage: "Swarm account key file",
	}
	SwarmPortFlag = cli.StringFlag{
		Name:  "bzzport",
		Usage: "Swarm local http api port",
	}
	SwarmNetworkIdFlag = cli.IntFlag{
		Name:  "bzznetworkid",
		Usage: "Network identifier (integer, default 322=swarm testnet)",
		Value: network.NetworkId,
	}
	SwarmConfigPathFlag = cli.StringFlag{
		Name:  "bzzconfig",
		Usage: "Swarm config file path (datadir/bzz)",
	}
	SwarmSwapEnabled = cli.BoolFlag{
		Name:  "swap",
		Usage: "Swarm SWAP enabled (default false)",
	}
	SwarmSyncEnabled = cli.BoolTFlag{
		Name:  "sync",
		Usage: "Swarm Syncing enabled (default true)",
	}
	EthAPI = cli.StringFlag{
		Name:  "ethapi",
		Usage: "URL of the Ethereum API provider",
		Value: node.DefaultIPCEndpoint("geth"),
	}
)

var defaultBootnodes = []string{}

func init() {
	// Override flag defaults so bzzd can run alongside geth.
	utils.ListenPortFlag.Value = 30399
	utils.IPCPathFlag.Value = utils.DirectoryString{Value: "bzzd.ipc"}
	utils.IPCApiFlag.Value = "admin, bzz, chequebook, debug, rpc, web3"

	// Set up the cli app.
	app.Commands = nil
	app.Action = bzzd
	app.Flags = []cli.Flag{
		utils.IdentityFlag,
		utils.DataDirFlag,
		utils.BootnodesFlag,
		utils.KeyStoreDirFlag,
		utils.ListenPortFlag,
		utils.NoDiscoverFlag,
		utils.DiscoveryV5Flag,
		utils.NetrestrictFlag,
		utils.NodeKeyFileFlag,
		utils.NodeKeyHexFlag,
		utils.MaxPeersFlag,
		utils.NATFlag,
		utils.IPCDisabledFlag,
		utils.IPCApiFlag,
		utils.IPCPathFlag,
		// bzzd-specific flags
		EthAPI,
		SwarmConfigPathFlag,
		SwarmSwapEnabled,
		SwarmSyncEnabled,
		SwarmPortFlag,
		SwarmAccountFlag,
		SwarmNetworkIdFlag,
		ChequebookAddrFlag,
	}
	app.Flags = append(app.Flags, debug.Flags...)
	app.Before = func(ctx *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		return debug.Setup(ctx)
	}
	app.After = func(ctx *cli.Context) error {
		debug.Exit()
		return nil
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func bzzd(ctx *cli.Context) error {
	stack := utils.MakeNode(ctx, clientIdentifier, gitCommit)
	registerBzzService(ctx, stack)
	utils.StartNode(stack)

	// Add bootnodes as initial peers.
	if ctx.GlobalIsSet(utils.BootnodesFlag.Name) {
		bootnodes := strings.Split(ctx.GlobalString(utils.BootnodesFlag.Name), ",")
		injectBootnodes(stack.Server(), bootnodes)
	} else {
		injectBootnodes(stack.Server(), defaultBootnodes)
	}

	stack.Wait()
	return nil
}

func registerBzzService(ctx *cli.Context, stack *node.Node) {
	prvkey := getAccount(ctx, stack)

	chbookaddr := common.HexToAddress(ctx.GlobalString(ChequebookAddrFlag.Name))
	bzzdir := ctx.GlobalString(SwarmConfigPathFlag.Name)
	if bzzdir == "" {
		bzzdir = stack.InstanceDir()
	}
	bzzconfig, err := bzzapi.NewConfig(bzzdir, chbookaddr, prvkey, ctx.GlobalUint64(SwarmNetworkIdFlag.Name))
	if err != nil {
		utils.Fatalf("unable to configure swarm: %v", err)
	}
	bzzport := ctx.GlobalString(SwarmPortFlag.Name)
	if len(bzzport) > 0 {
		bzzconfig.Port = bzzport
	}
	swapEnabled := ctx.GlobalBool(SwarmSwapEnabled.Name)
	syncEnabled := ctx.GlobalBoolT(SwarmSyncEnabled.Name)

	ethapi := ctx.GlobalString(EthAPI.Name)

	boot := func(ctx *node.ServiceContext) (node.Service, error) {
		var client *ethclient.Client
		if ethapi == "" {
			err = fmt.Errorf("use ethapi flag to connect to a an eth client and talk to the blockchain")
		} else {
			client, err = ethclient.Dial(ethapi)
		}
		if err != nil {
			utils.Fatalf("Can't connect: %v", err)
		}
		return swarm.NewSwarm(ctx, client, bzzconfig, swapEnabled, syncEnabled)
	}
	if err := stack.Register(boot); err != nil {
		utils.Fatalf("Failed to register the Swarm service: %v", err)
	}
}

func getAccount(ctx *cli.Context, stack *node.Node) *ecdsa.PrivateKey {
	keyid := ctx.GlobalString(SwarmAccountFlag.Name)
	if keyid == "" {
		utils.Fatalf("Option %q is required", SwarmAccountFlag.Name)
	}
	// Try to load the arg as a hex key file.
	if key, err := crypto.LoadECDSA(keyid); err == nil {
		glog.V(logger.Info).Infof("swarm account key loaded: %#x", crypto.PubkeyToAddress(key.PublicKey))
		return key
	}
	// Otherwise try getting it from the keystore.
	return decryptStoreAccount(stack.AccountManager(), keyid)
}

func decryptStoreAccount(accman *accounts.Manager, account string) *ecdsa.PrivateKey {
	var a accounts.Account
	var err error
	if common.IsHexAddress(account) {
		a, err = accman.Find(accounts.Account{Address: common.HexToAddress(account)})
	} else if ix, ixerr := strconv.Atoi(account); ixerr == nil {
		a, err = accman.AccountByIndex(ix)
	} else {
		utils.Fatalf("Can't find swarm account key %s", account)
	}
	if err != nil {
		utils.Fatalf("Can't find swarm account key: %v", err)
	}
	keyjson, err := ioutil.ReadFile(a.File)
	if err != nil {
		utils.Fatalf("Can't load swarm account key: %v", err)
	}
	for i := 1; i <= 3; i++ {
		passphrase := promptPassphrase(fmt.Sprintf("Unlocking swarm account %s [%d/3]", a.Address.Hex(), i))
		key, err := accounts.DecryptKey(keyjson, passphrase)
		if err == nil {
			return key.PrivateKey
		}
	}
	utils.Fatalf("Can't decrypt swarm account key")
	return nil
}

func promptPassphrase(prompt string) string {
	if prompt != "" {
		fmt.Println(prompt)
	}
	password, err := console.Stdin.PromptPassword("Passphrase: ")
	if err != nil {
		utils.Fatalf("Failed to read passphrase: %v", err)
	}
	return password
}

func injectBootnodes(srv *p2p.Server, nodes []string) {
	for _, url := range nodes {
		n, err := discover.ParseNode(url)
		if err != nil {
			glog.Errorf("invalid bootnode %q", err)
			continue
		}
		srv.AddPeer(n)
	}
}
