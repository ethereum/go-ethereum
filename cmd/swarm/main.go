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
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm"
	bzzapi "github.com/ethereum/go-ethereum/swarm/api"
	swarmmetrics "github.com/ethereum/go-ethereum/swarm/metrics"
	"github.com/ethereum/go-ethereum/swarm/storage/mock"
	mockrpc "github.com/ethereum/go-ethereum/swarm/storage/mock/rpc"
	"github.com/ethereum/go-ethereum/swarm/tracing"
	sv "github.com/ethereum/go-ethereum/swarm/version"

	cli "gopkg.in/urfave/cli.v1"
)

const clientIdentifier = "swarm"
const helpTemplate = `NAME:
{{.HelpName}} - {{.Usage}}

USAGE:
{{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}}{{if .VisibleFlags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}{{if .Category}}

CATEGORY:
{{.Category}}{{end}}{{if .Description}}

DESCRIPTION:
{{.Description}}{{end}}{{if .VisibleFlags}}

OPTIONS:
{{range .VisibleFlags}}{{.}}
{{end}}{{end}}
`

// Git SHA1 commit hash of the release (set via linker flags)
// this variable will be assigned if corresponding parameter is passed with install, but not with test
// e.g.: go install -ldflags "-X main.gitCommit=ed1312d01b19e04ef578946226e5d8069d5dfd5a" ./cmd/swarm
var gitCommit string

//declare a few constant error messages, useful for later error check comparisons in test
var (
	SWARM_ERR_NO_BZZACCOUNT   = "bzzaccount option is required but not set; check your config file, command line or environment variables"
	SWARM_ERR_SWAP_SET_NO_API = "SWAP is enabled but --swap-api is not set"
)

// this help command gets added to any subcommand that does not define it explicitly
var defaultSubcommandHelp = cli.Command{
	Action:             func(ctx *cli.Context) { cli.ShowCommandHelpAndExit(ctx, "", 1) },
	CustomHelpTemplate: helpTemplate,
	Name:               "help",
	Usage:              "shows this help",
	Hidden:             true,
}

var defaultNodeConfig = node.DefaultConfig

// This init function sets defaults so cmd/swarm can run alongside geth.
func init() {
	sv.GitCommit = gitCommit
	defaultNodeConfig.Name = clientIdentifier
	defaultNodeConfig.Version = sv.VersionWithCommit(gitCommit)
	defaultNodeConfig.P2P.ListenAddr = ":30399"
	defaultNodeConfig.IPCPath = "bzzd.ipc"
	// Set flag defaults for --help display.
	utils.ListenPortFlag.Value = 30399
}

var app = utils.NewApp("", "Ethereum Swarm")

