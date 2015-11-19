// Copyright 2014 The go-ethereum Authors
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

// geth is the official command-line client for Ethereum.
package main

import (
	"fmt"
	"io/ioutil"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/codegangsta/cli"
	"github.com/ethereum/ethash"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/comms"
)

const (
	ClientIdentifier = "Geth"
	Version          = "1.4.0-unstable"
	VersionMajor     = 1
	VersionMinor     = 4
	VersionPatch     = 0
)

var (
	gitCommit       string // set via linker flagg
	nodeNameVersion string
	app             *cli.App
)

func init() {
	if gitCommit == "" {
		nodeNameVersion = Version
	} else {
		nodeNameVersion = Version + "-" + gitCommit[:8]
	}

	app = utils.NewApp(Version, "the go-ethereum command line interface")
	app.Action = run
	app.HideVersion = true // we have a command to print the version
	app.Commands = []cli.Command{
		{
			Action: blockRecovery,
			Name:   "recover",
			Usage:  "Attempts to recover a corrupted database by setting a new block by number or hash",
			Description: `
The recover commands will attempt to read out the last
block based on that.

recover #number recovers by number
recover <hex> recovers by hash
`,
		},
		blocktestCommand,
		importCommand,
		exportCommand,
		upgradedbCommand,
		removedbCommand,
		dumpCommand,
		monitorCommand,
		{
			Action: makedag,
			Name:   "makedag",
			Usage:  "generate ethash dag (for testing)",
			Description: `
The makedag command generates an ethash DAG in /tmp/dag.

This command exists to support the system testing project.
Regular users do not need to execute it.
`,
		},
		{
			Action: gpuinfo,
			Name:   "gpuinfo",
			Usage:  "gpuinfo",
			Description: `
Prints OpenCL device info for all found GPUs.
`,
		},
		{
			Action: gpubench,
			Name:   "gpubench",
			Usage:  "benchmark GPU",
			Description: `
Runs quick benchmark on first GPU found.
`,
		},
		{
			Action: version,
			Name:   "version",
			Usage:  "print ethereum version numbers",
			Description: `
The output of this command is supposed to be machine-readable.
`,
		},

		{
			Name:  "wallet",
			Usage: "ethereum presale wallet",
			Subcommands: []cli.Command{
				{
					Action: importWallet,
					Name:   "import",
					Usage:  "import ethereum presale wallet",
				},
			},
			Description: `

    get wallet import /path/to/my/presale.wallet

will prompt for your password and imports your ether presale account.
It can be used non-interactively with the --password option taking a
passwordfile as argument containing the wallet password in plaintext.

`},
		{
			Action: accountList,
			Name:   "account",
			Usage:  "manage accounts",
			Description: `

Manage accounts lets you create new accounts, list all existing accounts,
import a private key into a new account.

'            help' shows a list of subcommands or help for one subcommand.

It supports interactive mode, when you are prompted for password as well as
non-interactive mode where passwords are supplied via a given password file.
Non-interactive mode is only meant for scripted use on test networks or known
safe environments.

Make sure you remember the password you gave when creating a new account (with
either new or import). Without it you are not able to unlock your account.

Note that exporting your key in unencrypted format is NOT supported.

Keys are stored under <DATADIR>/keys.
It is safe to transfer the entire directory or the individual keys therein
between ethereum nodes by simply copying.
Make sure you backup your keys regularly.

In order to use your account to send transactions, you need to unlock them using
the '--unlock' option. The argument is a space separated list of addresses or
indexes. If used non-interactively with a passwordfile, the file should contain
the respective passwords one per line. If you unlock n accounts and the password
file contains less than n entries, then the last password is meant to apply to
all remaining accounts.

And finally. DO NOT FORGET YOUR PASSWORD.
`,
			Subcommands: []cli.Command{
				{
					Action: accountList,
					Name:   "list",
					Usage:  "print account addresses",
				},
				{
					Action: accountCreate,
					Name:   "new",
					Usage:  "create a new account",
					Description: `

    ethereum account new

Creates a new account. Prints the address.

The account is saved in encrypted format, you are prompted for a passphrase.

You must remember this passphrase to unlock your account in the future.

For non-interactive use the passphrase can be specified with the --password flag:

    ethereum --password <passwordfile> account new

Note, this is meant to be used for testing only, it is a bad idea to save your
password to file or expose in any other way.
					`,
				},
				{
					Action: accountUpdate,
					Name:   "update",
					Usage:  "update an existing account",
					Description: `

    ethereum account update <address>

Update an existing account.

The account is saved in the newest version in encrypted format, you are prompted
for a passphrase to unlock the account and another to save the updated file.

This same command can therefore be used to migrate an account of a deprecated
format to the newest format or change the password for an account.

For non-interactive use the passphrase can be specified with the --password flag:

    ethereum --password <passwordfile> account update <address>

Since only one password can be given, only format update can be performed,
changing your password is only possible interactively.

Note that account update has the a side effect that the order of your accounts
changes.
					`,
				},
				{
					Action: accountImport,
					Name:   "import",
					Usage:  "import a private key into a new account",
					Description: `

    ethereum account import <keyfile>

Imports an unencrypted private key from <keyfile> and creates a new account.
Prints the address.

The keyfile is assumed to contain an unencrypted private key in hexadecimal format.

The account is saved in encrypted format, you are prompted for a passphrase.

You must remember this passphrase to unlock your account in the future.

For non-interactive use the passphrase can be specified with the -password flag:

    ethereum --password <passwordfile> account import <keyfile>

Note:
As you can directly copy your encrypted accounts to another ethereum instance,
this import mechanism is not needed when you transfer an account between
nodes.
					`,
				},
			},
		},
		{
			Action: console,
			Name:   "console",
			Usage:  `Geth Console: interactive JavaScript environment`,
			Description: `
The Geth console is an interactive shell for the JavaScript runtime environment
which exposes a node admin interface as well as the Ðapp JavaScript API.
See https://github.com/ethereum/go-ethereum/wiki/Javascipt-Console
`},
		{
			Action: attach,
			Name:   "attach",
			Usage:  `Geth Console: interactive JavaScript environment (connect to node)`,
			Description: `
The Geth console is an interactive shell for the JavaScript runtime environment
which exposes a node admin interface as well as the Ðapp JavaScript API.
See https://github.com/ethereum/go-ethereum/wiki/Javascipt-Console.
This command allows to open a console on a running geth node.
`,
		},
		{
			Action: execJSFiles,
			Name:   "js",
			Usage:  `executes the given JavaScript files in the Geth JavaScript VM`,
			Description: `
The JavaScript VM exposes a node admin interface as well as the Ðapp
JavaScript API. See https://github.com/ethereum/go-ethereum/wiki/Javascipt-Console
`,
		},
	}
	app.Flags = []cli.Flag{
		utils.IdentityFlag,
		utils.UnlockedAccountFlag,
		utils.PasswordFileFlag,
		utils.GenesisFileFlag,
		utils.BootnodesFlag,
		utils.DataDirFlag,
		utils.BlockchainVersionFlag,
		utils.OlympicFlag,
		utils.FastSyncFlag,
		utils.CacheFlag,
		utils.LightKDFFlag,
		utils.JSpathFlag,
		utils.ListenPortFlag,
		utils.MaxPeersFlag,
		utils.MaxPendingPeersFlag,
		utils.EtherbaseFlag,
		utils.GasPriceFlag,
		utils.MinerThreadsFlag,
		utils.MiningEnabledFlag,
		utils.MiningGPUFlag,
		utils.AutoDAGFlag,
		utils.NATFlag,
		utils.NatspecEnabledFlag,
		utils.NoDiscoverFlag,
		utils.NodeKeyFileFlag,
		utils.NodeKeyHexFlag,
		utils.RPCEnabledFlag,
		utils.RPCListenAddrFlag,
		utils.RPCPortFlag,
		utils.RpcApiFlag,
		utils.IPCDisabledFlag,
		utils.IPCApiFlag,
		utils.IPCPathFlag,
		utils.ExecFlag,
		utils.WhisperEnabledFlag,
		utils.DevModeFlag,
		utils.TestNetFlag,
		utils.VMDebugFlag,
		utils.VMForceJitFlag,
		utils.VMJitCacheFlag,
		utils.VMEnableJitFlag,
		utils.NetworkIdFlag,
		utils.RPCCORSDomainFlag,
		utils.VerbosityFlag,
		utils.BacktraceAtFlag,
		utils.LogVModuleFlag,
		utils.LogFileFlag,
		utils.PProfEanbledFlag,
		utils.PProfPortFlag,
		utils.MetricsEnabledFlag,
		utils.SolcPathFlag,
		utils.GpoMinGasPriceFlag,
		utils.GpoMaxGasPriceFlag,
		utils.GpoFullBlockRatioFlag,
		utils.GpobaseStepDownFlag,
		utils.GpobaseStepUpFlag,
		utils.GpobaseCorrectionFactorFlag,
		utils.ExtraDataFlag,
	}
	app.Before = func(ctx *cli.Context) error {
		utils.SetupLogger(ctx)
		utils.SetupNetwork(ctx)
		utils.SetupVM(ctx)
		if ctx.GlobalBool(utils.PProfEanbledFlag.Name) {
			utils.StartPProf(ctx)
		}
		return nil
	}
	// Start system runtime metrics collection
	go metrics.CollectProcessMetrics(3 * time.Second)
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	defer logger.Flush()
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// makeExtra resolves extradata for the miner from a flag or returns a default.
func makeExtra(ctx *cli.Context) []byte {
	if ctx.GlobalIsSet(utils.ExtraDataFlag.Name) {
		return []byte(ctx.GlobalString(utils.ExtraDataFlag.Name))
	}
	return makeDefaultExtra()
}

func makeDefaultExtra() []byte {
	var clientInfo = struct {
		Version   uint
		Name      string
		GoVersion string
		Os        string
	}{uint(VersionMajor<<16 | VersionMinor<<8 | VersionPatch), ClientIdentifier, runtime.Version(), runtime.GOOS}
	extra, err := rlp.EncodeToBytes(clientInfo)
	if err != nil {
		glog.V(logger.Warn).Infoln("error setting canonical miner information:", err)
	}

	if uint64(len(extra)) > params.MaximumExtraDataSize.Uint64() {
		glog.V(logger.Warn).Infoln("error setting canonical miner information: extra exceeds", params.MaximumExtraDataSize)
		glog.V(logger.Debug).Infof("extra: %x\n", extra)
		return nil
	}
	return extra
}

func run(ctx *cli.Context) {
	cfg := utils.MakeEthConfig(ClientIdentifier, nodeNameVersion, ctx)
	cfg.ExtraData = makeExtra(ctx)

	ethereum, err := eth.New(cfg)
	if err != nil {
		utils.Fatalf("%v", err)
	}

	startEth(ctx, ethereum)
	// this blocks the thread
	ethereum.WaitForShutdown()
}

func attach(ctx *cli.Context) {
	var client comms.EthereumClient
	var err error
	if ctx.Args().Present() {
		client, err = comms.ClientFromEndpoint(ctx.Args().First(), codec.JSON)
	} else {
		cfg := comms.IpcConfig{
			Endpoint: utils.IpcSocketPath(ctx),
		}
		client, err = comms.NewIpcClient(cfg, codec.JSON)
	}

	if err != nil {
		utils.Fatalf("Unable to attach to geth node - %v", err)
	}

	repl := newLightweightJSRE(
		ctx.GlobalString(utils.JSpathFlag.Name),
		client,
		ctx.GlobalString(utils.DataDirFlag.Name),
		true,
	)

	if ctx.GlobalString(utils.ExecFlag.Name) != "" {
		repl.batch(ctx.GlobalString(utils.ExecFlag.Name))
	} else {
		repl.welcome()
		repl.interactive()
	}
}

func console(ctx *cli.Context) {
	cfg := utils.MakeEthConfig(ClientIdentifier, nodeNameVersion, ctx)
	cfg.ExtraData = makeExtra(ctx)

	ethereum, err := eth.New(cfg)
	if err != nil {
		utils.Fatalf("%v", err)
	}

	client := comms.NewInProcClient(codec.JSON)

	startEth(ctx, ethereum)
	repl := newJSRE(
		ethereum,
		ctx.GlobalString(utils.JSpathFlag.Name),
		ctx.GlobalString(utils.RPCCORSDomainFlag.Name),
		client,
		true,
		nil,
	)

	if ctx.GlobalString(utils.ExecFlag.Name) != "" {
		repl.batch(ctx.GlobalString(utils.ExecFlag.Name))
	} else {
		repl.welcome()
		repl.interactive()
	}

	ethereum.Stop()
	ethereum.WaitForShutdown()
}

func execJSFiles(ctx *cli.Context) {
	cfg := utils.MakeEthConfig(ClientIdentifier, nodeNameVersion, ctx)
	ethereum, err := eth.New(cfg)
	if err != nil {
		utils.Fatalf("%v", err)
	}

	client := comms.NewInProcClient(codec.JSON)
	startEth(ctx, ethereum)
	repl := newJSRE(
		ethereum,
		ctx.GlobalString(utils.JSpathFlag.Name),
		ctx.GlobalString(utils.RPCCORSDomainFlag.Name),
		client,
		false,
		nil,
	)
	for _, file := range ctx.Args() {
		repl.exec(file)
	}

	ethereum.Stop()
	ethereum.WaitForShutdown()
}

func unlockAccount(ctx *cli.Context, am *accounts.Manager, addr string, i int, inputpassphrases []string) (addrHex, auth string, passphrases []string) {
	var err error
	passphrases = inputpassphrases
	addrHex, err = utils.ParamToAddress(addr, am)
	if err == nil {
		// Attempt to unlock the account 3 times
		attempts := 3
		for tries := 0; tries < attempts; tries++ {
			msg := fmt.Sprintf("Unlocking account %s | Attempt %d/%d", addr, tries+1, attempts)
			auth, passphrases = getPassPhrase(ctx, msg, false, i, passphrases)
			err = am.Unlock(common.HexToAddress(addrHex), auth)
			if err == nil || passphrases != nil {
				break
			}
		}
	}

	if err != nil {
		utils.Fatalf("Unlock account '%s' (%v) failed: %v", addr, addrHex, err)
	}
	fmt.Printf("Account '%s' (%v) unlocked.\n", addr, addrHex)
	return
}

func blockRecovery(ctx *cli.Context) {
	if len(ctx.Args()) < 1 {
		glog.Fatal("recover requires block number or hash")
	}
	arg := ctx.Args().First()

	cfg := utils.MakeEthConfig(ClientIdentifier, nodeNameVersion, ctx)
	blockDb, err := ethdb.NewLDBDatabase(filepath.Join(cfg.DataDir, "blockchain"), cfg.DatabaseCache)
	if err != nil {
		glog.Fatalln("could not open db:", err)
	}

	var block *types.Block
	if arg[0] == '#' {
		block = core.GetBlock(blockDb, core.GetCanonicalHash(blockDb, common.String2Big(arg[1:]).Uint64()))
	} else {
		block = core.GetBlock(blockDb, common.HexToHash(arg))
	}

	if block == nil {
		glog.Fatalln("block not found. Recovery failed")
	}

	if err = core.WriteHeadBlockHash(blockDb, block.Hash()); err != nil {
		glog.Fatalln("block write err", err)
	}
	glog.Infof("Recovery succesful. New HEAD %x\n", block.Hash())
}

func startEth(ctx *cli.Context, eth *eth.Ethereum) {
	// Start Ethereum itself
	utils.StartEthereum(eth)

	am := eth.AccountManager()
	account := ctx.GlobalString(utils.UnlockedAccountFlag.Name)
	accounts := strings.Split(account, " ")
	var passphrases []string
	for i, account := range accounts {
		if len(account) > 0 {
			if account == "primary" {
				utils.Fatalf("the 'primary' keyword is deprecated. You can use integer indexes, but the indexes are not permanent, they can change if you add external keys, export your keys or copy your keystore to another node.")
			}
			_, _, passphrases = unlockAccount(ctx, am, account, i, passphrases)
		}
	}
	// Start auxiliary services if enabled.
	if !ctx.GlobalBool(utils.IPCDisabledFlag.Name) {
		if err := utils.StartIPC(eth, ctx); err != nil {
			utils.Fatalf("Error string IPC: %v", err)
		}
	}
	if ctx.GlobalBool(utils.RPCEnabledFlag.Name) {
		if err := utils.StartRPC(eth, ctx); err != nil {
			utils.Fatalf("Error starting RPC: %v", err)
		}
	}
	if ctx.GlobalBool(utils.MiningEnabledFlag.Name) {
		err := eth.StartMining(
			ctx.GlobalInt(utils.MinerThreadsFlag.Name),
			ctx.GlobalString(utils.MiningGPUFlag.Name))
		if err != nil {
			utils.Fatalf("%v", err)
		}
	}
}

func accountList(ctx *cli.Context) {
	am := utils.MakeAccountManager(ctx)
	accts, err := am.Accounts()
	if err != nil {
		utils.Fatalf("Could not list accounts: %v", err)
	}
	for i, acct := range accts {
		fmt.Printf("Account #%d: %x\n", i, acct)
	}
}

func getPassPhrase(ctx *cli.Context, desc string, confirmation bool, i int, inputpassphrases []string) (passphrase string, passphrases []string) {
	passfile := ctx.GlobalString(utils.PasswordFileFlag.Name)
	if len(passfile) == 0 {
		fmt.Println(desc)
		auth, err := utils.PromptPassword("Passphrase: ", true)
		if err != nil {
			utils.Fatalf("%v", err)
		}
		if confirmation {
			confirm, err := utils.PromptPassword("Repeat Passphrase: ", false)
			if err != nil {
				utils.Fatalf("%v", err)
			}
			if auth != confirm {
				utils.Fatalf("Passphrases did not match.")
			}
		}
		passphrase = auth

	} else {
		passphrases = inputpassphrases
		if passphrases == nil {
			passbytes, err := ioutil.ReadFile(passfile)
			if err != nil {
				utils.Fatalf("Unable to read password file '%s': %v", passfile, err)
			}
			// this is backwards compatible if the same password unlocks several accounts
			// it also has the consequence that trailing newlines will not count as part
			// of the password, so --password <(echo -n 'pass') will now work without -n
			passphrases = strings.Split(string(passbytes), "\n")
		}
		if i >= len(passphrases) {
			passphrase = passphrases[len(passphrases)-1]
		} else {
			passphrase = passphrases[i]
		}
	}
	return
}

func accountCreate(ctx *cli.Context) {
	am := utils.MakeAccountManager(ctx)
	passphrase, _ := getPassPhrase(ctx, "Your new account is locked with a password. Please give a password. Do not forget this password.", true, 0, nil)
	acct, err := am.NewAccount(passphrase)
	if err != nil {
		utils.Fatalf("Could not create the account: %v", err)
	}
	fmt.Printf("Address: %x\n", acct)
}

func accountUpdate(ctx *cli.Context) {
	am := utils.MakeAccountManager(ctx)
	arg := ctx.Args().First()
	if len(arg) == 0 {
		utils.Fatalf("account address or index must be given as argument")
	}

	addr, authFrom, passphrases := unlockAccount(ctx, am, arg, 0, nil)
	authTo, _ := getPassPhrase(ctx, "Please give a new password. Do not forget this password.", true, 0, passphrases)
	err := am.Update(common.HexToAddress(addr), authFrom, authTo)
	if err != nil {
		utils.Fatalf("Could not update the account: %v", err)
	}
}

func importWallet(ctx *cli.Context) {
	keyfile := ctx.Args().First()
	if len(keyfile) == 0 {
		utils.Fatalf("keyfile must be given as argument")
	}
	keyJson, err := ioutil.ReadFile(keyfile)
	if err != nil {
		utils.Fatalf("Could not read wallet file: %v", err)
	}

	am := utils.MakeAccountManager(ctx)
	passphrase, _ := getPassPhrase(ctx, "", false, 0, nil)

	acct, err := am.ImportPreSaleKey(keyJson, passphrase)
	if err != nil {
		utils.Fatalf("Could not create the account: %v", err)
	}
	fmt.Printf("Address: %x\n", acct)
}

func accountImport(ctx *cli.Context) {
	keyfile := ctx.Args().First()
	if len(keyfile) == 0 {
		utils.Fatalf("keyfile must be given as argument")
	}
	am := utils.MakeAccountManager(ctx)
	passphrase, _ := getPassPhrase(ctx, "Your new account is locked with a password. Please give a password. Do not forget this password.", true, 0, nil)
	acct, err := am.Import(keyfile, passphrase)
	if err != nil {
		utils.Fatalf("Could not create the account: %v", err)
	}
	fmt.Printf("Address: %x\n", acct)
}

func makedag(ctx *cli.Context) {
	args := ctx.Args()
	wrongArgs := func() {
		utils.Fatalf(`Usage: geth makedag <block number> <outputdir>`)
	}
	switch {
	case len(args) == 2:
		blockNum, err := strconv.ParseUint(args[0], 0, 64)
		dir := args[1]
		if err != nil {
			wrongArgs()
		} else {
			dir = filepath.Clean(dir)
			// seems to require a trailing slash
			if !strings.HasSuffix(dir, "/") {
				dir = dir + "/"
			}
			_, err = ioutil.ReadDir(dir)
			if err != nil {
				utils.Fatalf("Can't find dir")
			}
			fmt.Println("making DAG, this could take awhile...")
			ethash.MakeDAG(blockNum, dir)
		}
	default:
		wrongArgs()
	}
}

func gpuinfo(ctx *cli.Context) {
	eth.PrintOpenCLDevices()
}

func gpubench(ctx *cli.Context) {
	args := ctx.Args()
	wrongArgs := func() {
		utils.Fatalf(`Usage: geth gpubench <gpu number>`)
	}
	switch {
	case len(args) == 1:
		n, err := strconv.ParseUint(args[0], 0, 64)
		if err != nil {
			wrongArgs()
		}
		eth.GPUBench(n)
	case len(args) == 0:
		eth.GPUBench(0)
	default:
		wrongArgs()
	}
}

func version(c *cli.Context) {
	fmt.Println(ClientIdentifier)
	fmt.Println("Version:", Version)
	if gitCommit != "" {
		fmt.Println("Git Commit:", gitCommit)
	}
	fmt.Println("Protocol Versions:", eth.ProtocolVersions)
	fmt.Println("Network Id:", c.GlobalInt(utils.NetworkIdFlag.Name))
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("OS:", runtime.GOOS)
	fmt.Printf("GOPATH=%s\n", os.Getenv("GOPATH"))
	fmt.Printf("GOROOT=%s\n", runtime.GOROOT())
}
