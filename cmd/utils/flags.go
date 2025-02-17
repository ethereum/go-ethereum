// Copyright 2015 The go-ethereum Authors
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

// Package utils contains internal helper functions for go-ethereum commands.
package utils

import (
	"crypto/ecdsa"
	"fmt"
	"math"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"runtime"
	godebug "runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/XinFinOrg/XDPoSChain/XDCx"
	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/accounts/keystore"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/common/fdlimit"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/ethash"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/txpool"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/crypto/kzg4844"
	"github.com/XinFinOrg/XDPoSChain/eth/downloader"
	"github.com/XinFinOrg/XDPoSChain/eth/ethconfig"
	"github.com/XinFinOrg/XDPoSChain/eth/filters"
	"github.com/XinFinOrg/XDPoSChain/eth/gasprice"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/internal/ethapi"
	"github.com/XinFinOrg/XDPoSChain/internal/flags"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/metrics"
	"github.com/XinFinOrg/XDPoSChain/metrics/exp"
	"github.com/XinFinOrg/XDPoSChain/metrics/influxdb"
	"github.com/XinFinOrg/XDPoSChain/node"
	"github.com/XinFinOrg/XDPoSChain/p2p"
	"github.com/XinFinOrg/XDPoSChain/p2p/discover"
	"github.com/XinFinOrg/XDPoSChain/p2p/discv5"
	"github.com/XinFinOrg/XDPoSChain/p2p/nat"
	"github.com/XinFinOrg/XDPoSChain/p2p/netutil"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/rpc"
	pcsclite "github.com/gballet/go-libpcsclite"
	gopsutil "github.com/shirou/gopsutil/mem"
	"github.com/urfave/cli/v2"
)

// These are all the command line flags we support.
// If you add to this list, please remember to include the
// flag in the appropriate command definition.
//
// The flags are defined here so their names and help texts
// are the same for all commands.

