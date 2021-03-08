package ethash

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
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
	// TODO: we must check if we are configuring it properly now, for now maxItems and func below are hardcoded
	lruCache := newlru("cache", 12, newCache)
	lruDataset := newlru("dataset", 12, newDataset)

	// Start a simple web vanguardServer to capture notifications.
	sink := make(chan [3]string)
	vanguardServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		blob, err := ioutil.ReadAll(req.Body)
		if err != nil {
			t.Errorf("failed to read miner notification: %v", err)
		}

		var work [3]string

		if err := json.Unmarshal(blob, &work); err != nil {
			t.Errorf("failed to unmarshal miner notification: %v", err)
		}

		sink <- work

		//// TODO: seal header hash by bls validator private key
		//rlpHexHeader := work[2]
		//rlpHeader, err := hexutil.Decode(rlpHexHeader)
		//
		//if nil != err {
		//	t.Errorf("failed to encode hex header")
		//}

		// TODO: Extract this anonymous function without running to vanguard signing process
		//_ = func() {
		//	header := types.Header{}
		//	err = rlp.DecodeBytes(rlpHeader, &header)
		//
		//	if nil != err {
		//		t.Errorf("failed to cast header as rlp")
		//	}
		//
		//	// TODO: This is how it will be signed in vanguard side
		//	// Motivation: you should always be sure that what you sign is valid.
		//	signature := signer.Sign(rlpHeader)
		//	isValidSignature := signature.Verify(signer.PublicKey(), rlpHeader)
		//
		//	if !isValidSignature {
		//		t.Errorf("Invalid signature received")
		//	}
		//}
	}))
	defer vanguardServer.Close()

	ethash := Ethash{
		caches:   lruCache,
		datasets: lruDataset,
		config: Config{
			// In pandora-vanguard implementation we do not need to increase nonce and mixHash is sealed/calculated on the Vanguard side
			PowMode: ModePandora,
			Log:     log.Root(),
		},
		lock:      sync.Mutex{},
		closeOnce: sync.Once{},
	}
	defer func() {
		_ = ethash.Close()
	}()
	urls := make([]string, 0)
	urls = append(urls, vanguardServer.URL)
	remoteSealerServer := StartRemotePandora(&ethash, urls, true)
	ethash.remote = remoteSealerServer
	header := &types.Header{
		ParentHash:  common.Hash{},
		UncleHash:   common.Hash{},
		Coinbase:    common.Address{},
		Root:        common.Hash{},
		TxHash:      common.Hash{},
		ReceiptHash: common.Hash{},
		Difficulty:  big.NewInt(1),
		Number:      big.NewInt(1),
		GasLimit:    0,
		GasUsed:     0,
		Time:        uint64(time.Now().UnixNano()),
		Extra:       nil,
		Nonce:       types.BlockNonce{},
	}

	t.Run("Should make work and notify vanguard", func(t *testing.T) {
		block := types.NewBlockWithHeader(header)
		results := make(chan *types.Block)
		err := ethash.Seal(nil, block, results, nil)
		assert.Nil(t, err)

		select {
		case work := <-sink:
			t.Run("Should have encodable sealHash", func(t *testing.T) {
				sealHash := ethash.SealHash(header).Hex()
				assert.Equal(t, sealHash, work[0])
			})

			t.Run("Should have encodable receiptHash", func(t *testing.T) {
				receiptHash := header.ReceiptHash
				assert.Equal(t, receiptHash.Hex(), work[1])
			})

			t.Run("Should have encodable rlp header in third channel", func(t *testing.T) {
				rlpHexHeader := work[2]
				rlpHeader, err := hexutil.Decode(rlpHexHeader)

				if nil != err {
					t.Errorf("failed to encode hex header")
				}

				header := types.Header{}
				err = rlp.DecodeBytes(rlpHeader, &header)

				if nil != err {
					t.Errorf("failed to cast header as rlp")
				}
			})

			return
		case <-time.After(5 * time.Second):
			t.Fatalf("notification timed out")
		}
	})

	t.Run("Should push valid header with signed data", func(t *testing.T) {
		api := &API{&ethash}

		nonce, digest := header.Nonce, ethash.SealHash(header)

		testcases := []struct {
			headers     []*types.Header
			submitIndex int
			submitRes   bool
		}{
			// Case1: submit solution for the latest mining package
			{
				[]*types.Header{
					header,
				},
				0,
				false,
			},
		}
		results := make(chan *types.Block, 1)

		for id, c := range testcases {
			for _, h := range c.headers {
				ethash.Seal(nil, types.NewBlockWithHeader(h), results, nil)
			}
			res := api.SubmitWork(nonce, ethash.SealHash(c.headers[c.submitIndex]), digest)
			if res != c.submitRes {
				t.Errorf("case %d submit result mismatch, want %t, get %t", id+1, c.submitRes, res)
			}
			if !c.submitRes {
				continue
			}
			select {
			case res := <-results:
				if res.Header().Nonce != nonce {
					t.Errorf("case %d block nonce mismatch, want %x, get %x", id+1, nonce, res.Header().Nonce)
				}
				if res.Header().MixDigest != digest {
					t.Errorf("case %d block digest mismatch, want %x, get %x", id+1, digest, res.Header().MixDigest)
				}
				if res.Header().Difficulty.Uint64() != c.headers[c.submitIndex].Difficulty.Uint64() {
					t.Errorf("case %d block difficulty mismatch, want %d, get %d", id+1, c.headers[c.submitIndex].Difficulty, res.Header().Difficulty)
				}
				if res.Header().Number.Uint64() != c.headers[c.submitIndex].Number.Uint64() {
					t.Errorf("case %d block number mismatch, want %d, get %d", id+1, c.headers[c.submitIndex].Number.Uint64(), res.Header().Number.Uint64())
				}
				if res.Header().ParentHash != c.headers[c.submitIndex].ParentHash {
					t.Errorf("case %d block parent hash mismatch, want %s, get %s", id+1, c.headers[c.submitIndex].ParentHash.Hex(), res.Header().ParentHash.Hex())
				}
			case <-time.NewTimer(time.Second).C:
				t.Errorf("case %d fetch ethash result timeout", id+1)
			}
		}
	})
}
