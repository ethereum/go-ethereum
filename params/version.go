// Copyright 2016 The go-ethereum Authors
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

package params

import (
	"fmt"

	"github.com/ethereum/go-ethereum/metrics"
)

const (
	VersionMajor = 2  // Major version component of the current release
	VersionMinor = 1  // Minor version component of the current release
	VersionPatch = 1  // Patch version component of the current release
	VersionMeta  = "" // Version metadata to append to the version string
)

var (
	// borInfoGauge stores Bor git commit and version details.
	borInfoGauge = metrics.NewRegisteredGaugeInfo("bor/info", nil)

	GitCommit string
)

// UpdateBorInfo updates the bor_info metric with the current git commit and version details.
func UpdateBorInfo() {
	borInfoGauge.Update(metrics.GaugeInfoValue{
		"commit":  GitCommit,
		"version": VersionWithMeta,
	})
}

// Version holds the textual version string.
var Version = func() string {
	return fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
}()

// VersionWithMeta holds the textual version string including the metadata.
var VersionWithMeta = func() string {
	v := Version
	if VersionMeta != "" {
		v += "-" + VersionMeta
	}
	return v
}()

// VersionWithCommitDetails holds the textual version string including the metadata and Git Details.
var VersionWithMetaCommitDetails = func() string {
	v := Version
	if VersionMeta != "" {
		v += "-" + VersionMeta
	}
	v_git := fmt.Sprintf("Version: %s\nGitCommit: %s", v, GitCommit)
	return v_git
}()

// ArchiveVersion holds the textual version string used for Geth archives.
// e.g. "1.8.11-dea1ce05" for stable releases, or
func ArchiveVersion(gitCommit string) string {
	vsn := Version
	if VersionMeta != "stable" {
		vsn += "-" + VersionMeta
	}

	if len(gitCommit) >= 8 {
		vsn += "-" + gitCommit[:8]
	}

	return vsn
}

func VersionWithCommit(gitCommit, gitDate string) string {
	vsn := VersionWithMeta
	if len(gitCommit) >= 8 {
		vsn += "-" + gitCommit[:8]
	}

	if (VersionMeta != "stable") && (gitDate != "") {
		vsn += "-" + gitDate
	}

	return vsn
}
