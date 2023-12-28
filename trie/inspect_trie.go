package trie

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"

	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/olekukonko/tablewriter"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/sync/semaphore"
)

const (
	DEFAULT_TRIEDBCACHE_SIZE = 1024 * 1024 * 1024
)

type Account struct {
	Nonce    uint64
	Balance  *big.Int
	Root     common.Hash // merkle root of the storage trie
	CodeHash []byte
}

type Inspector struct {
	trie           *Trie // traverse trie
	db             *Database
	stateRootHash  common.Hash
	blocknum       uint64
	root           node // root of triedb
	totalNum       uint64
	wg             sync.WaitGroup
	statLock       sync.RWMutex
	result         map[string]*TrieTreeStat
	sem            *semaphore.Weighted
	eoaAccountNums uint64
}

type TrieTreeStat struct {
	is_account_trie    bool
	theNodeStatByLevel [15]NodeStat
	totalNodeStat      NodeStat
}

type NodeStat struct {
	ShortNodeCnt uint64
	FullNodeCnt  uint64
	ValueNodeCnt uint64
}

func (trieStat *TrieTreeStat) AtomicAdd(theNode node, height uint32) {
	switch (theNode).(type) {
	case *shortNode:
		atomic.AddUint64(&trieStat.totalNodeStat.ShortNodeCnt, 1)
		atomic.AddUint64(&(trieStat.theNodeStatByLevel[height].ShortNodeCnt), 1)
	case *fullNode:
		atomic.AddUint64(&trieStat.totalNodeStat.FullNodeCnt, 1)
		atomic.AddUint64(&trieStat.theNodeStatByLevel[height].FullNodeCnt, 1)
	case valueNode:
		atomic.AddUint64(&trieStat.totalNodeStat.ValueNodeCnt, 1)
		atomic.AddUint64(&((trieStat.theNodeStatByLevel[height]).ValueNodeCnt), 1)
	default:
		panic(errors.New("Invalid node type to statistics"))
	}
}

func (trieStat *TrieTreeStat) Display(ownerAddress string, treeType string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"-", "Level", "ShortNodeCnt", "FullNodeCnt", "ValueNodeCnt"})
	if ownerAddress == "" {
		table.SetCaption(true, fmt.Sprintf("%v", treeType))
	} else {
		table.SetCaption(true, fmt.Sprintf("%v-%v", treeType, ownerAddress))
	}
	table.SetAlignment(1)
	for i := 0; i < len(trieStat.theNodeStatByLevel); i++ {
		nodeStat := trieStat.theNodeStatByLevel[i]
		if nodeStat.FullNodeCnt == 0 && nodeStat.ShortNodeCnt == 0 && nodeStat.ValueNodeCnt == 0 {
			break
		}
		table.AppendBulk([][]string{
			{"-", strconv.Itoa(i), nodeStat.ShortNodeCount(), nodeStat.FullNodeCount(), nodeStat.ValueNodeCount()},
		})
	}
	table.AppendBulk([][]string{
		{"Total", "-", trieStat.totalNodeStat.ShortNodeCount(), trieStat.totalNodeStat.FullNodeCount(), trieStat.totalNodeStat.ValueNodeCount()},
	})
	table.Render()
}

func Uint64ToString(cnt uint64) string {
	return fmt.Sprintf("%v", cnt)
}

func (nodeStat *NodeStat) ShortNodeCount() string {
	return Uint64ToString(nodeStat.ShortNodeCnt)
}

func (nodeStat *NodeStat) FullNodeCount() string {
	return Uint64ToString(nodeStat.FullNodeCnt)
}
func (nodeStat *NodeStat) ValueNodeCount() string {
	return Uint64ToString(nodeStat.ValueNodeCnt)
}

// NewInspector return a inspector obj
func NewInspector(tr *Trie, db *Database, stateRootHash common.Hash, blocknum uint64, jobnum uint64) (*Inspector, error) {
	if tr == nil {
		return nil, errors.New("trie is nil")
	}

	if tr.root == nil {
		return nil, errors.New("trie root is nil")
	}

	ins := &Inspector{
		trie:           tr,
		db:             db,
		stateRootHash:  stateRootHash,
		blocknum:       blocknum,
		root:           tr.root,
		result:         make(map[string]*TrieTreeStat),
		totalNum:       (uint64)(0),
		wg:             sync.WaitGroup{},
		sem:            semaphore.NewWeighted(int64(jobnum)),
		eoaAccountNums: 0,
	}

	return ins, nil
}

// Run statistics, external call
func (inspect *Inspector) Run() {
	accountTrieStat := &TrieTreeStat{
		is_account_trie: true,
	}
	if inspect.db.Scheme() == rawdb.HashScheme {
		ticker := time.NewTicker(30 * time.Second)
		go func() {
			defer ticker.Stop()
			for range ticker.C {
				inspect.db.Cap(DEFAULT_TRIEDBCACHE_SIZE)
			}
		}()
	}

	if _, ok := inspect.result[""]; !ok {
		inspect.result[""] = accountTrieStat
	}
	log.Info("Find Account Trie Tree", "rootHash: ", inspect.trie.Hash().String(), "BlockNum: ", inspect.blocknum)

	inspect.ConcurrentTraversal(inspect.trie, accountTrieStat, inspect.root, 0, []byte{})
	inspect.wg.Wait()
}

