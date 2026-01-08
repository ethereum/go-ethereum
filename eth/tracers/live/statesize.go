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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
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

// depthStats holds node count and byte size for a single depth level.
type depthStats struct {
	Count int64
	Bytes int64
}

// stateSizeDelta represents state size delta for a single block.
type stateSizeDelta struct {
	AccountDelta              int64
	AccountBytesDelta         int64
	AccountTrienodeDelta      int64
	AccountTrienodeBytesDelta int64
	ContractCodeDelta         int64
	ContractCodeBytesDelta    int64
	StorageDelta              int64
	StorageBytesDelta         int64
	StorageTrienodeDelta      int64
	StorageTrienodeBytesDelta int64
}

// Default configuration values
const (
	defaultMaxQueueSize       = 51200
	defaultBatchTimeout       = 5 * time.Second
	defaultExportTimeout      = 30 * time.Second
	defaultMaxExportBatchSize = 512
	defaultWorkers            = 1
)

type stateSizeTracer struct {
	mu     sync.Mutex
	config stateSizeTracerConfig

	// gRPC connection
	conn   *grpc.ClientConn
	client xatu.EventIngesterClient

	// Event batching
	eventCh chan *xatu.DecoratedEvent
	done    chan struct{}
	wg      sync.WaitGroup

	// Client metadata
	clientID string
}

type stateSizeTracerConfig struct {
	// Xatu server configuration
	Address            string            `json:"address"`            // Required: Xatu server address
	Headers            map[string]string `json:"headers"`            // Optional: HTTP headers (auth)
	TLS                bool              `json:"tls"`                // Use TLS connection
	MaxQueueSize       int               `json:"maxQueueSize"`       // Event queue size
	BatchTimeout       string            `json:"batchTimeout"`       // Batch timeout duration
	ExportTimeout      string            `json:"exportTimeout"`      // Export timeout duration
	MaxExportBatchSize int               `json:"maxExportBatchSize"` // Max batch size
	Workers            int               `json:"workers"`            // Worker count

	// Client identification
	ClientName    string `json:"clientName"`    // Client name for metadata
	ClientVersion string `json:"clientVersion"` // Client version
	NetworkID     uint64 `json:"networkId"`     // Network ID

	// Parsed durations
	batchTimeout  time.Duration
	exportTimeout time.Duration
}