// This init function creates the cli.App.
func init() {
	app.Action = bzzd
	app.Version = sv.ArchiveVersion(gitCommit)
	app.Copyright = "Copyright 2013-2016 The go-ethereum Authors"
	app.Commands = []cli.Command{
		{
			Action:             version,
			CustomHelpTemplate: helpTemplate,
			Name:               "version",
			Usage:              "Print version numbers",
			Description:        "The output of this command is supposed to be machine-readable",
		},
		{
			Action:             keys,
			CustomHelpTemplate: helpTemplate,
			Name:               "print-keys",
			Flags:              []cli.Flag{SwarmCompressedFlag},
			Usage:              "Print public key information",
			Description:        "The output of this command is supposed to be machine-readable",
		},
		// See upload.go
		upCommand,
		// See access.go
		accessCommand,
		// See feeds.go
		feedCommand,
		// See list.go
		listCommand,
		// See hash.go
		hashCommand,
		// See download.go
		downloadCommand,
		// See manifest.go
		manifestCommand,
		// See fs.go
		fsCommand,
		// See db.go
		dbCommand,
		// See config.go
		DumpConfigCommand,
		// hashesCommand
		hashesCommand,
	}

	// append a hidden help subcommand to all commands that have subcommands
	// if a help command was already defined above, that one will take precedence.
	addDefaultHelpSubcommands(app.Commands)

	sort.Sort(cli.CommandsByName(app.Commands))

	app.Flags = []cli.Flag{
		utils.IdentityFlag,
		utils.DataDirFlag,
		utils.BootnodesFlag,
		utils.KeyStoreDirFlag,
		utils.ListenPortFlag,
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
		SwarmTomlConfigPathFlag,
		SwarmSwapEnabledFlag,
		SwarmSwapAPIFlag,
		SwarmSyncDisabledFlag,
		SwarmSyncUpdateDelay,
		SwarmMaxStreamPeerServersFlag,
		SwarmLightNodeEnabled,
		SwarmDeliverySkipCheckFlag,
		SwarmListenAddrFlag,
		SwarmPortFlag,
		SwarmAccountFlag,
		SwarmNetworkIdFlag,
		ChequebookAddrFlag,
		// upload flags
		SwarmApiFlag,
		SwarmRecursiveFlag,
		SwarmWantManifestFlag,
		SwarmUploadDefaultPath,
		SwarmUpFromStdinFlag,
		SwarmUploadMimeType,
		// bootnode mode
		SwarmBootnodeModeFlag,
		// storage flags
		SwarmStorePath,
		SwarmStoreCapacity,
		SwarmStoreCacheCapacity,
		SwarmGlobalStoreAPIFlag,
	}
	rpcFlags := []cli.Flag{
		utils.WSEnabledFlag,
		utils.WSListenAddrFlag,
		utils.WSPortFlag,
		utils.WSApiFlag,
		utils.WSAllowedOriginsFlag,
	}
	app.Flags = append(app.Flags, rpcFlags...)
	app.Flags = append(app.Flags, debug.Flags...)
	app.Flags = append(app.Flags, swarmmetrics.Flags...)
	app.Flags = append(app.Flags, tracing.Flags...)
	app.Before = func(ctx *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		if err := debug.Setup(ctx, ""); err != nil {
			return err
		}
		swarmmetrics.Setup(ctx)
		tracing.Setup(ctx)
		return nil
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

func keys(ctx *cli.Context) error {
	privateKey := getPrivKey(ctx)
	pubkey := crypto.FromECDSAPub(&privateKey.PublicKey)
	pubkeyhex := hex.EncodeToString(pubkey)
	pubCompressed := hex.EncodeToString(crypto.CompressPubkey(&privateKey.PublicKey))
	bzzkey := crypto.Keccak256Hash(pubkey).Hex()

	if !ctx.Bool(SwarmCompressedFlag.Name) {
		fmt.Println(fmt.Sprintf("bzzkey=%s", bzzkey[2:]))
		fmt.Println(fmt.Sprintf("publicKey=%s", pubkeyhex))
	}
	fmt.Println(fmt.Sprintf("publicKeyCompressed=%s", pubCompressed))

	return nil
}

func version(ctx *cli.Context) error {
	fmt.Println(strings.Title(clientIdentifier))
	fmt.Println("Version:", sv.VersionWithMeta)
	if gitCommit != "" {
		fmt.Println("Git Commit:", gitCommit)
	}
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("OS:", runtime.GOOS)
	return nil
}

func bzzd(ctx *cli.Context) error {
	//build a valid bzzapi.Config from all available sources:
	//default config, file config, command line and env vars

	bzzconfig, err := buildConfig(ctx)
	if err != nil {
		utils.Fatalf("unable to configure swarm: %v", err)
	}

	cfg := defaultNodeConfig

	//pss operates on ws
	cfg.WSModules = append(cfg.WSModules, "pss")

	//geth only supports --datadir via command line
	//in order to be consistent within swarm, if we pass --datadir via environment variable
	//or via config file, we get the same directory for geth and swarm
	if _, err := os.Stat(bzzconfig.Path); err == nil {
		cfg.DataDir = bzzconfig.Path
	}

	//optionally set the bootnodes before configuring the node
	setSwarmBootstrapNodes(ctx, &cfg)
	//setup the ethereum node
	utils.SetNodeConfig(ctx, &cfg)

	//always disable discovery from p2p package - swarm discovery is done with the `hive` protocol
	cfg.P2P.NoDiscovery = true

	stack, err := node.New(&cfg)
	if err != nil {
		utils.Fatalf("can't create node: %v", err)
	}

	//a few steps need to be done after the config phase is completed,
	//due to overriding behavior
	initSwarmNode(bzzconfig, stack, ctx)
	//register BZZ as node.Service in the ethereum node
	registerBzzService(bzzconfig, stack)
	//start the node
	utils.StartNode(stack)

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Got sigterm, shutting swarm down...")
		stack.Stop()
	}()

	// add swarm bootnodes, because swarm doesn't use p2p package's discovery discv5
	go func() {
		s := stack.Server()

		for _, n := range cfg.P2P.BootstrapNodes {
			s.AddPeer(n)
		}
	}()

	stack.Wait()
	return nil
}

func registerBzzService(bzzconfig *bzzapi.Config, stack *node.Node) {
	//define the swarm service boot function
	boot := func(_ *node.ServiceContext) (node.Service, error) {
		var nodeStore *mock.NodeStore
		if bzzconfig.GlobalStoreAPI != "" {
			// connect to global store
			client, err := rpc.Dial(bzzconfig.GlobalStoreAPI)
			if err != nil {
				return nil, fmt.Errorf("global store: %v", err)
			}
			globalStore := mockrpc.NewGlobalStore(client)
			// create a node store for this swarm key on global store
			nodeStore = globalStore.NewNodeStore(common.HexToAddress(bzzconfig.BzzKey))
		}
		return swarm.NewSwarm(bzzconfig, nodeStore)
	}
	//register within the ethereum node
	if err := stack.Register(boot); err != nil {
		utils.Fatalf("Failed to register the Swarm service: %v", err)
	}
}

func getAccount(bzzaccount string, ctx *cli.Context, stack *node.Node) *ecdsa.PrivateKey {
	//an account is mandatory
	if bzzaccount == "" {
		utils.Fatalf(SWARM_ERR_NO_BZZACCOUNT)
	}
	// Try to load the arg as a hex key file.
	if key, err := crypto.LoadECDSA(bzzaccount); err == nil {
		log.Info("Swarm account key loaded", "address", crypto.PubkeyToAddress(key.PublicKey))
		return key
	}
	// Otherwise try getting it from the keystore.
	am := stack.AccountManager()
	ks := am.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)

	return decryptStoreAccount(ks, bzzaccount, utils.MakePasswordList(ctx))
}

