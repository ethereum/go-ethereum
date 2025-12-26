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

package live

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

func init() {
	tracers.LiveDirectory.Register("statesize", newStateSizeTracer)
}

// Database key size constants matching core/state/state_sizer.go
var (
	accountKeySize            = int64(len(rawdb.SnapshotAccountPrefix) + common.HashLength)
	storageKeySize            = int64(len(rawdb.SnapshotStoragePrefix) + common.HashLength*2)
	accountTrienodePrefixSize = int64(len(rawdb.TrieNodeAccountPrefix))
	storageTrienodePrefixSize = int64(len(rawdb.TrieNodeStoragePrefix) + common.HashLength)
	codeKeySize               = int64(len(rawdb.CodePrefix) + common.HashLength)
)

// CSV column headers (cumulative only)
var csvHeaders = []string{
	"block_number",
	"root",
	"parent_root",
	"accounts",
	"account_bytes",
	"storages",
	"storage_bytes",
	"account_trienodes",
	"account_trienode_bytes",
	"storage_trienodes",
	"storage_trienode_bytes",
	"codes",
	"code_bytes",
}

// depthCSVHeaders generates headers for the depth CSV files.
// Format: block_number, root, parent_root, total_nodes, account_depth_0..64, storage_depth_0..64
func depthCSVHeaders() []string {
	headers := make([]string, 0, 4+65+65)
	headers = append(headers, "block_number", "root", "parent_root", "total_nodes")
	for i := 0; i <= 64; i++ {
		headers = append(headers, fmt.Sprintf("account_depth_%d", i))
	}
	for i := 0; i <= 64; i++ {
		headers = append(headers, fmt.Sprintf("storage_depth_%d", i))
	}
	return headers
}

// stateSizeStats represents cumulative state size statistics.
type stateSizeStats struct {
	Accounts             int64
	AccountBytes         int64
	Storages             int64
	StorageBytes         int64
	AccountTrienodes     int64
	AccountTrienodeBytes int64
	StorageTrienodes     int64
	StorageTrienodeBytes int64
	Codes                int64
	CodeBytes            int64
}

// stateSizeRecord represents a single CSV record with cumulative stats.
type stateSizeRecord struct {
	BlockNumber uint64
	Root        common.Hash
	ParentRoot  common.Hash
	Stats       stateSizeStats
}

type stateSizeTracer struct {
	mu       sync.Mutex
	file     *os.File
	writer   *csv.Writer
	filePath string

	// Depth tracking - separate files for created and deleted nodes
	depthCreatedFile     *os.File
	depthCreatedWriter   *csv.Writer
	depthCreatedFilePath string

	depthDeletedFile     *os.File
	depthDeletedWriter   *csv.Writer
	depthDeletedFilePath string

	// Map from state root to cumulative stats (for handling forks)
	stats map[common.Hash]stateSizeStats
}

type stateSizeTracerConfig struct {
	Path string `json:"path"` // Path to the directory where the tracer logs will be stored
}

