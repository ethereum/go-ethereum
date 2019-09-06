package lescdn

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/trie"
)

const (
	// tileTarget is the target number of trie nodes to place into each tile. The
	// actual count might be smaller for leaf tiles, or larger to ensure proper
	// tile barriers.
	tileTarget = 16

	// tileLimit is the maximum number of trie nodes to place into each tile, above
	// which the tile is forcefully split, even if that means breaking barriers.
	tileLimit = 256

	// tileBarrier is the trie depth multiplier where trie nodes need to end to
	// ensure that mutating tries still reuse the same non-mutated tiles.
	//
	// The number 2 was chosen experimentally, but the rationalization behind it is
	// that nodes close to the root will be very dense. The worst case where the nodes
	// are filled, a barrier of 3 would result in about 16^3 = 4096 hash pointers in
	// the leaves, which is 128KB + internal pointers + nodes + boilerplate. Depending
	// on trie shape, this can grow to even larger values, becoming useless, especially
	// at the root, so 2 seems to be a limit. A barrier of 2 produced about 8KB tiles
	// in our experiments.
	tileBarrier = 2
)

// serveState is responsible for serving HTTP requests for state data.
func (s *Service) serveState(w http.ResponseWriter, r *http.Request) {
	// TODO(karalabe): the non-defaults are for benchmarking, get rid when finalized
	cutTileTarget := int64(tileTarget)
	if target, ok := r.URL.Query()["target"]; ok {
		cutTileTarget, _ = strconv.ParseInt(target[0], 0, 64)
	}
	curTileLimit := int64(tileLimit)
	if limit, ok := r.URL.Query()["limit"]; ok {
		curTileLimit, _ = strconv.ParseInt(limit[0], 0, 64)
	}
	curTileBarrier := int64(tileBarrier)
	if barrier, ok := r.URL.Query()["barrier"]; ok {
		curTileBarrier, _ = strconv.ParseInt(barrier[0], 0, 64)
	}
	// Decode the root of the subtrie tile we should return
	root, err := hexutil.Decode(shift(&r.URL.Path))
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid state root: %v", err), http.StatusBadRequest)
		return
	}
	if len(root) != common.HashLength {
		http.Error(w, fmt.Sprintf("invalid state root: length %d != %d", len(root), common.HashLength), http.StatusBadRequest)
		return
	}
	// Do a breadth-first expansion to collect a fixed size tile
	triedb := s.chain.StateCache().TrieDB()

	nodes, refset, cutset, err := makeIdealTile(triedb, common.BytesToHash(root), int(cutTileTarget), int(curTileBarrier))
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to make tile: %v", err), http.StatusBadRequest)
		return
	}
	// If our cutset nodes won't result in meaningful tiles (they reach the leaves),
	// merge all of them into the current tile to avoid creating millions of subtiles.
	var (
		merged []common.Hash
		merges [][]byte
	)
	for !cutset.Empty() {
		// Fetch the deepest cutset node and merge in if it's a leaf
		hash := cutset.PopItem().(common.Hash)

		subnodes, _, subcutset, err := makeIdealTile(triedb, hash, int(cutTileTarget), int(curTileBarrier))
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to make subtile: %v", err), http.StatusBadRequest)
			return
		}
		if subcutset.Empty() {
			merged = append(merged, hash)
			merges = append(merges, subnodes...)
			continue
		}
		// Deepest cutset node produces non-leaf tile, don't bother with shallower node
		break
	}
	// If the final tile became huge, it means we packed in too many leaves due to
	// tile mergers. Shave off the nodes that caused tile mergers in the first place.
	if len(nodes)+len(merges) > int(curTileLimit) {
		for _, drop := range merged {
			for i, refs := range refset {
				if _, ok := refs[drop]; ok {
					nodes = append(nodes[:i], nodes[i+1:]...)
					refset = append(refset[:i], refset[i+1:]...)
					break
				}
			}
		}
	} else {
		nodes = append(nodes, merges...)
	}
	reply(w, nodes)
}

// makeIdealTile gathers trie nodes and assembles an ideal tile: one that barely
// exceeds the allowed node count and terminates at tile boundaries.
func makeIdealTile(triedb *trie.Database, root common.Hash, limit int, barrier int) ([][]byte, []map[common.Hash]struct{}, *prque.Prque, error) {
	queue := prque.New(nil)
	queue.Push(root, 0)

	var (
		nodes  [][]byte
		refset []map[common.Hash]struct{}
		cutset = prque.New(nil)
	)
	for !queue.Empty() {
		// Fetch the next trie node, which may or may not be included in the tile
		root, prio := queue.Pop()
		hash, depth := root.(common.Hash), -prio

		if len(nodes) > int(limit) {
			// Tile exceeded its recommended size. If the next node is on a tile barrier,
			// leave it to be collected in a next run (or retrieved from a cache).
			if int(depth)%barrier == 0 {
				cutset.Push(hash, depth)
				continue
			}
		}
		// Tile not done yet, fetch the next node and append it to the tile
		node, err := triedb.Node(hash)
		if err != nil {
			return nil, nil, nil, err
		}
		nodes = append(nodes, node)

		// Expand the trie node and queue all children up
		refs := make(map[common.Hash]struct{})
		trie.IterateRefs(node, func(path []byte, child common.Hash) error {
			queue.Push(child, -(depth + int64(len(path))))
			refs[child] = struct{}{}
			return nil
		})
		refset = append(refset, refs)
	}
	return nodes, refset, cutset, nil
}
