// Copyright 2025 the libevm authors.
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

package rawdb

import (
	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/libevm/options"
)

// An InspectDatabaseOption configures the behaviour of [InspectDatabase].
type InspectDatabaseOption = options.Option[inspectDatabaseConfig]

type inspectDatabaseConfig struct {
	statRecorders     []func([]byte, common.StorageSize) bool
	isMetas           []func([]byte) bool
	statsTransformers []func([][]string) [][]string
}

func (c inspectDatabaseConfig) recordStat(key []byte, size common.StorageSize) bool {
	matched := false
	for _, f := range c.statRecorders {
		if f(key, size) {
			matched = true
		}
	}
	return matched
}

func (c inspectDatabaseConfig) isMetadata(key []byte) bool {
	for _, f := range c.isMetas {
		if f(key) {
			return true
		}
	}
	return false
}

func (c inspectDatabaseConfig) transformStats(stats [][]string) [][]string {
	for _, f := range c.statsTransformers {
		stats = f(stats)
	}
	return stats
}

func newInspectOpt(fn func(*inspectDatabaseConfig)) InspectDatabaseOption {
	return options.Func[inspectDatabaseConfig](fn)
}

// WithDatabaseStatRecorder returns an option that results in `rec` being called
// for every `key` not otherwise matched by the [InspectDatabase] iterator loop.
// The returned boolean signals whether the recorder matches the key, thus
// stopping further matches.
func WithDatabaseStatRecorder(rec func(key []byte, size common.StorageSize) bool) InspectDatabaseOption {
	return newInspectOpt(func(c *inspectDatabaseConfig) {
		c.statRecorders = append(c.statRecorders, rec)
	})
}

// A DatabaseStat stores total size and counts for a parameter measured by
// [InspectDatabase]. It is exported for use with [WithDatabaseStatRecorder].
type DatabaseStat = stat

// WithDatabaseMetadataKeys returns an option that results in the `key` size
// being counted with the metadata statistic i.f.f. the function returns true.
func WithDatabaseMetadataKeys(isMetadata func(key []byte) bool) InspectDatabaseOption {
	return newInspectOpt(func(c *inspectDatabaseConfig) {
		c.isMetas = append(c.isMetas, isMetadata)
	})
}

// WithDatabaseStatsTransformer returns an option that causes all statistics rows to
// be passed to the provided function, with its return value being printed
// instead of the original values.
// Each row contains 4 columns: database, category, size and count.
func WithDatabaseStatsTransformer(transform func(rows [][]string) [][]string) InspectDatabaseOption {
	return newInspectOpt(func(c *inspectDatabaseConfig) {
		c.statsTransformers = append(c.statsTransformers, transform)
	})
}
