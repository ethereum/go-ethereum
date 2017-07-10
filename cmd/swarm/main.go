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
	"context"
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/contracts/ens"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm"
	bzzapi "github.com/ethereum/go-ethereum/swarm/api"
	"gopkg.in/urfave/cli.v1"
)

const clientIdentifier = "swarm"

var (
	gitCommit        string // Git SHA1 commit hash of the release (set via linker flags)
	testbetBootNodes = []string{
		"enode://ec8ae764f7cb0417bdfb009b9d0f18ab3818a3a4e8e7c67dd5f18971a93510a2e6f43cd0b69a27e439a9629457ea804104f37c85e41eed057d3faabbf7744cdf@13.74.157.139:30429",
		"enode://c2e1fceb3bf3be19dff71eec6cccf19f2dbf7567ee017d130240c670be8594bc9163353ca55dd8df7a4f161dd94b36d0615c17418b5a3cdcbb4e9d99dfa4de37@13.74.157.139:30430",
		"enode://fe29b82319b734ce1ec68b84657d57145fee237387e63273989d354486731e59f78858e452ef800a020559da22dcca759536e6aa5517c53930d29ce0b1029286@13.74.157.139:30431",
		"enode://1d7187e7bde45cf0bee489ce9852dd6d1a0d9aa67a33a6b8e6db8a4fbc6fcfa6f0f1a5419343671521b863b187d1c73bad3603bae66421d157ffef357669ddb8@13.74.157.139:30432",
		"enode://0e4cba800f7b1ee73673afa6a4acead4018f0149d2e3216be3f133318fd165b324cd71b81fbe1e80deac8dbf56e57a49db7be67f8b9bc81bd2b7ee496434fb5d@13.74.157.139:30433",
	}
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
	SwarmListenAddrFlag = cli.StringFlag{
		Name:  "httpaddr",
		Usage: "Swarm HTTP API listening interface",
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
	SwarmSwapAPIFlag = cli.StringFlag{
		Name:  "swap-api",
		Usage: "URL of the Ethereum API provider to use to settle SWAP payments",
	}
	SwarmSyncEnabledFlag = cli.BoolTFlag{
		Name:  "sync",
		Usage: "Swarm Syncing enabled (default true)",
	}
	EnsAPIFlag = cli.StringFlag{
		Name:  "ens-api",
		Usage: "URL of the Ethereum API provider to use for ENS record lookups",
		Value: node.DefaultIPCEndpoint("geth"),
	}
	EnsAddrFlag = cli.StringFlag{
		Name:  "ens-addr",
		Usage: "ENS contract address (default is detected as testnet or mainnet using --ens-api)",
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
	SwarmUpFromStdinFlag = cli.BoolFlag{
		Name:  "stdin",
		Usage: "reads data to be uploaded from stdin",
	}
	SwarmUploadMimeType = cli.StringFlag{
		Name:  "mime",
		Usage: "force mime type",
	}
	CorsStringFlag = cli.StringFlag{
		Name:  "corsdomain",
		Usage: "Domain on which to send Access-Control-Allow-Origin header (multiple domains can be supplied separated by a ',')",
	}

	// the following flags are deprecated and should be removed in the future
	DeprecatedEthAPIFlag = cli.StringFlag{
		Name:  "ethapi",
		Usage: "DEPRECATED: please use --ens-api and --swap-api",
	}
)

var defaultNodeConfig = node.DefaultConfig

// This init function sets defaults so cmd/swarm can run alongside geth.
func init() {
	defaultNodeConfig.Name = clientIdentifier
	defaultNodeConfig.Version = params.VersionWithCommit(gitCommit)
	defaultNodeConfig.P2P.ListenAddr = ":30399"
	defaultNodeConfig.IPCPath = "bzzd.ipc"
	// Set flag defaults for --help display.
	utils.ListenPortFlag.Value = 30399
}

var app = utils.NewApp(gitCommit, "Ethereum Swarm")

// This init function creates the cli.App.
func init() {
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
			Action:    list,
			Name:      "ls",
			Usage:     "list files and directories contained in a manifest",
			ArgsUsage: " <manifest> [<prefix>]",
			Description: `
Lists files and directories contained in a manifest.
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
		utils.IPCPathFlag,
		utils.PasswordFileFlag,
		// bzzd-specific flags
		CorsStringFlag,
		EnsAPIFlag,
		EnsAddrFlag,
		SwarmConfigPathFlag,
		SwarmSwapEnabledFlag,
		SwarmSwapAPIFlag,
		SwarmSyncEnabledFlag,
		SwarmListenAddrFlag,
		SwarmPortFlag,
		SwarmAccountFlag,
		SwarmNetworkIdFlag,
		ChequebookAddrFlag,
		// upload flags
		SwarmApiFlag,
		SwarmRecursiveUploadFlag,
		SwarmWantManifestFlag,
		SwarmUploadDefaultPath,
		SwarmUpFromStdinFlag,
		SwarmUploadMimeType,
		//deprecated flags
		DeprecatedEthAPIFlag,
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
	fmt.Println("Version:", params.Version)
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
	// exit if the deprecated --ethapi flag is set
	if ctx.GlobalString(DeprecatedEthAPIFlag.Name) != "" {
		utils.Fatalf("--ethapi is no longer a valid command line flag, please use --ens-api and/or --swap-api.")
	}

	cfg := defaultNodeConfig
	utils.SetNodeConfig(ctx, &cfg)
	stack, err := node.New(&cfg)
	if err != nil {
		utils.Fatalf("can't create node: %v", err)
	}

	registerBzzService(ctx, stack)
	utils.StartNode(stack)

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Got sigterm, shutting swarm down...")
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

// detectEnsAddr determines the ENS contract address by getting both the
// version and genesis hash using the client and matching them to either
// mainnet or testnet addresses
func detectEnsAddr(client *rpc.Client) (common.Address, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var version string
	if err := client.CallContext(ctx, &version, "net_version"); err != nil {
		return common.Address{}, err
	}

	block, err := ethclient.NewClient(client).BlockByNumber(ctx, big.NewInt(0))
	if err != nil {
		return common.Address{}, err
	}

	switch {

	case version == "1" && block.Hash() == params.MainnetGenesisHash:
		log.Info("using Mainnet ENS contract address", "addr", ens.MainNetAddress)
		return ens.MainNetAddress, nil

	case version == "3" && block.Hash() == params.TestnetGenesisHash:
		log.Info("using Testnet ENS contract address", "addr", ens.TestNetAddress)
		return ens.TestNetAddress, nil

	default:
		return common.Address{}, fmt.Errorf("unknown version and genesis hash: %s %s", version, block.Hash())
	}
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
	if bzzaddr := ctx.GlobalString(SwarmListenAddrFlag.Name); bzzaddr != "" {
		bzzconfig.ListenAddr = bzzaddr
	}
	swapEnabled := ctx.GlobalBool(SwarmSwapEnabledFlag.Name)
	syncEnabled := ctx.GlobalBoolT(SwarmSyncEnabledFlag.Name)

	swapapi := ctx.GlobalString(SwarmSwapAPIFlag.Name)
	if swapEnabled && swapapi == "" {
		utils.Fatalf("SWAP is enabled but --swap-api is not set")
	}

	ensapi := ctx.GlobalString(EnsAPIFlag.Name)
	ensAddr := ctx.GlobalString(EnsAddrFlag.Name)

	cors := ctx.GlobalString(CorsStringFlag.Name)

	boot := func(ctx *node.ServiceContext) (node.Service, error) {
		var swapClient *ethclient.Client
		if swapapi != "" {
			log.Info("connecting to SWAP API", "url", swapapi)
			swapClient, err = ethclient.Dial(swapapi)
			if err != nil {
				return nil, fmt.Errorf("error connecting to SWAP API %s: %s", swapapi, err)
			}
		}

		var ensClient *ethclient.Client
		if ensapi != "" {
			log.Info("connecting to ENS API", "url", ensapi)
			client, err := rpc.Dial(ensapi)
			if err != nil {
				return nil, fmt.Errorf("error connecting to ENS API %s: %s", ensapi, err)
			}
			ensClient = ethclient.NewClient(client)

			if ensAddr != "" {
				bzzconfig.EnsRoot = common.HexToAddress(ensAddr)
			} else {
				ensAddr, err := detectEnsAddr(client)
				if err == nil {
					bzzconfig.EnsRoot = ensAddr
				} else {
					log.Warn(fmt.Sprintf("could not determine ENS contract address, using default %s", bzzconfig.EnsRoot), "err", err)
				}
			}
		}

		return swarm.NewSwarm(ctx, swapClient, ensClient, bzzconfig, swapEnabled, syncEnabled, cors)
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
		log.Info("Swarm account key loaded", "address", crypto.PubkeyToAddress(key.PublicKey))
		return key
	}
	// Otherwise try getting it from the keystore.
	am := stack.AccountManager()
	ks := am.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)

	return decryptStoreAccount(ks, keyid, utils.MakePasswordList(ctx))
}

func decryptStoreAccount(ks *keystore.KeyStore, account string, passwords []string) *ecdsa.PrivateKey {
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
	for i := 0; i < 3; i++ {
		password := getPassPhrase(fmt.Sprintf("Unlocking swarm account %s [%d/3]", a.Address.Hex(), i+1), i, passwords)
		key, err := keystore.DecryptKey(keyjson, password)
		if err == nil {
			return key.PrivateKey
		}
	}
	utils.Fatalf("Can't decrypt swarm account key")
	return nil
}

// getPassPhrase retrieves the password associated with bzz account, either by fetching
// from a list of pre-loaded passwords, or by requesting it interactively from user.
func getPassPhrase(prompt string, i int, passwords []string) string {
	// non-interactive
	if len(passwords) > 0 {
		if i < len(passwords) {
			return passwords[i]
		}
		return passwords[len(passwords)-1]
	}

	// fallback to interactive mode
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
			log.Error("Invalid swarm bootnode", "err", err)
			continue
		}
		srv.AddPeer(n)
	}
}
