// Copyright 2026 the libevm authors.
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

package core

import (
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/ethdb"
)

// BloomThrottling is the time to wait between processing two consecutive index sections.
const BloomThrottling = bloomThrottling

// NewBloomIndexerBackend creates a [BloomIndexer] instance for the given database and section size,
// allowing users to provide custom functionality to the bloom indexer.
func NewBloomIndexerBackend(db ethdb.Database, size uint64) *BloomIndexer {
	return &BloomIndexer{
		db:   db,
		size: size,
	}
}

// ProcessWithBloomOverride is the same as [BloomIndexer.Process], but takes the header and bloom separately.
// This must obey the same invariates as [BloomIndexer.Process], including calling [BloomIndexer.Reset]
// to start a new section prior to this call, otherwise this function will panic.
func (b *BloomIndexer) ProcessWithBloomOverride(header *types.Header, bloom types.Bloom) error {
	index := uint(header.Number.Uint64() - b.section*b.size)
	if err := b.gen.AddBloom(index, bloom); err != nil {
		return err
	}
	b.head = header.Hash()
	return nil
}
