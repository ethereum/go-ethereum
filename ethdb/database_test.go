// Copyright 2014 The go-ethereum Authors
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

package ethdb

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/metrics"
)

func newTestLDB() (*LDBDatabase, func()) {
	dirname, err := ioutil.TempDir(os.TempDir(), "ethdb_test_")
	if err != nil {
		panic("failed to create test file: " + err.Error())
	}
	db, err := NewLDBDatabase(dirname, 0, 0)
	if err != nil {
		panic("failed to create test database: " + err.Error())
	}

	return db, func() {
		db.Close()
		os.RemoveAll(dirname)
	}
}

var test_values = []string{"", "a", "1251", "\x00123\x00"}

func TestLDB_PutGet(t *testing.T) {
	db, remove := newTestLDB()
	defer remove()
	//enable metrics
	metrics.Enabled = true
	db.Meter("prefix")
	testPutGet(db, t)
}

func TestMemoryDB_PutGet(t *testing.T) {
	db, _ := NewMemDatabase()
	testPutGet(db, t)
}

func testPutGet(db Database, t *testing.T) {
	t.Parallel()

	for _, v := range test_values {
		err := db.Put([]byte(v), []byte(v))
		if err != nil {
			t.Fatalf("put failed: %v", err)
		}
	}

	for _, v := range test_values {
		data, err := db.Get([]byte(v))
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		if !bytes.Equal(data, []byte(v)) {
			t.Fatalf("get returned wrong result, got %q expected %q", string(data), v)
		}
	}

	for _, v := range test_values {
		err := db.Put([]byte(v), []byte("?"))
		if err != nil {
			t.Fatalf("put override failed: %v", err)
		}
	}

	for _, v := range test_values {
		data, err := db.Get([]byte(v))
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		if !bytes.Equal(data, []byte("?")) {
			t.Fatalf("get returned wrong result, got %q expected ?", string(data))
		}
	}

	for _, v := range test_values {
		orig, err := db.Get([]byte(v))
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		orig[0] = byte(0xff)
		data, err := db.Get([]byte(v))
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		if !bytes.Equal(data, []byte("?")) {
			t.Fatalf("get returned wrong result, got %q expected ?", string(data))
		}
	}

	for _, v := range test_values {
		err := db.Delete([]byte(v))
		if err != nil {
			t.Fatalf("delete %q failed: %v", v, err)
		}
	}

	for _, v := range test_values {
		_, err := db.Get([]byte(v))
		if err == nil {
			t.Fatalf("got deleted value %q", v)
		}
	}
	//assert for LDBDatabase meters
	if ldb, ok := db.(*LDBDatabase); ok {
		valuesLen := int64(len(test_values))
		writeBytes, readBytes := int64(0), int64(0)
		for _, v := range test_values {
			writeBytes += int64(len(v))
			readBytes += int64(len(v))
		}
		writeBytes += int64(len("?")) * valuesLen
		readBytes += int64(len("?")) * valuesLen * 3
		expects := map[string]int64{
			"putCount":   valuesLen * 2,
			"writeBytes": writeBytes,
			"getCount":   valuesLen * 5,
			"readBytes":  readBytes,
			"delCount":   valuesLen,
			"missCount":  valuesLen,
		}
		assertForMeters(ldb, expects, t)
	}
}

func TestLDB_ParallelPutGet(t *testing.T) {
	db, remove := newTestLDB()
	defer remove()
	//enable metrics
	metrics.Enabled = true
	db.Meter("parallel-prefix")
	testParallelPutGet(db, t)
}

func TestMemoryDB_ParallelPutGet(t *testing.T) {
	db, _ := NewMemDatabase()
	testParallelPutGet(db, t)
}

