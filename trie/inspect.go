// Copyright 2025 The go-ethereum Authors
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

package trie

import (
	"bufio"
	"bytes"
	"cmp"
	"container/heap"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/tablewriter"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/triedb/database"
	"golang.org/x/sync/semaphore"
)

const (
	inspectDumpRecordSize = 32 + trieStatLevels*(3*4+8)
	inspectDefaultTopN    = 10
	inspectParallelism    = int64(16)
)

// inspector is used by the inner inspect function to coordinate across threads.
type inspector struct {
	triedb database.NodeDatabase
	root   common.Hash

	config      *InspectConfig
	accountStat *LevelStats

	sem *semaphore.Weighted
	wg  sync.WaitGroup

	// Pass 1: dump file writer.
	dumpMu   sync.Mutex
	dumpBuf  *bufio.Writer
	dumpFile *os.File

	errMu sync.Mutex
	err   error
}

// InspectConfig is a set of options to control inspection and format the output.
// TopN determines the maximum number of entries retained for each top-list.
// Path controls optional JSON output. DumpPath controls the pass-1 dump location.
type InspectConfig struct {
	NoStorage bool
	TopN      int
	Path      string
	DumpPath  string
}

// Inspect walks the trie with the given root and records the number and type of
// nodes at each depth. Storage trie stats are streamed to disk in fixed-size
// records, then summarized in a second pass.
func Inspect(triedb database.NodeDatabase, root common.Hash, config *InspectConfig) error {
	trie, err := New(TrieID(root), triedb)
	if err != nil {
		return fmt.Errorf("fail to open trie %s: %w", root, err)
	}
	config = normalizeInspectConfig(config)

	dumpFile, err := os.OpenFile(config.DumpPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("failed to create trie dump %s: %w", config.DumpPath, err)
	}
	in := inspector{
		triedb:      triedb,
		root:        root,
		config:      config,
		accountStat: NewLevelStats(),
		sem:         semaphore.NewWeighted(inspectParallelism),
		dumpBuf:     bufio.NewWriterSize(dumpFile, 1<<20),
		dumpFile:    dumpFile,
	}

	in.recordRootSize(trie, root, in.accountStat)
	in.inspect(trie, trie.root, 0, []byte{}, in.accountStat)
	in.wg.Wait()

	// Persist account trie stats as the sentinel record.
	in.writeDumpRecord(common.Hash{}, in.accountStat)
	if err := in.closeDump(); err != nil {
		in.setError(err)
	}
	if err := in.getError(); err != nil {
		return err
	}
	return Summarize(config.DumpPath, config)
}

func normalizeInspectConfig(config *InspectConfig) *InspectConfig {
	if config == nil {
		config = &InspectConfig{}
	}
	if config.TopN <= 0 {
		config.TopN = inspectDefaultTopN
	}
	if config.DumpPath == "" {
		config.DumpPath = "trie-dump.bin"
	}
	return config
}

func (in *inspector) recordRootSize(trie *Trie, root common.Hash, stat *LevelStats) {
	if root == (common.Hash{}) || root == types.EmptyRootHash {
		return
	}
	blob := trie.prevalueTracer.Get(nil)
	if len(blob) == 0 {
		resolved, err := trie.reader.Node(nil, root)
		if err != nil {
			log.Error("Failed to read trie root for size accounting", "trie", trie.Hash(), "root", root, "err", err)
			return
		}
		blob = resolved
	}
	stat.addSize(0, uint64(len(blob)))
}

func (in *inspector) closeDump() error {
	var ret error
	if in.dumpBuf != nil {
		if err := in.dumpBuf.Flush(); err != nil {
			ret = errors.Join(ret, fmt.Errorf("failed to flush trie dump %s: %w", in.config.DumpPath, err))
		}
	}
	if in.dumpFile != nil {
		if err := in.dumpFile.Close(); err != nil {
			ret = errors.Join(ret, fmt.Errorf("failed to close trie dump %s: %w", in.config.DumpPath, err))
		}
	}
	return ret
}

