// Copyright 2022 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

const ourPath = "github.com/ethereum/go-ethereum" // Path to our module

// summarizeBadBlock returns a string summarizing the bad block and other
// relevant information.
func summarizeBadBlock(block *types.Block, receipts []*types.Receipt, config *params.ChainConfig, err error) string {
	var receiptString string
	for i, receipt := range receipts {
		receiptString += fmt.Sprintf("  %d: cumulative: %v gas: %v contract: %v status: %v tx: %v logs: %v bloom: %x state: %x\n",
			i, receipt.CumulativeGasUsed, receipt.GasUsed, receipt.ContractAddress.Hex(),
			receipt.Status, receipt.TxHash.Hex(), receipt.Logs, receipt.Bloom, receipt.PostState)
	}
	return fmt.Sprintf(`
########## BAD BLOCK #########
Block: %v (%#x)
Error: %v
Version: %v
Chain config: %#v
Receipts:
%v##############################
`, block.Number(), block.Hash(), err, runtimeInfo(), config, receiptString)
}

// RuntimeInfo returns build and platform information about the current binary.
//
// If the package that is currently executing is a prefixed by our go-ethereum
// module path, it will print out commit and date VCS information. Otherwise,
// it will assume it's imported by a third-party and will return the imported
// version and whether it was replaced by another module.
func runtimeInfo() string {
	var (
		version       string
		buildInfo, ok = debug.ReadBuildInfo()
	)

	switch {
	case !ok:
		// BuildInfo should generally be set. Fallback to the coded
		// version if not.
		version = params.VersionWithMeta
	case strings.HasPrefix(buildInfo.Path, ourPath):
		// If the main package is from our repo, we can actually
		// retrieve the VCS information directly from the buildInfo.
		revision, date, dirty := vcsInfo(buildInfo)
		version = fmt.Sprintf("geth %s", params.VersionWithCommit(revision, date))
		if dirty {
			version += " (dirty)"
		}
	default:
		// Not our main package, probably imported by a different
		// project. VCS data less relevant here.
		mod := findModule(buildInfo, ourPath)
		version = fmt.Sprintf("%s%s %s@%s", buildInfo.Path, buildInfo.Main.Version, mod.Path, mod.Version)
		if mod.Replace != nil {
			version += fmt.Sprintf(" (replaced by %s@%s)", mod.Replace.Path, mod.Replace.Version)
		}
	}
	return fmt.Sprintf("%s %s %s %s", version, runtime.Version(), runtime.GOARCH, runtime.GOOS)
}

// findModule returns the module at path.
func findModule(info *debug.BuildInfo, path string) *debug.Module {
	if info.Path == ourPath {
		return &info.Main
	}
	for _, mod := range info.Deps {
		if mod.Path == path {
			return mod
		}
	}
	return nil
}

// vcsInfo returns VCS information of the build.
func vcsInfo(info *debug.BuildInfo) (revision, date string, dirty bool) {
	revision = "unknown"
	date = "unknown"

	for _, v := range info.Settings {
		switch v.Key {
		case "vcs.revision":
			revision = v.Value
		case "vcs.modified":
			if v.Value == "true" {
				dirty = true
			}
		case "vcs.time":
			date = v.Value
		}
	}
	return revision, date, dirty
}
