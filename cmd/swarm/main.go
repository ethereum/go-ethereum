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
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/ubiq/go-ubiq/accounts"
	"github.com/ubiq/go-ubiq/accounts/keystore"
	"github.com/ubiq/go-ubiq/cmd/utils"
	"github.com/ubiq/go-ubiq/common"
	"github.com/ubiq/go-ubiq/console"
	"github.com/ubiq/go-ubiq/crypto"
	"github.com/ubiq/go-ubiq/ethclient"
	"github.com/ubiq/go-ubiq/internal/debug"
	"github.com/ubiq/go-ubiq/logger"
	"github.com/ubiq/go-ubiq/logger/glog"
	"github.com/ubiq/go-ubiq/node"
	"github.com/ubiq/go-ubiq/p2p"
	"github.com/ubiq/go-ubiq/p2p/discover"
	"github.com/ubiq/go-ubiq/swarm"
	bzzapi "github.com/ubiq/go-ubiq/swarm/api"
	"gopkg.in/urfave/cli.v1"
)

const (
	clientIdentifier = "swarm"
	versionString    = "0.2"
)

var (
	gitCommit        string // Git SHA1 commit hash of the release (set via linker flags)
	app              = utils.NewApp(gitCommit, "Ubiq Swarm")
	testbetBootNodes = []string{}
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
		Usage: "Network identifier (integer, default 3=swarm testnet)",
	}
	SwarmConfigPathFlag = cli.StringFlag{
		Name:  "bzzconfig",
		Usage: "Swarm config file path (datadir/bzz)",
	}
	SwarmSwapEnabledFlag = cli.BoolFlag{
		Name:  "swap",
		Usage: "Swarm SWAP enabled (default false)",
	}
	SwarmSyncEnabledFlag = cli.BoolTFlag{
		Name:  "sync",
		Usage: "Swarm Syncing enabled (default true)",
	}
	EthAPIFlag = cli.StringFlag{
		Name:  "ethapi",
		Usage: "URL of the Ethereum API provider",
		Value: node.DefaultIPCEndpoint("gubiq"),
	}
	SwarmApiFlag = cli.StringFlag{
		Name:  "bzzapi",
		Usage: "Swarm HTTP endpoint",
		Value: "http://127.0.0.1:8500",
	}
	SwarmRecursiveUploadFlag = cli.BoolFlag{
		Name:  "recursive",
		Usage: "Upload directories recursively",
	}
	SwarmWantManifestFlag = cli.BoolTFlag{
		Name:  "manifest",
		Usage: "Automatic manifest upload",
	}
	SwarmUploadDefaultPath = cli.StringFlag{
		Name:  "defaultpath",
		Usage: "path to file served for empty url path (none)",
	}
	CorsStringFlag = cli.StringFlag{
		Name:  "corsdomain",
		Usage: "Domain on which to send Access-Control-Allow-Origin header (multiple domains can be supplied separated by a ',')",
	}
)

