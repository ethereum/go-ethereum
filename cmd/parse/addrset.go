package main

import (
	"github.com/ethereum/go-ethereum/common"
	"math"
	"os"
)

type AddressSet interface {
	Add(addr common.Address) bool
	Contains(addr common.Address) bool
}

type TreeSet struct {
	set map[common.Address]struct{}
}

func NewTreeSet() *TreeSet {
	return &TreeSet{
		set: make(map[common.Address]struct{}),
	}
}

func (ts *TreeSet) Add(addr common.Address) bool {
	_, ok := ts.set[addr]
	if ok {
		return false
	}
	ts.set[addr] = struct{}{}
	return true
}

func (ts *TreeSet) Contains(addr common.Address) bool {
	_, ok := ts.set[addr]
	return ok
}

type BloomFilterSet struct {
	k                    int    // number of filters
	data                 []byte // bits
	n                    int    // number of items inserted so far
	inefficiencyReported bool
}

func NewBloomFilterSet(sizeBytes, estimatedItems uint64) *BloomFilterSet {
	bfs := &BloomFilterSet{
		k:                    int(math.Round(float64(sizeBytes*8) * math.Log(2) / float64(estimatedItems))),
		data:                 make([]byte, sizeBytes),
		n:                    0,
		inefficiencyReported: false,
	}

	if bfs.k < 1 {
		bfs.k = 1
	}

	return bfs
}

func (bfs *BloomFilterSet) computeHashes(addr common.Address) []uint64 {
	hashes := make([]uint64, bfs.k)
	for i := 0; i < bfs.k; i++ {
		h := uint64(0)
		for j := 0; j < 8; j++ {
			h = (h << 8) | uint64(addr.Bytes()[(i+j*(i/20+1))%20])
		}
		hashes[i] = h
	}
	return hashes
}

func (bfs *BloomFilterSet) Add(addr common.Address) bool {
	hashes := bfs.computeHashes(addr)
	found := true
	for _, h := range hashes {
		bit := h & 3
		offs := (h >> 3) % uint64(len(bfs.data))
		b := int(bfs.data[offs])
		mask := 1 << bit
		if b&mask == 0 {
			found = false
			bfs.data[offs] = byte(b | mask)
		}
	}
	if !found && !bfs.inefficiencyReported {
		bfs.n++
		if (bfs.n & 0xff) == 0 { // check each 256 items
			m := float64(len(bfs.data) * 8)
			q := float64(bfs.k*bfs.n) / m
			t := 1 - math.Exp(-q)
			falseposprob := math.Pow(t, float64(bfs.k))
			if !math.IsNaN(falseposprob) && !math.IsInf(falseposprob, 0) && falseposprob > 0.0001 {
				os.Stderr.WriteString("The Bloom filter is inefficient: I recommend to set the -memsize option to a higher value or risk missing addresses in the output\n")
				bfs.inefficiencyReported = true
			}
		}
	}
	return !found
}

func (bfs *BloomFilterSet) Contains(addr common.Address) bool {
	hashes := bfs.computeHashes(addr)
	for _, h := range hashes {
		bit := h & 3
		offs := (h >> 3) % uint64(len(bfs.data))
		if (bfs.data[offs] & (1 << bit)) == 0 {
			return false
		}
	}
	return true
}
