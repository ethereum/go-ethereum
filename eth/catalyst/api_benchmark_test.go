// Copyright 2026 The go-ethereum Authors
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
package catalyst

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
)

// encodingType specifies which encoding to use in benchmarks
type encodingType int

const (
	encNone encodingType = iota
	encJSON
	encJSONPremarshaled
	encRLP
)

func (e encodingType) String() string {
	switch e {
	case encNone:
		return "none"
	case encJSON:
		return "json"
	case encJSONPremarshaled:
		return "json_premarshaled"
	case encRLP:
		return "rlp"
	default:
		return "unknown"
	}
}

var encodingTypes = []encodingType{encNone, encJSON, encJSONPremarshaled, encRLP}

// benchEncode encodes the value using the specified encoding type.
// It fails the benchmark if encoding fails.
func benchEncode(b *testing.B, enc encodingType, v any) {
	var err error
	switch enc {
	case encJSON:
		_, err = json.Marshal(v)
		if err != nil {
			b.Fatalf("JSON marshal failed: %v", err)
		}
	case encJSONPremarshaled:
		if pm, ok := v.(rpc.JSONPremarshaled); ok {
			_, err = pm.PremarshaledJSON()
		} else {
			_, err = json.Marshal(v)
		}
		if err != nil {
			b.Fatalf("JSON premarshaled marshal failed: %v", err)
		}
	case encRLP:
		_, err = rlp.EncodeToBytes(v)
		if err != nil {
			b.Fatalf("RLP encode failed: %v", err)
		}
	}
}

// benchmarkBlobCounts defines the blob counts for benchmarks
var benchmarkBlobCounts = []int{21, 72}

// maxBenchmarkBlobs is the maximum number of blobs we need for benchmarks
var maxBenchmarkBlobs = benchmarkBlobCounts[len(benchmarkBlobCounts)-1]

var (
	// Pre-computed blobs for benchmarks
	benchBlobs          []*kzg4844.Blob
	benchBlobCommits    []kzg4844.Commitment
	benchBlobProofs     []kzg4844.Proof
	benchBlobCellProofs [][]kzg4844.Proof
	benchBlobVHashes    []common.Hash
)

func init() {
	// Pre-compute blobs for benchmarks
	for i := 0; i < maxBenchmarkBlobs; i++ {
		blob := &kzg4844.Blob{byte(i), byte(i >> 8)}
		benchBlobs = append(benchBlobs, blob)

		commit, _ := kzg4844.BlobToCommitment(blob)
		benchBlobCommits = append(benchBlobCommits, commit)

		proof, _ := kzg4844.ComputeBlobProof(blob, commit)
		benchBlobProofs = append(benchBlobProofs, proof)

		cellProofs, _ := kzg4844.ComputeCellProofs(blob)
		benchBlobCellProofs = append(benchBlobCellProofs, cellProofs)

		vhash := kzg4844.CalcBlobHashV1(sha256.New(), &commit)
		benchBlobVHashes = append(benchBlobVHashes, vhash)
	}
}

// benchFork specifies which fork to use in benchmark environments
type benchFork int

const (
	forkCancun benchFork = iota
	forkPrague
	forkOsaka
)

// benchmarkBlobEnv holds the environment for blob benchmarks
type benchmarkBlobEnv struct {
	node      *node.Node
	eth       *eth.Ethereum
	api       *ConsensusAPI
	config    *params.ChainConfig
	keys      []*ecdsa.PrivateKey
	vhashes   []common.Hash
	version   byte
	blobCount int
	nonces    []uint64 // current nonce for each key
}

// makeBenchBlobTx creates a blob transaction with the specified number of blobs.
// blobOffset indicates which pre-computed blobs to use.
func makeBenchBlobTx(chainConfig *params.ChainConfig, nonce uint64, blobCount int, blobOffset int, key *ecdsa.PrivateKey, version byte) *types.Transaction {
	var (
		blobs       []kzg4844.Blob
		blobHashes  []common.Hash
		commitments []kzg4844.Commitment
		proofs      []kzg4844.Proof
	)
	for i := 0; i < blobCount; i++ {
		idx := blobOffset + i
		blobs = append(blobs, *benchBlobs[idx])
		commitments = append(commitments, benchBlobCommits[idx])
		if version == types.BlobSidecarVersion0 {
			proofs = append(proofs, benchBlobProofs[idx])
		} else {
			proofs = append(proofs, benchBlobCellProofs[idx]...)
		}
		blobHashes = append(blobHashes, benchBlobVHashes[idx])
	}
	blobtx := &types.BlobTx{
		ChainID:    uint256.MustFromBig(chainConfig.ChainID),
		Nonce:      nonce,
		GasTipCap:  uint256.NewInt(params.GWei),
		GasFeeCap:  uint256.NewInt(10 * params.GWei),
		Gas:        21000,
		BlobFeeCap: uint256.NewInt(params.GWei),
		BlobHashes: blobHashes,
		Value:      uint256.NewInt(100),
		Sidecar:    types.NewBlobTxSidecar(version, blobs, commitments, proofs),
	}
	return types.MustSignNewTx(key, types.LatestSigner(chainConfig), blobtx)
}

