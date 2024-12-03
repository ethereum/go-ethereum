// Copyright 2024 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

package params

import "fmt"

const (
	LibEVMVersionMajor = 0
	LibEVMVersionMinor = 1
	LibEVMVersionPatch = 0

	libEVMReleaseType      releaseType = betaRelease
	libEVMReleaseCandidate uint        = 0 // ignored unless [libEVMReleaseType] == [releaseCandidate]
)

// LibEVMVersion holds the textual version string of `libevm` modifications.
//
// Although compliant with [semver v2], it follows additional rules:
//
//  1. Major, minor, and patch MUST be the respective `geth` values;
//  2. The first three pre-release identifiers MUST be a semver-compliant
//     triplet denoting the `libevm` "version";
//  3. On the `main` (development) branch, the final identifier MUST be "alpha"
//     or "beta";
//  3. If a production version, the final identifier MUST be "release"; and
//  4. If a release candidate, the final two identifiers MUST be "rc" and an
//     incrementing numeric value.
//
// The benefits of this pattern are that (a) it captures all relevant
// information; and (b) it follows an intuitive ordering under semver rules.
// Precedence is determined first by the `geth` version then the `libevm`
// version, with release candidates being lower than actual releases.
//
// The primary drawbacks is that it requires an explicit "release" identifier
// because of the use of pre-release identifiers to capture the `libevm`
// triplet.
//
// [semver v2]: https://semver.org/
var LibEVMVersion = func() string {
	v := libEVMSemver{
		geth:   semverTriplet{VersionMajor, VersionMinor, VersionPatch},
		libEVM: semverTriplet{LibEVMVersionMajor, LibEVMVersionMinor, LibEVMVersionPatch},
		typ:    libEVMReleaseType,
		rc:     libEVMReleaseCandidate,
	}
	return v.String()
}()

type semverTriplet struct {
	major, minor, patch uint
}

func (t semverTriplet) String() string {
	return fmt.Sprintf("%d.%d.%d", t.major, t.minor, t.patch)
}

type releaseType string

const (
	// betaRelease MUST be used on `main` branch
	betaRelease = releaseType("beta")
	// Reserved for `release/*` branches
	releaseCandidate  = releaseType("rc")
	productionRelease = releaseType("release")
)

type libEVMSemver struct {
	geth, libEVM semverTriplet
	typ          releaseType
	rc           uint
}

func (v libEVMSemver) String() string {
	suffix := v.typ
	if suffix == releaseCandidate {
		suffix = releaseType(fmt.Sprintf("%s.%d", suffix, v.rc))
	}
	return fmt.Sprintf("%s-%s.%s", v.geth, v.libEVM, suffix)
}
