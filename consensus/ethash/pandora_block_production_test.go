package ethash

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// This file is used for exploration of possible ways to achieve pandora-vanguard block production

// Test RemoteSigner approach connected to each other
func TestProducePandoraBlockViaRemoteSealer(t *testing.T) {
	// Start a simple web server to capture notifications.
	sink := make(chan [3]string)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		blob, err := ioutil.ReadAll(req.Body)
		if err != nil {
			t.Errorf("failed to read miner notification: %v", err)
		}
		var work [3]string
		if err := json.Unmarshal(blob, &work); err != nil {
			t.Errorf("failed to unmarshal miner notification: %v", err)
		}
		sink <- work
	}))
	defer server.Close()

	ethash := Ethash{
		config: Config{
			PowMode: 0,
			Log:     log.Root(),
		},
		lock:      sync.Mutex{},
		closeOnce: sync.Once{},
	}
	defer func() {
		_ = ethash.Close()
	}()
	urls := make([]string, 0)
	urls = append(urls, server.URL)
	remoteSealer := startRemoteSealer(&ethash, urls, true)
	ethash.remote = remoteSealer

	t.Run("Should discard invalid block", func(t *testing.T) {
		header := &types.Header{
			ParentHash:  common.Hash{},
			UncleHash:   common.Hash{},
			Coinbase:    common.Address{},
			Root:        common.Hash{},
			TxHash:      common.Hash{},
			ReceiptHash: common.Hash{},
			Bloom:       types.Bloom{},
			Difficulty:  nil,
			Number:      nil,
			GasLimit:    0,
			GasUsed:     0,
			Time:        0,
			Extra:       nil,
			MixDigest:   common.Hash{},
			Nonce:       types.BlockNonce{},
		}
		block := types.NewBlockWithHeader(header)
		err := ethash.Seal(nil, block, nil, nil)
		assert.Nil(t, err)

		select {
		case work := <-sink:
			if want := ethash.SealHash(header).Hex(); work[0] != want {
				t.Errorf("work packet hash mismatch: have %s, want %s", work[0], want)
			}
			if want := common.BytesToHash(SeedHash(header.Number.Uint64())).Hex(); work[1] != want {
				t.Errorf("work packet seed mismatch: have %s, want %s", work[1], want)
			}
			target := new(big.Int).Div(new(big.Int).Lsh(big.NewInt(1), 256), header.Difficulty)
			if want := common.BytesToHash(target.Bytes()).Hex(); work[2] != want {
				t.Errorf("work packet target mismatch: have %s, want %s", work[2], want)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("notification timed out")
		}
	})

	t.Run("Should push valid header with signed data", func(t *testing.T) {

	})
}