// newBenchmarkBlobEnv creates an environment for blob benchmarks.
// It creates multiple keys and fills the pool with blob transactions totaling the specified blob count.
// version: 0 = BlobSidecarVersion0 (pre-Osaka), 1 = BlobSidecarVersion1 (Osaka+)
// fork: which fork to enable
func newBenchmarkBlobEnv(b *testing.B, blobCount int, version byte, fork benchFork) *benchmarkBlobEnv {
	// Create a configuration that allows enough blobs
	config := *params.MergedTestChainConfig
	// Set blob schedule to allow for large blob counts (up to 128 blobs per block)
	config.BlobScheduleConfig = &params.BlobScheduleConfig{
		Cancun: &params.BlobConfig{Target: 6, Max: 128, UpdateFraction: 3338477},
		Prague: &params.BlobConfig{Target: 6, Max: 128, UpdateFraction: 5007716},
		Osaka:  &params.BlobConfig{Target: 6, Max: 128, UpdateFraction: 5007716},
	}
	// Configure fork times based on requested fork
	switch fork {
	case forkCancun:
		config.PragueTime = nil
		config.OsakaTime = nil
	case forkPrague:
		config.OsakaTime = nil
	case forkOsaka:
		// All forks enabled (default)
	}

	// Generate enough keys for all the blob transactions
	// Each tx can have up to 6 blobs, so we need ceil(blobCount/6) keys
	numTxs := (blobCount + 5) / 6
	keys := make([]*ecdsa.PrivateKey, numTxs)
	addrs := make([]common.Address, numTxs)
	alloc := make(types.GenesisAlloc)
	alloc[testAddr] = types.Account{Balance: testBalance}

	for i := 0; i < numTxs; i++ {
		key, _ := crypto.GenerateKey()
		keys[i] = key
		addrs[i] = crypto.PubkeyToAddress(key.PublicKey)
		// Give each account enough balance for many transactions
		alloc[addrs[i]] = types.Account{Balance: new(big.Int).Mul(big.NewInt(1e18), big.NewInt(10000))}
	}

	gspec := &core.Genesis{
		Config:     &config,
		Alloc:      alloc,
		Difficulty: common.Big0,
	}
	n, ethServ := startEthService(b, gspec, nil)

	// Collect versioned hashes for the blobs we'll use
	var vhashes []common.Hash
	for i := 0; i < blobCount; i++ {
		vhashes = append(vhashes, benchBlobVHashes[i])
	}

	// Fill initial blob txs into the pool
	env := &benchmarkBlobEnv{
		node:      n,
		eth:       ethServ,
		api:       newConsensusAPIWithoutHeartbeat(ethServ),
		config:    &config,
		keys:      keys,
		vhashes:   vhashes,
		version:   version,
		blobCount: blobCount,
		nonces:    make([]uint64, numTxs),
	}
	env.addBlobTxs(b)
	return env
}

// addBlobTxs adds blob transactions to the pool using the stored blobCount and per-key nonces.
// It increments each key's nonce after adding transactions.
func (env *benchmarkBlobEnv) addBlobTxs(b *testing.B) {
	numTxs := (env.blobCount + 5) / 6
	var txs []*types.Transaction
	blobsRemaining := env.blobCount
	blobOffset := 0

	for i := 0; i < numTxs && blobsRemaining > 0; i++ {
		// Each tx gets up to 6 blobs
		txBlobCount := 6
		if blobsRemaining < 6 {
			txBlobCount = blobsRemaining
		}
		tx := makeBenchBlobTx(env.config, env.nonces[i], txBlobCount, blobOffset, env.keys[i], env.version)
		txs = append(txs, tx)
		blobOffset += txBlobCount
		blobsRemaining -= txBlobCount
	}
	errs := env.eth.TxPool().Add(txs, true)
	for i, err := range errs {
		if err != nil {
			b.Fatalf("Failed to add blob tx %d to pool: %v", i, err)
		}
	}
	// Increment nonce for each key used
	for i := 0; i < numTxs; i++ {
		env.nonces[i]++
	}
}

