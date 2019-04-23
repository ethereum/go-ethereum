package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/hashicorp/golang-lru"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var (
	dir          = flag.String("dir", "", "directory to TomoChain chaindata")
	cacheSize    = flag.Int("size", 1000000, "LRU cache size")
	sercureKey   = []byte("secure-key-")
	nWorker      = runtime.NumCPU() / 2
	cleanAddress = []common.Address{common.HexToAddress(common.BlockSigners)}
	cache        *lru.Cache
	finish       = int32(0)
	running      = true
	stateRoots   = make(chan TrieRoot)
)

type TrieRoot struct {
	trie   *trie.SecureTrie
	number uint64
}
type StateNode struct {
	node trie.Node
	path []byte
}
type ResultProcessNode struct {
	index    int
	number   int
	newNodes [17]*StateNode
	keys     [17]*[]byte
}

func main() {
	flag.Parse()
	lddb, _ := ethdb.NewLDBDatabase(*dir, eth.DefaultConfig.DatabaseCache, utils.MakeDatabaseHandles())
	head := core.GetHeadBlockHash(lddb)
	currentHeader := core.GetHeader(lddb, head, core.GetBlockNumber(lddb, head))
	tridb := trie.NewDatabase(lddb)
	catchEventInterupt(lddb.LDB())
	cache, _ = lru.New(*cacheSize)
	go func() {
		for i := uint64(1); i <= currentHeader.Number.Uint64(); i++ {
			hash := core.GetCanonicalHash(lddb, i)
			root := core.GetHeader(lddb, hash, i).Root
			trieRoot, err := trie.NewSecure(root, tridb, 0)
			if err != nil {
				continue
			}
			if running {
				stateRoots <- TrieRoot{trieRoot, i}
			} else {
				break
			}
		}
		if running {
			close(stateRoots)
		}
	}()
	for trieRoot := range stateRoots {
		atomic.StoreInt32(&finish, 1)
		if running {
			for _, address := range cleanAddress {
				enc := trieRoot.trie.Get(address.Bytes())
				var data state.Account
				rlp.DecodeBytes(enc, &data)
				fmt.Println(time.Now().Format(time.RFC3339), "Start clean state address ", address.Hex(), " at block ", trieRoot.number)
				signerRoot, err := resolveHash(data.Root[:], lddb.LDB())
				if err != nil {
					fmt.Println(time.Now().Format(time.RFC3339), "Not found clean state address ", address.Hex(), " at block ", trieRoot.number)
					continue
				}
				batch := new(leveldb.Batch)
				count := 1
				list := []*StateNode{{node: signerRoot}}
				for len(list) > 0 {
					newList, total := findNewNodes(list, lddb.LDB(), batch)
					count = count + 17*len(newList)
					list = removeNodesNil(newList, total)
				}
				fmt.Println(time.Now().Format(time.RFC3339), "Finish clean state address ", address.Hex(), " at block ", trieRoot.number, " keys ", count)
				err = lddb.LDB().Write(batch, nil)
				if err != nil {
					fmt.Println(time.Now().Format(time.RFC3339), "Write batch leveldb error", err)
					os.Exit(1)
				}
			}
		} else {
			break
		}
		atomic.StoreInt32(&finish, 0)
	}
	fmt.Println(time.Now(), "compact")
	lddb.LDB().CompactRange(util.Range{})
	lddb.Close()
	fmt.Println(time.Now(), "end")
}

func removeNodesNil(list [][17]*StateNode, length int) []*StateNode {
	results := make([]*StateNode, length)
	index := 0
	for _, nodes := range list {
		for _, node := range nodes {
			if node != nil {
				results[index] = node
				index++
			}
		}
	}
	return results
}
func catchEventInterupt(db *leveldb.DB) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			fmt.Println("catch event interrupt ", sig, running, finish)
			running = false
			if atomic.LoadInt32(&finish) == 0 {
				close(stateRoots)
				db.Close()
				os.Exit(1)
			}
		}
	}()
}
func resolveHash(n trie.HashNode, db *leveldb.DB) (trie.Node, error) {
	if cache.Contains(common.BytesToHash(n)) {
		return nil, &trie.MissingNodeError{}
	}
	enc, err := db.Get(n, nil)
	if err != nil || enc == nil {
		return nil, &trie.MissingNodeError{}
	}
	return trie.MustDecodeNode(n, enc, 0), nil
}