func (in *inspector) setError(err error) {
	if err == nil {
		return
	}
	in.errMu.Lock()
	defer in.errMu.Unlock()
	in.err = errors.Join(in.err, err)
}

func (in *inspector) getError() error {
	in.errMu.Lock()
	defer in.errMu.Unlock()
	return in.err
}

func (in *inspector) hasError() bool {
	return in.getError() != nil
}

func (in *inspector) spawn(fn func()) bool {
	if !in.sem.TryAcquire(1) {
		return false
	}
	in.wg.Add(1)
	go func() {
		defer in.sem.Release(1)
		defer in.wg.Done()
		fn()
	}()
	return true
}

func (in *inspector) writeDumpRecord(owner common.Hash, s *LevelStats) {
	if in.hasError() {
		return
	}
	var buf [inspectDumpRecordSize]byte
	copy(buf[:32], owner[:])

	off := 32
	for i := 0; i < trieStatLevels; i++ {
		binary.LittleEndian.PutUint32(buf[off:], uint32(s.level[i].short.Load()))
		off += 4
		binary.LittleEndian.PutUint32(buf[off:], uint32(s.level[i].full.Load()))
		off += 4
		binary.LittleEndian.PutUint32(buf[off:], uint32(s.level[i].value.Load()))
		off += 4
		binary.LittleEndian.PutUint64(buf[off:], s.level[i].size.Load())
		off += 8
	}
	in.dumpMu.Lock()
	_, err := in.dumpBuf.Write(buf[:])
	in.dumpMu.Unlock()
	if err != nil {
		in.setError(fmt.Errorf("failed writing trie dump record: %w", err))
	}
}

// inspect is called recursively down the trie. At each level it records the
// node type encountered.
func (in *inspector) inspect(trie *Trie, n node, height uint32, path []byte, stat *LevelStats) {
	if n == nil {
		return
	}

	// Four types of nodes can be encountered:
	// - short: extend path with key, inspect single value.
	// - full: inspect all 17 children, spin up new threads when possible.
	// - hash: need to resolve node from disk, retry inspect on result.
	// - value: if account, begin inspecting storage trie.
	switch n := (n).(type) {
	case *shortNode:
		nextPath := slices.Concat(path, n.Key)
		in.inspect(trie, n.Val, height+1, nextPath, stat)
	case *fullNode:
		for idx, child := range n.Children {
			if child == nil {
				continue
			}
			childPath := slices.Concat(path, []byte{byte(idx)})
			childNode := child
			if in.spawn(func() {
				in.inspect(trie, childNode, height+1, childPath, stat)
			}) {
				continue
			}
			in.inspect(trie, childNode, height+1, childPath, stat)
		}
	case hashNode:
		blob, err := trie.reader.Node(path, common.BytesToHash(n))
		if err != nil {
			log.Error("Failed to resolve HashNode", "err", err, "trie", trie.Hash(), "height", height+1, "path", path)
			return
		}
		stat.addSize(height, uint64(len(blob)))
		resolved := mustDecodeNode(n, blob)
		in.inspect(trie, resolved, height, path, stat)

		// Return early here so this level isn't recorded twice.
		return
	case valueNode:
		if !hasTerm(path) {
			break
		}
		var account types.StateAccount
		if err := rlp.Decode(bytes.NewReader(n), &account); err != nil {
			// Not an account value.
			break
		}
		if account.Root == (common.Hash{}) || account.Root == types.EmptyRootHash {
			// Account is empty, nothing further to inspect.
			break
		}

		if !in.config.NoStorage {
			owner := common.BytesToHash(hexToCompact(path))
			storage, err := New(StorageTrieID(in.root, owner, account.Root), in.triedb)
			if err != nil {
				log.Error("Failed to open account storage trie", "node", n, "error", err, "height", height, "path", common.Bytes2Hex(path))
				break
			}
			storageStat := NewLevelStats()
			run := func() {
				in.recordRootSize(storage, account.Root, storageStat)
				in.inspect(storage, storage.root, 0, []byte{}, storageStat)
				in.writeDumpRecord(owner, storageStat)
			}
			if in.spawn(run) {
				break
			}
			run()
		}
	default:
		panic(fmt.Sprintf("%T: invalid node: %v", n, n))
	}

	// Record stats for current height.
	stat.add(n, height)
}