// Close closes the environment
func (env *benchmarkBlobEnv) Close() {
	env.node.Close()
}

// BenchmarkGetBlobsV1 benchmarks the GetBlobsV1 method with various blob counts.
// GetBlobsV1 is available at Cancun/Prague (pre-Osaka).
func BenchmarkGetBlobsV1(b *testing.B) {
	for _, blobCount := range benchmarkBlobCounts {
		for _, enc := range encodingTypes {
			b.Run(fmt.Sprintf("blobs=%d/enc=%s", blobCount, enc), func(b *testing.B) {
				env := newBenchmarkBlobEnv(b, blobCount, 0, forkPrague)
				defer env.Close()

				b.ResetTimer()
				for b.Loop() {
					result, err := env.api.GetBlobsV1(env.vhashes)
					if err != nil {
						b.Fatalf("GetBlobsV1 failed: %v", err)
					}
					// Verify we got the expected number of blobs
					if len(result) != blobCount {
						b.Fatalf("expected %d blobs, got %d", blobCount, len(result))
					}
					benchEncode(b, enc, result)
				}
				b.ReportMetric(float64(b.Elapsed().Milliseconds())/float64(b.N), "ms/op")
			})
		}
	}
}

// BenchmarkGetBlobsV2Extended benchmarks the GetBlobsV2 method with various blob counts.
// GetBlobsV2 is available at Osaka+.
func BenchmarkGetBlobsV2Extended(b *testing.B) {
	for _, blobCount := range benchmarkBlobCounts {
		for _, enc := range encodingTypes {
			b.Run(fmt.Sprintf("blobs=%d/enc=%s", blobCount, enc), func(b *testing.B) {
				env := newBenchmarkBlobEnv(b, blobCount, 1, forkOsaka)
				defer env.Close()

				b.ResetTimer()
				for b.Loop() {
					result, err := env.api.GetBlobsV2(env.vhashes)
					if err != nil {
						b.Fatalf("GetBlobsV2 failed: %v", err)
					}
					// Verify we got the expected number of blobs
					if len(result) != blobCount {
						b.Fatalf("expected %d blobs, got %d", blobCount, len(result))
					}
					benchEncode(b, enc, result)
				}
				b.ReportMetric(float64(b.Elapsed().Milliseconds())/float64(b.N), "ms/op")
			})
		}
	}
}

// BenchmarkGetBlobsV3 benchmarks the GetBlobsV3 method with various blob counts.
// GetBlobsV3 is available at Osaka+.
func BenchmarkGetBlobsV3(b *testing.B) {
	for _, blobCount := range benchmarkBlobCounts {
		for _, enc := range encodingTypes {
			b.Run(fmt.Sprintf("blobs=%d/enc=%s", blobCount, enc), func(b *testing.B) {
				env := newBenchmarkBlobEnv(b, blobCount, 1, forkOsaka)
				defer env.Close()

				b.ResetTimer()
				for b.Loop() {
					result, err := env.api.GetBlobsV3(env.vhashes)
					if err != nil {
						b.Fatalf("GetBlobsV3 failed: %v", err)
					}
					// Verify we got the expected number of blobs
					if len(result) != blobCount {
						b.Fatalf("expected %d blobs, got %d", blobCount, len(result))
					}
					benchEncode(b, enc, result)
				}
				b.ReportMetric(float64(b.Elapsed().Milliseconds())/float64(b.N), "ms/op")
			})
		}
	}
}