func testParallelPutGet(db Database, t *testing.T) {
	const n = 8
	var pending sync.WaitGroup

	pending.Add(n)
	for i := 0; i < n; i++ {
		go func(key string) {
			defer pending.Done()
			err := db.Put([]byte(key), []byte("v"+key))
			if err != nil {
				panic("put failed: " + err.Error())
			}
		}(strconv.Itoa(i))
	}
	pending.Wait()

	pending.Add(n)
	for i := 0; i < n; i++ {
		go func(key string) {
			defer pending.Done()
			data, err := db.Get([]byte(key))
			if err != nil {
				panic("get failed: " + err.Error())
			}
			if !bytes.Equal(data, []byte("v"+key)) {
				panic(fmt.Sprintf("get failed, got %q expected %q", []byte(data), []byte("v"+key)))
			}
		}(strconv.Itoa(i))
	}
	pending.Wait()

	pending.Add(n)
	for i := 0; i < n; i++ {
		go func(key string) {
			defer pending.Done()
			err := db.Delete([]byte(key))
			if err != nil {
				panic("delete failed: " + err.Error())
			}
		}(strconv.Itoa(i))
	}
	pending.Wait()

	pending.Add(n)
	for i := 0; i < n; i++ {
		go func(key string) {
			defer pending.Done()
			_, err := db.Get([]byte(key))
			if err == nil {
				panic("get succeeded")
			}
		}(strconv.Itoa(i))
	}
	pending.Wait()
	//assert for LDBDatabase meters
	if ldb, ok := db.(*LDBDatabase); ok {
		totalBytes := int64(0)
		for i := 0; i < n; i++ {
			totalBytes += int64(len([]byte("v" + strconv.Itoa(i))))
		}
		expects := map[string]int64{
			"putCount":   n,
			"writeBytes": totalBytes,
			"getCount":   n * 2,
			"readBytes":  totalBytes,
			"delCount":   n,
			"missCount":  n,
		}
		assertForMeters(ldb, expects, t)
	}
}

func assertForMeters(ldb *LDBDatabase, expects map[string]int64, t *testing.T) {
	//test putTimer
	ret := ldb.putTimer.Count()
	if ret != expects["putCount"] {
		t.Errorf("putTimer count: expected %d, got %d", expects["putCount"], ret)
	}
	min := ldb.putTimer.Min()
	if min == 0 {
		t.Error("putTimer: expected min time larger than zero, got 0")
	}
	mean := ldb.putTimer.Mean()
	if mean == 0 {
		t.Error("putTimer: expected mean time larger than zero, got 0")
	}
	max := ldb.putTimer.Max()
	if max == 0 {
		t.Error("putTimer: expected max time larger than zero, got 0")
	}
	//test writeMeter
	ret = ldb.writeMeter.Count()
	if ret != expects["writeBytes"] {
		t.Errorf("writeMeter: expected %d, got %d", expects["writeBytes"], ret)
	}
	//test getTimer
	ret = ldb.getTimer.Count()
	if ret != expects["getCount"] {
		t.Errorf("getTimer count: expected %d, got %d", expects["getCount"], ret)
	}
	min = ldb.getTimer.Min()
	if min == 0 {
		t.Error("getTimer: expected min time larger than zero, got 0")
	}
	mean = ldb.getTimer.Mean()
	if mean == 0 {
		t.Error("getTimer: expected mean time larger than zero, got 0")
	}
	max = ldb.getTimer.Max()
	if max == 0 {
		t.Error("getTimer: expected max time larger than zero, got 0")
	}
	//test readMeter
	ret = ldb.readMeter.Count()
	if ret != expects["readBytes"] {
		t.Errorf("readMeter: expected %d got %d", expects["readBytes"], ret)
	}
	//test delTimer
	ret = ldb.delTimer.Count()
	if ret != expects["delCount"] {
		t.Errorf("delTimer count: expected %d got %d", expects["delCount"], ret)
	}
	min = ldb.delTimer.Min()
	if min == 0 {
		t.Error("delTimer: expected min time larger than zero, got 0")
	}
	mean = ldb.delTimer.Mean()
	if mean == 0 {
		t.Error("delTimer: expected mean time larger than zero, got 0")
	}
	max = ldb.delTimer.Max()
	if max == 0 {
		t.Error("delTimer: expected max time larger than zero, got 0")
	}
	//test missMeter
	ret = ldb.missMeter.Count()
	if ret != expects["missCount"] {
		t.Errorf("missMeter: expected %d got %d", expects["missCount"], ret)
	}
}