// getPrivKey returns the private key of the specified bzzaccount
// Used only by client commands, such as `feed`
func getPrivKey(ctx *cli.Context) *ecdsa.PrivateKey {
	// booting up the swarm node just as we do in bzzd action
	bzzconfig, err := buildConfig(ctx)
	if err != nil {
		utils.Fatalf("unable to configure swarm: %v", err)
	}
	cfg := defaultNodeConfig
	if _, err := os.Stat(bzzconfig.Path); err == nil {
		cfg.DataDir = bzzconfig.Path
	}
	utils.SetNodeConfig(ctx, &cfg)
	stack, err := node.New(&cfg)
	if err != nil {
		utils.Fatalf("can't create node: %v", err)
	}
	return getAccount(bzzconfig.BzzAccount, ctx, stack)
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
		utils.Fatalf("Can't find swarm account key: %v - Is the provided bzzaccount(%s) from the right datadir/Path?", err, account)
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

// addDefaultHelpSubcommand scans through defined CLI commands and adds
// a basic help subcommand to each
// if a help command is already defined, it will take precedence over the default.
func addDefaultHelpSubcommands(commands []cli.Command) {
	for i := range commands {
		cmd := &commands[i]
		if cmd.Subcommands != nil {
			cmd.Subcommands = append(cmd.Subcommands, defaultSubcommandHelp)
			addDefaultHelpSubcommands(cmd.Subcommands)
		}
	}
}

func setSwarmBootstrapNodes(ctx *cli.Context, cfg *node.Config) {
	if ctx.GlobalIsSet(utils.BootnodesFlag.Name) || ctx.GlobalIsSet(utils.BootnodesV4Flag.Name) {
		return
	}

	cfg.P2P.BootstrapNodes = []*enode.Node{}

	for _, url := range SwarmBootnodes {
		node, err := enode.ParseV4(url)
		if err != nil {
			log.Error("Bootstrap URL invalid", "enode", url, "err", err)
		}
		cfg.P2P.BootstrapNodes = append(cfg.P2P.BootstrapNodes, node)
	}

}