// BenchmarkGetPayloadV5WithBlobs benchmarks GetPayloadV5 (Osaka fork) with blobs.
// Note: Measures single iteration performance due to NewPayload complexity at Osaka.
func BenchmarkGetPayloadV5WithBlobs(b *testing.B) {
	for _, blobCount := range benchmarkBlobCounts {
		for _, enc := range encodingTypes {
			b.Run(fmt.Sprintf("blobs=%d/enc=%s", blobCount, enc), func(b *testing.B) {
				env := newBenchmarkBlobEnv(b, blobCount, 1, forkOsaka)
				defer env.Close()

				parent := env.api.eth.BlockChain().CurrentHeader()
				beaconRoot := common.Hash{0x42}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					// Note: We don't call addBlobTxs here because we can't advance the chain
					// (NewPayloadV5 requires execution requests). The same transactions are
					// reused for each iteration, which still benchmarks the GetPayload performance.
					timestamp := parent.Time + 12
					fcState := engine.ForkchoiceStateV1{
						HeadBlockHash:      parent.Hash(),
						SafeBlockHash:      parent.Hash(),
						FinalizedBlockHash: parent.Hash(),
					}
					payloadAttr := &engine.PayloadAttributes{
						Timestamp:             timestamp,
						Random:                common.Hash{byte(i)},
						SuggestedFeeRecipient: testAddr,
						Withdrawals:           []*types.Withdrawal{},
						BeaconRoot:            &beaconRoot,
					}
					resp, err := env.api.ForkchoiceUpdatedV3(fcState, payloadAttr)
					if err != nil {
						b.Fatalf("ForkchoiceUpdatedV3 failed: %v", err)
					}
					if resp.PayloadID == nil {
						b.Fatalf("ForkchoiceUpdatedV3 returned nil PayloadID")
					}
					// Wait for the payload to be built with transactions
					time.Sleep(100 * time.Millisecond)

					envelope, err := env.api.GetPayloadV5(*resp.PayloadID)
					if err != nil {
						b.Fatalf("GetPayloadV5 failed: %v", err)
					}
					if envelope.BlobsBundle == nil {
						b.Fatalf("BlobsBundle is nil")
					}
					// Verify we got the expected number of blobs
					if len(envelope.BlobsBundle.Blobs) != blobCount {
						b.Fatalf("expected %d blobs, got %d", blobCount, len(envelope.BlobsBundle.Blobs))
					}
					benchEncode(b, enc, envelope)
				}
				b.ReportMetric(float64(b.Elapsed().Milliseconds())/float64(b.N), "ms/op")
			})
		}
	}
}

// BenchmarkNewPayloadV3WithBlobs benchmarks the NewPayloadV3 method with various blob counts.
// Each iteration processes a payload with the full blob count.
func BenchmarkNewPayloadV3WithBlobs(b *testing.B) {
	for _, blobCount := range benchmarkBlobCounts {
		for _, enc := range encodingTypes {
			b.Run(fmt.Sprintf("blobs=%d/enc=%s", blobCount, enc), func(b *testing.B) {
				env := newBenchmarkBlobEnv(b, blobCount, 0, forkCancun)
				defer env.Close()

				parent := env.api.eth.BlockChain().CurrentHeader()
				beaconRoot := common.Hash{0x42}

				// Build a payload first to get valid executable data
				timestamp := parent.Time + 12
				fcState := engine.ForkchoiceStateV1{
					HeadBlockHash:      parent.Hash(),
					SafeBlockHash:      parent.Hash(),
					FinalizedBlockHash: parent.Hash(),
				}
				payloadAttr := &engine.PayloadAttributes{
					Timestamp:             timestamp,
					Random:                common.Hash{0x01},
					SuggestedFeeRecipient: testAddr,
					Withdrawals:           []*types.Withdrawal{},
					BeaconRoot:            &beaconRoot,
				}
				resp, err := env.api.ForkchoiceUpdatedV3(fcState, payloadAttr)
				if err != nil {
					b.Fatalf("ForkchoiceUpdatedV3 failed: %v", err)
				}
				if resp.PayloadID == nil {
					b.Fatalf("ForkchoiceUpdatedV3 returned nil PayloadID")
				}
				// Wait for the payload to be built with transactions
				time.Sleep(100 * time.Millisecond)

				// Get the payload
				envelope, err := env.api.GetPayloadV3(*resp.PayloadID)
				if err != nil {
					b.Fatalf("GetPayloadV3 failed: %v", err)
				}
				// Verify we got the expected number of blobs
				if len(envelope.BlobsBundle.Blobs) != blobCount {
					b.Fatalf("expected %d blobs in setup, got %d", blobCount, len(envelope.BlobsBundle.Blobs))
				}

				execData := envelope.ExecutionPayload
				// Collect versioned hashes from blobs bundle
				vhashes := make([]common.Hash, len(envelope.BlobsBundle.Commitments))
				for j, commitment := range envelope.BlobsBundle.Commitments {
					var commit kzg4844.Commitment
					copy(commit[:], commitment)
					vhashes[j] = kzg4844.CalcBlobHashV1(sha256.New(), &commit)
				}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					// NewPayload is idempotent, calling it multiple times with the same data
					// should return the same result. The payload contains blobCount blobs.
					result, err := env.api.NewPayloadV3(context.Background(), *execData, vhashes, &beaconRoot)
					if err != nil {
						b.Fatalf("NewPayloadV3 failed: %v", err)
					}
					benchEncode(b, enc, result)
				}
				b.ReportMetric(float64(b.Elapsed().Milliseconds())/float64(b.N), "ms/op")
			})
		}
	}
}

