// Copyright 2024-2025 the libevm authors.
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

const (
	LibEVMVersionMajor = 0
	LibEVMVersionMinor = 4
	LibEVMVersionPatch = 0

	LibEVMReleaseType      ReleaseType = BetaRelease
	libEVMReleaseCandidate uint        = 0 // ignored unless [LibEVMReleaseType] == [ReleaseCandidate]
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
const LibEVMVersion = "1.13.14-0.4.0.beta"

// A ReleaseType is a suffix for [LibEVMVersion].
type ReleaseType string

const (
	// BetaRelease MUST be used on `main` branch.
	BetaRelease = ReleaseType("beta")
	// Reserved for `release/*` branches.
	ReleaseCandidate  = ReleaseType("rc")
	ProductionRelease = ReleaseType("release")
)

// ForReleaseBranch returns true i.f.f. `t` is suitable for use on a release
// branch. The sets of [ReleaseType] values suitable for release vs default
// branches is disjoint so the negation of the return value is equivalent to
// "ForDefaultBranch".
func (t ReleaseType) ForReleaseBranch() bool {
	return t == ReleaseCandidate || t == ProductionRelease
}