func getAllChilds(n StateNode, db *leveldb.DB) ([17]*StateNode, error) {
	childs := [17]*StateNode{}
	switch node := n.node.(type) {
	case *trie.FullNode:
		// Full Node, move to the first non-nil child.
		for i := 0; i < len(node.Children); i++ {
			child := node.Children[i]
			if child != nil {
				childNode := child
				var err error = nil
				if _, ok := child.(trie.HashNode); ok {
					childNode, err = resolveHash(child.(trie.HashNode), db)
				}
				if err == nil {
					childs[i] = &StateNode{node: childNode, path: append(n.path, byte(i))}
				} else if err != nil {
					_, ok := err.(*trie.MissingNodeError)
					if !ok {
						return childs, err
					}
				}
			}
		}
	case *trie.ShortNode:
		// Short Node, return the pointer singleton child
		childNode := node.Val
		var err error = nil
		if _, ok := node.Val.(trie.HashNode); ok {
			childNode, err = resolveHash(node.Val.(trie.HashNode), db)
		}
		if err == nil {
			childs[0] = &StateNode{node: childNode, path: append(n.path, node.Key...)}
		} else if err != nil {
			_, ok := err.(*trie.MissingNodeError)
			if !ok {
				return childs, err
			}
		}
	}
	return childs, nil
}
func processNodes(node StateNode, db *leveldb.DB) ([17]*StateNode, [17]*[]byte, int) {
	hash, _ := node.node.Cache()
	commonHash := common.BytesToHash(hash)
	newNodes := [17]*StateNode{}
	keys := [17]*[]byte{}
	number := 0
	if !cache.Contains(commonHash) {
		childNodes, err := getAllChilds(node, db)
		if err != nil {
			fmt.Println("Error when get all childs node : ", common.Bytes2Hex(node.path), err)
			os.Exit(1)
		}
		for i, child := range childNodes {
			if child != nil {
				if _, ok := child.node.(trie.ValueNode); ok {
					buf := append(sercureKey, child.path...)
					keys[i] = &buf
				} else {
					hash, _ := child.node.Cache()
					var bytes []byte = hash
					keys[i] = &bytes
					newNodes[i] = child
					number++
				}
			}
		}
		cache.Add(commonHash, true)
	}
	return newNodes, keys, number
}

func findNewNodes(nodes []*StateNode, db *leveldb.DB, batchlvdb *leveldb.Batch) ([][17]*StateNode, int) {
	length := len(nodes)
	chunkSize := length / nWorker
	if len(nodes)%nWorker != 0 {
		chunkSize++
	}
	childNodes := make([][17]*StateNode, length)
	results := make(chan ResultProcessNode)
	wg := sync.WaitGroup{}
	wg.Add(length)
	for i := 0; i < nWorker; i++ {
		from := i * chunkSize
		to := from + chunkSize
		if to > length {
			to = length
		}
		go func(from int, to int) {
			for j := from; j < to; j++ {
				childs, keys, number := processNodes(*nodes[j], db)
				go func(result ResultProcessNode) {
					results <- result
				}(ResultProcessNode{j, number, childs, keys})
			}
		}(from, to)
	}
	total := 0
	go func() {
		for result := range results {
			childNodes[result.index] = result.newNodes
			total = total + result.number
			for _, key := range result.keys {
				if key != nil {
					batchlvdb.Delete(*key)
				}
			}
			wg.Done()
		}
	}()
	wg.Wait()
	close(results)
	return childNodes, total
}