// BenchmarkForkchoiceUpdatedWithBlobPayload benchmarks forkchoice updates that trigger
// payload building with blob transactions.
// Note: Measures ForkchoiceUpdated performance with blob transactions in the pool.
func BenchmarkForkchoiceUpdatedWithBlobPayload(b *testing.B) {
	for _, blobCount := range benchmarkBlobCounts {
		for _, enc := range encodingTypes {
			b.Run(fmt.Sprintf("blobs=%d/enc=%s", blobCount, enc), func(b *testing.B) {
				env := newBenchmarkBlobEnv(b, blobCount, 0, forkCancun)
				defer env.Close()

				parent := env.api.eth.BlockChain().CurrentHeader()
				beaconRoot := common.Hash{0x42}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					// Note: We don't call addBlobTxs here because the blob pool has
					// a per-account limit of 16 transactions. The same transactions are
					// reused for each iteration, which still benchmarks the ForkchoiceUpdated
					// performance with blob transactions in the pool.
					timestamp := parent.Time + 12
					fcState := engine.ForkchoiceStateV1{
						HeadBlockHash:      parent.Hash(),
						SafeBlockHash:      parent.Hash(),
						FinalizedBlockHash: parent.Hash(),
					}
					payloadAttr := &engine.PayloadAttributes{
						Timestamp:             timestamp,
						Random:                common.Hash{byte(i)},
						SuggestedFeeRecipient: testAddr,
						Withdrawals:           []*types.Withdrawal{},
						BeaconRoot:            &beaconRoot,
					}
					resp, err := env.api.ForkchoiceUpdatedV3(fcState, payloadAttr)
					if err != nil {
						b.Fatalf("ForkchoiceUpdatedV3 failed: %v", err)
					}
					if resp.PayloadID == nil {
						b.Fatalf("ForkchoiceUpdatedV3 returned nil PayloadID")
					}
					benchEncode(b, enc, resp)
				}
				b.ReportMetric(float64(b.Elapsed().Milliseconds())/float64(b.N), "ms/op")
			})
		}
	}
}

// BenchmarkFullBlobWorkflowOsaka benchmarks the complete blob workflow at Osaka:
// ForkchoiceUpdated -> GetPayload
// Note: Measures single iteration performance due to NewPayload complexity at Osaka.
func BenchmarkFullBlobWorkflowOsaka(b *testing.B) {
	for _, blobCount := range benchmarkBlobCounts {
		for _, enc := range encodingTypes {
			b.Run(fmt.Sprintf("blobs=%d/enc=%s", blobCount, enc), func(b *testing.B) {
				env := newBenchmarkBlobEnv(b, blobCount, 1, forkOsaka)
				defer env.Close()

				parent := env.api.eth.BlockChain().CurrentHeader()
				beaconRoot := common.Hash{0x42}

				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					// Note: We don't call addBlobTxs here because we can't advance the chain
					// (NewPayloadV5 requires execution requests). The same transactions are
					// reused for each iteration, which still benchmarks the workflow performance.

					// 1. ForkchoiceUpdated to build payload
					timestamp := parent.Time + 12
					fcState := engine.ForkchoiceStateV1{
						HeadBlockHash:      parent.Hash(),
						SafeBlockHash:      parent.Hash(),
						FinalizedBlockHash: parent.Hash(),
					}
					payloadAttr := &engine.PayloadAttributes{
						Timestamp:             timestamp,
						Random:                common.Hash{byte(i)},
						SuggestedFeeRecipient: testAddr,
						Withdrawals:           []*types.Withdrawal{},
						BeaconRoot:            &beaconRoot,
					}
					resp, err := env.api.ForkchoiceUpdatedV3(fcState, payloadAttr)
					if err != nil {
						b.Fatalf("ForkchoiceUpdatedV3 failed: %v", err)
					}
					if resp.PayloadID == nil {
						b.Fatalf("ForkchoiceUpdatedV3 returned nil PayloadID")
					}
					// Encode ForkchoiceUpdated response
					benchEncode(b, enc, resp)

					// Wait for the payload to be built with transactions
					time.Sleep(100 * time.Millisecond)

					// 2. GetPayload
					envelope, err := env.api.GetPayloadV5(*resp.PayloadID)
					if err != nil {
						b.Fatalf("GetPayloadV5 failed: %v", err)
					}
					if envelope.BlobsBundle == nil {
						b.Fatalf("BlobsBundle is nil")
					}
					// Verify we got the expected number of blobs
					if len(envelope.BlobsBundle.Blobs) != blobCount {
						b.Fatalf("expected %d blobs, got %d", blobCount, len(envelope.BlobsBundle.Blobs))
					}
					// Encode GetPayload response
					benchEncode(b, enc, envelope)
				}
				b.ReportMetric(float64(b.Elapsed().Milliseconds())/float64(b.N), "ms/op")
			})
		}
	}
}

