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

package utils

import (
	"fmt"

	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/urfave/cli/v2"
)

var ShowDeprecated = &cli.Command{
	Action:      showDeprecated,
	Name:        "show-deprecated-flags",
	Usage:       "Show flags that have been deprecated",
	ArgsUsage:   " ",
	Description: "Show flags that have been deprecated and will soon be removed",
}

var DeprecatedFlags = []cli.Flag{
	NoUSBFlag,
	LegacyWhitelistFlag,
	CacheTrieJournalFlag,
	CacheTrieRejournalFlag,
	LegacyDiscoveryV5Flag,
	TxLookupLimitFlag,
	LightServeFlag,
	LightIngressFlag,
	LightEgressFlag,
	LightMaxPeersFlag,
	LightNoPruneFlag,
	LightNoSyncServeFlag,
	LogBacktraceAtFlag,
	LogDebugFlag,
	MinerNewPayloadTimeoutFlag,
	MinerEtherbaseFlag,
	MiningEnabledFlag,
}

var (
	// Deprecated May 2020, shown in aliased flags section
	NoUSBFlag = &cli.BoolFlag{
		Name:     "nousb",
		Usage:    "Disables monitoring for and managing USB hardware wallets (deprecated)",
		Category: flags.DeprecatedCategory,
	}
	// Deprecated March 2022
	LegacyWhitelistFlag = &cli.StringFlag{
		Name:     "whitelist",
		Usage:    "Comma separated block number-to-hash mappings to enforce (<number>=<hash>) (deprecated in favor of --eth.requiredblocks)",
		Category: flags.DeprecatedCategory,
	}
	// Deprecated July 2023
	CacheTrieJournalFlag = &cli.StringFlag{
		Name:     "cache.trie.journal",
		Usage:    "Disk journal directory for trie cache to survive node restarts",
		Category: flags.DeprecatedCategory,
	}
	CacheTrieRejournalFlag = &cli.DurationFlag{
		Name:     "cache.trie.rejournal",
		Usage:    "Time interval to regenerate the trie cache journal",
		Category: flags.DeprecatedCategory,
	}
	LegacyDiscoveryV5Flag = &cli.BoolFlag{
		Name:     "v5disc",
		Usage:    "Enables the experimental RLPx V5 (Topic Discovery) mechanism (deprecated, use --discv5 instead)",
		Category: flags.DeprecatedCategory,
	}
	// Deprecated August 2023
	TxLookupLimitFlag = &cli.Uint64Flag{
		Name:     "txlookuplimit",
		Usage:    "Number of recent blocks to maintain transactions index for (default = about one year, 0 = entire chain) (deprecated, use history.transactions instead)",
		Value:    ethconfig.Defaults.TransactionHistory,
		Category: flags.DeprecatedCategory,
	}
	// Light server and client settings, Deprecated November 2023
	LightServeFlag = &cli.IntFlag{
		Name:     "light.serve",
		Usage:    "Maximum percentage of time allowed for serving LES requests (deprecated)",
		Category: flags.DeprecatedCategory,
	}
	LightIngressFlag = &cli.IntFlag{
		Name:     "light.ingress",
		Usage:    "Incoming bandwidth limit for serving light clients (deprecated)",
		Category: flags.DeprecatedCategory,
	}
	LightEgressFlag = &cli.IntFlag{
		Name:     "light.egress",
		Usage:    "Outgoing bandwidth limit for serving light clients (deprecated)",
		Category: flags.DeprecatedCategory,
	}
	LightMaxPeersFlag = &cli.IntFlag{
		Name:     "light.maxpeers",
		Usage:    "Maximum number of light clients to serve, or light servers to attach to (deprecated)",
		Category: flags.DeprecatedCategory,
	}
	LightNoPruneFlag = &cli.BoolFlag{
		Name:     "light.nopruning",
		Usage:    "Disable ancient light chain data pruning (deprecated)",
		Category: flags.DeprecatedCategory,
	}
	LightNoSyncServeFlag = &cli.BoolFlag{
		Name:     "light.nosyncserve",
		Usage:    "Enables serving light clients before syncing (deprecated)",
		Category: flags.DeprecatedCategory,
	}
	// Deprecated November 2023
	LogBacktraceAtFlag = &cli.StringFlag{
		Name:     "log.backtrace",
		Usage:    "Request a stack trace at a specific logging statement (deprecated)",
		Value:    "",
		Category: flags.DeprecatedCategory,
	}
	LogDebugFlag = &cli.BoolFlag{
		Name:     "log.debug",
		Usage:    "Prepends log messages with call-site location (deprecated)",
		Category: flags.DeprecatedCategory,
	}
	// Deprecated February 2024
	MinerNewPayloadTimeoutFlag = &cli.DurationFlag{
		Name:     "miner.newpayload-timeout",
		Usage:    "Specify the maximum time allowance for creating a new payload (deprecated)",
		Value:    ethconfig.Defaults.Miner.Recommit,
		Category: flags.DeprecatedCategory,
	}
	MinerEtherbaseFlag = &cli.StringFlag{
		Name:     "miner.etherbase",
		Usage:    "0x prefixed public address for block mining rewards (deprecated)",
		Category: flags.DeprecatedCategory,
	}
	MiningEnabledFlag = &cli.BoolFlag{
		Name:     "mine",
		Usage:    "Enable mining (deprecated)",
		Category: flags.DeprecatedCategory,
	}
	MetricsEnabledExpensiveFlag = &cli.BoolFlag{
		Name:     "metrics.expensive",
		Usage:    "Enable expensive metrics collection and reporting (deprecated)",
		Category: flags.DeprecatedCategory,
	}
)

// showDeprecated displays deprecated flags that will be soon removed from the codebase.
func showDeprecated(*cli.Context) error {
	fmt.Println("--------------------------------------------------------------------")
	fmt.Println("The following flags are deprecated and will be removed in the future!")
	fmt.Println("--------------------------------------------------------------------")
	fmt.Println()
	for _, flag := range DeprecatedFlags {
		fmt.Println(flag.String())
	}
	fmt.Println()
	return nil
}
