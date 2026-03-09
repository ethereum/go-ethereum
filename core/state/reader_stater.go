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

package state

// ContractCodeReaderStats aggregates statistics for the contract code reader.
type ContractCodeReaderStats struct {
	CacheHit       int64 // Number of cache hits
	CacheMiss      int64 // Number of cache misses
	CacheHitBytes  int64 // Total bytes served from cache
	CacheMissBytes int64 // Total bytes read on cache misses
}

// HitRate returns the cache hit rate in percentage.
func (s ContractCodeReaderStats) HitRate() float64 {
	total := s.CacheHit + s.CacheMiss
	if total == 0 {
		return 0
	}
	return float64(s.CacheHit) / float64(total) * 100
}

// ContractCodeReaderStater wraps the method to retrieve the statistics of
// contract code reader.
type ContractCodeReaderStater interface {
	GetCodeStats() ContractCodeReaderStats
}

// StateReaderStats aggregates statistics for the state reader.
type StateReaderStats struct {
	AccountCacheHit  int64 // Number of account cache hits
	AccountCacheMiss int64 // Number of account cache misses
	StorageCacheHit  int64 // Number of storage cache hits
	StorageCacheMiss int64 // Number of storage cache misses
}

// AccountCacheHitRate returns the cache hit rate of account requests in percentage.
func (s StateReaderStats) AccountCacheHitRate() float64 {
	total := s.AccountCacheHit + s.AccountCacheMiss
	if total == 0 {
		return 0
	}
	return float64(s.AccountCacheHit) / float64(total) * 100
}

// StorageCacheHitRate returns the cache hit rate of storage requests in percentage.
func (s StateReaderStats) StorageCacheHitRate() float64 {
	total := s.StorageCacheHit + s.StorageCacheMiss
	if total == 0 {
		return 0
	}
	return float64(s.StorageCacheHit) / float64(total) * 100
}

// StateReaderStater wraps the method to retrieve the statistics of state reader.
type StateReaderStater interface {
	GetStateStats() StateReaderStats
}

// ReaderStats wraps the statistics of reader.
type ReaderStats struct {
	CodeStats  ContractCodeReaderStats
	StateStats StateReaderStats
}

// ReaderStater defines the capability to retrieve aggregated statistics.
type ReaderStater interface {
	GetStats() ReaderStats
}