func init() {
	// Override flag defaults so bzzd can run alongside gubiq.
	utils.ListenPortFlag.Value = 30399
	utils.IPCPathFlag.Value = utils.DirectoryString{Value: "bzzd.ipc"}
	utils.IPCApiFlag.Value = "admin, bzz, chequebook, debug, rpc, web3"

	// Set up the cli app.
	app.Action = bzzd
	app.HideVersion = true // we have a command to print the version
	app.Copyright = "Copyright 2013-2016 The go-ethereum Authors"
	app.Commands = []cli.Command{
		{
			Action:    version,
			Name:      "version",
			Usage:     "Print version numbers",
			ArgsUsage: " ",
			Description: `
The output of this command is supposed to be machine-readable.
`,
		},
		{
			Action:    upload,
			Name:      "up",
			Usage:     "upload a file or directory to swarm using the HTTP API",
			ArgsUsage: " <file>",
			Description: `
"upload a file or directory to swarm using the HTTP API and prints the root hash",
`,
		},
		{
			Action:    hash,
			Name:      "hash",
			Usage:     "print the swarm hash of a file or directory",
			ArgsUsage: " <file>",
			Description: `
Prints the swarm hash of file or directory.
`,
		},
		{
			Name:      "manifest",
			Usage:     "update a MANIFEST",
			ArgsUsage: "manifest COMMAND",
			Description: `
Updates a MANIFEST by adding/removing/updating the hash of a path.
`,
			Subcommands: []cli.Command{
				{
					Action:    add,
					Name:      "add",
					Usage:     "add a new path to the manifest",
					ArgsUsage: "<MANIFEST> <path> <hash> [<content-type>]",
					Description: `
Adds a new path to the manifest
`,
				},
				{
					Action:    update,
					Name:      "update",
					Usage:     "update the hash for an already existing path in the manifest",
					ArgsUsage: "<MANIFEST> <path> <newhash> [<newcontent-type>]",
					Description: `
Update the hash for an already existing path in the manifest
`,
				},
				{
					Action:    remove,
					Name:      "remove",
					Usage:     "removes a path from the manifest",
					ArgsUsage: "<MANIFEST> <path>",
					Description: `
Removes a path from the manifest
`,
				},
			},
		},
		{
			Action:    cleandb,
			Name:      "cleandb",
			Usage:     "Cleans database of corrupted entries",
			ArgsUsage: " ",
			Description: `
Cleans database of corrupted entries.
`,
		},
	}

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
		CorsStringFlag,
		EthAPIFlag,
		SwarmConfigPathFlag,
		SwarmSwapEnabledFlag,
		SwarmSyncEnabledFlag,
		SwarmPortFlag,
		SwarmAccountFlag,
		SwarmNetworkIdFlag,
		ChequebookAddrFlag,
		// upload flags
		SwarmApiFlag,
		SwarmRecursiveUploadFlag,
		SwarmWantManifestFlag,
		SwarmUploadDefaultPath,
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

func version(ctx *cli.Context) error {
	fmt.Println(strings.Title(clientIdentifier))
	fmt.Println("Version:", versionString)
	if gitCommit != "" {
		fmt.Println("Git Commit:", gitCommit)
	}
	fmt.Println("Network Id:", ctx.GlobalInt(utils.NetworkIdFlag.Name))
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("OS:", runtime.GOOS)
	fmt.Printf("GOPATH=%s\n", os.Getenv("GOPATH"))
	fmt.Printf("GOROOT=%s\n", runtime.GOROOT())
	return nil
}

func bzzd(ctx *cli.Context) error {
	stack := utils.MakeNode(ctx, clientIdentifier, gitCommit)
	registerBzzService(ctx, stack)
	utils.StartNode(stack)
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		glog.V(logger.Info).Infoln("Got sigterm, shutting down...")
		stack.Stop()
	}()
	networkId := ctx.GlobalUint64(SwarmNetworkIdFlag.Name)
	// Add bootnodes as initial peers.
	if ctx.GlobalIsSet(utils.BootnodesFlag.Name) {
		bootnodes := strings.Split(ctx.GlobalString(utils.BootnodesFlag.Name), ",")
		injectBootnodes(stack.Server(), bootnodes)
	} else {
		if networkId == 3 {
			injectBootnodes(stack.Server(), testbetBootNodes)
		}
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
	swapEnabled := ctx.GlobalBool(SwarmSwapEnabledFlag.Name)
	syncEnabled := ctx.GlobalBoolT(SwarmSyncEnabledFlag.Name)

	ethapi := ctx.GlobalString(EthAPIFlag.Name)
	cors := ctx.GlobalString(CorsStringFlag.Name)

	boot := func(ctx *node.ServiceContext) (node.Service, error) {
		var client *ethclient.Client
		if len(ethapi) > 0 {
			client, err = ethclient.Dial(ethapi)
			if err != nil {
				utils.Fatalf("Can't connect: %v", err)
			}
		}
		return swarm.NewSwarm(ctx, client, bzzconfig, swapEnabled, syncEnabled, cors)
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
	am := stack.AccountManager()
	ks := am.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)

	return decryptStoreAccount(ks, keyid)
}

func decryptStoreAccount(ks *keystore.KeyStore, account string) *ecdsa.PrivateKey {
	var a accounts.Account
	var err error
	if common.IsHexAddress(account) {
		a, err = ks.Find(accounts.Account{Address: common.HexToAddress(account)})
	} else if ix, ixerr := strconv.Atoi(account); ixerr == nil && ix > 0 {
		if accounts := ks.Accounts(); len(accounts) > ix {
			a = accounts[ix]
		} else {
			err = fmt.Errorf("index %d higher than number of accounts %d", ix, len(accounts))
		}
	} else {
		utils.Fatalf("Can't find swarm account key %s", account)
	}
	if err != nil {
		utils.Fatalf("Can't find swarm account key: %v", err)
	}
	keyjson, err := ioutil.ReadFile(a.URL.Path)
	if err != nil {
		utils.Fatalf("Can't load swarm account key: %v", err)
	}
	for i := 1; i <= 3; i++ {
		passphrase := promptPassphrase(fmt.Sprintf("Unlocking swarm account %s [%d/3]", a.Address.Hex(), i))
		key, err := keystore.DecryptKey(keyjson, passphrase)
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