// discardConn is a net.Conn-like writer that discards all output.
// Used to measure server-side RPC cost without client-side decoding.
type discardConn struct {
	io.Reader
	io.Writer
}

func (discardConn) Close() error                     { return nil }
func (discardConn) SetWriteDeadline(time.Time) error { return nil }

// BenchmarkGetPayloadV5RPCServerOnly benchmarks only the EL server-side cost of
// engine_getPayloadV5: method dispatch, JSON serialization, and wire encoding.
// Client-side decoding is excluded by writing to io.Discard.
func BenchmarkGetPayloadV5RPCServerOnly(b *testing.B) {
	blobCount := 72
	env := newBenchmarkBlobEnv(b, blobCount, 1, forkOsaka)
	defer env.Close()

	// Register the engine API on the running node's in-process RPC server.
	rpcServer, err := env.node.RPCHandler()
	if err != nil {
		b.Fatalf("RPCHandler failed: %v", err)
	}
	rpcServer.RegisterName("engine", env.api)

	parent := env.api.eth.BlockChain().CurrentHeader()
	beaconRoot := common.Hash{0x42}

	// Build one payload to get a valid payloadID.
	fcState := engine.ForkchoiceStateV1{
		HeadBlockHash:      parent.Hash(),
		SafeBlockHash:      parent.Hash(),
		FinalizedBlockHash: parent.Hash(),
	}
	payloadAttr := &engine.PayloadAttributes{
		Timestamp:             parent.Time + 12,
		Random:                common.Hash{0x01},
		SuggestedFeeRecipient: testAddr,
		Withdrawals:           []*types.Withdrawal{},
		BeaconRoot:            &beaconRoot,
	}
	resp, err := env.api.ForkchoiceUpdatedV3(fcState, payloadAttr)
	if err != nil {
		b.Fatalf("ForkchoiceUpdatedV3 failed: %v", err)
	}
	if resp.PayloadID == nil {
		b.Fatalf("ForkchoiceUpdatedV3 returned nil PayloadID")
	}
	time.Sleep(100 * time.Millisecond)

	// Verify the payload has the expected blobs via the direct API first.
	envelope, err := env.api.GetPayloadV5(*resp.PayloadID)
	if err != nil {
		b.Fatalf("GetPayloadV5 failed: %v", err)
	}
	if len(envelope.BlobsBundle.Blobs) != blobCount {
		b.Fatalf("expected %d blobs, got %d", blobCount, len(envelope.BlobsBundle.Blobs))
	}
	b.Logf("payload size: %d blobs, %d txs", len(envelope.BlobsBundle.Blobs), len(envelope.ExecutionPayload.Transactions))

	// Build the JSON-RPC request bytes once.
	reqJSON := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"engine_getPayloadV5","params":["%s"]}`, resp.PayloadID.String())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn := discardConn{
			Reader: strings.NewReader(reqJSON),
			Writer: io.Discard,
		}
		codec := rpc.NewCodec(conn)
		rpcServer.ServeCodec(codec, 0)
	}
	b.StopTimer()
	b.ReportMetric(float64(b.Elapsed().Milliseconds())/float64(b.N), "ms/op")
}
