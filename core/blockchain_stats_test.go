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

package core

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// TestLogSlowBlockJSON tests that logSlow outputs valid JSON in the expected format.
func TestLogSlowBlockJSON(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	handler := log.NewTerminalHandler(&buf, false)
	log.SetDefault(log.NewLogger(handler))

	// Create a test block
	header := &types.Header{
		Number:   common.Big1,
		GasUsed:  21000000,
		GasLimit: 30000000,
	}
	block := types.NewBlockWithHeader(header)

	// Create test stats with cache data
	stats := &ExecuteStats{
		Execution:      500 * time.Millisecond,
		TotalTime:      1200 * time.Millisecond, // > 1s threshold
		MgasPerSecond:  17.5,
		AccountLoaded:  100,
		StorageLoaded:  500,
		CodeLoaded:     20,
		AccountUpdated: 50,
		StorageUpdated: 200,
		StateReadCacheStats: state.ReaderStats{
			AccountCacheHit:  4,
			AccountCacheMiss: 6,
			StorageCacheHit:  0,
			StorageCacheMiss: 11,
			CodeStats: state.ContractCodeReaderStats{
				CacheHit:       4,
				CacheMiss:      0,
				CacheHitBytes:  102400, // ~100KB served from cache
				CacheMissBytes: 0,
			},
		},
	}

	// Log the slow block (threshold of 1s, so 1.2s total time should trigger)
	stats.logSlow(block, 1*time.Second)

	// Get the log output
	output := buf.String()
	t.Logf("Log output:\n%s", output)

	// The output should contain the JSON
	if len(output) == 0 {
		t.Fatal("Expected log output, got empty string")
	}

	// Try to extract and parse the JSON from the log line
	// The log format is: WARN [...] {"level":"warn",...}
	// We need to find the JSON part
	jsonStart := bytes.Index(buf.Bytes(), []byte(`{"level"`))
	if jsonStart == -1 {
		t.Logf("Full output: %s", output)
		t.Fatal("Could not find JSON in log output")
	}

	jsonBytes := buf.Bytes()[jsonStart:]
	// Find the end of the JSON (newline or end of buffer)
	jsonEnd := bytes.IndexByte(jsonBytes, '\n')
	if jsonEnd != -1 {
		jsonBytes = jsonBytes[:jsonEnd]
	}

	// Parse the JSON
	var logEntry slowBlockLog
	if err := json.Unmarshal(jsonBytes, &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON: %v\nJSON: %s", err, string(jsonBytes))
	}

	// Verify the fields
	if logEntry.Level != "warn" {
		t.Errorf("Expected level 'warn', got '%s'", logEntry.Level)
	}
	if logEntry.Msg != "Slow block" {
		t.Errorf("Expected msg 'Slow block', got '%s'", logEntry.Msg)
	}
	if logEntry.Block.Number != 1 {
		t.Errorf("Expected block number 1, got %d", logEntry.Block.Number)
	}
	if logEntry.Block.GasUsed != 21000000 {
		t.Errorf("Expected gas used 21000000, got %d", logEntry.Block.GasUsed)
	}
	if logEntry.Timing.ExecutionMs != 500.0 {
		t.Errorf("Expected execution_ms 500.0, got %v", logEntry.Timing.ExecutionMs)
	}
	if logEntry.Timing.TotalMs != 1200.0 {
		t.Errorf("Expected total_ms 1200.0, got %v", logEntry.Timing.TotalMs)
	}
	if logEntry.StateReads.Accounts != 100 {
		t.Errorf("Expected accounts 100, got %d", logEntry.StateReads.Accounts)
	}
	if logEntry.StateReads.StorageSlots != 500 {
		t.Errorf("Expected storage_slots 500, got %d", logEntry.StateReads.StorageSlots)
	}
	if logEntry.StateWrites.Accounts != 50 {
		t.Errorf("Expected accounts updated 50, got %d", logEntry.StateWrites.Accounts)
	}

	// Verify cache statistics
	if logEntry.Cache.Account.Hits != 4 {
		t.Errorf("Expected account cache hits 4, got %d", logEntry.Cache.Account.Hits)
	}
	if logEntry.Cache.Account.Misses != 6 {
		t.Errorf("Expected account cache misses 6, got %d", logEntry.Cache.Account.Misses)
	}
	// 4/(4+6) = 0.4 = 40%
	if logEntry.Cache.Account.HitRate != 40.0 {
		t.Errorf("Expected account cache hit_rate 40.0, got %f", logEntry.Cache.Account.HitRate)
	}
	if logEntry.Cache.Storage.Hits != 0 {
		t.Errorf("Expected storage cache hits 0, got %d", logEntry.Cache.Storage.Hits)
	}
	if logEntry.Cache.Storage.Misses != 11 {
		t.Errorf("Expected storage cache misses 11, got %d", logEntry.Cache.Storage.Misses)
	}
	// 0/(0+11) = 0%
	if logEntry.Cache.Storage.HitRate != 0.0 {
		t.Errorf("Expected storage cache hit_rate 0.0, got %f", logEntry.Cache.Storage.HitRate)
	}
	if logEntry.Cache.Code.Hits != 4 {
		t.Errorf("Expected code cache hits 4, got %d", logEntry.Cache.Code.Hits)
	}
	if logEntry.Cache.Code.Misses != 0 {
		t.Errorf("Expected code cache misses 0, got %d", logEntry.Cache.Code.Misses)
	}
	// 4/(4+0) = 100%
	if logEntry.Cache.Code.HitRate != 100.0 {
		t.Errorf("Expected code cache hit_rate 100.0, got %f", logEntry.Cache.Code.HitRate)
	}
	// Verify new byte-level cache statistics
	if logEntry.Cache.Code.HitBytes != 102400 {
		t.Errorf("Expected code cache hit_bytes 102400, got %d", logEntry.Cache.Code.HitBytes)
	}
	if logEntry.Cache.Code.MissBytes != 0 {
		t.Errorf("Expected code cache miss_bytes 0, got %d", logEntry.Cache.Code.MissBytes)
	}

	t.Logf("Parsed JSON:\n%+v", logEntry)
}