func newStateSizeTracer(cfg json.RawMessage) (*tracing.Hooks, error) {
	var config stateSizeTracerConfig
	if err := json.Unmarshal(cfg, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if config.Address == "" {
		return nil, errors.New("statesize tracer: xatu server address is required")
	}

	// Apply defaults
	if config.MaxQueueSize == 0 {
		config.MaxQueueSize = defaultMaxQueueSize
	}
	if config.MaxExportBatchSize == 0 {
		config.MaxExportBatchSize = defaultMaxExportBatchSize
	}
	if config.Workers == 0 {
		config.Workers = defaultWorkers
	}

	// Parse durations
	var err error
	if config.BatchTimeout != "" {
		config.batchTimeout, err = time.ParseDuration(config.BatchTimeout)
		if err != nil {
			return nil, fmt.Errorf("failed to parse batchTimeout: %w", err)
		}
	} else {
		config.batchTimeout = defaultBatchTimeout
	}

	if config.ExportTimeout != "" {
		config.exportTimeout, err = time.ParseDuration(config.ExportTimeout)
		if err != nil {
			return nil, fmt.Errorf("failed to parse exportTimeout: %w", err)
		}
	} else {
		config.exportTimeout = defaultExportTimeout
	}

	// Set default client name
	if config.ClientName == "" {
		config.ClientName = "geth"
	}

	// Create gRPC connection
	var opts []grpc.DialOption
	if config.TLS {
		host, _, err := net.SplitHostPort(config.Address)
		if err != nil {
			return nil, fmt.Errorf("failed to parse address for TLS: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, host)))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(config.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	t := &stateSizeTracer{
		config:   config,
		conn:     conn,
		client:   xatu.NewEventIngesterClient(conn),
		eventCh:  make(chan *xatu.DecoratedEvent, config.MaxQueueSize),
		done:     make(chan struct{}),
		clientID: uuid.New().String(),
	}

	// Start batch processor workers
	for i := 0; i < config.Workers; i++ {
		t.wg.Add(1)
		go t.runBatchProcessor()
	}

	log.Info("State size tracer initialized",
		"address", config.Address,
		"tls", config.TLS,
		"workers", config.Workers,
		"maxQueueSize", config.MaxQueueSize,
		"batchTimeout", config.batchTimeout,
	)

	return &tracing.Hooks{
		OnStateUpdate: t.onStateUpdate,
		OnClose:       t.onClose,
	}, nil
}

func (s *stateSizeTracer) onStateUpdate(update *tracing.StateUpdate) {
	if update == nil {
		return
	}

	// Calculate state size delta and depth stats
	delta, accountDepthCreated, storageDepthCreated, accountDepthDeleted, storageDepthDeleted := calculateStateSizeDelta(update)

	// Create decorated event
	event := s.createDecoratedEvent(update, delta, accountDepthCreated, storageDepthCreated, accountDepthDeleted, storageDepthDeleted)

	// Queue for sending (non-blocking)
	select {
	case s.eventCh <- event:
	default:
		log.Warn("State size tracer event queue full, dropping event", "block", update.BlockNumber)
	}
}

func (s *stateSizeTracer) createDecoratedEvent(
	update *tracing.StateUpdate,
	delta stateSizeDelta,
	accountDepthCreated, storageDepthCreated, accountDepthDeleted, storageDepthDeleted [65]depthStats,
) *xatu.DecoratedEvent {
	now := time.Now()

	// Build the state metrics message
	metrics := &xatu.ExecutionBlockStateMetrics{
		BlockNumber: strconv.FormatUint(update.BlockNumber, 10),
		StateRoot:   update.Root.Hex(),
		OriginRoot:  update.OriginRoot.Hex(),

		AccountDelta:              strconv.FormatInt(delta.AccountDelta, 10),
		AccountBytesDelta:         strconv.FormatInt(delta.AccountBytesDelta, 10),
		AccountTrienodeDelta:      strconv.FormatInt(delta.AccountTrienodeDelta, 10),
		AccountTrienodeBytesDelta: strconv.FormatInt(delta.AccountTrienodeBytesDelta, 10),
		ContractCodeDelta:         strconv.FormatInt(delta.ContractCodeDelta, 10),
		ContractCodeBytesDelta:    strconv.FormatInt(delta.ContractCodeBytesDelta, 10),
		StorageDelta:              strconv.FormatInt(delta.StorageDelta, 10),
		StorageBytesDelta:         strconv.FormatInt(delta.StorageBytesDelta, 10),
		StorageTrienodeDelta:      strconv.FormatInt(delta.StorageTrienodeDelta, 10),
		StorageTrienodeBytesDelta: strconv.FormatInt(delta.StorageTrienodeBytesDelta, 10),

		AccountTrieDepthCreated: convertDepthStats(accountDepthCreated),
		StorageTrieDepthCreated: convertDepthStats(storageDepthCreated),
		AccountTrieDepthDeleted: convertDepthStats(accountDepthDeleted),
		StorageTrieDepthDeleted: convertDepthStats(storageDepthDeleted),
	}

	// Build client metadata
	clientMeta := &xatu.ClientMeta{
		Name:       s.config.ClientName,
		Version:    s.config.ClientVersion,
		Id:         s.clientID,
		ModuleName: xatu.ModuleName_EL_MIMICRY,
		Ethereum: &xatu.ClientMeta_Ethereum{
			Network: &xatu.ClientMeta_Ethereum_Network{
				Id: s.config.NetworkID,
			},
			Execution: &xatu.ClientMeta_Ethereum_Execution{
				Implementation: s.config.ClientName,
				Version:        s.config.ClientVersion,
			},
		},
	}

	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_BLOCK_STATE_METRICS,
			DateTime: timestamppb.New(now),
			Id:       uuid.New().String(),
		},
		Meta: &xatu.Meta{
			Client: clientMeta,
		},
		Data: &xatu.DecoratedEvent_ExecutionBlockStateMetrics{
			ExecutionBlockStateMetrics: metrics,
		},
	}
}

// convertDepthStats converts [65]depthStats to []*xatu.DepthStats, including only non-zero entries.
func convertDepthStats(stats [65]depthStats) []*xatu.DepthStats {
	result := make([]*xatu.DepthStats, 0, 10) // pre-allocate for typical case
	for i, s := range stats {
		if s.Count > 0 || s.Bytes > 0 {
			result = append(result, &xatu.DepthStats{
				Depth: wrapperspb.UInt32(uint32(i)),
				Count: strconv.FormatInt(s.Count, 10),
				Bytes: strconv.FormatInt(s.Bytes, 10),
			})
		}
	}
	return result
}

func (s *stateSizeTracer) runBatchProcessor() {
	defer s.wg.Done()

	batch := make([]*xatu.DecoratedEvent, 0, s.config.MaxExportBatchSize)
	timer := time.NewTimer(s.config.batchTimeout)
	defer timer.Stop()

	sendBatch := func() {
		if len(batch) == 0 {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), s.config.exportTimeout)
		defer cancel()

		// Add headers to context
		if len(s.config.Headers) > 0 {
			md := metadata.New(s.config.Headers)
			ctx = metadata.NewOutgoingContext(ctx, md)
		}

		req := &xatu.CreateEventsRequest{
			Events: batch,
		}

		resp, err := s.client.CreateEvents(ctx, req, grpc.UseCompressor(gzip.Name))
		if err != nil {
			log.Warn("Failed to send state size events to Xatu", "error", err, "count", len(batch))
		} else {
			log.Debug("Sent state size events to Xatu",
				"sent", len(batch),
				"ingested", resp.GetEventsIngested().GetValue(),
			)
		}

		// Reset batch
		batch = batch[:0]
	}

	for {
		select {
		case <-s.done:
			// Final flush
			sendBatch()
			return

		case event := <-s.eventCh:
			batch = append(batch, event)
			if len(batch) >= s.config.MaxExportBatchSize {
				sendBatch()
				timer.Reset(s.config.batchTimeout)
			}

		case <-timer.C:
			sendBatch()
			timer.Reset(s.config.batchTimeout)
		}
	}
}

