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

import (
	"testing"

	"golang.org/x/mod/semver"
)

func TestLibEVMVersioning(t *testing.T) {
	// We have an unusual version structure as defined by [LibEVMVersion] that
	// is easy to mess up, so it's easier to just automate it and test the
	// ordering assumptions.

	// This is a deliberate change-detector test to provide us with a copyable
	// string of the current version, useful for git tagging.
	const curr = "1.13.14-0.1.0.beta"
	if got, want := LibEVMVersion, curr; got != want {
		t.Errorf("got LibEVMVersion %q; want %q", got, want)
	}

	ordered := []libEVMSemver{
		{
			semverTriplet{1, 13, 14},
			semverTriplet{0, 1, 0},
			betaRelease,
			0, // ignored
		},
		{
			semverTriplet{1, 13, 14},
			semverTriplet{0, 1, 0},
			releaseCandidate, 1,
		},
		{
			semverTriplet{1, 13, 14},
			semverTriplet{0, 1, 0},
			releaseCandidate, 2,
		},
		{
			semverTriplet{1, 13, 14},
			semverTriplet{0, 1, 0},
			productionRelease,
			0, // ignored,
		},
		{
			semverTriplet{1, 13, 14},
			semverTriplet{0, 1, 1}, // bump takes precedence
			betaRelease, 0,
		},
		{
			semverTriplet{1, 13, 14},
			semverTriplet{0, 1, 1},
			productionRelease, 0,
		},
		{
			semverTriplet{1, 13, 15}, // bump takes precedence
			semverTriplet{0, 1, 1},
			betaRelease, 0,
		},
	}

	for i, low := range ordered[:len(ordered)-1] {
		// The `go mod` semver package requires the "v" prefix, which
		// technically isn't valid semver.
		lo := "v" + low.String()
		hi := "v" + ordered[i+1].String()
		if got := semver.Compare(lo, hi); got != -1 {
			t.Errorf("Version pattern is not strictly ordered; semver.Compare(%q, %q) = %d", lo, hi, got)
		}
	}
}