// TestLogSlowBlockEIP7702 tests that EIP-7702 delegation fields are properly serialized.
func TestLogSlowBlockEIP7702(t *testing.T) {
	var buf bytes.Buffer
	handler := log.NewTerminalHandler(&buf, false)
	log.SetDefault(log.NewLogger(handler))

	header := &types.Header{
		Number:   common.Big1,
		GasUsed:  21000000,
		GasLimit: 30000000,
	}
	block := types.NewBlockWithHeader(header)

	// Create test stats with EIP-7702 delegation data
	stats := &ExecuteStats{
		Execution:                 500 * time.Millisecond,
		TotalTime:                 1200 * time.Millisecond,
		MgasPerSecond:             17.5,
		AccountLoaded:             100,
		StorageLoaded:             500,
		CodeLoaded:                20,
		CodeBytesRead:             4096,
		AccountUpdated:            50,
		StorageUpdated:            200,
		CodeUpdated:               5,
		CodeBytesWrite:            2048,
		Eip7702DelegationsSet:     3,
		Eip7702DelegationsCleared: 1,
		StateReadCacheStats: state.ReaderStats{
			AccountCacheHit:  4,
			AccountCacheMiss: 6,
		},
	}

	stats.logSlow(block, 1*time.Second)

	// Find and parse the JSON
	jsonStart := bytes.Index(buf.Bytes(), []byte(`{"level"`))
	if jsonStart == -1 {
		t.Fatal("Could not find JSON in log output")
	}
	jsonBytes := buf.Bytes()[jsonStart:]
	if jsonEnd := bytes.IndexByte(jsonBytes, '\n'); jsonEnd != -1 {
		jsonBytes = jsonBytes[:jsonEnd]
	}

	var logEntry slowBlockLog
	if err := json.Unmarshal(jsonBytes, &logEntry); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify EIP-7702 fields
	if logEntry.StateWrites.Eip7702DelegationsSet != 3 {
		t.Errorf("Expected eip7702_delegations_set 3, got %d", logEntry.StateWrites.Eip7702DelegationsSet)
	}
	if logEntry.StateWrites.Eip7702DelegationsCleared != 1 {
		t.Errorf("Expected eip7702_delegations_cleared 1, got %d", logEntry.StateWrites.Eip7702DelegationsCleared)
	}

	// Verify code bytes fields
	if logEntry.StateReads.CodeBytes != 4096 {
		t.Errorf("Expected code_bytes read 4096, got %d", logEntry.StateReads.CodeBytes)
	}
	if logEntry.StateWrites.CodeBytes != 2048 {
		t.Errorf("Expected code_bytes write 2048, got %d", logEntry.StateWrites.CodeBytes)
	}
	if logEntry.StateWrites.Code != 5 {
		t.Errorf("Expected code writes 5, got %d", logEntry.StateWrites.Code)
	}
}

// TestLogSlowBlockThreshold tests that logSlow respects the threshold.
func TestLogSlowBlockThreshold(t *testing.T) {
	var buf bytes.Buffer
	handler := log.NewTerminalHandler(&buf, false)
	log.SetDefault(log.NewLogger(handler))

	header := &types.Header{Number: common.Big1}
	block := types.NewBlockWithHeader(header)

	stats := &ExecuteStats{
		TotalTime: 500 * time.Millisecond, // 0.5s, below 1s threshold
	}

	// Should NOT log (below threshold)
	stats.logSlow(block, 1*time.Second)

	if buf.Len() > 0 {
		t.Errorf("Expected no output for fast block, got: %s", buf.String())
	}

	// Reset buffer
	buf.Reset()

	// Test with zero threshold (logs all blocks)
	stats.logSlow(block, 0)

	if buf.Len() == 0 {
		t.Errorf("Expected output for zero threshold (logs all), got nothing")
	}

	// Reset buffer
	buf.Reset()

	// Test with negative threshold (disabled)
	stats.logSlow(block, -1)

	if buf.Len() > 0 {
		t.Errorf("Expected no output for negative threshold (disabled), got: %s", buf.String())
	}
}