var (
	// General settings
	DataDirFlag = &flags.DirectoryFlag{
		Name:     "datadir",
		Usage:    "Data directory for the databases and keystore",
		Value:    flags.DirectoryString(node.DefaultDataDir()),
		Category: flags.EthCategory,
	}
	KeyStoreDirFlag = &flags.DirectoryFlag{
		Name:     "keystore",
		Usage:    "Directory for the keystore (default = inside the datadir)",
		Category: flags.AccountCategory,
	}
	USBFlag = &cli.BoolFlag{
		Name:     "usb",
		Usage:    "Enable monitoring and management of USB hardware wallets",
		Category: flags.AccountCategory,
	}
	SmartCardDaemonPathFlag = &cli.StringFlag{
		Name:     "pcscdpath",
		Usage:    "Path to the smartcard daemon (pcscd) socket file",
		Value:    pcsclite.PCSCDSockName,
		Category: flags.AccountCategory,
	}
	NetworkIdFlag = &cli.Uint64Flag{
		Name:     "networkid",
		Usage:    "Network identifier (integer, 89=XDPoSChain)",
		Value:    ethconfig.Defaults.NetworkId,
		Category: flags.EthCategory,
	}
	TestnetFlag = &cli.BoolFlag{
		Name:     "testnet",
		Usage:    "Ropsten network: pre-configured proof-of-work test network",
		Category: flags.EthCategory,
	}
	XDCTestnetFlag = &cli.BoolFlag{
		Name:     "apothem",
		Usage:    "XDC Apothem Network",
		Category: flags.EthCategory,
	}
	RinkebyFlag = &cli.BoolFlag{
		Name:     "rinkeby",
		Usage:    "Rinkeby network: pre-configured proof-of-authority test network",
		Category: flags.EthCategory,
	}

	// Dev mode
	DeveloperFlag = &cli.BoolFlag{
		Name:     "dev",
		Usage:    "Ephemeral proof-of-authority network with a pre-funded developer account, mining enabled",
		Category: flags.DevCategory,
	}
	DeveloperPeriodFlag = &cli.IntFlag{
		Name:     "dev-period",
		Aliases:  []string{"dev.period"},
		Usage:    "Block period to use in developer mode (0 = mine only if transaction pending)",
		Category: flags.DevCategory,
	}
	IdentityFlag = &cli.StringFlag{
		Name:     "identity",
		Usage:    "Custom node name",
		Category: flags.NetworkingCategory,
	}
	DocRootFlag = &flags.DirectoryFlag{
		Name:     "docroot",
		Usage:    "Document Root for HTTPClient file scheme",
		Value:    flags.DirectoryString(flags.HomeDir()),
		Category: flags.APICategory,
	}

	SyncModeFlag = &cli.StringFlag{
		Name:     "syncmode",
		Usage:    `Blockchain sync mode ("fast", "full", or "light")`,
		Value:    ethconfig.Defaults.SyncMode.String(),
		Category: flags.EthCategory,
	}
	GCModeFlag = &cli.StringFlag{
		Name:     "gcmode",
		Usage:    `Blockchain garbage collection mode ("full", "archive")`,
		Value:    "full",
		Category: flags.EthCategory,
	}
	LightKDFFlag = &cli.BoolFlag{
		Name:     "lightkdf",
		Usage:    "Reduce key-derivation RAM & CPU usage at some expense of KDF strength",
		Category: flags.AccountCategory,
	}

	// Light server and client settings
	LightServFlag = &cli.IntFlag{
		Name:     "light-serv",
		Aliases:  []string{"lightserv"},
		Usage:    "Maximum percentage of time allowed for serving LES requests (0-90)",
		Value:    ethconfig.Defaults.LightServ,
		Category: flags.LightCategory,
	}
	LightPeersFlag = &cli.IntFlag{
		Name:     "light-peers",
		Aliases:  []string{"lightpeers"},
		Usage:    "Maximum number of LES client peers",
		Value:    ethconfig.Defaults.LightPeers,
		Category: flags.LightCategory,
	}

	// Ethash settings
	EthashCacheDirFlag = &flags.DirectoryFlag{
		Name:     "ethash-cachedir",
		Aliases:  []string{"ethash.cachedir"},
		Usage:    "Directory to store the ethash verification caches (default = inside the datadir)",
		Category: flags.EthashCategory,
	}
	EthashCachesInMemoryFlag = &cli.IntFlag{
		Name:     "ethash-cachesinmem",
		Aliases:  []string{"ethash.cachesinmem"},
		Usage:    "Number of recent ethash caches to keep in memory (16MB each)",
		Value:    ethconfig.Defaults.Ethash.CachesInMem,
		Category: flags.EthashCategory,
	}
	EthashCachesOnDiskFlag = &cli.IntFlag{
		Name:     "ethash-cachesondisk",
		Aliases:  []string{"ethash.cachesondisk"},
		Usage:    "Number of recent ethash caches to keep on disk (16MB each)",
		Value:    ethconfig.Defaults.Ethash.CachesOnDisk,
		Category: flags.EthashCategory,
	}
	EthashDatasetDirFlag = &flags.DirectoryFlag{
		Name:     "ethash-dagdir",
		Aliases:  []string{"ethash.dagdir"},
		Usage:    "Directory to store the ethash mining DAGs (default = inside home folder)",
		Value:    flags.DirectoryString(ethconfig.Defaults.Ethash.DatasetDir),
		Category: flags.EthashCategory,
	}
	EthashDatasetsInMemoryFlag = &cli.IntFlag{
		Name:     "ethash-dagsinmem",
		Aliases:  []string{"ethash.dagsinmem"},
		Usage:    "Number of recent ethash mining DAGs to keep in memory (1+GB each)",
		Value:    ethconfig.Defaults.Ethash.DatasetsInMem,
		Category: flags.EthashCategory,
	}
	EthashDatasetsOnDiskFlag = &cli.IntFlag{
		Name:     "ethash-dagsondisk",
		Aliases:  []string{"ethash.dagsondisk"},
		Usage:    "Number of recent ethash mining DAGs to keep on disk (1+GB each)",
		Value:    ethconfig.Defaults.Ethash.DatasetsOnDisk,
		Category: flags.EthashCategory,
	}

	// Transaction pool settings
	TxPoolNoLocalsFlag = &cli.BoolFlag{
		Name:     "txpool-nolocals",
		Aliases:  []string{"txpool.nolocals"},
		Usage:    "Disables price exemptions for locally submitted transactions",
		Category: flags.TxPoolCategory,
	}
	TxPoolJournalFlag = &cli.StringFlag{
		Name:     "txpool-journal",
		Aliases:  []string{"txpool.journal"},
		Usage:    "Disk journal for local transaction to survive node restarts",
		Value:    txpool.DefaultConfig.Journal,
		Category: flags.TxPoolCategory,
	}
	TxPoolRejournalFlag = &cli.DurationFlag{
		Name:     "txpool-rejournal",
		Aliases:  []string{"txpool.rejournal"},
		Usage:    "Time interval to regenerate the local transaction journal",
		Value:    txpool.DefaultConfig.Rejournal,
		Category: flags.TxPoolCategory,
	}
	TxPoolPriceLimitFlag = &cli.Uint64Flag{
		Name:     "txpool-pricelimit",
		Aliases:  []string{"txpool.pricelimit"},
		Usage:    "Minimum gas price limit to enforce for acceptance into the pool",
		Value:    ethconfig.Defaults.TxPool.PriceLimit,
		Category: flags.TxPoolCategory,
	}
	TxPoolPriceBumpFlag = &cli.Uint64Flag{
		Name:     "txpool-pricebump",
		Aliases:  []string{"txpool.pricebump"},
		Usage:    "Price bump percentage to replace an already existing transaction",
		Value:    ethconfig.Defaults.TxPool.PriceBump,
		Category: flags.TxPoolCategory,
	}
	TxPoolAccountSlotsFlag = &cli.Uint64Flag{
		Name:     "txpool-accountslots",
		Aliases:  []string{"txpool.accountslots"},
		Usage:    "Minimum number of executable transaction slots guaranteed per account",
		Value:    ethconfig.Defaults.TxPool.AccountSlots,
		Category: flags.TxPoolCategory,
	}
	TxPoolGlobalSlotsFlag = &cli.Uint64Flag{
		Name:     "txpool-globalslots",
		Aliases:  []string{"txpool.globalslots"},
		Usage:    "Maximum number of executable transaction slots for all accounts",
		Value:    ethconfig.Defaults.TxPool.GlobalSlots,
		Category: flags.TxPoolCategory,
	}
	TxPoolAccountQueueFlag = &cli.Uint64Flag{
		Name:     "txpool-accountqueue",
		Aliases:  []string{"txpool.accountqueue"},
		Usage:    "Maximum number of non-executable transaction slots permitted per account",
		Value:    ethconfig.Defaults.TxPool.AccountQueue,
		Category: flags.TxPoolCategory,
	}
	TxPoolGlobalQueueFlag = &cli.Uint64Flag{
		Name:     "txpool-globalqueue",
		Aliases:  []string{"txpool.globalqueue"},
		Usage:    "Maximum number of non-executable transaction slots for all accounts",
		Value:    ethconfig.Defaults.TxPool.GlobalQueue,
		Category: flags.TxPoolCategory,
	}
	TxPoolLifetimeFlag = &cli.DurationFlag{
		Name:     "txpool-lifetime",
		Aliases:  []string{"txpool.lifetime"},
		Usage:    "Maximum amount of time non-executable transaction are queued",
		Value:    ethconfig.Defaults.TxPool.Lifetime,
		Category: flags.TxPoolCategory,
	}

	// Performance tuning settings
	CacheFlag = &cli.IntFlag{
		Name:     "cache",
		Usage:    "Megabytes of memory allocated to internal caching",
		Value:    1024,
		Category: flags.PerfCategory,
	}
	CacheDatabaseFlag = &cli.IntFlag{
		Name:     "cache-database",
		Aliases:  []string{"cache.database"},
		Usage:    "Percentage of cache memory allowance to use for database io",
		Value:    50,
		Category: flags.PerfCategory,
	}
	CacheGCFlag = &cli.IntFlag{
		Name:     "cache-gc",
		Aliases:  []string{"cache.gc"},
		Usage:    "Percentage of cache memory allowance to use for trie pruning (default = 25% full mode, 0% archive mode)",
		Value:    25,
		Category: flags.PerfCategory,
	}
	CacheLogSizeFlag = &cli.IntFlag{
		Name:     "cache-blocklogs",
		Aliases:  []string{"cache.blocklogs"},
		Usage:    "Size (in number of blocks) of the log cache for filtering",
		Value:    ethconfig.Defaults.FilterLogCacheSize,
		Category: flags.PerfCategory,
	}
	FDLimitFlag = &cli.IntFlag{
		Name:     "fdlimit",
		Usage:    "Raise the open file descriptor resource limit (default = system fd limit)",
		Category: flags.PerfCategory,
	}
	CryptoKZGFlag = &cli.StringFlag{
		Name:     "crypto-kzg",
		Usage:    "KZG library implementation to use; gokzg (recommended) or ckzg",
		Value:    "gokzg",
		Category: flags.PerfCategory,
	}

	// Miner settings
	MinerThreadsFlag = &cli.IntFlag{
		Name:     "miner-threads",
		Aliases:  []string{"minerthreads"},
		Usage:    "Number of CPU threads to use for mining",
		Value:    runtime.NumCPU(),
		Category: flags.MinerCategory,
	}
	MinerGasLimitFlag = &cli.Uint64Flag{
		Name:     "miner-gaslimit",
		Aliases:  []string{"targetgaslimit"},
		Usage:    "Target gas limit sets the artificial target gas floor for the blocks to mine",
		Value:    50000000,
		Category: flags.MinerCategory,
	}
	MinerGasPriceFlag = &flags.BigFlag{
		Name:     "miner-gasprice",
		Aliases:  []string{"gasprice"},
		Usage:    "Minimal gas price to accept for mining a transactions",
		Value:    big.NewInt(1),
		Category: flags.MinerCategory,
	}
	MinerEtherbaseFlag = &cli.StringFlag{
		Name:     "miner-etherbase",
		Aliases:  []string{"etherbase"},
		Usage:    "Public address for block mining rewards (default = first account created)",
		Value:    "0x000000000000000000000000000000000000abcd",
		Category: flags.MinerCategory,
	}
	MinerExtraDataFlag = &cli.StringFlag{
		Name:     "miner-extradata",
		Aliases:  []string{"extradata"},
		Usage:    "Block extra data set by the miner (default = client version)",
		Category: flags.MinerCategory,
	}

	// Account settings
	UnlockedAccountFlag = &cli.StringFlag{
		Name:     "unlock",
		Usage:    "Comma separated list of accounts to unlock",
		Value:    "",
		Category: flags.AccountCategory,
	}
	PasswordFileFlag = &cli.StringFlag{
		Name:     "password",
		Usage:    "Password file to use for non-interactive password input",
		Value:    "",
		Category: flags.AccountCategory,
	}

	// EVM settings
	VMEnableDebugFlag = &cli.BoolFlag{
		Name:     "vmdebug",
		Usage:    "Record information useful for VM and contract debugging",
		Category: flags.VMCategory,
	}

	// API options
	RPCGlobalGasCapFlag = &cli.Uint64Flag{
		Name:     "rpc-gascap",
		Usage:    "Sets a cap on gas that can be used in eth_call/estimateGas (0=infinite)",
		Value:    ethconfig.Defaults.RPCGasCap,
		Category: flags.APICategory,
	}
	RPCGlobalTxFeeCap = &cli.Float64Flag{
		Name:     "rpc-txfeecap",
		Aliases:  []string{"rpc.txfeecap"},
		Usage:    "Sets a cap on transaction fee (in ether) that can be sent via the RPC APIs (0 = no cap)",
		Value:    ethconfig.Defaults.RPCTxFeeCap,
		Category: flags.APICategory,
	}

	// Logging and debug settings
	EthStatsURLFlag = &cli.StringFlag{
		Name:     "ethstats",
		Usage:    "Reporting URL of a ethstats service (nodename:secret@host:port)",
		Category: flags.MetricsCategory,
	}
	FakePoWFlag = &cli.BoolFlag{
		Name:     "fakepow",
		Usage:    "Disables proof-of-work verification",
		Category: flags.LoggingCategory,
	}
	NoCompactionFlag = &cli.BoolFlag{
		Name:     "nocompaction",
		Usage:    "Disables db compaction after import",
		Category: flags.LoggingCategory,
	}

	// RPC settings
	IPCDisabledFlag = &cli.BoolFlag{
		Name:     "ipcdisable",
		Usage:    "Disable the IPC-RPC server",
		Category: flags.APICategory,
	}
	IPCPathFlag = &flags.DirectoryFlag{
		Name:     "ipcpath",
		Usage:    "Filename for IPC socket/pipe within the datadir (explicit paths escape it)",
		Category: flags.APICategory,
	}
	HTTPEnabledFlag = &cli.BoolFlag{
		Name:     "http",
		Aliases:  []string{"rpc"},
		Usage:    "Enable the HTTP-RPC server",
		Value:    true,
		Category: flags.APICategory,
	}
	HTTPListenAddrFlag = &cli.StringFlag{
		Name:     "http-addr",
		Aliases:  []string{"rpcaddr"},
		Usage:    "HTTP-RPC server listening interface",
		Value:    "0.0.0.0",
		Category: flags.APICategory,
	}
	HTTPPortFlag = &cli.IntFlag{
		Name:     "http-port",
		Aliases:  []string{"rpcport"},
		Usage:    "HTTP-RPC server listening port",
		Value:    node.DefaultHTTPPort,
		Category: flags.APICategory,
	}
	HTTPCORSDomainFlag = &cli.StringFlag{
		Name:     "http-corsdomain",
		Aliases:  []string{"rpccorsdomain"},
		Usage:    "Comma separated list of domains from which to accept cross origin requests (browser enforced)",
		Value:    "*",
		Category: flags.APICategory,
	}
	HTTPVirtualHostsFlag = &cli.StringFlag{
		Name:     "http-vhosts",
		Aliases:  []string{"rpcvhosts"},
		Usage:    "Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard.",
		Value:    "*",
		Category: flags.APICategory,
	}
	HTTPApiFlag = &cli.StringFlag{
		Name:     "http-api",
		Aliases:  []string{"rpcapi"},
		Usage:    "API's offered over the HTTP-RPC interface",
		Value:    "debug,eth,net,personal,txpool,web3,XDPoS",
		Category: flags.APICategory,
	}
	HTTPReadTimeoutFlag = &cli.DurationFlag{
		Name:     "http-readtimeout",
		Aliases:  []string{"rpcreadtimeout"},
		Usage:    "HTTP-RPC server read timeout",
		Value:    rpc.DefaultHTTPTimeouts.ReadTimeout,
		Category: flags.APICategory,
	}
	HTTPWriteTimeoutFlag = &cli.DurationFlag{
		Name:     "http-writetimeout",
		Aliases:  []string{"rpcwritetimeout"},
		Usage:    "HTTP-RPC server write timeout",
		Value:    rpc.DefaultHTTPTimeouts.WriteTimeout,
		Category: flags.APICategory,
	}
	HTTPIdleTimeoutFlag = &cli.DurationFlag{
		Name:     "http-idletimeout",
		Aliases:  []string{"rpcidletimeout"},
		Usage:    "HTTP-RPC server idle timeout",
		Value:    rpc.DefaultHTTPTimeouts.IdleTimeout,
		Category: flags.APICategory,
	}
	WSEnabledFlag = &cli.BoolFlag{
		Name:     "ws",
		Usage:    "Enable the WS-RPC server",
		Value:    true,
		Category: flags.APICategory,
	}
	WSListenAddrFlag = &cli.StringFlag{
		Name:     "ws-addr",
		Aliases:  []string{"wsaddr"},
		Usage:    "WS-RPC server listening interface",
		Value:    "0.0.0.0",
		Category: flags.APICategory,
	}
	WSPortFlag = &cli.IntFlag{
		Name:     "ws-port",
		Aliases:  []string{"wsport"},
		Usage:    "WS-RPC server listening port",
		Value:    node.DefaultWSPort,
		Category: flags.APICategory,
	}
	WSApiFlag = &cli.StringFlag{
		Name:     "ws-api",
		Aliases:  []string{"wsapi"},
		Usage:    "API's offered over the WS-RPC interface",
		Value:    "debug,eth,net,personal,txpool,web3,XDPoS",
		Category: flags.APICategory,
	}
	WSAllowedOriginsFlag = &cli.StringFlag{
		Name:     "ws-origins",
		Aliases:  []string{"wsorigins"},
		Usage:    "Origins from which to accept websockets requests",
		Value:    "*",
		Category: flags.APICategory,
	}
	ExecFlag = &cli.StringFlag{
		Name:     "exec",
		Usage:    "Execute JavaScript statement",
		Category: flags.APICategory,
	}
	PreloadJSFlag = &cli.StringFlag{
		Name:     "preload",
		Usage:    "Comma separated list of JavaScript files to preload into the console",
		Category: flags.APICategory,
	}

	// Network Settings
	MaxPeersFlag = &cli.IntFlag{
		Name:     "maxpeers",
		Usage:    "Maximum number of network peers (network disabled if set to 0)",
		Value:    node.DefaultConfig.P2P.MaxPeers,
		Category: flags.NetworkingCategory,
	}
	MaxPendingPeersFlag = &cli.IntFlag{
		Name:     "maxpendpeers",
		Usage:    "Maximum number of pending connection attempts (defaults used if set to 0)",
		Value:    node.DefaultConfig.P2P.MaxPendingPeers,
		Category: flags.NetworkingCategory,
	}
	ListenPortFlag = &cli.IntFlag{
		Name:     "port",
		Usage:    "Network listening port",
		Value:    30303,
		Category: flags.NetworkingCategory,
	}
	BootnodesFlag = &cli.StringFlag{
		Name:     "bootnodes",
		Usage:    "Comma separated enode URLs for P2P discovery bootstrap (set v4+v5 instead for light servers)",
		Value:    "",
		Category: flags.NetworkingCategory,
	}
	BootnodesV4Flag = &cli.StringFlag{
		Name:     "bootnodesv4",
		Usage:    "Comma separated enode URLs for P2P v4 discovery bootstrap (light server, full nodes)",
		Value:    "",
		Category: flags.NetworkingCategory,
	}
	BootnodesV5Flag = &cli.StringFlag{
		Name:     "bootnodesv5",
		Usage:    "Comma separated enode URLs for P2P v5 discovery bootstrap (light server, light nodes)",
		Value:    "",
		Category: flags.NetworkingCategory,
	}
	NodeKeyFileFlag = &cli.StringFlag{
		Name:     "nodekey",
		Usage:    "P2P node key file",
		Category: flags.NetworkingCategory,
	}
	NodeKeyHexFlag = &cli.StringFlag{
		Name:     "nodekeyhex",
		Usage:    "P2P node key as hex (for testing)",
		Category: flags.NetworkingCategory,
	}
	NATFlag = &cli.StringFlag{
		Name:     "nat",
		Usage:    "NAT port mapping mechanism (any|none|upnp|pmp|extip:<IP>)",
		Value:    "any",
		Category: flags.NetworkingCategory,
	}
	NoDiscoverFlag = &cli.BoolFlag{
		Name:     "nodiscover",
		Usage:    "Disables the peer discovery mechanism (manual peer addition)",
		Category: flags.NetworkingCategory,
	}
	DiscoveryV5Flag = &cli.BoolFlag{
		Name:     "v5disc",
		Usage:    "Enables the experimental RLPx V5 (Topic Discovery) mechanism",
		Category: flags.NetworkingCategory,
	}
	NetrestrictFlag = &cli.StringFlag{
		Name:     "netrestrict",
		Usage:    "Restricts network communication to the given IP networks (CIDR masks)",
		Category: flags.NetworkingCategory,
	}

	// Console
	JSpathFlag = &cli.StringFlag{
		Name:     "jspath",
		Usage:    "JavaScript root path for `loadScript`",
		Value:    ".",
		Category: flags.APICategory,
	}

	// Gas price oracle settings
	GpoBlocksFlag = &cli.IntFlag{
		Name:     "gpo-blocks",
		Aliases:  []string{"gpoblocks"},
		Usage:    "Number of recent blocks to check for gas prices",
		Value:    ethconfig.Defaults.GPO.Blocks,
		Category: flags.GasPriceCategory,
	}
	GpoPercentileFlag = &cli.IntFlag{
		Name:     "gpo-percentile",
		Aliases:  []string{"gpopercentile"},
		Usage:    "Suggested gas price is the given percentile of a set of recent transaction gas prices",
		Value:    ethconfig.Defaults.GPO.Percentile,
		Category: flags.GasPriceCategory,
	}
	GpoMaxGasPriceFlag = &cli.Int64Flag{
		Name:     "gpo-maxprice",
		Aliases:  []string{"gpo.maxprice"},
		Usage:    "Maximum gas price will be recommended by gpo",
		Value:    ethconfig.Defaults.GPO.MaxPrice.Int64(),
		Category: flags.GasPriceCategory,
	}
	GpoIgnoreGasPriceFlag = &cli.Int64Flag{
		Name:     "gpo-ignoreprice",
		Aliases:  []string{"gpo.ignoreprice"},
		Usage:    "Gas price below which gpo will ignore transactions",
		Value:    ethconfig.Defaults.GPO.IgnorePrice.Int64(),
		Category: flags.GasPriceCategory,
	}

	// Metrics flags
	MetricsEnabledFlag = &cli.BoolFlag{
		Name:     "metrics",
		Usage:    "Enable metrics collection and reporting",
		Category: flags.MetricsCategory,
	}
	// MetricsHTTPFlag defines the endpoint for a stand-alone metrics HTTP endpoint.
	// Since the pprof service enables sensitive/vulnerable behavior, this allows a user
	// to enable a public-OK metrics endpoint without having to worry about ALSO exposing
	// other profiling behavior or information.
	MetricsHTTPFlag = &cli.StringFlag{
		Name:     "metrics-addr",
		Aliases:  []string{"metrics.addr"},
		Usage:    "Enable stand-alone metrics HTTP server listening interface",
		Value:    metrics.DefaultConfig.HTTP,
		Category: flags.MetricsCategory,
	}
	MetricsPortFlag = &cli.IntFlag{
		Name:     "metrics-port",
		Aliases:  []string{"metrics.port"},
		Usage:    "Metrics HTTP server listening port",
		Value:    metrics.DefaultConfig.Port,
		Category: flags.MetricsCategory,
	}
	MetricsEnableInfluxDBFlag = &cli.BoolFlag{
		Name:     "metrics-influxdb",
		Usage:    "Enable metrics export/push to an external InfluxDB database",
		Category: flags.MetricsCategory,
	}
	MetricsInfluxDBEndpointFlag = &cli.StringFlag{
		Name:     "metrics-influxdb.endpoint",
		Usage:    "InfluxDB API endpoint to report metrics to",
		Value:    metrics.DefaultConfig.InfluxDBEndpoint,
		Category: flags.MetricsCategory,
	}
	MetricsInfluxDBDatabaseFlag = &cli.StringFlag{
		Name:     "metrics-influxdb.database",
		Usage:    "InfluxDB database name to push reported metrics to",
		Value:    metrics.DefaultConfig.InfluxDBDatabase,
		Category: flags.MetricsCategory,
	}
	MetricsInfluxDBUsernameFlag = &cli.StringFlag{
		Name:     "metrics-influxdb.username",
		Usage:    "Username to authorize access to the database",
		Value:    metrics.DefaultConfig.InfluxDBUsername,
		Category: flags.MetricsCategory,
	}
	MetricsInfluxDBPasswordFlag = &cli.StringFlag{
		Name:     "metrics-influxdb.password",
		Usage:    "Password to authorize access to the database",
		Value:    metrics.DefaultConfig.InfluxDBPassword,
		Category: flags.MetricsCategory,
	}
	// Tags are part of every measurement sent to InfluxDB. Queries on tags are faster in InfluxDB.
	// For example `host` tag could be used so that we can group all nodes and average a measurement
	// across all of them, but also so that we can select a specific node and inspect its measurements.
	// https://docs.influxdata.com/influxdb/v1.4/concepts/key_concepts/#tag-key
	MetricsInfluxDBTagsFlag = &cli.StringFlag{
		Name:     "metrics-influxdb.tags",
		Usage:    "Comma-separated InfluxDB tags (key/values) attached to all measurements",
		Value:    metrics.DefaultConfig.InfluxDBTags,
		Category: flags.MetricsCategory,
	}

	MetricsEnableInfluxDBV2Flag = &cli.BoolFlag{
		Name:     "metrics-influxdbv2",
		Usage:    "Enable metrics export/push to an external InfluxDB v2 database",
		Category: flags.MetricsCategory,
	}
	MetricsInfluxDBTokenFlag = &cli.StringFlag{
		Name:     "metrics-influxdb.token",
		Usage:    "Token to authorize access to the database (v2 only)",
		Value:    metrics.DefaultConfig.InfluxDBToken,
		Category: flags.MetricsCategory,
	}
	MetricsInfluxDBBucketFlag = &cli.StringFlag{
		Name:     "metrics-influxdb.bucket",
		Usage:    "InfluxDB bucket name to push reported metrics to (v2 only)",
		Value:    metrics.DefaultConfig.InfluxDBBucket,
		Category: flags.MetricsCategory,
	}
	MetricsInfluxDBOrganizationFlag = &cli.StringFlag{
		Name:     "metrics-influxdb.organization",
		Usage:    "InfluxDB organization name (v2 only)",
		Value:    metrics.DefaultConfig.InfluxDBOrganization,
		Category: flags.MetricsCategory,
	}

	// MISC settings
	RollbackFlag = &cli.StringFlag{
		Name:     "rollback",
		Usage:    "Rollback chain at hash",
		Value:    "",
		Category: flags.MiscCategory,
	}
	AnnounceTxsFlag = &cli.BoolFlag{
		Name:     "announce-txs",
		Usage:    "Always commit transactions",
		Value:    false,
		Category: flags.MiscCategory,
	}
	StoreRewardFlag = &cli.BoolFlag{
		Name:     "store-reward",
		Usage:    "Store reward to file",
		Value:    false,
		Category: flags.MiscCategory,
	}
	RewoundFlag = &cli.IntFlag{
		Name:     "rewound",
		Usage:    "Rewound blocks",
		Value:    0,
		Category: flags.MiscCategory,
	}

	// XDC settings
	Enable0xPrefixFlag = &cli.BoolFlag{
		Name:     "enable-0x-prefix",
		Usage:    "Addres use 0x-prefix (Deprecated: this is on by default, to use xdc prefix use --enable-xdc-prefix)",
		Value:    true,
		Category: flags.XdcCategory,
	}
	EnableXDCPrefixFlag = &cli.BoolFlag{
		Name:     "enable-xdc-prefix",
		Usage:    "Addres use xdc-prefix (default = false)",
		Value:    false,
		Category: flags.XdcCategory,
	}
	XDCSlaveModeFlag = &cli.BoolFlag{
		Name:     "slave",
		Usage:    "Enable slave mode",
		Category: flags.XdcCategory,
	}

	// XDCX settings
	XDCXEnabledFlag = &cli.BoolFlag{
		Name:     "XDCx",
		Usage:    "Enable the XDCX protocol",
		Category: flags.XdcxCategory,
	}
	XDCXDBEngineFlag = &cli.StringFlag{
		Name:     "XDCx-dbengine",
		Aliases:  []string{"XDCx.dbengine"},
		Usage:    "Database engine for XDCX (leveldb, mongodb)",
		Value:    "leveldb",
		Category: flags.XdcxCategory,
	}
	XDCXDBNameFlag = &cli.StringFlag{
		Name:     "XDCx-dbName",
		Aliases:  []string{"XDCx.dbName"},
		Usage:    "Database name for XDCX",
		Value:    "XDCdex",
		Category: flags.XdcxCategory,
	}
	XDCXDBConnectionUrlFlag = &cli.StringFlag{
		Name:     "XDCx-dbConnectionUrl",
		Aliases:  []string{"XDCx.dbConnectionUrl"},
		Usage:    "ConnectionUrl to database if dbEngine is mongodb. Host:port. If there are multiple instances, separated by comma. Eg: localhost:27017,localhost:27018",
		Value:    "localhost:27017",
		Category: flags.XdcxCategory,
	}
	XDCXDBReplicaSetNameFlag = &cli.StringFlag{
		Name:     "XDCx-dbReplicaSetName",
		Aliases:  []string{"XDCx.dbReplicaSetName"},
		Usage:    "ReplicaSetName if Master-Slave is setup",
		Category: flags.XdcxCategory,
	}
)

