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

package main

import (
	"fmt"
	"os"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/accounts/keystore"
	"github.com/XinFinOrg/XDPoSChain/cmd/utils"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/console/prompt"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/eth"
	"github.com/XinFinOrg/XDPoSChain/ethclient"
	"github.com/XinFinOrg/XDPoSChain/internal/debug"
	"github.com/XinFinOrg/XDPoSChain/internal/ethapi"
	"github.com/XinFinOrg/XDPoSChain/internal/flags"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/metrics"
	"github.com/XinFinOrg/XDPoSChain/node"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/urfave/cli/v2"

	// Force-load the tracer engines to trigger registration
	_ "github.com/XinFinOrg/XDPoSChain/eth/tracers/js"
	_ "github.com/XinFinOrg/XDPoSChain/eth/tracers/live"
	_ "github.com/XinFinOrg/XDPoSChain/eth/tracers/native"
)

const (
	clientIdentifier = "XDC" // Client identifier to advertise over the network
)

var (
	// The app that holds all commands and flags.
	app = flags.NewApp("the XDPoSChain command line interface")

	// The app that holds all commands and flags.
	nodeFlags = slices.Concat([]cli.Flag{
		utils.IdentityFlag,
		utils.UnlockedAccountFlag,
		utils.PasswordFileFlag,
		utils.PeersAllowlistFlag,
		utils.PeersDenylistFlag,
		utils.BootnodesFlag,
		utils.BootnodesV4Flag,
		utils.BootnodesV5Flag,
		utils.KeyStoreDirFlag,
		utils.NoUSBFlag, // deprecated
		utils.USBFlag,
		utils.SmartCardDaemonPathFlag,
		utils.EnablePersonal,
		utils.XDCXDBNameFlag,
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
		utils.GCModeFlag,
		// utils.LightServFlag,  // deprecated
		// utils.LightPeersFlag, // deprecated
		//utils.LightKDFFlag,
		utils.CacheFlag,
		utils.CacheDatabaseFlag,
		utils.CacheTrieFlag,
		utils.CacheGCFlag,
		utils.CachePrefetchFlag,
		//utils.TrieCacheGenFlag,
		utils.CachePreimagesFlag,
		utils.CacheLogSizeFlag,
		utils.FDLimitFlag,
		utils.CryptoKZGFlag,
		utils.ListenPortFlag,
		utils.MaxPeersFlag,
		utils.MaxPendingPeersFlag,
		utils.MinerEtherbaseFlag,
		utils.MinerGasPriceFlag,
		utils.MinerThreadsFlag,
		utils.MiningEnabledFlag,
		utils.MinerGasLimitFlag,
		utils.NATFlag,
		utils.NoDiscoverFlag,
		//utils.DiscoveryV5Flag,
		//utils.NetrestrictFlag,
		utils.NodeKeyFileFlag,
		utils.NodeKeyHexFlag,
		//utils.DeveloperFlag,
		//utils.DeveloperPeriodFlag,
		//utils.VMEnableDebugFlag,
		utils.Enable0xPrefixFlag,
		utils.EnableXDCPrefixFlag,
		utils.VMTraceFlag,
		utils.VMTraceJsonConfigFlag,
		utils.NetworkIdFlag,
		utils.HTTPCORSDomainFlag,
		utils.AuthListenFlag,
		utils.AuthPortFlag,
		utils.AuthVirtualHostsFlag,
		utils.JWTSecretFlag,
		utils.HTTPVirtualHostsFlag,
		utils.EthStatsURLFlag,
		//utils.NoCompactionFlag,
		//utils.GpoBlocksFlag,
		//utils.GpoPercentileFlag,
		utils.GpoMaxGasPriceFlag,
		utils.GpoIgnoreGasPriceFlag,
		//utils.ExtraDataFlag,
		configFileFlag,
		utils.LogDebugFlag,
		utils.LogBacktraceAtFlag,
		utils.AnnounceTxsFlag,
		utils.StoreRewardFlag,
		utils.SetHeadFlag,
		utils.DeleteAllBadBlocksFlag,
	}, utils.NetworkFlags, utils.DatabaseFlags)

	rpcFlags = []cli.Flag{
		utils.HTTPEnabledFlag,
		utils.RPCGlobalGasCapFlag,
		utils.RPCGlobalEVMTimeoutFlag,
		utils.HTTPListenAddrFlag,
		utils.HTTPPortFlag,
		utils.HTTPReadTimeoutFlag,
		utils.HTTPReadHeaderTimeoutFlag,
		utils.HTTPWriteTimeoutFlag,
		utils.HTTPIdleTimeoutFlag,
		utils.HTTPApiFlag,
		utils.HTTPPathPrefixFlag,
		utils.WSEnabledFlag,
		utils.WSListenAddrFlag,
		utils.WSPortFlag,
		utils.WSApiFlag,
		utils.WSAllowedOriginsFlag,
		utils.WSPathPrefixFlag,
		utils.IPCDisabledFlag,
		utils.IPCPathFlag,
		utils.RPCGlobalTxFeeCapFlag,
		utils.RPCGlobalLogQueryLimit,
		utils.AllowUnprotectedTxsFlag,
		utils.BatchRequestLimitFlag,
		utils.BatchResponseMaxSizeFlag,
		utils.RPCGlobalRangeLimitFlag,
	}

	metricsFlags = []cli.Flag{
		utils.MetricsEnabledFlag,
		utils.MetricsEnabledExpensiveFlag,
		utils.MetricsHTTPFlag,
		utils.MetricsPortFlag,
		utils.MetricsEnableInfluxDBFlag,
		utils.MetricsInfluxDBEndpointFlag,
		utils.MetricsInfluxDBDatabaseFlag,
		utils.MetricsInfluxDBUsernameFlag,
		utils.MetricsInfluxDBPasswordFlag,
		utils.MetricsInfluxDBTagsFlag,
		utils.MetricsInfluxDBIntervalFlag,
		utils.MetricsEnableInfluxDBV2Flag,
		utils.MetricsInfluxDBTokenFlag,
		utils.MetricsInfluxDBBucketFlag,
		utils.MetricsInfluxDBOrganizationFlag,
	}
)

