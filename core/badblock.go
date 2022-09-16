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
		receiptString += fmt.Sprintf("\n  %d: cumulative: %v gas: %v contract: %v status: %v tx: %v logs: %v bloom: %x state: %x",
			i, receipt.CumulativeGasUsed, receipt.GasUsed, receipt.ContractAddress.Hex(),
			receipt.Status, receipt.TxHash.Hex(), receipt.Logs, receipt.Bloom, receipt.PostState)
	}
	version, vcs := runtimeInfo()
	platform := fmt.Sprintf("%s %s %s %s", version, runtime.Version(), runtime.GOARCH, runtime.GOOS)
	if vcs != "" {
		vcs = fmt.Sprintf("\nVCS: %s", vcs)
	}
	return fmt.Sprintf(`
########## BAD BLOCK #########
Block: %v (%#x)
Error: %v
Platform: %v%v
Chain config: %#v
Receipts: %v
##############################
`, block.Number(), block.Hash(), err, platform, vcs, config, receiptString)
}

// runtimeInfo returns build and platform information about the current binary.
//
// If the package that is currently executing is a prefixed by our go-ethereum
// module path, it will print out commit and date VCS information. Otherwise,
// it will assume it's imported by a third-party and will return the imported
// version and whether it was replaced by another module.
func runtimeInfo() (string, string) {
	var (
		version       = params.VersionWithMeta
		vcs           = ""
		buildInfo, ok = debug.ReadBuildInfo()
	)
	if ok {
		version = versionInfo(buildInfo)
		if status, ok := vcsInfo(buildInfo); ok {
			modified := ""
			if status.modified {
				modified = " (dirty)"
			}
			vcs = status.revision + "-" + status.time + modified
		}
	}
	return version, vcs
}

// versionInfo returns version information for the currently executing
// implementation.
//
// Depending on how the code is instansiated, it returns different amounts of
// information. If it is unable to determine which module is related to our
// package it falls back to the hardcoded values in the params package.
func versionInfo(info *debug.BuildInfo) string {
	// If the main package is from our repo, prefix version with "geth".
	if strings.HasPrefix(info.Path, ourPath) {
		return fmt.Sprintf("geth %s", info.Main.Version)
	}
	// Not our main package, so explicitly print out the module path and
	// version.
	var version string
	if info.Main.Path != "" && info.Main.Version != "" {
		// These can be empty when invoked with "go run".
		version = fmt.Sprintf("%s@%s ", info.Main.Path, info.Main.Version)
	}
	mod := findModule(info, ourPath)
	if mod == nil {
		// If our module path wasn't imported, it's unclear which
		// version of our code they are running. Fallback to hardcoded
		// version.
		return version + fmt.Sprintf("geth %s", params.VersionWithMeta)
	}
	// Our package is a dependency for the main module. Return path and
	// version data for both.
	version += fmt.Sprintf("%s@%s", mod.Path, mod.Version)
	if mod.Replace != nil {
		// If our package was replaced by something else, also note that.
		version += fmt.Sprintf(" (replaced by %s@%s)", mod.Replace.Path, mod.Replace.Version)
	}
	return version
}

type status struct {
	revision string
	time     string
	modified bool
}

// vcsInfo returns VCS information of the build.
func vcsInfo(info *debug.BuildInfo) (s status, ok bool) {
	for _, v := range info.Settings {
		switch v.Key {
		case "vcs.revision":
			if len(v.Value) < 8 {
				s.revision = v.Value
			} else {
				s.revision = v.Value[:8]
			}
		case "vcs.modified":
			if v.Value == "true" {
				s.modified = true
			}
		case "vcs.time":
			s.time = v.Value
		}
	}
	if s.revision != "" && s.time != "" {
		ok = true
	}
	return
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
