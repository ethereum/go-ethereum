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
	"log"
	"os/exec"
	"strconv"
	"time"
)

const (
	VersionMajor = 1          // Major version component of the current release
	VersionMinor = 9          // Minor version component of the current release
	VersionPatch = 0          // Patch version component of the current release
	VersionMeta  = "unstable" // Version metadata to append to the version string
)

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

// ArchiveVersion holds the textual version string used for Geth archives.
// e.g. "1.8.11-dea1ce05" for stable releases, or
//      "1.8.13-unstable-21c059b6" for unstable releases
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

func VersionWithCommit(gitCommit string) string {
	vsn := VersionWithMeta
	if len(gitCommit) >= 8 {
		vsn += "-" + gitCommit[:8]
	}
	if VersionMeta != "stable" {
		d := getCommitDate(gitCommit)
		if d != "" {
			vsn += "-" + d
		}
	}
	return vsn
}

func getCommitDate(commit string) string {
	if commit != "" {
		out, err := exec.Command("git", "show", "-s", "--format=%ct", commit).CombinedOutput()
		if err != nil {
			log.Println("Could not get gitCommit date: " + string(out))
			return ""
		}
		ti, err := strconv.ParseInt(string(out), 10, 64)
		if err != nil {
			log.Println("Could not convert gitCommit date. Parsed timestap is: " + string(out))
			return ""
		}
		return time.Unix(ti, 0).Format("20060102")
	}
	return ""
}
