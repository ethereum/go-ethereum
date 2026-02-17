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
	inspectDumpRecordSize = 32 + trieStatLevels*3*4
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
		resolved, err := trie.resolveWithoutTrack(n, path)
		if err != nil {
			log.Error("Failed to resolve HashNode", "err", err, "trie", trie.Hash(), "height", height+1, "path", path)
			return
		}
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

	depthTop := newStorageTopN(config.TopN, compareStorageByDepth)
	totalTop := newStorageTopN(config.TopN, compareStorageByTotal)
	valueTop := newStorageTopN(config.TopN, compareStorageByValue)

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
		snapshot := newStorageSnapshot(record.Owner, record.Levels)
		if record.Owner == (common.Hash{}) {
			summary.Account = snapshot
			continue
		}
		summary.StorageCount++
		summary.DepthHistogram[snapshot.MaxDepth]++
		summary.StorageTotals.Short += snapshot.Summary.Short
		summary.StorageTotals.Full += snapshot.Summary.Full
		summary.StorageTotals.Value += snapshot.Summary.Value

		depthTop.TryInsert(snapshot)
		totalTop.TryInsert(snapshot)
		valueTop.TryInsert(snapshot)
	}
	if summary.Account == nil {
		return fmt.Errorf("dump file %s does not contain the account trie sentinel record", dumpPath)
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
		}
		off += 12
	}
	return record
}

type storageSnapshot struct {
	Owner      common.Hash
	Levels     [trieStatLevels]jsonLevel
	Summary    jsonLevel
	MaxDepth   int
	TotalNodes uint64
}

func newStorageSnapshot(owner common.Hash, levels [trieStatLevels]jsonLevel) *storageSnapshot {
	snapshot := &storageSnapshot{Owner: owner, Levels: levels}
	for i := 0; i < trieStatLevels; i++ {
		level := levels[i]
		if level.Short != 0 || level.Full != 0 || level.Value != 0 {
			snapshot.MaxDepth = i
		}
		snapshot.Summary.Short += level.Short
		snapshot.Summary.Full += level.Full
		snapshot.Summary.Value += level.Value
	}
	snapshot.TotalNodes = snapshot.Summary.Short + snapshot.Summary.Full + snapshot.Summary.Value
	return snapshot
}

func (s *storageSnapshot) toLevelStats() *LevelStats {
	stats := NewLevelStats()
	for i := 0; i < trieStatLevels; i++ {
		stats.level[i].short.Store(s.Levels[i].Short)
		stats.level[i].full.Store(s.Levels[i].Full)
		stats.level[i].value.Store(s.Levels[i].Value)
	}
	return stats
}

type storageCompare func(a, b *storageSnapshot) int

type topStorage struct {
	limit int
	cmp   storageCompare
	heap  storageHeap
}

type storageHeap struct {
	items []*storageSnapshot
	cmp   storageCompare
}

func (h storageHeap) Len() int { return len(h.items) }

func (h storageHeap) Less(i, j int) bool {
	// Keep the weakest entry at the root (min-heap semantics).
	return h.cmp(h.items[i], h.items[j]) < 0
}

func (h storageHeap) Swap(i, j int) { h.items[i], h.items[j] = h.items[j], h.items[i] }

func (h *storageHeap) Push(x any) {
	h.items = append(h.items, x.(*storageSnapshot))
}

func (h *storageHeap) Pop() any {
	item := h.items[len(h.items)-1]
	h.items = h.items[:len(h.items)-1]
	return item
}

func newStorageTopN(limit int, cmp storageCompare) *topStorage {
	h := storageHeap{cmp: cmp}
	heap.Init(&h)
	return &topStorage{limit: limit, cmp: cmp, heap: h}
}

