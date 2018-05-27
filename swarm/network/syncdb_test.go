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

package network

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

func init() {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlCrit, log.StreamHandler(os.Stderr, log.TerminalFormat(false))))
}

type testSyncDb struct {
	*syncDb
	c         int
	t         *testing.T
	fromDb    chan bool
	delivered [][]byte
	sent      []int
	dbdir     string
	at        int
}

func newTestSyncDb(priority, bufferSize, batchSize int, dbdir string, t *testing.T) *testSyncDb {
	if len(dbdir) == 0 {
		tmp, err := ioutil.TempDir(os.TempDir(), "syncdb-test")
		if err != nil {
			t.Fatalf("unable to create temporary direcory %v: %v", tmp, err)
		}
		dbdir = tmp
	}
	db, err := storage.NewLDBDatabase(filepath.Join(dbdir, "requestdb"))
	if err != nil {
		t.Fatalf("unable to create db: %v", err)
	}
	self := &testSyncDb{
		fromDb: make(chan bool),
		dbdir:  dbdir,
		t:      t,
	}
	h := crypto.Keccak256Hash([]byte{0})
	key := storage.Key(h[:])
	self.syncDb = newSyncDb(db, key, uint(priority), uint(bufferSize), uint(batchSize), self.deliver)
	// kick off db iterator right away, if no items on db this will allow
	// reading from the buffer
	return self

}

func (self *testSyncDb) close() {
	self.db.Close()
	os.RemoveAll(self.dbdir)
}

func (self *testSyncDb) push(n int) {
	for i := 0; i < n; i++ {
		self.buffer <- storage.Key(crypto.Keccak256([]byte{byte(self.c)}))
		self.sent = append(self.sent, self.c)
		self.c++
	}
	log.Debug(fmt.Sprintf("pushed %v requests", n))
}

func (self *testSyncDb) draindb() {
	it := self.db.NewIterator()
	defer it.Release()
	for {
		it.Seek(self.start)
		if !it.Valid() {
			return
		}
		k := it.Key()
		if len(k) == 0 || k[0] == 1 {
			return
		}
		it.Release()
		it = self.db.NewIterator()
	}
}

func (self *testSyncDb) deliver(req interface{}, quit chan bool) bool {
	_, db := req.(*syncDbEntry)
	key, _, _, _, err := parseRequest(req)
	if err != nil {
		self.t.Fatalf("unexpected error of key %v: %v", key, err)
	}
	self.delivered = append(self.delivered, key)
	select {
	case self.fromDb <- db:
		return true
	case <-quit:
		return false
	}
}

func (self *testSyncDb) expect(n int, db bool) {
	var ok bool
	// for n items
	for i := 0; i < n; i++ {
		ok = <-self.fromDb
		if self.at+1 > len(self.delivered) {
			self.t.Fatalf("expected %v, got %v", self.at+1, len(self.delivered))
		}
		if len(self.sent) > self.at && !bytes.Equal(crypto.Keccak256([]byte{byte(self.sent[self.at])}), self.delivered[self.at]) {
			self.t.Fatalf("expected delivery %v/%v/%v to be hash of  %v, from db: %v = %v", i, n, self.at, self.sent[self.at], ok, db)
			log.Debug(fmt.Sprintf("%v/%v/%v to be hash of  %v, from db: %v = %v", i, n, self.at, self.sent[self.at], ok, db))
		}
		if !ok && db {
			self.t.Fatalf("expected delivery %v/%v/%v from db", i, n, self.at)
		}
		if ok && !db {
			self.t.Fatalf("expected delivery %v/%v/%v from cache", i, n, self.at)
		}
		self.at++
	}
}

func TestSyncDb(t *testing.T) {
	t.Skip("fails randomly on all platforms")

	priority := High
	bufferSize := 5
	batchSize := 2 * bufferSize
	s := newTestSyncDb(priority, bufferSize, batchSize, "", t)
	defer s.close()
	defer s.stop()
	s.dbRead(false, 0, s.deliver)
	s.draindb()

	s.push(4)
	s.expect(1, false)
	// 3 in buffer
	time.Sleep(100 * time.Millisecond)
	s.push(3)
	// push over limit
	s.expect(1, false)
	// one popped from the buffer, then contention detected
	s.expect(4, true)
	s.push(4)
	s.expect(5, true)
	// depleted db, switch back to buffer
	s.draindb()
	s.push(5)
	s.expect(4, false)
	s.push(3)
	s.expect(4, false)
	// buffer depleted
	time.Sleep(100 * time.Millisecond)
	s.push(6)
	s.expect(1, false)
	// push into buffer full, switch to db
	s.expect(5, true)
	s.draindb()
	s.push(1)
	s.expect(1, false)
}

func TestSaveSyncDb(t *testing.T) {
	amount := 30
	priority := High
	bufferSize := amount
	batchSize := 10
	s := newTestSyncDb(priority, bufferSize, batchSize, "", t)
	go s.dbRead(false, 0, s.deliver)
	s.push(amount)
	s.stop()
	s.db.Close()

	s = newTestSyncDb(priority, bufferSize, batchSize, s.dbdir, t)
	go s.dbRead(false, 0, s.deliver)
	s.expect(amount, true)
	for i, key := range s.delivered {
		expKey := crypto.Keccak256([]byte{byte(i)})
		if !bytes.Equal(key, expKey) {
			t.Fatalf("delivery %v expected to be key %x, got %x", i, expKey, key)
		}
	}
	s.push(amount)
	s.expect(amount, false)
	for i := amount; i < 2*amount; i++ {
		key := s.delivered[i]
		expKey := crypto.Keccak256([]byte{byte(i - amount)})
		if !bytes.Equal(key, expKey) {
			t.Fatalf("delivery %v expected to be key %x, got %x", i, expKey, key)
		}
	}
	s.stop()
	s.db.Close()

	s = newTestSyncDb(priority, bufferSize, batchSize, s.dbdir, t)
	defer s.close()
	defer s.stop()

	go s.dbRead(false, 0, s.deliver)
	s.push(1)
	s.expect(1, false)

}