// MakeDataDir retrieves the currently requested data directory, terminating
// if none (or the empty string) is specified. If the node is starting a testnet,
// the a subdirectory of the specified datadir will be used.
func MakeDataDir(ctx *cli.Context) string {
	if path := ctx.String(DataDirFlag.Name); path != "" {
		if ctx.Bool(TestnetFlag.Name) {
			return filepath.Join(path, "testnet")
		}
		if ctx.Bool(RinkebyFlag.Name) {
			return filepath.Join(path, "rinkeby")
		}
		return path
	}
	Fatalf("Cannot determine default data directory, please set manually (--datadir)")
	return ""
}

// setNodeKey creates a node key from set command line flags, either loading it
// from a file or as a specified hex value. If neither flags were provided, this
// method returns nil and an emphemeral key is to be generated.
func setNodeKey(ctx *cli.Context, cfg *p2p.Config) {
	var (
		hex  = ctx.String(NodeKeyHexFlag.Name)
		file = ctx.String(NodeKeyFileFlag.Name)
		key  *ecdsa.PrivateKey
		err  error
	)
	switch {
	case file != "" && hex != "":
		Fatalf("Options %q and %q are mutually exclusive", NodeKeyFileFlag.Name, NodeKeyHexFlag.Name)
	case file != "":
		if key, err = crypto.LoadECDSA(file); err != nil {
			Fatalf("Option %q: %v", NodeKeyFileFlag.Name, err)
		}
		cfg.PrivateKey = key
	case hex != "":
		if key, err = crypto.HexToECDSA(hex); err != nil {
			Fatalf("Option %q: %v", NodeKeyHexFlag.Name, err)
		}
		cfg.PrivateKey = key
	}
}

