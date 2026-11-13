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

package rawdb

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethdb"
)

// TestChainFreezerConcurrentReadAncients checks that a second reader can enter
// ReadAncients while another reader is still inside it.
func TestChainFreezerConcurrentReadAncients(t *testing.T) {
	f, err := newChainFreezer(t.TempDir(), "", "", false)
	if err != nil {
		t.Fatalf("failed to create chain freezer: %v", err)
	}
	defer f.Close()

	var (
		entered = make(chan struct{})
		release = make(chan struct{})
		first   = make(chan struct{})
		second  = make(chan struct{})
	)
	go func() {
		defer close(first)
		f.ReadAncients(func(ethdb.AncientReaderOp) error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered

	go func() {
		defer close(second)
		f.ReadAncients(func(ethdb.AncientReaderOp) error { return nil })
	}()
	select {
	case <-second:
	case <-time.After(5 * time.Second):
		t.Error("second reader blocked behind the first one")
	}
	close(release)
	<-first
	<-second
}
