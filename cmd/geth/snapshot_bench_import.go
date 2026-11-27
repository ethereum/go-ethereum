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

package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/sync/errgroup"
)

const (
	flatStateImportWorkers   = 8
	flatStateImportBatchSize = 16 * 1024 * 1024
)

type flatStateImportStats struct {
	Accounts int64
	Slots    int64
	Bytes    int64
}

// importBenchmarkFlatState copies snapshot accounts and storage into a clean
// database. Each worker handles one disjoint prefix and account-hash nibble at
// a time so the SSTs can be installed without overlapping key ranges.
func importBenchmarkFlatState(ctx context.Context, src ethdb.Iteratee, dst ethdb.Ingester) (flatStateImportStats, error) {
	var (
		accounts atomic.Int64
		slots    atomic.Int64
		bytes    atomic.Int64

		stats = func() flatStateImportStats {
			return flatStateImportStats{
				Accounts: accounts.Load(),
				Slots:    slots.Load(),
				Bytes:    bytes.Load(),
			}
		}
	)
	done := make(chan struct{})
	defer close(done)

	go func() {
		start := time.Now()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				s := stats()
				elapsed := time.Since(start)
				rate := common.StorageSize(float64(s.Bytes) / elapsed.Seconds())
				log.Info("Importing flat state", "accounts", s.Accounts, "slots", s.Slots, "data", common.StorageSize(s.Bytes),
					"elapsed", common.PrettyDuration(elapsed), "rate", fmt.Sprintf("%v/s", rate))
			}
		}
	}()

	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(flatStateImportWorkers)

	for _, section := range [...]struct {
		prefix     []byte
		suffixSize int
		entries    *atomic.Int64
	}{
		{rawdb.SnapshotAccountPrefix, common.HashLength, &accounts},
		{rawdb.SnapshotStoragePrefix, 2 * common.HashLength, &slots},
	} {
		for partition := range 16 {
			group.Go(func() error {
				if err := ctx.Err(); err != nil {
					return err
				}
				nibble := byte(partition)
				it := src.NewIterator(section.prefix, []byte{nibble << 4})
				defer it.Release()

				if err := importFlatStateRange(ctx, it, dst, len(section.prefix), section.suffixSize, nibble, section.entries, &bytes); err != nil {
					return fmt.Errorf("import flat state %q partition %x: %w", section.prefix, nibble, err)
				}
				return nil
			})
		}
	}
	err := group.Wait()
	return stats(), err
}

func importFlatStateRange(ctx context.Context, it ethdb.Iterator, dst ethdb.Ingester, prefixSize, suffixSize int, nibble byte, entries, bytes *atomic.Int64) error {
	batch := dst.NewIngestBatch(flatStateImportBatchSize)
	defer batch.Close()

	var (
		count int64
		size  int
	)
	flush := func() error {
		if err := batch.Write(); err != nil {
			return err
		}
		entries.Add(count)
		bytes.Add(int64(size))
		batch.Reset()
		count, size = 0, 0
		return nil
	}
	for it.Next() {
		if err := ctx.Err(); err != nil {
			return err
		}
		key := it.Key()
		if len(key) <= prefixSize || key[prefixSize]>>4 != nibble {
			break
		}
		if len(key) != prefixSize+suffixSize {
			continue
		}
		value := it.Value()
		entrySize := len(key) + len(value)
		if size > 0 && size+entrySize > flatStateImportBatchSize {
			if err := flush(); err != nil {
				return err
			}
		}
		if err := batch.Put(key, value); err != nil {
			return err
		}
		count++
		size += entrySize
	}
	if err := it.Error(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if size > 0 {
		return flush()
	}
	return nil
}