func newStateSizeTracer(cfg json.RawMessage) (*tracing.Hooks, error) {
	var config stateSizeTracerConfig
	if err := json.Unmarshal(cfg, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}
	if config.Path == "" {
		return nil, errors.New("statesize tracer output path is required")
	}

	filePath := filepath.Join(config.Path, "statesize.csv")
	depthCreatedFilePath := filepath.Join(config.Path, "statesize_depth_created.csv")
	depthDeletedFilePath := filepath.Join(config.Path, "statesize_depth_deleted.csv")

	// Ensure the directory exists
	if err := os.MkdirAll(config.Path, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create statesize directory: %v", err)
	}

	t := &stateSizeTracer{
		filePath:             filePath,
		depthCreatedFilePath: depthCreatedFilePath,
		depthDeletedFilePath: depthDeletedFilePath,
		stats:                make(map[common.Hash]stateSizeStats),
	}

	// Load existing data if file exists
	if err := t.loadExisting(); err != nil {
		return nil, fmt.Errorf("failed to load existing statesize data: %v", err)
	}

	// Open statesize file for appending
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open statesize file: %v", err)
	}
	t.file = file
	t.writer = csv.NewWriter(file)

	// Write header if file is new (empty)
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat statesize file: %v", err)
	}
	if info.Size() == 0 {
		if err := t.writer.Write(csvHeaders); err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to write CSV headers: %v", err)
		}
		t.writer.Flush()
	}

	// Open depth created file for appending
	depthCreatedFile, err := os.OpenFile(depthCreatedFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to open depth created file: %v", err)
	}
	t.depthCreatedFile = depthCreatedFile
	t.depthCreatedWriter = csv.NewWriter(depthCreatedFile)

	// Write header if depth created file is new (empty)
	depthCreatedInfo, err := depthCreatedFile.Stat()
	if err != nil {
		file.Close()
		depthCreatedFile.Close()
		return nil, fmt.Errorf("failed to stat depth created file: %v", err)
	}
	if depthCreatedInfo.Size() == 0 {
		if err := t.depthCreatedWriter.Write(depthCSVHeaders()); err != nil {
			file.Close()
			depthCreatedFile.Close()
			return nil, fmt.Errorf("failed to write depth created CSV headers: %v", err)
		}
		t.depthCreatedWriter.Flush()
	}

	// Open depth deleted file for appending
	depthDeletedFile, err := os.OpenFile(depthDeletedFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		file.Close()
		depthCreatedFile.Close()
		return nil, fmt.Errorf("failed to open depth deleted file: %v", err)
	}
	t.depthDeletedFile = depthDeletedFile
	t.depthDeletedWriter = csv.NewWriter(depthDeletedFile)

	// Write header if depth deleted file is new (empty)
	depthDeletedInfo, err := depthDeletedFile.Stat()
	if err != nil {
		file.Close()
		depthCreatedFile.Close()
		depthDeletedFile.Close()
		return nil, fmt.Errorf("failed to stat depth deleted file: %v", err)
	}
	if depthDeletedInfo.Size() == 0 {
		if err := t.depthDeletedWriter.Write(depthCSVHeaders()); err != nil {
			file.Close()
			depthCreatedFile.Close()
			depthDeletedFile.Close()
			return nil, fmt.Errorf("failed to write depth deleted CSV headers: %v", err)
		}
		t.depthDeletedWriter.Flush()
	}

	return &tracing.Hooks{
		OnStateUpdate: t.onStateUpdate,
		OnClose:       t.onClose,
	}, nil
}

// loadExisting reads the existing CSV file and loads the latest records by block number.
func (s *stateSizeTracer) loadExisting() error {
	file, err := os.Open(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, nothing to load
		}
		return err
	}
	defer file.Close()

	reader := csv.NewReader(bufio.NewReader(file))

	// Read and skip header
	if _, err := reader.Read(); err != nil {
		if err == io.EOF {
			return nil // Empty file
		}
		return fmt.Errorf("failed to read CSV header: %v", err)
	}

	// Read all records to find the latest block number
	var (
		latestBlockNum uint64
		latestRecords  = make(map[common.Hash]stateSizeStats)
	)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read CSV record: %v", err)
		}
		if len(record) < len(csvHeaders) {
			continue // Skip malformed records
		}

		blockNum, err := strconv.ParseUint(record[0], 10, 64)
		if err != nil {
			continue
		}

		// If we found a new latest block, clear the map
		if blockNum > latestBlockNum {
			latestBlockNum = blockNum
			latestRecords = make(map[common.Hash]stateSizeStats)
		}

		// Only store records from the latest block
		if blockNum == latestBlockNum {
			root := common.HexToHash(record[1])
			stats, err := parseStats(record)
			if err != nil {
				log.Warn("Failed to parse statesize record", "error", err)
				continue
			}
			latestRecords[root] = stats
		}
	}

	s.stats = latestRecords
	if len(latestRecords) > 0 {
		log.Info("Loaded statesize tracer state", "block", latestBlockNum, "roots", len(latestRecords))
	}
	return nil
}