func (inspect *Inspector) SubConcurrentTraversal(theTrie *Trie, theTrieTreeStat *TrieTreeStat, theNode node, height uint32, path []byte) {
	inspect.ConcurrentTraversal(theTrie, theTrieTreeStat, theNode, height, path)
	inspect.wg.Done()
}

func (inspect *Inspector) ConcurrentTraversal(theTrie *Trie, theTrieTreeStat *TrieTreeStat, theNode node, height uint32, path []byte) {
	// print process progress
	total_num := atomic.AddUint64(&inspect.totalNum, 1)
	if total_num%100000 == 0 {
		fmt.Printf("Complete progress: %v, go routines Num: %v\n", total_num, runtime.NumGoroutine())
	}

	// nil node
	if theNode == nil {
		return
	}

	switch current := (theNode).(type) {
	case *shortNode:
		inspect.ConcurrentTraversal(theTrie, theTrieTreeStat, current.Val, height, append(path, current.Key...))
	case *fullNode:
		for idx, child := range current.Children {
			if child == nil {
				continue
			}
			childPath := append(path, byte(idx))
			if inspect.sem.TryAcquire(1) {
				inspect.wg.Add(1)
				dst := make([]byte, len(childPath))
				copy(dst, childPath)
				go inspect.SubConcurrentTraversal(theTrie, theTrieTreeStat, child, height+1, dst)
			} else {
				inspect.ConcurrentTraversal(theTrie, theTrieTreeStat, child, height+1, childPath)
			}
		}
	case hashNode:
		n, err := theTrie.resloveWithoutTrack(current, path)
		if err != nil {
			fmt.Printf("Resolve HashNode error: %v, TrieRoot: %v, Height: %v, Path: %v\n", err, theTrie.Hash().String(), height+1, path)
			return
		}
		inspect.ConcurrentTraversal(theTrie, theTrieTreeStat, n, height, path)
		return
	case valueNode:
		if !hasTerm(path) {
			break
		}
		var account Account
		if err := rlp.Decode(bytes.NewReader(current), &account); err != nil {
			break
		}
		if common.BytesToHash(account.CodeHash) == types.EmptyCodeHash {
			inspect.eoaAccountNums++
		}
		if account.Root == (common.Hash{}) || account.Root == types.EmptyRootHash {
			break
		}
		ownerAddress := common.BytesToHash(hexToCompact(path))
		contractTrie, err := New(StorageTrieID(inspect.stateRootHash, ownerAddress, account.Root), inspect.db)
		if err != nil {
			fmt.Printf("New contract trie node: %v, error: %v, Height: %v, Path: %v\n", theNode, err, height, path)
			break
		}
		contractTrie.tracer.reset()
		trieStat := &TrieTreeStat{
			is_account_trie: false,
		}

		inspect.statLock.Lock()
		if _, ok := inspect.result[ownerAddress.String()]; !ok {
			inspect.result[ownerAddress.String()] = trieStat
		}
		inspect.statLock.Unlock()

		// log.Info("Find Contract Trie Tree, rootHash: ", contractTrie.Hash().String(), "")
		inspect.wg.Add(1)
		go inspect.SubConcurrentTraversal(contractTrie, trieStat, contractTrie.root, 0, []byte{})
	default:
		panic(errors.New("Invalid node type to traverse."))
	}
	theTrieTreeStat.AtomicAdd(theNode, height)
}

func (inspect *Inspector) DisplayResult() {
	// display root hash
	if _, ok := inspect.result[""]; !ok {
		log.Info("Display result error", "missing account trie")
		return
	}
	inspect.result[""].Display("", "AccountTrie")

	type SortedTrie struct {
		totalNum     uint64
		ownerAddress string
	}
	// display contract trie
	var sortedTriesByNums []SortedTrie
	var totalContactsNodeStat NodeStat
	var contractTrieCnt uint64 = 0

	for ownerAddress, stat := range inspect.result {
		if ownerAddress == "" {
			continue
		}
		contractTrieCnt++
		totalContactsNodeStat.ShortNodeCnt += stat.totalNodeStat.ShortNodeCnt
		totalContactsNodeStat.FullNodeCnt += stat.totalNodeStat.FullNodeCnt
		totalContactsNodeStat.ValueNodeCnt += stat.totalNodeStat.ValueNodeCnt
		totalNodeCnt := stat.totalNodeStat.ShortNodeCnt + stat.totalNodeStat.ValueNodeCnt + stat.totalNodeStat.FullNodeCnt
		sortedTriesByNums = append(sortedTriesByNums, SortedTrie{totalNum: totalNodeCnt, ownerAddress: ownerAddress})
	}
	sort.Slice(sortedTriesByNums, func(i, j int) bool {
		return sortedTriesByNums[i].totalNum > sortedTriesByNums[j].totalNum
	})
	fmt.Println("EOA accounts num: ", inspect.eoaAccountNums)
	// only display top 5
	for i, t := range sortedTriesByNums {
		if i > 5 {
			break
		}
		if stat, ok := inspect.result[t.ownerAddress]; !ok {
			log.Error("Storage trie stat not found", "ownerAddress", t.ownerAddress)
		} else {
			stat.Display(t.ownerAddress, "ContractTrie")
		}
	}
	fmt.Printf("Contract Trie, total trie num: %v, ShortNodeCnt: %v, FullNodeCnt: %v, ValueNodeCnt: %v\n",
		contractTrieCnt, totalContactsNodeStat.ShortNodeCnt, totalContactsNodeStat.FullNodeCnt, totalContactsNodeStat.ValueNodeCnt)
}