// Summarize performs pass 2 over a trie dump and reports account stats,
// aggregate storage statistics, and top-N rankings.
func Summarize(dumpPath string, config *InspectConfig) error {
	config = normalizeInspectConfig(config)
	if dumpPath == "" {
		dumpPath = config.DumpPath
	}
	if dumpPath == "" {
		return errors.New("missing dump path")
	}
	file, err := os.Open(dumpPath)
	if err != nil {
		return fmt.Errorf("failed to open trie dump %s: %w", dumpPath, err)
	}
	defer file.Close()

	if info, err := file.Stat(); err == nil {
		if info.Size()%inspectDumpRecordSize != 0 {
			return fmt.Errorf("invalid trie dump size %d (not a multiple of %d)", info.Size(), inspectDumpRecordSize)
		}
	}

	depthTop := newStorageStatsTopN(config.TopN, compareStorageStatsByDepth)
	totalTop := newStorageStatsTopN(config.TopN, compareStorageStatsByTotal)
	valueTop := newStorageStatsTopN(config.TopN, compareStorageStatsByValue)

	summary := &inspectSummary{}
	reader := bufio.NewReaderSize(file, 1<<20)
	var buf [inspectDumpRecordSize]byte

	for {
		_, err := io.ReadFull(reader, buf[:])
		if errors.Is(err, io.EOF) {
			break
		}
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return fmt.Errorf("truncated trie dump %s", dumpPath)
		}
		if err != nil {
			return fmt.Errorf("failed reading trie dump %s: %w", dumpPath, err)
		}

		record := decodeDumpRecord(buf[:])
		snapshot := newStorageStats(record.Owner, record.Levels)
		if record.Owner == (common.Hash{}) {
			summary.Account = snapshot
			continue
		}
		summary.StorageCount++
		summary.DepthHistogram[snapshot.MaxDepth]++
		for i := 0; i < trieStatLevels; i++ {
			summary.StorageLevels[i].Short += record.Levels[i].Short
			summary.StorageLevels[i].Full += record.Levels[i].Full
			summary.StorageLevels[i].Value += record.Levels[i].Value
			summary.StorageLevels[i].Size += record.Levels[i].Size
		}

		depthTop.TryInsert(snapshot)
		totalTop.TryInsert(snapshot)
		valueTop.TryInsert(snapshot)
	}
	if summary.Account == nil {
		return fmt.Errorf("dump file %s does not contain the account trie sentinel record", dumpPath)
	}
	for i := 0; i < trieStatLevels; i++ {
		summary.StorageTotals.Short += summary.StorageLevels[i].Short
		summary.StorageTotals.Full += summary.StorageLevels[i].Full
		summary.StorageTotals.Value += summary.StorageLevels[i].Value
		summary.StorageTotals.Size += summary.StorageLevels[i].Size
	}
	summary.TopByDepth = depthTop.Sorted()
	summary.TopByTotalNodes = totalTop.Sorted()
	summary.TopByValueNodes = valueTop.Sorted()

	if config.Path != "" {
		return summary.writeJSON(config.Path)
	}
	summary.display()
	return nil
}

type dumpRecord struct {
	Owner  common.Hash
	Levels [trieStatLevels]jsonLevel
}

func decodeDumpRecord(raw []byte) dumpRecord {
	var (
		record dumpRecord
		off    = 32
	)
	copy(record.Owner[:], raw[:32])
	for i := 0; i < trieStatLevels; i++ {
		record.Levels[i] = jsonLevel{
			Short: uint64(binary.LittleEndian.Uint32(raw[off:])),
			Full:  uint64(binary.LittleEndian.Uint32(raw[off+4:])),
			Value: uint64(binary.LittleEndian.Uint32(raw[off+8:])),
			Size:  binary.LittleEndian.Uint64(raw[off+12:]),
		}
		off += 20
	}
	return record
}

