// Copyright 2024 The go-ethereum Authors
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

package txpool

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/txpool/fastfeed"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

func TestTxFastFeedBasic(t *testing.T) {
	feed := fastfeed.NewTxFastFeed()
	
	// Create a test transaction
	key, _ := crypto.GenerateKey()
	signer := types.LatestSigner(params.TestChainConfig)
	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		Nonce:    0,
		GasPrice: big.NewInt(1000000000),
		Gas:      21000,
		To:       &common.Address{1},
		Value:    big.NewInt(1000000000000000000),
	})
	
	// Subscribe
	sub, err := feed.Subscribe(nil)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()
	
	// Publish transaction
	feed.Publish(tx, fastfeed.TxEventAdded)
	
	// Receive event
	select {
	case event := <-sub.Events():
		if event.EventType != fastfeed.TxEventAdded {
			t.Errorf("Expected TxEventAdded, got %d", event.EventType)
		}
		expectedHash := tx.Hash()
		receivedHash := common.BytesToHash(event.Hash[:])
		if receivedHash != expectedHash {
			t.Errorf("Hash mismatch: expected %s, got %s", expectedHash.Hex(), receivedHash.Hex())
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for event")
	}
}

func TestTxFastFeedFiltering(t *testing.T) {
	feed := fastfeed.NewTxFastFeed()
	
	// Create test transactions
	key, _ := crypto.GenerateKey()
	signer := types.LatestSigner(params.TestChainConfig)
	
	targetAddr := common.Address{1}
	otherAddr := common.Address{2}
	
	// Transaction to target address
	targetTx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		Nonce:    0,
		GasPrice: big.NewInt(1000000000),
		Gas:      21000,
		To:       &targetAddr,
		Value:    big.NewInt(1000),
	})
	
	// Transaction to other address
	otherTx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		Nonce:    1,
		GasPrice: big.NewInt(1000000000),
		Gas:      21000,
		To:       &otherAddr,
		Value:    big.NewInt(2000),
	})
	
	// Subscribe with address filter
	filter := &fastfeed.TxFilter{
		Addresses: map[common.Address]struct{}{
			targetAddr: {},
		},
	}
	sub, err := feed.Subscribe(filter)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()
	
	// Publish both transactions
	feed.Publish(otherTx, fastfeed.TxEventAdded)
	feed.Publish(targetTx, fastfeed.TxEventAdded)
	
	// Should only receive target address tx
	select {
	case event := <-sub.Events():
		receivedHash := common.BytesToHash(event.Hash[:])
		if receivedHash != targetTx.Hash() {
			t.Errorf("Expected target tx %s, got %s", targetTx.Hash().Hex(), receivedHash.Hex())
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for filtered event")
	}
	
	// Should not receive other address tx
	select {
	case event := <-sub.Events():
		t.Errorf("Unexpected event received: %s", common.BytesToHash(event.Hash[:]).Hex())
	case <-time.After(50 * time.Millisecond):
		// Expected timeout
	}
}

func TestTxFastFeedMultipleConsumers(t *testing.T) {
	feed := fastfeed.NewTxFastFeed()
	
	// Create test transaction
	key, _ := crypto.GenerateKey()
	signer := types.LatestSigner(params.TestChainConfig)
	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		Nonce:    0,
		GasPrice: big.NewInt(1000000000),
		Gas:      21000,
		To:       &common.Address{1},
		Value:    big.NewInt(1000),
	})
	
	// Create multiple subscribers
	const numSubs = 5
	subs := make([]*fastfeed.Subscription, numSubs)
	for i := 0; i < numSubs; i++ {
		sub, err := feed.Subscribe(nil)
		if err != nil {
			t.Fatalf("Failed to subscribe #%d: %v", i, err)
		}
		defer sub.Unsubscribe()
		subs[i] = sub
	}
	
	// Publish transaction
	feed.Publish(tx, fastfeed.TxEventAdded)
	
	// All subscribers should receive the event
	for i, sub := range subs {
		select {
		case event := <-sub.Events():
			receivedHash := common.BytesToHash(event.Hash[:])
			if receivedHash != tx.Hash() {
				t.Errorf("Subscriber %d: hash mismatch", i)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("Subscriber %d: timeout waiting for event", i)
		}
	}
}

func BenchmarkTxFastFeedPublish(b *testing.B) {
	feed := fastfeed.NewTxFastFeed()
	
	key, _ := crypto.GenerateKey()
	signer := types.LatestSigner(params.TestChainConfig)
	tx := types.MustSignNewTx(key, signer, &types.LegacyTx{
		Nonce:    0,
		GasPrice: big.NewInt(1000000000),
		Gas:      21000,
		To:       &common.Address{1},
		Value:    big.NewInt(1000),
	})
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		feed.Publish(tx, fastfeed.TxEventAdded)
	}
}

func BenchmarkTxFastFeedLatency(b *testing.B) {
	feed := fastfeed.NewTxFastFeed()
	
	sub, err := feed.Subscribe(nil)
	if err != nil {
		b.Fatalf("Failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()
	
	key, _ := crypto.GenerateKey()
	signer := types.LatestSigner(params.TestChainConfig)
	
	// Pre-generate transactions
	txs := make([]*types.Transaction, b.N)
	for i := 0; i < b.N; i++ {
		txs[i] = types.MustSignNewTx(key, signer, &types.LegacyTx{
			Nonce:    uint64(i),
			GasPrice: big.NewInt(1000000000),
			Gas:      21000,
			To:       &common.Address{1},
			Value:    big.NewInt(1000),
		})
	}
	
	var maxLatency time.Duration
	var totalLatency time.Duration
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		feed.Publish(txs[i], fastfeed.TxEventAdded)
		
		select {
		case <-sub.Events():
			latency := time.Since(start)
			totalLatency += latency
			if latency > maxLatency {
				maxLatency = latency
			}
		case <-time.After(100 * time.Millisecond):
			b.Fatalf("Timeout waiting for event %d", i)
		}
	}
	b.StopTimer()
	
	avgLatency := totalLatency / time.Duration(b.N)
	b.ReportMetric(float64(avgLatency.Nanoseconds()), "ns/event")
	b.ReportMetric(float64(maxLatency.Nanoseconds())/1000, "μs-max")
}

