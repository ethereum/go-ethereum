package ethash

import (
	"crypto/rand"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/silesiacoin/bls/herumi"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
	vbls "vuvuzela.io/crypto/bls"
)

// This file is used for exploration of possible ways to achieve pandora-vanguard block production
// Test RemoteSigner approach connected to each other
func TestCreateBlockByPandoraAndVanguard(t *testing.T) {
	// TODO: we must check if we are configuring it properly now, for now maxItems and func below are hardcoded
	lruCache := newlru("cache", 12, newCache)
	lruDataset := newlru("dataset", 12, newDataset)

	randomReader := rand.Reader
	pubKey, privKey, err := herumi.GenerateKey(randomReader)
	assert.Nil(t, err)

	pubKeySet := make([]*vbls.PublicKey, 0)
	pubKeySet = append(pubKeySet, pubKey)

	workSubmittedLock := sync.WaitGroup{}
	workSubmittedLock.Add(1)

	// Start a simple web vanguardServer to capture notifications.
	workChannel := make(chan [4]string)
	submitWorkChannel := make(chan *mineResult)

	// This is used to mimic server on vanguard which will consume pandora
	vanguardServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		blob, err := ioutil.ReadAll(req.Body)
		if err != nil {
			t.Errorf("failed to read miner notification: %v", err)
		}

		var work [4]string

		if err := json.Unmarshal(blob, &work); err != nil {
			t.Errorf("failed to unmarshal miner notification: %v", err)
		}

		workChannel <- work

		// TODO: seal header hash by bls validator private key
		rlpHexHeader := work[2]
		rlpHeader, err := hexutil.Decode(rlpHexHeader)

		if nil != err {
			t.Errorf("failed to encode hex header")
		}

		//TODO: Extract this function without running to vanguard signing process
		signerFunc := func() {
			header := types.Header{}
			err = rlp.DecodeBytes(rlpHeader, &header)

			if nil != err {
				t.Errorf("failed to cast header as rlp")
			}

			// TODO: This is how it will be signed on the vanguard side
			// Motivation: you should always be sure that what you sign is valid.
			// We sign hash
			signatureBytes, err := hexutil.Decode(work[0])
			assert.Nil(t, err)
			signature := herumi.Sign(privKey, signatureBytes)
			compressedSignature := signature.Compress()
			messages := make([][]byte, 0)
			messages = append(messages, signatureBytes)
			isValidSignature := herumi.VerifyCompressed(pubKeySet, messages, compressedSignature)

			if !isValidSignature {
				t.Errorf("Invalid signature received")
			}

			// Cast to []byte from [32]byte. This should prevent cropping
			header.MixDigest = common.BytesToHash(compressedSignature[:])
			workSubmittedLock.Done()

			// TODO: With networking: return header via channel `submitWork`
			// This in test is using work channel to push sealed block to pandora back
			submitWorkChannel <- &mineResult{
				nonce:     types.BlockNonce{},
				mixDigest: header.MixDigest,
				hash:      common.HexToHash(work[0]),
				errc:      nil,
			}
		}

		signerFunc()
	}))
	defer vanguardServer.Close()

	// This is how ethash would be designed to serve vanguard
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
	ethashAPI := API{ethash: &ethash}

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
		case work := <-workChannel:
			t.Run("Should have encodable sealHash", func(t *testing.T) {
				sealHash := ethash.SealHash(header).Hex()
				assert.Equal(t, sealHash, work[0])
			})

			t.Run("Should have encodable receiptHash", func(t *testing.T) {
				receiptHash := header.ReceiptHash
				assert.Equal(t, receiptHash.Hex(), work[1])
			})

			t.Run("Should have encodable rlp header", func(t *testing.T) {
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

			t.Run("Should have encodable block number", func(t *testing.T) {
				assert.Equal(t, hexutil.Encode(header.Number.Bytes()), work[3])
			})

		//	Whole procedure should take no more than 1/2 of a slot. Lets test that for 6s / 4 ~ 2s
		case <-time.After(2 * time.Second):
			t.Fatalf("notification timed out")
		}

		// Wait until work is submitted back
		workSubmittedLock.Wait()

		select {
		// This is created by channel to remove network complexity for test scenario.
		// Full E2E layer should be somwhere else, or we should consider stress test
		case submittedWork := <-submitWorkChannel:
			submitted := ethashAPI.SubmitWork(
				submittedWork.nonce,
				submittedWork.hash,
				submittedWork.mixDigest,
			)

			mixDigest := submittedWork.mixDigest
			// Check if signature of header is valid
			messages := make([][]byte, 0)
			messages = append(messages, submittedWork.hash.Bytes())
			signature := [32]byte{}
			copy(signature[:], mixDigest.Bytes())
			signatureValid := herumi.VerifyCompressed(pubKeySet, messages, &signature)
			assert.True(t, signatureValid)

			// This will return false, if anything goes wrong
			// TODO: debug why non-blocking result channel returns false
			// see: `https://gobyexample.com/non-blocking-channel-operations`
			//  Catch it at: consensus/ethash/sealer.go:447
			assert.False(t, submitted)
		case <-time.After(2 * time.Second):
			t.Fatalf("notification timed out")
		}
	})
}