type storageStats struct {
	Owner      common.Hash
	Levels     [trieStatLevels]jsonLevel
	Summary    jsonLevel
	MaxDepth   int
	TotalNodes uint64
	TotalSize  uint64
}

func newStorageStats(owner common.Hash, levels [trieStatLevels]jsonLevel) *storageStats {
	snapshot := &storageStats{Owner: owner, Levels: levels}
	for i := 0; i < trieStatLevels; i++ {
		level := levels[i]
		if level.Short != 0 || level.Full != 0 || level.Value != 0 {
			snapshot.MaxDepth = i
		}
		snapshot.Summary.Short += level.Short
		snapshot.Summary.Full += level.Full
		snapshot.Summary.Value += level.Value
		snapshot.Summary.Size += level.Size
	}
	snapshot.TotalNodes = snapshot.Summary.Short + snapshot.Summary.Full + snapshot.Summary.Value
	snapshot.TotalSize = snapshot.Summary.Size
	return snapshot
}

func trimLevels(levels [trieStatLevels]jsonLevel) []jsonLevel {
	n := len(levels)
	for n > 0 && levels[n-1] == (jsonLevel{}) {
		n--
	}
	return levels[:n]
}

func (s *storageStats) MarshalJSON() ([]byte, error) {
	type jsonStorageSnapshot struct {
		Owner      common.Hash `json:"Owner"`
		MaxDepth   int         `json:"MaxDepth"`
		TotalNodes uint64      `json:"TotalNodes"`
		TotalSize  uint64      `json:"TotalSize"`
		ValueNodes uint64      `json:"ValueNodes"`
		Levels     []jsonLevel `json:"Levels"`
		Summary    jsonLevel   `json:"Summary"`
	}
	return json.Marshal(jsonStorageSnapshot{
		Owner:      s.Owner,
		MaxDepth:   s.MaxDepth,
		TotalNodes: s.TotalNodes,
		TotalSize:  s.TotalSize,
		ValueNodes: s.Summary.Value,
		Levels:     trimLevels(s.Levels),
		Summary:    s.Summary,
	})
}

func (s *storageStats) toLevelStats() *LevelStats {
	stats := NewLevelStats()
	for i := 0; i < trieStatLevels; i++ {
		stats.level[i].short.Store(s.Levels[i].Short)
		stats.level[i].full.Store(s.Levels[i].Full)
		stats.level[i].value.Store(s.Levels[i].Value)
		stats.level[i].size.Store(s.Levels[i].Size)
	}
	return stats
}

type storageStatsCompare func(a, b *storageStats) int

type storageStatsTopN struct {
	limit int
	cmp   storageStatsCompare
	heap  storageStatsHeap
}

type storageStatsHeap struct {
	items []*storageStats
	cmp   storageStatsCompare
}

func (h storageStatsHeap) Len() int { return len(h.items) }

func (h storageStatsHeap) Less(i, j int) bool {
	// Keep the weakest entry at the root (min-heap semantics).
	return h.cmp(h.items[i], h.items[j]) < 0
}

func (h storageStatsHeap) Swap(i, j int) { h.items[i], h.items[j] = h.items[j], h.items[i] }

func (h *storageStatsHeap) Push(x any) {
	h.items = append(h.items, x.(*storageStats))
}

func (h *storageStatsHeap) Pop() any {
	item := h.items[len(h.items)-1]
	h.items = h.items[:len(h.items)-1]
	return item
}

func newStorageStatsTopN(limit int, cmp storageStatsCompare) *storageStatsTopN {
	h := storageStatsHeap{cmp: cmp}
	heap.Init(&h)
	return &storageStatsTopN{limit: limit, cmp: cmp, heap: h}
}

func (t *storageStatsTopN) TryInsert(item *storageStats) {
	if t.limit <= 0 {
		return
	}
	if t.heap.Len() < t.limit {
		heap.Push(&t.heap, item)
		return
	}
	if t.cmp(item, t.heap.items[0]) <= 0 {
		return
	}
	heap.Pop(&t.heap)
	heap.Push(&t.heap, item)
}