// setNodeUserIdent creates the user identifier from CLI flags.
func setNodeUserIdent(ctx *cli.Context, cfg *node.Config) {
	if identity := ctx.String(IdentityFlag.Name); len(identity) > 0 {
		cfg.UserIdent = identity
	}
}

// setBootstrapNodes creates a list of bootstrap nodes from the command line
// flags, reverting to pre-configured ones if none have been specified.
// Priority order for bootnodes configuration:
//
// 1. --bootnodes flag
// 2. Config file
// 3. Network preset flags (e.g. --holesky)
// 4. default to mainnet nodes
func setBootstrapNodes(ctx *cli.Context, cfg *p2p.Config) {
	urls := params.MainnetBootnodes
	if ctx.IsSet(BootnodesFlag.Name) {
		urls = SplitAndTrim(ctx.String(BootnodesFlag.Name))
	} else if ctx.IsSet(BootnodesV4Flag.Name) {
		urls = SplitAndTrim(ctx.String(BootnodesV4Flag.Name))
	} else {
		if cfg.BootstrapNodes != nil {
			return // Already set by config file, don't apply defaults.
		}
		networkID := uint64(0)
		if ctx.IsSet(NetworkIdFlag.Name) {
			networkID = ctx.Uint64(NetworkIdFlag.Name)
		}
		switch {
		case ctx.Bool(XDCTestnetFlag.Name) || networkID == params.TestnetChainConfig.ChainId.Uint64():
			urls = params.TestnetBootnodes
		case networkID == params.DevnetChainConfig.ChainId.Uint64():
			urls = params.DevnetBootnodes
		}
	}
	cfg.BootstrapNodes = mustParseBootnodes(urls)
}

