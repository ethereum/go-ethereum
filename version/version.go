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

package version

import (
	"fmt"
)

const (
	Major = 1          // Major version component of the current release
	Minor = 14         // Minor version component of the current release
	Patch = 12         // Patch version component of the current release
	Meta  = "unstable" // Version metadata to append to the version string
)

// Semantic holds the textual version string.
var Semantic = func() string {
	return fmt.Sprintf("%d.%d.%d", Major, Minor, Patch)
}()

// WithMeta holds the textual version string including the metadata.
var WithMeta = func() string {
	v := Semantic
	if Meta != "" {
		v += "-" + Meta
	}
	return v
}()

func WithCommit(gitCommit, gitDate string) string {
	vsn := WithMeta
	if len(gitCommit) >= 8 {
		vsn += "-" + gitCommit[:8]
	}
	if (Meta != "stable") && (gitDate != "") {
		vsn += "-" + gitDate
	}
	return vsn
}

// Archive holds the textual version string used for Geth archives. e.g.
// "1.8.11-dea1ce05" for stable releases, or "1.8.13-unstable-21c059b6" for unstable
// releases.
func Archive(gitCommit string) string {
	vsn := Semantic
	if Meta != "stable" {
		vsn += "-" + Meta
	}
	if len(gitCommit) >= 8 {
		vsn += "-" + gitCommit[:8]
	}
	return vsn
}