func init() {
	// Initialize the CLI app and start XDC
	app.Action = XDC
	app.Copyright = "Copyright 2019-2026 XDC Network"
	app.Commands = []*cli.Command{
		// See chaincmd.go:
		initCommand,
		importCommand,
		exportCommand,
		importPreimagesCommand,
		exportPreimagesCommand,
		removedbCommand,
		dumpCommand,
		// See accountcmd.go:
		accountCommand,
		walletCommand,
		// See consolecmd.go:
		consoleCommand,
		attachCommand,
		javascriptCommand,
		// See misccmd.go:
		versionCommand,
		licenseCommand,
		// See config.go
		dumpConfigCommand,
		// see dbcmd.go
		dbCommand,
		// See cmd/utils/flags_legacy.go
		utils.ShowDeprecated,
	}
	sort.Sort(cli.CommandsByName(app.Commands))

	app.Flags = append(app.Flags, nodeFlags...)
	app.Flags = append(app.Flags, rpcFlags...)
	app.Flags = append(app.Flags, consoleFlags...)
	app.Flags = append(app.Flags, debug.Flags...)
	app.Flags = append(app.Flags, metricsFlags...)
	flags.AutoEnvVars(app.Flags, "XDC")

	app.Before = func(ctx *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		flags.MigrateGlobalFlags(ctx)
		if err := debug.Setup(ctx); err != nil {
			return err
		}
		flags.CheckEnvVars(ctx, app.Flags, "XDC")

		// Start system runtime metrics collection
		go metrics.CollectProcessMetrics(3 * time.Second)

		params.TargetGasLimit = ctx.Uint64(utils.MinerGasLimitFlag.Name)
		return nil
	}

	app.After = func(ctx *cli.Context) error {
		debug.Exit()
		prompt.Stdin.Close() // Resets terminal mode.
		return nil
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// XDC is the main entry point into the system if no special subcommand is ran.
// It creates a default node based on the command line arguments and runs it in
// blocking mode, waiting for it to be shut down.
func XDC(ctx *cli.Context) error {
	stack, backend, cfg := makeFullNode(ctx)
	defer stack.Close()
	startNode(ctx, stack, backend, cfg, false)
	stack.Wait()
	if engine, ok := backend.Engine().(*XDPoS.XDPoS); ok {
		engine.Stop()
	}
	return nil
}

// startNode boots up the system node and all registered protocols, after which
// it unlocks any requested accounts, and starts the RPC/IPC interfaces and the
// miner.
func startNode(ctx *cli.Context, stack *node.Node, backend ethapi.Backend, cfg XDCConfig, isConsole bool) {
	// Start up the node itself
	utils.StartNode(stack, isConsole)

	// Unlock any account specifically requested
	backends := stack.AccountManager().Backends(keystore.KeyStoreType)
	if len(backends) == 0 {
		log.Warn("Failed to unlock accounts, keystore is not available")
		return
	}
	ks := backends[0].(*keystore.KeyStore)

	if ctx.IsSet(utils.UnlockedAccountFlag.Name) {
		cfg.Account.Unlocks = strings.Split(ctx.String(utils.UnlockedAccountFlag.Name), ",")
	}

	if ctx.IsSet(utils.PasswordFileFlag.Name) {
		cfg.Account.Passwords = utils.MakePasswordList(ctx)
	}

	for i, account := range cfg.Account.Unlocks {
		if trimmed := strings.TrimSpace(account); trimmed != "" {
			unlockAccount(ctx, ks, trimmed, i, cfg.Account.Passwords)
		}
	}
	// Register wallet event handlers to open and auto-derive wallets
	events := make(chan accounts.WalletEvent, 16)
	stack.AccountManager().Subscribe(events)

	go func() {
		// Create an chain state reader for self-derivation
		rpcClient := stack.Attach()
		stateReader := ethclient.NewClient(rpcClient)

		// Open any wallets already attached
		for _, wallet := range stack.AccountManager().Wallets() {
			if err := wallet.Open(""); err != nil {
				log.Warn("Failed to open wallet", "url", wallet.URL(), "err", err)
			}
		}
		// Listen for wallet event till termination
		for event := range events {
			switch event.Kind {
			case accounts.WalletArrived:
				if err := event.Wallet.Open(""); err != nil {
					log.Warn("New wallet appeared, failed to open", "url", event.Wallet.URL(), "err", err)
				}
			case accounts.WalletOpened:
				status, _ := event.Wallet.Status()
				log.Info("New wallet appeared", "url", event.Wallet.URL(), "status", status)

				var derivationPaths []accounts.DerivationPath
				if event.Wallet.URL().Scheme == "ledger" {
					derivationPaths = append(derivationPaths, accounts.LegacyLedgerBaseDerivationPath)
				}
				derivationPaths = append(derivationPaths, accounts.DefaultBaseDerivationPath)

				event.Wallet.SelfDerive(derivationPaths, stateReader)

			case accounts.WalletDropped:
				log.Info("Old wallet dropped", "url", event.Wallet.URL())
				event.Wallet.Close()
			}
		}
	}()
	// Start auxiliary services if enabled

	ethBackend, ok := backend.(*eth.EthAPIBackend)
	if !ok {
		utils.Fatalf("Ethereum service not running")
	}
	if engine, ok := ethBackend.Engine().(*XDPoS.XDPoS); ok {
		go func() {
			started := false
			ok := false
			var err error
			ok, err = ethBackend.ValidateMasternode()
			if err != nil {
				utils.Fatalf("Can't verify masternode permission: %v", err)
			}
			if ok {
				log.Info("Masternode found. Enabling staking mode...")
				// Use a reduced number of threads if requested
				if threads := ctx.Int(utils.MinerThreadsFlag.Name); threads > 0 {
					type threaded interface {
						SetThreads(threads int)
					}
					if th, ok := ethBackend.Engine().(threaded); ok {
						th.SetThreads(threads)
					}
				}
				if err := ethBackend.StartStaking(true); err != nil {
					utils.Fatalf("Failed to start staking: %v", err)
				}
				started = true
				log.Info("Enabled staking node!!!")
			}
			defer close(core.CheckpointCh)
			for range core.CheckpointCh {
				log.Info("Checkpoint!!! It's time to reconcile node's state...")
				log.Info("Update consensus parameters")
				chain := ethBackend.BlockChain()
				engine.UpdateParams(chain.CurrentHeader())

				ok, err = ethBackend.ValidateMasternode()
				if err != nil {
					utils.Fatalf("Can't verify masternode permission: %v", err)
				}
				if !ok {
					if started {
						log.Info("Only masternode can propose and verify blocks. Cancelling staking on this node...")
						ethBackend.StopStaking()
						started = false
						log.Info("Cancelled mining mode!!!")
					}
				} else if !started {
					log.Info("Masternode found. Enabling staking mode...")
					// Use a reduced number of threads if requested
					if threads := ctx.Int(utils.MinerThreadsFlag.Name); threads > 0 {
						type threaded interface {
							SetThreads(threads int)
						}
						if th, ok := ethBackend.Engine().(threaded); ok {
							th.SetThreads(threads)
						}
					}
					if err := ethBackend.StartStaking(true); err != nil {
						utils.Fatalf("Failed to start staking: %v", err)
					}
					started = true
					log.Info("Enabled staking node!!!")
				}
			}
		}()
	}
}
