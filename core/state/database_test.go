// Copyright 2020 The go-ethereum Authors
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

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/trie"
)

func randBytes(n int) []byte {
	r := make([]byte, n)
	rand.Read(r)
	return r
}

func newCommiTasks(n int, db *trie.Database) []*commitTask {
	var tasks []*commitTask
	for i := 0; i < n; i++ {
		t, _ := trie.NewSecure(emptyRoot, db)
		for j := 0; j < 10; j++ {
			t.Update(randBytes(20), randBytes(32))
		}
		root := t.Hash()
		tasks = append(tasks, &commitTask{
			root:    root,
			number:  uint64(i + 1),
			state:   t,
			storage: make(map[common.Hash]Trie),
		})
	}
	return tasks
}

func TestRunCommitTask(t *testing.T) {
	cdb := NewDatabase(rawdb.NewMemoryDatabase())
	tasks := newCommiTasks(2, cdb.TrieDB())

	task := tasks[0]
	cdb.Commit(task.root, task.number, task.state, task.storage, nil, nil)
	cdb.WaitCommits(0)
	_, err := cdb.OpenTrie(task.root)
	if err != nil {
		t.Fatalf("Failed to commit state")
	}
	task = tasks[1]
	signal := make(chan struct{})
	cdb.Commit(task.root, task.number, task.state, task.storage, nil, func() {
		signal <- struct{}{} // Will block the commiting
	})
	_, err = cdb.OpenTrie(task.root)
	if err != nil {
		t.Fatalf("Failed to fetch the trie in pending list")
	}
	<-signal
	cdb.Close()
}

func TestCachingDBClose(t *testing.T) {
	cdb := NewDatabase(rawdb.NewMemoryDatabase())
	tasks := newCommiTasks(10, cdb.TrieDB())

	signal := make(chan struct{})
	for i := 0; i < len(tasks); i++ {
		task := tasks[i]
		cdb.Commit(task.root, task.number, task.state, task.storage, nil, func() {
			signal <- struct{}{}
		})
	}
	// Drain the blocking channel
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(time.Second) // Hack! ensure the close is called.
		for range signal {
		}
	}()
	cdb.Close()
	close(signal)
	wg.Wait()
}