func (s *stateSizeTracer) onClose() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Signal workers to stop
	close(s.done)

	// Wait for workers to finish
	s.wg.Wait()

	// Close gRPC connection
	if s.conn != nil {
		if err := s.conn.Close(); err != nil {
			log.Warn("Failed to close gRPC connection", "error", err)
		}
	}

	log.Info("State size tracer closed")
}

// calculateStateSizeDelta computes the state size delta from a state update.
// It returns the delta and depth stats (count + bytes) for account/storage trie nodes (created and deleted).
func calculateStateSizeDelta(update *tracing.StateUpdate) (
	delta stateSizeDelta,
	accountDepthCreated, storageDepthCreated, accountDepthDeleted, storageDepthDeleted [65]depthStats,
) {
	// Calculate account size changes
	for _, change := range update.AccountChanges {
		prevLen := slimAccountSize(change.Prev)
		newLen := slimAccountSize(change.New)

		switch {
		case prevLen > 0 && newLen == 0:
			delta.AccountDelta--
			delta.AccountBytesDelta -= accountKeySize + int64(prevLen)
		case prevLen == 0 && newLen > 0:
			delta.AccountDelta++
			delta.AccountBytesDelta += accountKeySize + int64(newLen)
		default:
			delta.AccountBytesDelta += int64(newLen - prevLen)
		}
	}

	// Calculate storage size changes
	for _, slots := range update.StorageChanges {
		for _, change := range slots {
			prevLen := len(encodeStorageValue(change.Prev))
			newLen := len(encodeStorageValue(change.New))

			switch {
			case prevLen > 0 && newLen == 0:
				delta.StorageDelta--
				delta.StorageBytesDelta -= storageKeySize + int64(prevLen)
			case prevLen == 0 && newLen > 0:
				delta.StorageDelta++
				delta.StorageBytesDelta += storageKeySize + int64(newLen)
			default:
				delta.StorageBytesDelta += int64(newLen - prevLen)
			}
		}
	}

	// Calculate trie node size changes and depth counts
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

		// Calculate depth stats for created/modified and deleted nodes
		createdStats, deletedStats := calculateDepthStatsByType(nodes)

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
					delta.AccountTrienodeDelta--
					delta.AccountTrienodeBytesDelta -= keySize + int64(prevLen)
				} else {
					delta.StorageTrienodeDelta--
					delta.StorageTrienodeBytesDelta -= keySize + int64(prevLen)
				}
			case prevLen == 0 && newLen > 0:
				if isAccount {
					delta.AccountTrienodeDelta++
					delta.AccountTrienodeBytesDelta += keySize + int64(newLen)
				} else {
					delta.StorageTrienodeDelta++
					delta.StorageTrienodeBytesDelta += keySize + int64(newLen)
				}
			default:
				if isAccount {
					delta.AccountTrienodeBytesDelta += int64(newLen - prevLen)
				} else {
					delta.StorageTrienodeBytesDelta += int64(newLen - prevLen)
				}
			}
		}

		// Accumulate depth stats
		if isAccount {
			for i := range 65 {
				accountDepthCreated[i].Count += createdStats[i].Count
				accountDepthCreated[i].Bytes += createdStats[i].Bytes
				accountDepthDeleted[i].Count += deletedStats[i].Count
				accountDepthDeleted[i].Bytes += deletedStats[i].Bytes
			}
		} else {
			for i := range 65 {
				storageDepthCreated[i].Count += createdStats[i].Count
				storageDepthCreated[i].Bytes += createdStats[i].Bytes
				storageDepthDeleted[i].Count += deletedStats[i].Count
				storageDepthDeleted[i].Bytes += deletedStats[i].Bytes
			}
		}
	}

	// Calculate contract code size changes
	// Only count new codes that didn't exist before
	codeExists := make(map[common.Hash]struct{})
	for _, change := range update.CodeChanges {
		if change.New == nil {
			continue
		}
		// Skip if we've already counted this code hash or if it existed before
		if _, ok := codeExists[change.New.Hash]; ok || change.New.Exists {
			continue
		}
		delta.ContractCodeDelta++
		delta.ContractCodeBytesDelta += codeKeySize + int64(len(change.New.Code))
		codeExists[change.New.Hash] = struct{}{}
	}

	return
}

// encodeStorageValue RLP-encodes a storage value for size calculation.
func encodeStorageValue(val common.Hash) []byte {
	if val == (common.Hash{}) {
		return nil
	}
	blob, _ := rlp.EncodeToBytes(common.TrimLeftZeroes(val[:]))
	return blob
}

// slimAccountSize calculates the RLP-encoded size of an account in slim format.
func slimAccountSize(acct *types.StateAccount) int {
	if acct == nil {
		return 0
	}
	data := types.SlimAccountRLP(*acct)
	return len(data)
}

// calculateDepthStatsByType calculates the depth of each node and separates stats
// (count and bytes) into created/modified nodes and deleted nodes.
// - Created/Modified: nodes that exist after the update (New has data)
// - Deleted: nodes that existed before but don't exist after (Prev has data, New is empty)
func calculateDepthStatsByType(pathMap map[string]*tracing.TrieNodeChange) (created, deleted [65]depthStats) {
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
			created[depth].Count++
			created[depth].Bytes += int64(newLen)
		}
		// Deleted: Prev has data but New is empty (node removed)
		if prevLen > 0 && newLen == 0 {
			deleted[depth].Count++
			deleted[depth].Bytes += int64(prevLen)
		}
	}

	return
}
