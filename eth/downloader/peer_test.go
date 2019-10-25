// Copyright 2020 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package downloader

import (
	"sort"
	"testing"
)

func TestPeerThroughputSorting(t *testing.T) {
	a := &peerConnection{
		id:               "a",
		headerThroughput: 1.25,
	}
	b := &peerConnection{
		id:               "b",
		headerThroughput: 1.21,
	}
	c := &peerConnection{
		id:               "c",
		headerThroughput: 1.23,
	}

	peers := []*peerConnection{a, b, c}
	tps := []float64{a.headerThroughput,
		b.headerThroughput, c.headerThroughput}
	sortPeers := &peerThroughputSort{peers, tps}
	sort.Sort(sortPeers)
	if got, exp := sortPeers.p[0].id, "a"; got != exp {
		t.Errorf("sort fail, got %v exp %v", got, exp)
	}
	if got, exp := sortPeers.p[1].id, "c"; got != exp {
		t.Errorf("sort fail, got %v exp %v", got, exp)
	}
	if got, exp := sortPeers.p[2].id, "b"; got != exp {
		t.Errorf("sort fail, got %v exp %v", got, exp)
	}

}