// parseStats extracts cumulative statistics from a CSV record.
func parseStats(record []string) (stateSizeStats, error) {
	if len(record) < len(csvHeaders) {
		return stateSizeStats{}, errors.New("record too short")
	}

	// Cumulative columns start at index 3
	stats := stateSizeStats{}
	var err error

	stats.Accounts, err = strconv.ParseInt(record[3], 10, 64)
	if err != nil {
		return stats, err
	}
	stats.AccountBytes, err = strconv.ParseInt(record[4], 10, 64)
	if err != nil {
		return stats, err
	}
	stats.Storages, err = strconv.ParseInt(record[5], 10, 64)
	if err != nil {
		return stats, err
	}
	stats.StorageBytes, err = strconv.ParseInt(record[6], 10, 64)
	if err != nil {
		return stats, err
	}
	stats.AccountTrienodes, err = strconv.ParseInt(record[7], 10, 64)
	if err != nil {
		return stats, err
	}
	stats.AccountTrienodeBytes, err = strconv.ParseInt(record[8], 10, 64)
	if err != nil {
		return stats, err
	}
	stats.StorageTrienodes, err = strconv.ParseInt(record[9], 10, 64)
	if err != nil {
		return stats, err
	}
	stats.StorageTrienodeBytes, err = strconv.ParseInt(record[10], 10, 64)
	if err != nil {
		return stats, err
	}
	stats.Codes, err = strconv.ParseInt(record[11], 10, 64)
	if err != nil {
		return stats, err
	}
	stats.CodeBytes, err = strconv.ParseInt(record[12], 10, 64)
	if err != nil {
		return stats, err
	}

	return stats, nil
}