func (t *topStorage) TryInsert(item *storageSnapshot) {
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

func (t *topStorage) Sorted() []*storageSnapshot {
	out := append([]*storageSnapshot(nil), t.heap.items...)
	sort.Slice(out, func(i, j int) bool { return t.cmp(out[i], out[j]) > 0 })
	return out
}

func compareStorageByDepth(a, b *storageSnapshot) int {
	if cmp := compareInt(a.MaxDepth, b.MaxDepth); cmp != 0 {
		return cmp
	}
	if cmp := compareUint64(a.TotalNodes, b.TotalNodes); cmp != 0 {
		return cmp
	}
	if cmp := compareUint64(a.Summary.Value, b.Summary.Value); cmp != 0 {
		return cmp
	}
	return bytes.Compare(a.Owner[:], b.Owner[:])
}

func compareStorageByTotal(a, b *storageSnapshot) int {
	if cmp := compareUint64(a.TotalNodes, b.TotalNodes); cmp != 0 {
		return cmp
	}
	if cmp := compareInt(a.MaxDepth, b.MaxDepth); cmp != 0 {
		return cmp
	}
	if cmp := compareUint64(a.Summary.Value, b.Summary.Value); cmp != 0 {
		return cmp
	}
	return bytes.Compare(a.Owner[:], b.Owner[:])
}

func compareStorageByValue(a, b *storageSnapshot) int {
	if cmp := compareUint64(a.Summary.Value, b.Summary.Value); cmp != 0 {
		return cmp
	}
	if cmp := compareInt(a.MaxDepth, b.MaxDepth); cmp != 0 {
		return cmp
	}
	if cmp := compareUint64(a.TotalNodes, b.TotalNodes); cmp != 0 {
		return cmp
	}
	return bytes.Compare(a.Owner[:], b.Owner[:])
}

func compareInt(a, b int) int {
	switch {
	case a > b:
		return 1
	case a < b:
		return -1
	default:
		return 0
	}
}

func compareUint64(a, b uint64) int {
	switch {
	case a > b:
		return 1
	case a < b:
		return -1
	default:
		return 0
	}
}

type inspectSummary struct {
	Account         *storageSnapshot
	StorageCount    uint64
	StorageTotals   jsonLevel
	DepthHistogram  [trieStatLevels]uint64
	TopByDepth      []*storageSnapshot
	TopByTotalNodes []*storageSnapshot
	TopByValueNodes []*storageSnapshot
}

func (s *inspectSummary) display() {
	s.Account.toLevelStats().display("Accounts trie")
	fmt.Println("Storage trie aggregate summary")
	fmt.Printf("Total storage tries: %d\n", s.StorageCount)
	totalNodes := s.StorageTotals.Short + s.StorageTotals.Full + s.StorageTotals.Value
	fmt.Printf("Total nodes: %d\n", totalNodes)
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

func (s *inspectSummary) displayTop(title string, list []*storageSnapshot) {
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

func (s *inspectSummary) writeJSON(path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(s.toJSON())
}

func (s *inspectSummary) toJSON() *jsonSummary {
	jsonSummary := &jsonSummary{
		AccountTrie: newJsonStat(s.Account.toLevelStats(), "account trie"),
		StorageSummary: jsonStorageSummary{
			TotalStorageTries: s.StorageCount,
			Totals:            s.StorageTotals,
			DepthHistogram:    s.DepthHistogram,
		},
		TopByDepth:      snapshotsToJSON(s.TopByDepth),
		TopByTotalNodes: snapshotsToJSON(s.TopByTotalNodes),
		TopByValueNodes: snapshotsToJSON(s.TopByValueNodes),
	}
	return jsonSummary
}

type jsonSummary struct {
	AccountTrie     *jsonStat
	StorageSummary  jsonStorageSummary
	TopByDepth      []jsonStorageStat
	TopByTotalNodes []jsonStorageStat
	TopByValueNodes []jsonStorageStat
}

type jsonStorageSummary struct {
	TotalStorageTries uint64
	Totals            jsonLevel
	DepthHistogram    [trieStatLevels]uint64
}

type jsonStorageStat struct {
	Owner      string
	MaxDepth   int
	TotalNodes uint64
	ValueNodes uint64
	Levels     []jsonLevel
	Summary    jsonLevel
}

func snapshotsToJSON(list []*storageSnapshot) []jsonStorageStat {
	out := make([]jsonStorageStat, 0, len(list))
	for _, item := range list {
		stat := newJsonStat(item.toLevelStats(), item.Owner.Hex())
		out = append(out, jsonStorageStat{
			Owner:      item.Owner.Hex(),
			MaxDepth:   item.MaxDepth,
			TotalNodes: item.TotalNodes,
			ValueNodes: item.Summary.Value,
			Levels:     stat.Levels,
			Summary:    stat.Summary,
		})
	}
	return out
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
		short, full, value := s.level[i].load()
		table.AppendBulk([][]string{{"-", fmt.Sprint(i), fmt.Sprint(short), fmt.Sprint(full), fmt.Sprint(value)}})
		stat.add(&s.level[i])
	}
	short, full, value := stat.load()
	table.SetFooter([]string{"Total", "", fmt.Sprint(short), fmt.Sprint(full), fmt.Sprint(value)})
	table.Render()
	fmt.Print(b.String())
	fmt.Println("Max depth", s.maxDepth())
	fmt.Println()
}

type jsonLevel struct {
	Short uint64
	Full  uint64
	Value uint64
}

type jsonStat struct {
	Name    string
	Levels  []jsonLevel
	Summary jsonLevel
}

func newJsonStat(s *LevelStats, name string) *jsonStat {
	ret := jsonStat{Name: name, Summary: jsonLevel{}}
	for i := 0; i < len(s.level); i++ {
		if s.level[i].empty() {
			continue
		}
		level := jsonLevel{
			Short: s.level[i].short.Load(),
			Full:  s.level[i].full.Load(),
			Value: s.level[i].value.Load(),
		}
		ret.Summary.Full += level.Full
		ret.Summary.Short += level.Short
		ret.Summary.Value += level.Value
		ret.Levels = append(ret.Levels, level)
	}
	return &ret
}
