package downloader

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"gopkg.in/fatih/set.v0"
)

func createHashSet(hashes []common.Hash) *set.Set {
	hset := set.New()

	for _, hash := range hashes {
		hset.Add(hash)
	}

	return hset
}

func createBlocksFromHashSet(hashes *set.Set) []*types.Block {
	blocks := make([]*types.Block, hashes.Size())

	var i int
	hashes.Each(func(v interface{}) bool {
		blocks[i] = createBlock(i, common.Hash{}, v.(common.Hash))
		i++
		return true
	})

	return blocks
}

func TestChunking(t *testing.T) {
	queue := newQueue()
	peer1 := newPeer("peer1", common.Hash{}, nil, nil)
	peer2 := newPeer("peer2", common.Hash{}, nil, nil)

	// 99 + 1 (1 == known genesis hash)
	hashes := createHashes(0, 99)
	queue.Insert(hashes)

	chunk1 := queue.Reserve(peer1, 99)
	if chunk1 == nil {
		t.Errorf("chunk1 is nil")
		t.FailNow()
	}
	chunk2 := queue.Reserve(peer2, 99)
	if chunk2 == nil {
		t.Errorf("chunk2 is nil")
		t.FailNow()
	}

	if len(chunk1.Hashes) != 99 {
		t.Error("expected chunk1 hashes to be 99, got", len(chunk1.Hashes))
	}

	if len(chunk2.Hashes) != 1 {
		t.Error("expected chunk1 hashes to be 1, got", len(chunk2.Hashes))
	}
}
