// Copyright 2018 The go-ethereum Authors
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

// Command feed allows the user to create and update signed Swarm feeds
package main

import cli "gopkg.in/urfave/cli.v1"

var (
	ChequebookAddrFlag = cli.StringFlag{
		Name:   "chequebook",
		Usage:  "chequebook contract address",
		EnvVar: SwarmEnvChequebookAddr,
	}
	SwarmAccountFlag = cli.StringFlag{
		Name:   "bzzaccount",
		Usage:  "Swarm account key file",
		EnvVar: SwarmEnvAccount,
	}
	SwarmListenAddrFlag = cli.StringFlag{
		Name:   "httpaddr",
		Usage:  "Swarm HTTP API listening interface",
		EnvVar: SwarmEnvListenAddr,
	}
	SwarmPortFlag = cli.StringFlag{
		Name:   "bzzport",
		Usage:  "Swarm local http api port",
		EnvVar: SwarmEnvPort,
	}
	SwarmNetworkIdFlag = cli.IntFlag{
		Name:   "bzznetworkid",
		Usage:  "Network identifier (integer, default 3=swarm testnet)",
		EnvVar: SwarmEnvNetworkID,
	}
	SwarmSwapEnabledFlag = cli.BoolFlag{
		Name:   "swap",
		Usage:  "Swarm SWAP enabled (default false)",
		EnvVar: SwarmEnvSwapEnable,
	}
	SwarmSwapAPIFlag = cli.StringFlag{
		Name:   "swap-api",
		Usage:  "URL of the Ethereum API provider to use to settle SWAP payments",
		EnvVar: SwarmEnvSwapAPI,
	}
	SwarmSyncDisabledFlag = cli.BoolTFlag{
		Name:   "nosync",
		Usage:  "Disable swarm syncing",
		EnvVar: SwarmEnvSyncDisable,
	}
	SwarmSyncUpdateDelay = cli.DurationFlag{
		Name:   "sync-update-delay",
		Usage:  "Duration for sync subscriptions update after no new peers are added (default 15s)",
		EnvVar: SwarmEnvSyncUpdateDelay,
	}
	SwarmMaxStreamPeerServersFlag = cli.IntFlag{
		Name:   "max-stream-peer-servers",
		Usage:  "Limit of Stream peer servers, 0 denotes unlimited",
		EnvVar: SwarmEnvMaxStreamPeerServers,
		Value:  10000, // A very large default value is possible as stream servers have very small memory footprint
	}
	SwarmLightNodeEnabled = cli.BoolFlag{
		Name:   "lightnode",
		Usage:  "Enable Swarm LightNode (default false)",
		EnvVar: SwarmEnvLightNodeEnable,
	}
	SwarmDeliverySkipCheckFlag = cli.BoolFlag{
		Name:   "delivery-skip-check",
		Usage:  "Skip chunk delivery check (default false)",
		EnvVar: SwarmEnvDeliverySkipCheck,
	}
	EnsAPIFlag = cli.StringSliceFlag{
		Name:   "ens-api",
		Usage:  "ENS API endpoint for a TLD and with contract address, can be repeated, format [tld:][contract-addr@]url",
		EnvVar: SwarmEnvENSAPI,
	}
	SwarmApiFlag = cli.StringFlag{
		Name:  "bzzapi",
		Usage: "Specifies the Swarm HTTP endpoint to connect to",
		Value: "http://127.0.0.1:8500",
	}
	SwarmRecursiveFlag = cli.BoolFlag{
		Name:  "recursive",
		Usage: "Upload directories recursively",
	}
	SwarmWantManifestFlag = cli.BoolTFlag{
		Name:  "manifest",
		Usage: "Automatic manifest upload (default true)",
	}
	SwarmUploadDefaultPath = cli.StringFlag{
		Name:  "defaultpath",
		Usage: "path to file served for empty url path (none)",
	}
	SwarmAccessGrantKeyFlag = cli.StringFlag{
		Name:  "grant-key",
		Usage: "grants a given public key access to an ACT",
	}
	SwarmAccessGrantKeysFlag = cli.StringFlag{
		Name:  "grant-keys",
		Usage: "grants a given list of public keys in the following file (separated by line breaks) access to an ACT",
	}
	SwarmUpFromStdinFlag = cli.BoolFlag{
		Name:  "stdin",
		Usage: "reads data to be uploaded from stdin",
	}
	SwarmUploadMimeType = cli.StringFlag{
		Name:  "mime",
		Usage: "Manually specify MIME type",
	}
	SwarmEncryptedFlag = cli.BoolFlag{
		Name:  "encrypt",
		Usage: "use encrypted upload",
	}
	SwarmAccessPasswordFlag = cli.StringFlag{
		Name:   "password",
		Usage:  "Password",
		EnvVar: SwarmAccessPassword,
	}
	SwarmDryRunFlag = cli.BoolFlag{
		Name:  "dry-run",
		Usage: "dry-run",
	}
	CorsStringFlag = cli.StringFlag{
		Name:   "corsdomain",
		Usage:  "Domain on which to send Access-Control-Allow-Origin header (multiple domains can be supplied separated by a ',')",
		EnvVar: SwarmEnvCORS,
	}
	SwarmStorePath = cli.StringFlag{
		Name:   "store.path",
		Usage:  "Path to leveldb chunk DB (default <$GETH_ENV_DIR>/swarm/bzz-<$BZZ_KEY>/chunks)",
		EnvVar: SwarmEnvStorePath,
	}
	SwarmStoreCapacity = cli.Uint64Flag{
		Name:   "store.size",
		Usage:  "Number of chunks (5M is roughly 20-25GB) (default 5000000)",
		EnvVar: SwarmEnvStoreCapacity,
	}
	SwarmStoreCacheCapacity = cli.UintFlag{
		Name:   "store.cache.size",
		Usage:  "Number of recent chunks cached in memory",
		EnvVar: SwarmEnvStoreCacheCapacity,
		Value:  10000,
	}
	SwarmCompressedFlag = cli.BoolFlag{
		Name:  "compressed",
		Usage: "Prints encryption keys in compressed form",
	}
	SwarmBootnodeModeFlag = cli.BoolFlag{
		Name:  "bootnode-mode",
		Usage: "Run Swarm in Bootnode mode",
	}
	SwarmFeedNameFlag = cli.StringFlag{
		Name:  "name",
		Usage: "User-defined name for the new feed, limited to 32 characters. If combined with topic, it will refer to a subtopic with this name",
	}
	SwarmFeedTopicFlag = cli.StringFlag{
		Name:  "topic",
		Usage: "User-defined topic this feed is tracking, hex encoded. Limited to 64 hexadecimal characters",
	}
	SwarmFeedManifestFlag = cli.StringFlag{
		Name:  "manifest",
		Usage: "Refers to the feed through a manifest",
	}
	SwarmFeedUserFlag = cli.StringFlag{
		Name:  "user",
		Usage: "Indicates the user who updates the feed",
	}
	SwarmGlobalStoreAPIFlag = cli.StringFlag{
		Name:   "globalstore-api",
		Usage:  "URL of the Global Store API provider (only for testing)",
		EnvVar: SwarmGlobalstoreAPI,
	}
)