func (s *stateSizeTracer) onStateUpdate(update *tracing.StateUpdate) {
	if update == nil {
		return
	}

	// Calculate deltas
	var (
		accountsDelta         int64
		accountBytesDelta     int64
		storagesDelta         int64
		storageBytesDelta     int64
		accountTrienodesDelta int64
		accountTrienodeBytes  int64
		storageTrienodesDelta int64
		storageTrienodeBytes  int64
		codesDelta            int64
		codeBytesDelta        int64
	)

	// Calculate account size changes
	for _, change := range update.AccountChanges {
		prevLen := slimAccountSize(change.Prev)
		newLen := slimAccountSize(change.New)

		switch {
		case prevLen > 0 && newLen == 0:
			accountsDelta--
			accountBytesDelta -= accountKeySize + int64(prevLen)
		case prevLen == 0 && newLen > 0:
			accountsDelta++
			accountBytesDelta += accountKeySize + int64(newLen)
		default:
			accountBytesDelta += int64(newLen - prevLen)
		}
	}

	encode := func(val common.Hash) []byte {
		if val == (common.Hash{}) {
			return nil
		}
		blob, _ := rlp.EncodeToBytes(common.TrimLeftZeroes(val[:]))
		return blob
	}

	// Calculate storage size changes
	for _, slots := range update.StorageChanges {
		for _, change := range slots {
			prevLen := len(encode(change.Prev))
			newLen := len(encode(change.New))

			switch {
			case prevLen > 0 && newLen == 0:
				storagesDelta--
				storageBytesDelta -= storageKeySize + int64(prevLen)
			case prevLen == 0 && newLen > 0:
				storagesDelta++
				storageBytesDelta += storageKeySize + int64(newLen)
			default:
				storageBytesDelta += int64(newLen - prevLen)
			}
		}
	}

	// Calculate trie node size changes and depth counts
	var (
		accountDepthCreated [65]int64
		storageDepthCreated [65]int64
		accountDepthDeleted [65]int64
		storageDepthDeleted [65]int64
	)

	for owner, nodes := range update.TrieChanges {
		var (
			keyPrefix int64
			isAccount = owner == (common.Hash{})
		)
		if isAccount {
			keyPrefix = accountTrienodePrefixSize
		} else {
			keyPrefix = storageTrienodePrefixSize
		}

		// Calculate depth counts for created/modified and deleted nodes
		createdCounts, deletedCounts := calculateDepthCountsByType(nodes)

		for path, change := range nodes {
			var prevLen, newLen int
			if change.Prev != nil {
				prevLen = len(change.Prev.Blob)
			}
			if change.New != nil {
				newLen = len(change.New.Blob)
			}
			keySize := keyPrefix + int64(len(path))

			switch {
			case prevLen > 0 && newLen == 0:
				if isAccount {
					accountTrienodesDelta--
					accountTrienodeBytes -= keySize + int64(prevLen)
				} else {
					storageTrienodesDelta--
					storageTrienodeBytes -= keySize + int64(prevLen)
				}
			case prevLen == 0 && newLen > 0:
				if isAccount {
					accountTrienodesDelta++
					accountTrienodeBytes += keySize + int64(newLen)
				} else {
					storageTrienodesDelta++
					storageTrienodeBytes += keySize + int64(newLen)
				}
			default:
				if isAccount {
					accountTrienodeBytes += int64(newLen - prevLen)
				} else {
					storageTrienodeBytes += int64(newLen - prevLen)
				}
			}
		}

		// Accumulate depth counts
		if isAccount {
			for i := range 65 {
				accountDepthCreated[i] += createdCounts[i]
				accountDepthDeleted[i] += deletedCounts[i]
			}
		} else {
			for i := range 65 {
				storageDepthCreated[i] += createdCounts[i]
				storageDepthDeleted[i] += deletedCounts[i]
			}
		}
	}

	// Calculate contract code size changes
	codeExists := make(map[common.Hash]struct{})
	for _, code := range update.CodeChanges {
		if _, ok := codeExists[code.Hash]; ok || code.Exists {
			continue
		}
		codesDelta++
		codeBytesDelta += codeKeySize + int64(len(code.Code))
		codeExists[code.Hash] = struct{}{}
	}

	// Calculate cumulative statistics
	s.mu.Lock()
	defer s.mu.Unlock()

	// Look up parent stats
	parentStats := s.stats[update.OriginRoot] // zero value if not found

	// Apply deltas to get new cumulative stats
	newStats := stateSizeStats{
		Accounts:             parentStats.Accounts + accountsDelta,
		AccountBytes:         parentStats.AccountBytes + accountBytesDelta,
		Storages:             parentStats.Storages + storagesDelta,
		StorageBytes:         parentStats.StorageBytes + storageBytesDelta,
		AccountTrienodes:     parentStats.AccountTrienodes + accountTrienodesDelta,
		AccountTrienodeBytes: parentStats.AccountTrienodeBytes + accountTrienodeBytes,
		StorageTrienodes:     parentStats.StorageTrienodes + storageTrienodesDelta,
		StorageTrienodeBytes: parentStats.StorageTrienodeBytes + storageTrienodeBytes,
		Codes:                parentStats.Codes + codesDelta,
		CodeBytes:            parentStats.CodeBytes + codeBytesDelta,
	}

	// Store the new stats for this root
	s.stats[update.Root] = newStats

	// Calculate total nodes for created and deleted
	var totalCreated, totalDeleted int64
	for _, c := range accountDepthCreated {
		totalCreated += c
	}
	for _, c := range storageDepthCreated {
		totalCreated += c
	}
	for _, c := range accountDepthDeleted {
		totalDeleted += c
	}
	for _, c := range storageDepthDeleted {
		totalDeleted += c
	}

	// Write to statesize CSV
	s.writeRecord(stateSizeRecord{
		BlockNumber: update.BlockNumber,
		Root:        update.Root,
		ParentRoot:  update.OriginRoot,
		Stats:       newStats,
	})

	// Write to depth CSV files
	s.writeDepthRecord(s.depthCreatedWriter, update.BlockNumber, update.Root, update.OriginRoot, totalCreated, accountDepthCreated, storageDepthCreated)
	s.writeDepthRecord(s.depthDeletedWriter, update.BlockNumber, update.Root, update.OriginRoot, totalDeleted, accountDepthDeleted, storageDepthDeleted)
}

func (s *stateSizeTracer) writeRecord(r stateSizeRecord) {
	row := []string{
		strconv.FormatUint(r.BlockNumber, 10),
		r.Root.Hex(),
		r.ParentRoot.Hex(),
		strconv.FormatInt(r.Stats.Accounts, 10),
		strconv.FormatInt(r.Stats.AccountBytes, 10),
		strconv.FormatInt(r.Stats.Storages, 10),
		strconv.FormatInt(r.Stats.StorageBytes, 10),
		strconv.FormatInt(r.Stats.AccountTrienodes, 10),
		strconv.FormatInt(r.Stats.AccountTrienodeBytes, 10),
		strconv.FormatInt(r.Stats.StorageTrienodes, 10),
		strconv.FormatInt(r.Stats.StorageTrienodeBytes, 10),
		strconv.FormatInt(r.Stats.Codes, 10),
		strconv.FormatInt(r.Stats.CodeBytes, 10),
	}

	if err := s.writer.Write(row); err != nil {
		log.Warn("Failed to write statesize record", "error", err)
		return
	}
	s.writer.Flush()
	if err := s.writer.Error(); err != nil {
		log.Warn("Failed to flush statesize record", "error", err)
	}
}

