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