func (t *storageStatsTopN) Sorted() []*storageStats {
	out := append([]*storageStats(nil), t.heap.items...)
	sort.Slice(out, func(i, j int) bool { return t.cmp(out[i], out[j]) > 0 })
	return out
}

func compareStorageStatsByDepth(a, b *storageStats) int {
	return cmp.Or(
		cmp.Compare(a.MaxDepth, b.MaxDepth),
		cmp.Compare(a.TotalNodes, b.TotalNodes),
		cmp.Compare(a.Summary.Value, b.Summary.Value),
		bytes.Compare(a.Owner[:], b.Owner[:]),
	)
}

func compareStorageStatsByTotal(a, b *storageStats) int {
	return cmp.Or(
		cmp.Compare(a.TotalNodes, b.TotalNodes),
		cmp.Compare(a.MaxDepth, b.MaxDepth),
		cmp.Compare(a.Summary.Value, b.Summary.Value),
		bytes.Compare(a.Owner[:], b.Owner[:]),
	)
}

func compareStorageStatsByValue(a, b *storageStats) int {
	return cmp.Or(
		cmp.Compare(a.Summary.Value, b.Summary.Value),
		cmp.Compare(a.MaxDepth, b.MaxDepth),
		cmp.Compare(a.TotalNodes, b.TotalNodes),
		bytes.Compare(a.Owner[:], b.Owner[:]),
	)
}

type inspectSummary struct {
	Account         *storageStats
	StorageCount    uint64
	StorageTotals   jsonLevel
	StorageLevels   [trieStatLevels]jsonLevel
	DepthHistogram  [trieStatLevels]uint64
	TopByDepth      []*storageStats
	TopByTotalNodes []*storageStats
	TopByValueNodes []*storageStats
}

func (s *inspectSummary) display() {
	s.displayCombinedDepthTable()
	s.Account.toLevelStats().display("Accounts trie")
	fmt.Println("Storage trie aggregate summary")
	fmt.Printf("Total storage tries: %d\n", s.StorageCount)
	totalNodes := s.StorageTotals.Short + s.StorageTotals.Full + s.StorageTotals.Value
	fmt.Printf("Total nodes: %d\n", totalNodes)
	fmt.Printf("Total size: %s\n", common.StorageSize(s.StorageTotals.Size))
	fmt.Printf("  Short nodes: %d\n", s.StorageTotals.Short)
	fmt.Printf("  Full nodes:  %d\n", s.StorageTotals.Full)
	fmt.Printf("  Value nodes: %d\n", s.StorageTotals.Value)

	b := new(strings.Builder)
	table := tablewriter.NewWriter(b)
	table.SetHeader([]string{"Max Depth", "Storage Tries"})
	for i, count := range s.DepthHistogram {
		table.AppendBulk([][]string{{fmt.Sprint(i), fmt.Sprint(count)}})
	}
	table.Render()
	fmt.Print(b.String())
	fmt.Println()

	s.displayTop("Top storage tries by max depth", s.TopByDepth)
	s.displayTop("Top storage tries by total node count", s.TopByTotalNodes)
	s.displayTop("Top storage tries by value (slot) count", s.TopByValueNodes)
}

func (s *inspectSummary) displayCombinedDepthTable() {
	accountTotal := s.Account.Summary.Short + s.Account.Summary.Full + s.Account.Summary.Value
	storageTotal := s.StorageTotals.Short + s.StorageTotals.Full + s.StorageTotals.Value
	accountTotalSize := s.Account.Summary.Size
	storageTotalSize := s.StorageTotals.Size

	fmt.Println("Trie Depth Distribution")
	fmt.Printf("Account Trie: %d nodes (%s)\n", accountTotal, common.StorageSize(accountTotalSize))
	fmt.Printf("Storage Tries: %d nodes (%s) across %d tries\n", storageTotal, common.StorageSize(storageTotalSize), s.StorageCount)

	b := new(strings.Builder)
	table := tablewriter.NewWriter(b)
	table.SetHeader([]string{"Depth", "Account Nodes", "Account Size", "Storage Nodes", "Storage Size"})
	for i := 0; i < trieStatLevels; i++ {
		accountNodes := s.Account.Levels[i].Short + s.Account.Levels[i].Full + s.Account.Levels[i].Value
		accountSize := s.Account.Levels[i].Size
		storageNodes := s.StorageLevels[i].Short + s.StorageLevels[i].Full + s.StorageLevels[i].Value
		storageSize := s.StorageLevels[i].Size
		if accountNodes == 0 && storageNodes == 0 {
			continue
		}
		table.AppendBulk([][]string{{
			fmt.Sprint(i),
			fmt.Sprint(accountNodes),
			common.StorageSize(accountSize).String(),
			fmt.Sprint(storageNodes),
			common.StorageSize(storageSize).String(),
		}})
	}
	table.Render()
	fmt.Print(b.String())
	fmt.Println()
}

