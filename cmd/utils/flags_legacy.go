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

	"github.com/XinFinOrg/XDPoSChain/eth/ethconfig"
	"github.com/XinFinOrg/XDPoSChain/internal/flags"
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
	LogBacktraceAtFlag,
	LogDebugFlag,
	MiningEnabledFlag,
	XDCXDataDirFlag,
	LightServFlag,
	LightPeersFlag,
}

var (
	// Deprecated May 2020, shown in aliased flags section
	NoUSBFlag = &cli.BoolFlag{
		Name:     "nousb",
		Usage:    "Disables monitoring for and managing USB hardware wallets (deprecated)",
		Category: flags.DeprecatedCategory,
	}
	// Deprecated November 2023
	LogBacktraceAtFlag = &cli.StringFlag{
		Name:     "log-backtrace",
		Usage:    "Request a stack trace at a specific logging statement (deprecated)",
		Value:    "",
		Category: flags.DeprecatedCategory,
	}
	LogDebugFlag = &cli.BoolFlag{
		Name:     "log-debug",
		Usage:    "Prepends log messages with call-site location (deprecated)",
		Category: flags.DeprecatedCategory,
	}
	// Deprecated February 2024
	MetricsEnabledExpensiveFlag = &cli.BoolFlag{
		Name:     "metrics-expensive",
		Usage:    "Enable expensive metrics collection and reporting (deprecated)",
		Category: flags.DeprecatedCategory,
	}
	// Deprecated February 2025
	MiningEnabledFlag = &cli.BoolFlag{
		Name:     "mine",
		Usage:    "Enable mining (deprecated)",
		Category: flags.DeprecatedCategory,
	}
	XDCXDataDirFlag = &flags.DirectoryFlag{
		Name:     "XDCx-datadir",
		Aliases:  []string{"XDCx.datadir"},
		Usage:    "Data directory for the XDCX databases (deprecated)",
		Category: flags.DeprecatedCategory,
	}
	// Deprecated March 2025
	LightServFlag = &cli.IntFlag{
		Name:     "light-serv",
		Aliases:  []string{"lightserv"},
		Usage:    "Maximum percentage of time allowed for serving LES requests (0-90)",
		Value:    ethconfig.Defaults.LightServ,
		Category: flags.DeprecatedCategory,
	}
	LightPeersFlag = &cli.IntFlag{
		Name:     "light-peers",
		Aliases:  []string{"lightpeers"},
		Usage:    "Maximum number of LES client peers",
		Value:    ethconfig.Defaults.LightPeers,
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