func mustParseBootnodes(urls []string) []*discover.Node {
	nodes := make([]*discover.Node, 0, len(urls))
	for _, url := range urls {
		if url != "" {
			node, err := discover.ParseNode(url)
			if err != nil {
				log.Crit("Bootstrap URL invalid", "enode", url, "err", err)
				return nil
			}
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// setBootstrapNodesV5 creates a list of bootstrap nodes from the command line
// flags, reverting to pre-configured ones if none have been specified.
func setBootstrapNodesV5(ctx *cli.Context, cfg *p2p.Config) {
	urls := params.DiscoveryV5Bootnodes
	switch {
	case ctx.IsSet(BootnodesFlag.Name):
		urls = SplitAndTrim(ctx.String(BootnodesFlag.Name))
	case ctx.IsSet(BootnodesV5Flag.Name):
		urls = SplitAndTrim(ctx.String(BootnodesV5Flag.Name))
	case ctx.Bool(XDCTestnetFlag.Name):
		urls = params.TestnetBootnodes
	case cfg.BootstrapNodesV5 != nil:
		return // already set, don't apply defaults.
	}

	cfg.BootstrapNodesV5 = make([]*discv5.Node, 0, len(urls))
	for _, url := range urls {
		if url != "" {
			node, err := discv5.ParseNode(url)
			if err != nil {
				log.Error("Bootstrap URL invalid", "enode", url, "err", err)
				continue
			}
			cfg.BootstrapNodesV5 = append(cfg.BootstrapNodesV5, node)
		}
	}
}

// setListenAddress creates a TCP listening address string from set command
// line flags.
func setListenAddress(ctx *cli.Context, cfg *p2p.Config) {
	if ctx.IsSet(ListenPortFlag.Name) {
		cfg.ListenAddr = fmt.Sprintf(":%d", ctx.Int(ListenPortFlag.Name))
	}
}

// setNAT creates a port mapper from command line flags.
func setNAT(ctx *cli.Context, cfg *p2p.Config) {
	if ctx.IsSet(NATFlag.Name) {
		log.Info("NAT is setted", "value", ctx.String(NATFlag.Name))
		natif, err := nat.Parse(ctx.String(NATFlag.Name))
		if err != nil {
			Fatalf("Option %s: %v", NATFlag.Name, err)
		}
		cfg.NAT = natif
	}
}

// SplitAndTrim splits input separated by a comma
// and trims excessive white space from the substrings.
func SplitAndTrim(input string) (ret []string) {
	l := strings.Split(input, ",")
	for _, r := range l {
		if r = strings.TrimSpace(r); r != "" {
			ret = append(ret, r)
		}
	}
	return ret
}

// setHTTP creates the HTTP RPC listener interface string from the set
// command line flags, returning empty if the HTTP endpoint is disabled.
func setHTTP(ctx *cli.Context, cfg *node.Config) {
	if ctx.Bool(HTTPEnabledFlag.Name) {
		if cfg.HTTPHost == "" {
			cfg.HTTPHost = "127.0.0.1"
		}
		cfg.HTTPHost = ctx.String(HTTPListenAddrFlag.Name)
	}

	if ctx.IsSet(HTTPPortFlag.Name) {
		cfg.HTTPPort = ctx.Int(HTTPPortFlag.Name)
	}
	if ctx.IsSet(HTTPReadTimeoutFlag.Name) {
		cfg.HTTPTimeouts.ReadTimeout = ctx.Duration(HTTPReadTimeoutFlag.Name)
	}
	if ctx.IsSet(HTTPWriteTimeoutFlag.Name) {
		cfg.HTTPTimeouts.WriteTimeout = ctx.Duration(HTTPWriteTimeoutFlag.Name)
	}
	if ctx.IsSet(HTTPIdleTimeoutFlag.Name) {
		cfg.HTTPTimeouts.IdleTimeout = ctx.Duration(HTTPIdleTimeoutFlag.Name)
	}
	cfg.HTTPCors = SplitAndTrim(ctx.String(HTTPCORSDomainFlag.Name))
	cfg.HTTPModules = SplitAndTrim(ctx.String(HTTPApiFlag.Name))
	cfg.HTTPVirtualHosts = SplitAndTrim(ctx.String(HTTPVirtualHostsFlag.Name))
}

// setWS creates the WebSocket RPC listener interface string from the set
// command line flags, returning empty if the HTTP endpoint is disabled.
func setWS(ctx *cli.Context, cfg *node.Config) {
	if ctx.Bool(WSEnabledFlag.Name) {
		if cfg.WSHost == "" {
			cfg.WSHost = "127.0.0.1"
		}
		cfg.WSHost = ctx.String(WSListenAddrFlag.Name)
	}

	if ctx.IsSet(WSPortFlag.Name) {
		cfg.WSPort = ctx.Int(WSPortFlag.Name)
	}
	cfg.WSOrigins = SplitAndTrim(ctx.String(WSAllowedOriginsFlag.Name))
	cfg.WSModules = SplitAndTrim(ctx.String(WSApiFlag.Name))
}

// setIPC creates an IPC path configuration from the set command line flags,
// returning an empty string if IPC was explicitly disabled, or the set path.
func setIPC(ctx *cli.Context, cfg *node.Config) {
	CheckExclusive(ctx, IPCDisabledFlag, IPCPathFlag)
	switch {
	case ctx.Bool(IPCDisabledFlag.Name):
		cfg.IPCPath = ""
	case ctx.IsSet(IPCPathFlag.Name):
		cfg.IPCPath = ctx.String(IPCPathFlag.Name)
	}
}

func setPrefix(ctx *cli.Context, cfg *node.Config) {
	CheckExclusive(ctx, Enable0xPrefixFlag, EnableXDCPrefixFlag)
}

// MakeDatabaseHandles raises out the number of allowed file handles per process
// for XDC and returns half of the allowance to assign to the database.
func MakeDatabaseHandles(max int) int {
	limit, err := fdlimit.Maximum()
	if err != nil {
		Fatalf("Failed to retrieve file descriptor allowance: %v", err)
	}
	switch {
	case max == 0:
		// User didn't specify a meaningful value, use system limits
	case max < 128:
		// User specified something unhealthy, just use system defaults
		log.Error("File descriptor limit invalid (<128)", "had", max, "updated", limit)
	case max > limit:
		// User requested more than the OS allows, notify that we can't allocate it
		log.Warn("Requested file descriptors denied by OS", "req", max, "limit", limit)
	default:
		// User limit is meaningful and within allowed range, use that
		limit = max
	}
	raised, err := fdlimit.Raise(uint64(limit))
	if err != nil {
		Fatalf("Failed to raise file descriptor allowance: %v", err)
	}
	return int(raised / 2) // Leave half for networking and other stuff
}

// MakeAddress converts an account specified directly as a hex encoded string or
// a key index in the key store to an internal account representation.
func MakeAddress(ks *keystore.KeyStore, account string) (accounts.Account, error) {
	// If the specified account is a valid address, return it
	if common.IsHexAddress(account) {
		return accounts.Account{Address: common.HexToAddress(account)}, nil
	}
	// Otherwise try to interpret the account as a keystore index
	index, err := strconv.Atoi(account)
	if err != nil || index < 0 {
		return accounts.Account{}, fmt.Errorf("invalid account address or index %q", account)
	}
	log.Warn("-------------------------------------------------------------------")
	log.Warn("Referring to accounts by order in the keystore folder is dangerous!")
	log.Warn("This functionality is deprecated and will be removed in the future!")
	log.Warn("Please use explicit addresses! (can search via `XDC account list`)")
	log.Warn("-------------------------------------------------------------------")

	accs := ks.Accounts()
	if len(accs) <= index {
		return accounts.Account{}, fmt.Errorf("index %d higher than number of accounts %d", index, len(accs))
	}
	return accs[index], nil
}

// setEtherbase retrieves the etherbase either from the directly specified
// command line flags or from the keystore if CLI indexed.
func setEtherbase(ctx *cli.Context, ks *keystore.KeyStore, cfg *ethconfig.Config) {
	if ctx.IsSet(MinerEtherbaseFlag.Name) {
		account, err := MakeAddress(ks, ctx.String(MinerEtherbaseFlag.Name))
		if err != nil {
			Fatalf("Option %q: %v", MinerEtherbaseFlag.Name, err)
		}
		cfg.Etherbase = account.Address
	} else {
		cfg.Etherbase = common.HexToAddress(ctx.String(MinerEtherbaseFlag.Name))
	}
}

// MakePasswordList reads password lines from the file specified by the global --password flag.
func MakePasswordList(ctx *cli.Context) []string {
	path := ctx.String(PasswordFileFlag.Name)
	if path == "" {
		return nil
	}
	text, err := os.ReadFile(path)
	if err != nil {
		Fatalf("Failed to read password file: %v", err)
	}
	lines := strings.Split(string(text), "\n")
	// Sanitise DOS line endings.
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}
	return lines
}

func SetP2PConfig(ctx *cli.Context, cfg *p2p.Config) {
	setNodeKey(ctx, cfg)
	setNAT(ctx, cfg)
	setListenAddress(ctx, cfg)
	setBootstrapNodes(ctx, cfg)
	// setBootstrapNodesV5(ctx, cfg)

	lightClient := ctx.Bool(LightModeFlag.Name) || ctx.String(SyncModeFlag.Name) == "light"
	lightServer := ctx.Int(LightServFlag.Name) != 0
	lightPeers := ctx.Int(LightPeersFlag.Name)

	if ctx.IsSet(MaxPeersFlag.Name) {
		cfg.MaxPeers = ctx.Int(MaxPeersFlag.Name)
		if lightServer && !ctx.IsSet(LightPeersFlag.Name) {
			cfg.MaxPeers += lightPeers
		}
	} else {
		if lightServer {
			cfg.MaxPeers += lightPeers
		}
		if lightClient && ctx.IsSet(LightPeersFlag.Name) && cfg.MaxPeers < lightPeers {
			cfg.MaxPeers = lightPeers
		}
	}
	if !(lightClient || lightServer) {
		lightPeers = 0
	}
	ethPeers := cfg.MaxPeers - lightPeers
	if lightClient {
		ethPeers = 0
	}
	log.Info("Maximum peer count", "ETH", ethPeers, "LES", lightPeers, "total", cfg.MaxPeers)

	if ctx.IsSet(MaxPendingPeersFlag.Name) {
		cfg.MaxPendingPeers = ctx.Int(MaxPendingPeersFlag.Name)
	}
	if ctx.IsSet(NoDiscoverFlag.Name) || lightClient {
		cfg.NoDiscovery = true
	}

	// if we're running a light client or server, force enable the v5 peer discovery
	// unless it is explicitly disabled with --nodiscover note that explicitly specifying
	// --v5disc overrides --nodiscover, in which case the later only disables v4 discovery
	forceV5Discovery := (lightClient || lightServer) && !ctx.Bool(NoDiscoverFlag.Name)
	if ctx.IsSet(DiscoveryV5Flag.Name) {
		cfg.DiscoveryV5 = ctx.Bool(DiscoveryV5Flag.Name)
	} else if forceV5Discovery {
		cfg.DiscoveryV5 = true
	}

	if netrestrict := ctx.String(NetrestrictFlag.Name); netrestrict != "" {
		list, err := netutil.ParseNetlist(netrestrict)
		if err != nil {
			Fatalf("Option %q: %v", NetrestrictFlag.Name, err)
		}
		cfg.NetRestrict = list
	}

	if ctx.Bool(DeveloperFlag.Name) {
		// --dev mode can't use p2p networking.
		cfg.MaxPeers = 0
		cfg.ListenAddr = ":0"
		cfg.NoDiscovery = true
		cfg.DiscoveryV5 = false
	}
}

// SetNodeConfig applies node-related command line flags to the config.
func SetNodeConfig(ctx *cli.Context, cfg *node.Config) {
	SetP2PConfig(ctx, &cfg.P2P)
	setIPC(ctx, cfg)
	setHTTP(ctx, cfg)
	setWS(ctx, cfg)
	setNodeUserIdent(ctx, cfg)
	setPrefix(ctx, cfg)
	setSmartCard(ctx, cfg)

	switch {
	case ctx.IsSet(DataDirFlag.Name):
		cfg.DataDir = ctx.String(DataDirFlag.Name)
	case ctx.Bool(DeveloperFlag.Name):
		cfg.DataDir = "" // unless explicitly requested, use memory databases
	case ctx.Bool(TestnetFlag.Name):
		cfg.DataDir = filepath.Join(node.DefaultDataDir(), "testnet")
	case ctx.Bool(RinkebyFlag.Name):
		cfg.DataDir = filepath.Join(node.DefaultDataDir(), "rinkeby")
	}

	if ctx.IsSet(KeyStoreDirFlag.Name) {
		cfg.KeyStoreDir = ctx.String(KeyStoreDirFlag.Name)
	}
	if ctx.IsSet(LightKDFFlag.Name) {
		cfg.UseLightweightKDF = ctx.Bool(LightKDFFlag.Name)
	}
	if ctx.IsSet(NoUSBFlag.Name) || cfg.NoUSB {
		log.Warn("Option nousb is deprecated and USB is deactivated by default. Use --usb to enable")
	}
	if ctx.IsSet(USBFlag.Name) {
		cfg.USB = ctx.Bool(USBFlag.Name)
	}
	if ctx.IsSet(AnnounceTxsFlag.Name) {
		cfg.AnnounceTxs = ctx.Bool(AnnounceTxsFlag.Name)
	}
	// deprecation notice for log debug flags (TODO: find a more appropriate place to put these?)
	if ctx.IsSet(LogBacktraceAtFlag.Name) {
		log.Warn("log.backtrace flag is deprecated")
	}
	if ctx.IsSet(LogDebugFlag.Name) {
		log.Warn("log.debug flag is deprecated")
	}
}

func setSmartCard(ctx *cli.Context, cfg *node.Config) {
	// Skip enabling smartcards if no path is set
	path := ctx.String(SmartCardDaemonPathFlag.Name)
	if path == "" {
		return
	}
	// Sanity check that the smartcard path is valid
	fi, err := os.Stat(path)
	if err != nil {
		log.Info("Smartcard socket not found, disabling", "err", err)
		return
	}
	if fi.Mode()&os.ModeType != os.ModeSocket {
		log.Error("Invalid smartcard daemon path", "path", path, "type", fi.Mode().String())
		return
	}
	// Smartcard daemon path exists and is a socket, enable it
	cfg.SmartCardDaemonPath = path
}

func setGPO(ctx *cli.Context, cfg *gasprice.Config, light bool) {
	// If we are running the light client, apply another group
	// settings for gas oracle.
	if light {
		*cfg = ethconfig.LightClientGPO
	}
	if ctx.IsSet(GpoBlocksFlag.Name) {
		cfg.Blocks = ctx.Int(GpoBlocksFlag.Name)
	}
	if ctx.IsSet(GpoPercentileFlag.Name) {
		cfg.Percentile = ctx.Int(GpoPercentileFlag.Name)
	}
	if ctx.IsSet(GpoMaxGasPriceFlag.Name) {
		cfg.MaxPrice = big.NewInt(ctx.Int64(GpoMaxGasPriceFlag.Name))
	}
	if ctx.IsSet(GpoIgnoreGasPriceFlag.Name) {
		cfg.IgnorePrice = big.NewInt(ctx.Int64(GpoIgnoreGasPriceFlag.Name))
	}
}

func setTxPool(ctx *cli.Context, cfg *txpool.Config) {
	if ctx.IsSet(TxPoolNoLocalsFlag.Name) {
		cfg.NoLocals = ctx.Bool(TxPoolNoLocalsFlag.Name)
	}
	if ctx.IsSet(TxPoolJournalFlag.Name) {
		cfg.Journal = ctx.String(TxPoolJournalFlag.Name)
	}
	if ctx.IsSet(TxPoolRejournalFlag.Name) {
		cfg.Rejournal = ctx.Duration(TxPoolRejournalFlag.Name)
	}
	if ctx.IsSet(TxPoolPriceLimitFlag.Name) {
		cfg.PriceLimit = ctx.Uint64(TxPoolPriceLimitFlag.Name)
	}
	if ctx.IsSet(TxPoolPriceBumpFlag.Name) {
		cfg.PriceBump = ctx.Uint64(TxPoolPriceBumpFlag.Name)
	}
	if ctx.IsSet(TxPoolAccountSlotsFlag.Name) {
		cfg.AccountSlots = ctx.Uint64(TxPoolAccountSlotsFlag.Name)
	}
	if ctx.IsSet(TxPoolGlobalSlotsFlag.Name) {
		cfg.GlobalSlots = ctx.Uint64(TxPoolGlobalSlotsFlag.Name)
	}
	if ctx.IsSet(TxPoolAccountQueueFlag.Name) {
		cfg.AccountQueue = ctx.Uint64(TxPoolAccountQueueFlag.Name)
	}
	if ctx.IsSet(TxPoolGlobalQueueFlag.Name) {
		cfg.GlobalQueue = ctx.Uint64(TxPoolGlobalQueueFlag.Name)
	}
	if ctx.IsSet(TxPoolLifetimeFlag.Name) {
		cfg.Lifetime = ctx.Duration(TxPoolLifetimeFlag.Name)
	}
}

func setEthash(ctx *cli.Context, cfg *ethconfig.Config) {
	if ctx.IsSet(EthashCacheDirFlag.Name) {
		cfg.Ethash.CacheDir = ctx.String(EthashCacheDirFlag.Name)
	}
	if ctx.IsSet(EthashDatasetDirFlag.Name) {
		cfg.Ethash.DatasetDir = ctx.String(EthashDatasetDirFlag.Name)
	}
	if ctx.IsSet(EthashCachesInMemoryFlag.Name) {
		cfg.Ethash.CachesInMem = ctx.Int(EthashCachesInMemoryFlag.Name)
	}
	if ctx.IsSet(EthashCachesOnDiskFlag.Name) {
		cfg.Ethash.CachesOnDisk = ctx.Int(EthashCachesOnDiskFlag.Name)
	}
	if ctx.IsSet(EthashDatasetsInMemoryFlag.Name) {
		cfg.Ethash.DatasetsInMem = ctx.Int(EthashDatasetsInMemoryFlag.Name)
	}
	if ctx.IsSet(EthashDatasetsOnDiskFlag.Name) {
		cfg.Ethash.DatasetsOnDisk = ctx.Int(EthashDatasetsOnDiskFlag.Name)
	}
}

// CheckExclusive verifies that only a single isntance of the provided flags was
// set by the user. Each flag might optionally be followed by a string type to
// specialize it further.
func CheckExclusive(ctx *cli.Context, args ...interface{}) {
	set := make([]string, 0, 1)
	for i := 0; i < len(args); i++ {
		// Make sure the next argument is a flag and skip if not set
		flag, ok := args[i].(cli.Flag)
		if !ok {
			panic(fmt.Sprintf("invalid argument, not cli.Flag type: %T", args[i]))
		}
		// Check if next arg extends current and expand its name if so
		name := flag.Names()[0]

		if i+1 < len(args) {
			switch option := args[i+1].(type) {
			case string:
				// Extended flag check, make sure value set doesn't conflict with passed in option
				if ctx.String(flag.Names()[0]) == option {
					name += "=" + option
					set = append(set, "--"+name)
				}
				// shift arguments and continue
				i++
				continue

			case cli.Flag:
			default:
				panic(fmt.Sprintf("invalid argument, not cli.Flag or string extension: %T", args[i+1]))
			}
		}
		// Mark the flag if it's set
		if ctx.IsSet(flag.Names()[0]) {
			set = append(set, "--"+name)
		}
	}
	if len(set) > 1 {
		Fatalf("Flags %v can't be used at the same time", strings.Join(set, ", "))
	}
}

func SetXDCXConfig(ctx *cli.Context, cfg *XDCx.Config, XDCDataDir string) {
	if ctx.IsSet(XDCXDataDirFlag.Name) {
		log.Warn("The flag XDCx-datadir or XDCx.datadir is deprecated, please remove this flag")
	}
	// XDCx datadir: XDCDataDir/XDCx
	cfg.DataDir = filepath.Join(XDCDataDir, "XDCx")
	log.Info("XDCX datadir", "path", cfg.DataDir)
	if ctx.IsSet(XDCXDBEngineFlag.Name) {
		cfg.DBEngine = ctx.String(XDCXDBEngineFlag.Name)
	} else {
		cfg.DBEngine = XDCXDBEngineFlag.Value
	}
	if ctx.IsSet(XDCXDBNameFlag.Name) {
		cfg.DBName = ctx.String(XDCXDBNameFlag.Name)
	} else {
		cfg.DBName = XDCXDBNameFlag.Value
	}
	if ctx.IsSet(XDCXDBConnectionUrlFlag.Name) {
		cfg.ConnectionUrl = ctx.String(XDCXDBConnectionUrlFlag.Name)
	} else {
		cfg.ConnectionUrl = XDCXDBConnectionUrlFlag.Value
	}
	if ctx.IsSet(XDCXDBReplicaSetNameFlag.Name) {
		cfg.ReplicaSetName = ctx.String(XDCXDBReplicaSetNameFlag.Name)
	}
}

// SetEthConfig applies eth-related command line flags to the config.
func SetEthConfig(ctx *cli.Context, stack *node.Node, cfg *ethconfig.Config) {
	// Avoid conflicting network flags
	CheckExclusive(ctx, DeveloperFlag, TestnetFlag, RinkebyFlag)
	CheckExclusive(ctx, FastSyncFlag, LightModeFlag, SyncModeFlag)
	CheckExclusive(ctx, LightServFlag, LightModeFlag)
	CheckExclusive(ctx, LightServFlag, SyncModeFlag, "light")

	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	setEtherbase(ctx, ks, cfg)
	setGPO(ctx, &cfg.GPO, ctx.String(SyncModeFlag.Name) == "light")
	setTxPool(ctx, &cfg.TxPool)
	setEthash(ctx, cfg)

	// Cap the cache allowance and tune the garbage collector
	mem, err := gopsutil.VirtualMemory()
	if err == nil {
		if 32<<(^uintptr(0)>>63) == 32 && mem.Total > 2*1024*1024*1024 {
			log.Warn("Lowering memory allowance on 32bit arch", "available", mem.Total/1024/1024, "addressable", 2*1024)
			mem.Total = 2 * 1024 * 1024 * 1024
		}
		allowance := int(mem.Total / 1024 / 1024 / 3)
		if cache := ctx.Int(CacheFlag.Name); cache > allowance {
			log.Warn("Sanitizing cache to Go's GC limits", "provided", cache, "updated", allowance)
			ctx.Set(CacheFlag.Name, strconv.Itoa(allowance))
		}
	}
	// Ensure Go's GC ignores the database cache for trigger percentage
	cache := ctx.Int(CacheFlag.Name)
	gogc := math.Max(20, math.Min(100, 100/(float64(cache)/1024)))

	log.Debug("Sanitizing Go's GC trigger", "percent", int(gogc))
	godebug.SetGCPercent(int(gogc))

	switch {
	case ctx.IsSet(SyncModeFlag.Name):
		if err = cfg.SyncMode.UnmarshalText([]byte(ctx.String(SyncModeFlag.Name))); err != nil {
			Fatalf("invalid --syncmode flag: %v", err)
		}
	case ctx.Bool(FastSyncFlag.Name):
		cfg.SyncMode = downloader.FastSync
	case ctx.Bool(LightModeFlag.Name):
		cfg.SyncMode = downloader.LightSync
	}
	if ctx.IsSet(LightServFlag.Name) {
		cfg.LightServ = ctx.Int(LightServFlag.Name)
	}
	if ctx.IsSet(LightPeersFlag.Name) {
		cfg.LightPeers = ctx.Int(LightPeersFlag.Name)
	}
	if ctx.IsSet(NetworkIdFlag.Name) {
		cfg.NetworkId = ctx.Uint64(NetworkIdFlag.Name)
	}
	if ctx.Bool(XDCTestnetFlag.Name) {
		cfg.NetworkId = 51
	}
	common.CopyConstans(cfg.NetworkId)

	if ctx.IsSet(CacheFlag.Name) || ctx.IsSet(CacheDatabaseFlag.Name) {
		cfg.DatabaseCache = ctx.Int(CacheFlag.Name) * ctx.Int(CacheDatabaseFlag.Name) / 100
	}
	cfg.DatabaseHandles = MakeDatabaseHandles(ctx.Int(FDLimitFlag.Name))

	if gcmode := ctx.String(GCModeFlag.Name); gcmode != "full" && gcmode != "archive" {
		Fatalf("--%s must be either 'full' or 'archive'", GCModeFlag.Name)
	}
	cfg.NoPruning = ctx.String(GCModeFlag.Name) == "archive"

	if ctx.IsSet(CacheFlag.Name) || ctx.IsSet(CacheGCFlag.Name) {
		cfg.TrieCache = ctx.Int(CacheFlag.Name) * ctx.Int(CacheGCFlag.Name) / 100
	}
	if ctx.IsSet(MinerThreadsFlag.Name) {
		cfg.MinerThreads = ctx.Int(MinerThreadsFlag.Name)
	}
	if ctx.IsSet(DocRootFlag.Name) {
		cfg.DocRoot = ctx.String(DocRootFlag.Name)
	}
	if ctx.IsSet(RPCGlobalGasCapFlag.Name) {
		cfg.RPCGasCap = ctx.Uint64(RPCGlobalGasCapFlag.Name)
	}
	if ctx.IsSet(RPCGlobalTxFeeCap.Name) {
		cfg.RPCTxFeeCap = ctx.Float64(RPCGlobalTxFeeCap.Name)
	}
	if ctx.IsSet(RPCGlobalGasCapFlag.Name) {
		cfg.RPCGasCap = ctx.Uint64(RPCGlobalGasCapFlag.Name)
	}
	if ctx.IsSet(MinerExtraDataFlag.Name) {
		cfg.ExtraData = []byte(ctx.String(MinerExtraDataFlag.Name))
	}
	cfg.GasPrice = flags.GlobalBig(ctx, MinerGasPriceFlag.Name)
	if ctx.IsSet(CacheLogSizeFlag.Name) {
		cfg.FilterLogCacheSize = ctx.Int(CacheLogSizeFlag.Name)
	}
	if ctx.IsSet(VMEnableDebugFlag.Name) {
		// TODO(fjl): force-enable this in --dev mode
		cfg.EnablePreimageRecording = ctx.Bool(VMEnableDebugFlag.Name)
	}
	if cfg.RPCGasCap != 0 {
		log.Info("Set global gas cap", "cap", cfg.RPCGasCap)
	} else {
		log.Info("Global gas cap disabled")
	}
	if ctx.IsSet(StoreRewardFlag.Name) {
		common.StoreRewardFolder = filepath.Join(stack.DataDir(), "XDC", "rewards")
		if _, err := os.Stat(common.StoreRewardFolder); os.IsNotExist(err) {
			os.Mkdir(common.StoreRewardFolder, os.ModePerm)
		}
	}
	// Override any default configs for hard coded networks.
	switch {
	case ctx.Bool(TestnetFlag.Name):
		if !ctx.IsSet(NetworkIdFlag.Name) {
			cfg.NetworkId = 3
		}
		cfg.Genesis = core.DefaultTestnetGenesisBlock()
	case ctx.Bool(RinkebyFlag.Name):
		if !ctx.IsSet(NetworkIdFlag.Name) {
			cfg.NetworkId = 4
		}
		cfg.Genesis = core.DefaultRinkebyGenesisBlock()
	case ctx.Bool(DeveloperFlag.Name):
		// Create new developer account or reuse existing one
		var (
			developer accounts.Account
			err       error
		)
		if accs := ks.Accounts(); len(accs) > 0 {
			developer = ks.Accounts()[0]
		} else {
			developer, err = ks.NewAccount("")
			if err != nil {
				Fatalf("Failed to create developer account: %v", err)
			}
		}
		if err := ks.Unlock(developer, ""); err != nil {
			Fatalf("Failed to unlock developer account: %v", err)
		}
		log.Info("Using developer account", "address", developer.Address)

		cfg.Genesis = core.DeveloperGenesisBlock(uint64(ctx.Int(DeveloperPeriodFlag.Name)), developer.Address)
		if !ctx.IsSet(MinerGasPriceFlag.Name) {
			cfg.GasPrice = big.NewInt(1)
		}
	}
	// Set any dangling config values
	if ctx.String(CryptoKZGFlag.Name) != "gokzg" && ctx.String(CryptoKZGFlag.Name) != "ckzg" {
		Fatalf("--%s flag must be 'gokzg' or 'ckzg'", CryptoKZGFlag.Name)
	}
	log.Info("Initializing the KZG library", "backend", ctx.String(CryptoKZGFlag.Name))
	if err := kzg4844.UseCKZG(ctx.String(CryptoKZGFlag.Name) == "ckzg"); err != nil {
		Fatalf("Failed to set KZG library implementation to %s: %v", ctx.String(CryptoKZGFlag.Name), err)
	}
}

// SetupNetwork configures the system for either the main net or some test network.
func SetupNetwork(ctx *cli.Context) {
	// TODO(fjl): move target gas limit into config
	params.TargetGasLimit = ctx.Uint64(MinerGasLimitFlag.Name)
}

// SetupMetrics configures the metrics system.
func SetupMetrics(cfg *metrics.Config) {
	if !cfg.Enabled {
		return
	}
	log.Info("Enabling metrics collection")
	metrics.Enable()

	// InfluxDB exporter.
	var (
		enableExport   = cfg.EnableInfluxDB
		enableExportV2 = cfg.EnableInfluxDBV2
	)
	if cfg.EnableInfluxDB && cfg.EnableInfluxDBV2 {
		Fatalf("Flags %v can't be used at the same time", strings.Join([]string{MetricsEnableInfluxDBFlag.Name, MetricsEnableInfluxDBV2Flag.Name}, ", "))
	}
	var (
		endpoint = cfg.InfluxDBEndpoint
		database = cfg.InfluxDBDatabase
		username = cfg.InfluxDBUsername
		password = cfg.InfluxDBPassword

		token        = cfg.InfluxDBToken
		bucket       = cfg.InfluxDBBucket
		organization = cfg.InfluxDBOrganization
		tagsMap      = SplitTagsFlag(cfg.InfluxDBTags)
	)
	if enableExport {
		log.Info("Enabling metrics export to InfluxDB")
		go influxdb.InfluxDBWithTags(metrics.DefaultRegistry, 10*time.Second, endpoint, database, username, password, "geth.", tagsMap)
	} else if enableExportV2 {
		tagsMap := SplitTagsFlag(cfg.InfluxDBTags)
		log.Info("Enabling metrics export to InfluxDB (v2)")
		go influxdb.InfluxDBV2WithTags(metrics.DefaultRegistry, 10*time.Second, endpoint, token, bucket, organization, "geth.", tagsMap)
	}

	// Expvar exporter.
	if cfg.HTTP != "" {
		address := net.JoinHostPort(cfg.HTTP, fmt.Sprintf("%d", cfg.Port))
		log.Info("Enabling stand-alone metrics HTTP endpoint", "address", address)
		exp.Setup(address)
	} else if cfg.HTTP == "" && cfg.Port != 0 {
		log.Warn(fmt.Sprintf("--%s specified without --%s, metrics server will not start.", MetricsPortFlag.Name, MetricsHTTPFlag.Name))
	}

	// Enable system metrics collection.
	go metrics.CollectProcessMetrics(3 * time.Second)
}

// SplitTagsFlag parses a comma-separated list of k=v metrics tags.
func SplitTagsFlag(tagsFlag string) map[string]string {
	tags := strings.Split(tagsFlag, ",")
	tagsMap := map[string]string{}

	for _, t := range tags {
		if t != "" {
			kv := strings.Split(t, "=")

			if len(kv) == 2 {
				tagsMap[kv[0]] = kv[1]
			}
		}
	}

	return tagsMap
}

// MakeChainDatabase open an LevelDB using the flags passed to the client and will hard crash if it fails.
func MakeChainDatabase(ctx *cli.Context, stack *node.Node, readonly bool) ethdb.Database {
	var (
		cache   = ctx.Int(CacheFlag.Name) * ctx.Int(CacheDatabaseFlag.Name) / 100
		handles = MakeDatabaseHandles(ctx.Int(FDLimitFlag.Name))
	)
	name := "chaindata"
	if ctx.Bool(LightModeFlag.Name) {
		name = "lightchaindata"
	}
	chainDb, err := stack.OpenDatabase(name, cache, handles, "", readonly)
	if err != nil {
		Fatalf("Could not open database: %v", err)
	}
	return chainDb
}

func MakeGenesis(ctx *cli.Context) *core.Genesis {
	var genesis *core.Genesis
	switch {
	case ctx.Bool(TestnetFlag.Name):
		genesis = core.DefaultTestnetGenesisBlock()
	case ctx.Bool(RinkebyFlag.Name):
		genesis = core.DefaultRinkebyGenesisBlock()
	case ctx.Bool(DeveloperFlag.Name):
		Fatalf("Developer chains are ephemeral")
	}
	return genesis
}

// MakeChain creates a chain manager from set command line flags.
func MakeChain(ctx *cli.Context, stack *node.Node, readonly bool) (chain *core.BlockChain, chainDb ethdb.Database) {
	var err error
	chainDb = MakeChainDatabase(ctx, stack, readonly)

	config, _, err := core.SetupGenesisBlock(chainDb, MakeGenesis(ctx))
	if err != nil {
		Fatalf("%v", err)
	}
	var engine consensus.Engine
	if config.XDPoS != nil {
		engine = XDPoS.New(config, chainDb)
	} else {
		engine = ethash.NewFaker()
		if !ctx.Bool(FakePoWFlag.Name) {
			engine = ethash.New(ethash.Config{
				CacheDir:       stack.ResolvePath(ethconfig.Defaults.Ethash.CacheDir),
				CachesInMem:    ethconfig.Defaults.Ethash.CachesInMem,
				CachesOnDisk:   ethconfig.Defaults.Ethash.CachesOnDisk,
				DatasetDir:     stack.ResolvePath(ethconfig.Defaults.Ethash.DatasetDir),
				DatasetsInMem:  ethconfig.Defaults.Ethash.DatasetsInMem,
				DatasetsOnDisk: ethconfig.Defaults.Ethash.DatasetsOnDisk,
			})
		}
		Fatalf("Only support XDPoS consensus")
	}
	if gcmode := ctx.String(GCModeFlag.Name); gcmode != "full" && gcmode != "archive" {
		Fatalf("--%s must be either 'full' or 'archive'", GCModeFlag.Name)
	}
	cache := &core.CacheConfig{
		Disabled:      ctx.String(GCModeFlag.Name) == "archive",
		TrieNodeLimit: ethconfig.Defaults.TrieCache,
		TrieTimeLimit: ethconfig.Defaults.TrieTimeout,
	}
	if ctx.IsSet(CacheFlag.Name) || ctx.IsSet(CacheGCFlag.Name) {
		cache.TrieNodeLimit = ctx.Int(CacheFlag.Name) * ctx.Int(CacheGCFlag.Name) / 100
	}
	vmcfg := vm.Config{EnablePreimageRecording: ctx.Bool(VMEnableDebugFlag.Name)}
	chain, err = core.NewBlockChain(chainDb, cache, config, engine, vmcfg)
	if err != nil {
		Fatalf("Can't create BlockChain: %v", err)
	}
	return chain, chainDb
}

// MakeConsolePreloads retrieves the absolute paths for the console JavaScript
// scripts to preload before starting.
func MakeConsolePreloads(ctx *cli.Context) []string {
	// Skip preloading if there's nothing to preload
	if ctx.String(PreloadJSFlag.Name) == "" {
		return nil
	}
	// Otherwise resolve absolute paths and return them
	preloads := []string{}

	assets := ctx.String(JSpathFlag.Name)
	for _, file := range strings.Split(ctx.String(PreloadJSFlag.Name), ",") {
		preloads = append(preloads, common.AbsolutePath(assets, strings.TrimSpace(file)))
	}
	return preloads
}

// find all filenames match the given pattern in the given root directory
func WalkMatch(root, pattern string) ([]string, error) {
	matches := []string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if matched, err := filepath.Match(pattern, filepath.Base(path)); err != nil {
			return err
		} else if matched {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return matches, nil
}

// RegisterFilterAPI adds the eth log filtering RPC API to the node.
func RegisterFilterAPI(stack *node.Node, backend ethapi.Backend, ethcfg *ethconfig.Config) *filters.FilterSystem {
	isLightClient := ethcfg.SyncMode == downloader.LightSync
	filterSystem := filters.NewFilterSystem(backend, filters.Config{
		LogCacheSize: ethcfg.FilterLogCacheSize,
	})
	stack.RegisterAPIs([]rpc.API{{
		Namespace: "eth",
		Service:   filters.NewFilterAPI(filterSystem, isLightClient),
	}})
	return filterSystem
}
