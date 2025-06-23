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

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/mod/semver"
)

// libEVMServer automates the version rules described by the [LibEVMVersion]
// documentation.
type libEVMSemver struct {
	geth, libEVM semverTriplet
	typ          ReleaseType
	rc           uint
}

func (v libEVMSemver) String() string {
	suffix := v.typ
	if suffix == ReleaseCandidate {
		suffix = ReleaseType(fmt.Sprintf("%s.%d", suffix, v.rc))
	}
	return fmt.Sprintf("%s-%s.%s", v.geth, v.libEVM, suffix)
}

type semverTriplet struct {
	major, minor, patch uint
}

func (t semverTriplet) String() string {
	return fmt.Sprintf("%d.%d.%d", t.major, t.minor, t.patch)
}

func TestLibEVMVersioning(t *testing.T) {
	t.Run("current", func(t *testing.T) {
		want := libEVMSemver{
			geth:   semverTriplet{VersionMajor, VersionMinor, VersionPatch},
			libEVM: semverTriplet{LibEVMVersionMajor, LibEVMVersionMinor, LibEVMVersionPatch},
			typ:    LibEVMReleaseType,
			rc:     libEVMReleaseCandidate,
		}.String()
		assert.Equal(t, want, LibEVMVersion, "LibEVMVersion")
	})

	ordered := []libEVMSemver{
		{
			semverTriplet{1, 13, 14},
			semverTriplet{0, 1, 0},
			BetaRelease,
			0, // ignored
		},
		{
			semverTriplet{1, 13, 14},
			semverTriplet{0, 1, 0},
			ReleaseCandidate, 1,
		},
		{
			semverTriplet{1, 13, 14},
			semverTriplet{0, 1, 0},
			ReleaseCandidate, 2,
		},
		{
			semverTriplet{1, 13, 14},
			semverTriplet{0, 1, 0},
			ProductionRelease,
			0, // ignored,
		},
		{
			semverTriplet{1, 13, 14},
			semverTriplet{0, 1, 1}, // bump takes precedence
			BetaRelease, 0,
		},
		{
			semverTriplet{1, 13, 14},
			semverTriplet{0, 1, 1},
			ProductionRelease, 0,
		},
		{
			semverTriplet{1, 13, 15}, // bump takes precedence
			semverTriplet{0, 1, 1},
			BetaRelease, 0,
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
