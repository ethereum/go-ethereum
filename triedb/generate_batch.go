// Copyright 2026 The go-ethereum Authors
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

package triedb

import "github.com/ethereum/go-ethereum/ethdb"

// Bound the memory held by each stream while writing sufficiently large SSTs.
const trieIngestBatchSize = 8 * 1024 * 1024

type generationBatch struct {
	ethdb.Batch
	limit int
}

func (b *generationBatch) full() bool {
	return b.ValueSize() >= b.limit
}

func (b *generationBatch) flush() error {
	if b.ValueSize() == 0 {
		return nil
	}
	if err := b.Write(); err != nil {
		return err
	}
	b.Reset()
	return nil
}

func newGenerationBatch(db ethdb.Database, ingester ethdb.Ingester) *generationBatch {
	if ingester == nil {
		return &generationBatch{
			Batch: db.NewBatchWithSize(ethdb.IdealBatchSize),
			limit: ethdb.IdealBatchSize + 1,
		}
	}
	return &generationBatch{
		Batch: ingester.NewIngestBatch(trieIngestBatchSize),
		limit: trieIngestBatchSize,
	}
}

// partitionBatches keeps account and storage nodes in separate SST streams.
// Flat repairs and assembled roots use ordinary writes. On other backends all
// three streams share the ordinary batch.
type partitionBatches struct {
	flat    *generationBatch
	account *generationBatch
	storage *generationBatch
}

func newPartitionBatches(db ethdb.Database, ingester ethdb.Ingester) *partitionBatches {
	flat := newGenerationBatch(db, nil)
	if ingester == nil {
		return &partitionBatches{
			flat:    flat,
			account: flat,
			storage: flat,
		}
	}
	return &partitionBatches{
		flat:    flat,
		account: newGenerationBatch(db, ingester),
		storage: newGenerationBatch(db, ingester),
	}
}

func (b *partitionBatches) close() {
	if b.account != b.flat {
		b.account.Close()
		b.storage.Close()
	}
	b.flat.Close()
}

func (b *partitionBatches) flush() error {
	for _, batch := range [...]*generationBatch{b.account, b.storage, b.flat} {
		if err := batch.flush(); err != nil {
			return err
		}
	}
	return nil
}
