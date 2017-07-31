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

package storage

import "testing"

func newTestMemStore() *MemStore {
	return NewMemStore(nil, defaultCacheCapacity)
}

func testMemStoreRandom(n int, processors int, chunksize int, t *testing.T) {
	m := newTestMemStore()
	defer m.Close()
	testStoreRandom(m, processors, n, chunksize, t)
}

func testMemStoreCorrect(n int, processors int, chunksize int, t *testing.T) {
	m := newTestMemStore()
	defer m.Close()
	testStoreCorrect(m, processors, n, chunksize, t)
}

func TestMemStoreRandom_1(t *testing.T) {
	testMemStoreRandom(1, 1, 0, t)
}

func TestMemStoreCorrect_1(t *testing.T) {
	testMemStoreCorrect(1, 1, 4104, t)
}

func TestMemStoreRandom_1_10k(t *testing.T) {
	testMemStoreRandom(1, 5000, 0, t)
}

func TestMemStoreCorrect_1_10k(t *testing.T) {
	testMemStoreCorrect(1, 5000, 4096, t)
}

func TestMemStoreRandom_8_10k(t *testing.T) {
	testMemStoreRandom(8, 5000, 0, t)
}

func TestMemStoreCorrect_8_10k(t *testing.T) {
	testMemStoreCorrect(8, 5000, 4096, t)
}

func TestMemStoreNotFound(t *testing.T) {
	m := newTestMemStore()
	defer m.Close()

	_, err := m.Get(ZeroKey)
	if err != notFound {
		t.Errorf("Expected notFound, got %v", err)
	}
}

func benchmarkMemStorePut(n int, processors int, chunksize int, b *testing.B) {
	m := newTestMemStore()
	defer m.Close()
	benchmarkStorePut(m, processors, n, chunksize, b)
}

func benchmarkMemStoreGet(n int, processors int, chunksize int, b *testing.B) {
	m := newTestMemStore()
	defer m.Close()
	benchmarkStoreGet(m, processors, n, chunksize, b)
}

func BenchmarkMemStorePut_1_5k(b *testing.B) {
	benchmarkMemStorePut(5000, 1, 4096, b)
}

func BenchmarkMemStorePut_8_5k(b *testing.B) {
	benchmarkMemStorePut(5000, 8, 4096, b)
}

func BenchmarkMemStoreGet_1_5k(b *testing.B) {
	benchmarkMemStoreGet(5000, 1, 4096, b)
}

func BenchmarkMemStoreGet_8_5k(b *testing.B) {
	benchmarkMemStoreGet(5000, 8, 4096, b)
}