func (s *stateSizeTracer) writeDepthRecord(writer *csv.Writer, blockNumber uint64, root, parentRoot common.Hash, totalNodes int64, accountDepths, storageDepths [65]int64) {
	// Build row: block_number, root, parent_root, total_nodes, account_depth_0..64, storage_depth_0..64
	row := make([]string, 0, 4+65+65)
	row = append(row, strconv.FormatUint(blockNumber, 10))
	row = append(row, root.Hex())
	row = append(row, parentRoot.Hex())
	row = append(row, strconv.FormatInt(totalNodes, 10))

	for i := 0; i < 65; i++ {
		row = append(row, strconv.FormatInt(accountDepths[i], 10))
	}
	for i := 0; i < 65; i++ {
		row = append(row, strconv.FormatInt(storageDepths[i], 10))
	}

	if err := writer.Write(row); err != nil {
		log.Warn("Failed to write depth record", "error", err)
		return
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		log.Warn("Failed to flush depth record", "error", err)
	}
}

func (s *stateSizeTracer) onClose() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.writer != nil {
		s.writer.Flush()
	}
	if s.file != nil {
		if err := s.file.Close(); err != nil {
			log.Warn("Failed to close statesize tracer file", "error", err)
		}
	}
	if s.depthCreatedWriter != nil {
		s.depthCreatedWriter.Flush()
	}
	if s.depthCreatedFile != nil {
		if err := s.depthCreatedFile.Close(); err != nil {
			log.Warn("Failed to close depth created file", "error", err)
		}
	}
	if s.depthDeletedWriter != nil {
		s.depthDeletedWriter.Flush()
	}
	if s.depthDeletedFile != nil {
		if err := s.depthDeletedFile.Close(); err != nil {
			log.Warn("Failed to close depth deleted file", "error", err)
		}
	}
}

// slimAccountSize calculates the RLP-encoded size of an account in slim format.
func slimAccountSize(acct *types.StateAccount) int {
	if acct == nil {
		return 0
	}
	data := types.SlimAccountRLP(*acct)
	return len(data)
}

// calculateDepthCountsByType calculates the depth of each node and separates counts
// into created/modified nodes and deleted nodes.
// - Created/Modified: nodes that exist after the update (New has data)
// - Deleted: nodes that existed before but don't exist after (Prev has data, New is empty)
func calculateDepthCountsByType(pathMap map[string]*tracing.TrieNodeChange) (created, deleted [65]int64) {
	n := len(pathMap)
	if n == 0 {
		return
	}

	// First, calculate depth for all nodes using the tree structure
	paths := make([]string, 0, n)
	for path := range pathMap {
		paths = append(paths, path)
	}
	slices.Sort(paths)

	// Map from path to its depth
	depthMap := make(map[string]int, n)

	// Stack stores paths of ancestors
	stack := make([]string, 0, 65)

	for _, path := range paths {
		// Pop until stack top is a strict prefix of path
		for len(stack) > 0 {
			top := stack[len(stack)-1]
			if len(top) < len(path) && path[:len(top)] == top {
				break
			}
			stack = stack[:len(stack)-1]
		}

		depth := len(stack)
		depthMap[path] = depth

		stack = append(stack, path)
	}

	// Now classify each node based on Prev/New status
	for path, change := range pathMap {
		depth := depthMap[path]

		var prevLen, newLen int
		if change.Prev != nil {
			prevLen = len(change.Prev.Blob)
		}
		if change.New != nil {
			newLen = len(change.New.Blob)
		}

		// Created/Modified: New has data (node exists after update)
		if newLen > 0 {
			created[depth]++
		}
		// Deleted: Prev has data but New is empty (node removed)
		if prevLen > 0 && newLen == 0 {
			deleted[depth]++
		}
	}

	return
}