func (s *inspectSummary) displayTop(title string, list []*storageStats) {
	fmt.Println(title)
	if len(list) == 0 {
		fmt.Println("No storage tries found")
		fmt.Println()
		return
	}
	for i, item := range list {
		fmt.Printf("%d: %s\n", i+1, item.Owner)
		item.toLevelStats().display("storage trie")
	}
}

func (s *inspectSummary) MarshalJSON() ([]byte, error) {
	type jsonAccountTrie struct {
		Name    string      `json:"Name"`
		Levels  []jsonLevel `json:"Levels"`
		Summary jsonLevel   `json:"Summary"`
	}
	type jsonStorageSummary struct {
		TotalStorageTries uint64                 `json:"TotalStorageTries"`
		Totals            jsonLevel              `json:"Totals"`
		Levels            []jsonLevel            `json:"Levels"`
		DepthHistogram    [trieStatLevels]uint64 `json:"DepthHistogram"`
	}
	type jsonInspectSummary struct {
		AccountTrie     jsonAccountTrie    `json:"AccountTrie"`
		StorageSummary  jsonStorageSummary `json:"StorageSummary"`
		TopByDepth      []*storageStats    `json:"TopByDepth"`
		TopByTotalNodes []*storageStats    `json:"TopByTotalNodes"`
		TopByValueNodes []*storageStats    `json:"TopByValueNodes"`
	}
	return json.Marshal(jsonInspectSummary{
		AccountTrie: jsonAccountTrie{
			Name:    "account trie",
			Levels:  trimLevels(s.Account.Levels),
			Summary: s.Account.Summary,
		},
		StorageSummary: jsonStorageSummary{
			TotalStorageTries: s.StorageCount,
			Totals:            s.StorageTotals,
			Levels:            trimLevels(s.StorageLevels),
			DepthHistogram:    s.DepthHistogram,
		},
		TopByDepth:      s.TopByDepth,
		TopByTotalNodes: s.TopByTotalNodes,
		TopByValueNodes: s.TopByValueNodes,
	})
}

func (s *inspectSummary) writeJSON(path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(s)
}

// display will print a table displaying the trie's node statistics.
func (s *LevelStats) display(title string) {
	// Shorten title if too long.
	if len(title) > 32 {
		title = title[0:8] + "..." + title[len(title)-8:]
	}

	b := new(strings.Builder)
	table := tablewriter.NewWriter(b)
	table.SetHeader([]string{title, "Level", "Short Nodes", "Full Node", "Value Node"})

	stat := &stat{}
	for i := range s.level {
		if s.level[i].empty() {
			continue
		}
		short, full, value, _ := s.level[i].load()
		table.AppendBulk([][]string{{"-", fmt.Sprint(i), fmt.Sprint(short), fmt.Sprint(full), fmt.Sprint(value)}})
		stat.add(&s.level[i])
	}
	short, full, value, _ := stat.load()
	table.SetFooter([]string{"Total", "", fmt.Sprint(short), fmt.Sprint(full), fmt.Sprint(value)})
	table.Render()
	fmt.Print(b.String())
	fmt.Println("Max depth", s.MaxDepth())
	fmt.Println()
}

type jsonLevel struct {
	Short uint64
	Full  uint64
	Value uint64
	Size  uint64
}
