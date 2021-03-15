package ethash

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/silesiacoin/bls/herumi"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math/big"
	mathRand "math/rand"
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
		rlpHexHeader := work[2]
		rlpHeader, err := hexutil.Decode(rlpHexHeader)

		if nil != err {
			t.Errorf("failed to encode hex header")
		}

		signerFunc := func() {
			header := types.Header{}
			err = rlp.DecodeBytes(rlpHeader, &header)

			if nil != err {
				t.Errorf("failed to cast header as rlp")
			}

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
		// Full E2E layer should be somewhere else, or we should consider stress test
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

func TestReceiveValidatorsForEpoch(t *testing.T) {

}

func TestMinimalEpochConsensusInfo_AssignEpochStartFromGenesis(t *testing.T) {
	random := 2 ^ 7
	genesisTime := time.Now()

	for i := 0; i < random; i++ {
		minimalEpochConsensusInfo := NewMinimalConsensusInfo(uint64(i))
		consensusInfo := minimalEpochConsensusInfo.(*MinimalEpochConsensusInfo)
		consensusInfo.AssignEpochStartFromGenesis(genesisTime)
		epochTimeStart := consensusInfo.epochTimeStart
		seconds := time.Duration(slotTimeDuration) * time.Second * time.Duration(i)
		expectedEpochTime := genesisTime.Add(seconds)
		assert.Equal(t, expectedEpochTime.Unix(), epochTimeStart.Unix())
	}
}

func TestVerifySeal(t *testing.T) {
	lruCache := newlru("cache", 12, newCache)
	lruDataset := newlru("dataset", 12, newDataset)
	lruEpochSet := newlru("epochSet", 12, NewMinimalConsensusInfo)
	validatorPublicList := [32]*vbls.PublicKey{}
	validatorPrivateList := [32]*vbls.PrivateKey{}

	for index, _ := range validatorPublicList {
		randomReader := rand.Reader
		pubKey, privKey, err := herumi.GenerateKey(randomReader)
		assert.Nil(t, err)
		validatorPublicList[index] = pubKey
		validatorPrivateList[index] = privKey
	}

	genesisEpoch := NewMinimalConsensusInfo(0).(*MinimalEpochConsensusInfo)
	genesisStart := time.Now()
	genesisEpoch.AssignEpochStartFromGenesis(genesisStart)
	genesisEpoch.AssignValidators(validatorPublicList)
	// Should not be evicted
	assert.False(t, lruEpochSet.cache.Add(0, genesisEpoch))
	assert.True(t, lruEpochSet.cache.Contains(0))
	assert.False(t, lruEpochSet.cache.Add(1, genesisEpoch))
	assert.True(t, lruEpochSet.cache.Contains(1))
	// Check epochs from cache
	genesisEpochFromCache, ok := lruEpochSet.cache.Get(0)
	assert.Equal(t, genesisEpochFromCache, genesisEpoch)
	assert.True(t, ok)
	nextEpochSet, ok := lruEpochSet.cache.Get(1)
	assert.Equal(t, nextEpochSet, genesisEpoch)
	assert.True(t, ok)

	// This is how ethash would be designed to serve vanguard
	ethash := Ethash{
		caches:   lruCache,
		datasets: lruDataset,
		mci:      lruEpochSet,
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

	t.Run("Should increment over slots in first epoch", func(t *testing.T) {
		headers := make([]*types.Header, 0)
		for index, privateKey := range validatorPrivateList {
			headerTime := genesisEpoch.epochTimeStart.Add(slotTimeDuration * time.Second * time.Duration(index))
			// Add additional second to not be on start of the slot
			randMin := 0
			randMax := 5
			randomInterval := mathRand.Intn(randMax-randMin) + randMin
			headerTime = headerTime.Add(time.Second * time.Duration(randomInterval))
			extraData := &PandoraExtraData{
				Slot:          uint64(index),
				Epoch:         uint64(0),
				ProposerIndex: uint64(index),
			}
			header, _, _ := generatePandoraSealedHeaderByKey(privateKey, int64(index), headerTime, extraData)
			// This will take long time to run for whole suite. Consider running it in other manner
			headers = append(headers, header)
			assert.Nil(
				t,
				ethash.verifySeal(nil, header, false),
				fmt.Sprintf("failed on index: %d, with header: %v", index, header),
			)
		}

		t.Run("Should fail before overflowing the validator set length", func(t *testing.T) {
			// Check next epoch
			nextEpochHeaderNumber := len(validatorPrivateList) + 1
			headerTime := genesisEpoch.epochTimeStart.Add(
				slotTimeDuration * time.Second * time.Duration(nextEpochHeaderNumber))
			header, _, _ := generatePandoraSealedHeaderByKey(
				validatorPrivateList[0],
				int64(nextEpochHeaderNumber),
				headerTime,
				// This is checked at last so can be empty
				&PandoraExtraData{},
			)
			assert.Error(
				t,
				ethash.verifySeal(nil, header, false),
				fmt.Sprintf("should fail on index: %d, with header: %v", nextEpochHeaderNumber, header),
			)
		})
	})

	t.Run("Should discard invalid slot sealer in second epoch", func(t *testing.T) {
		headers := make([]*types.Header, 0)
		randomReader := rand.Reader
		_, privateKey, err := herumi.GenerateKey(randomReader)
		assert.Nil(t, err)

		for index, _ := range validatorPrivateList {
			headerTime := genesisEpoch.epochTimeStart.Add(slotTimeDuration * time.Second * time.Duration(index))
			// Add additional second to not be on start of the slot
			randMin := 0
			randMax := 5
			randomInterval := mathRand.Intn(randMax-randMin) + randMin
			headerTime = headerTime.Add(time.Second * time.Duration(randomInterval))
			header, sealHash, mixDigest := generatePandoraSealedHeaderByKey(
				privateKey,
				int64(index),
				headerTime,
				&PandoraExtraData{},
			)
			headers = append(headers, header)
			expectedErr := fmt.Errorf(
				"invalid mixDigest: %s in header hash: %s with sealHash: %s",
				mixDigest.String(),
				header.Hash().String(),
				sealHash.String(),
			)
			assert.Equal(
				t,
				expectedErr,
				ethash.verifySeal(nil, header, false),
			)
		}
	})
}

func generatePandoraSealedHeaderByKey(
	privKey *vbls.PrivateKey,
	headerNumber int64,
	headerTime time.Time,
	extraData *PandoraExtraData,
) (
	header *types.Header,
	headerHash common.Hash,
	mixDigest common.Hash,
) {
	ethash := NewTester(nil, true)
	extraDataBytes, _ := rlp.EncodeToBytes(extraData)

	header = &types.Header{
		ParentHash:  common.Hash{},
		UncleHash:   common.Hash{},
		Coinbase:    common.Address{},
		Root:        common.Hash{},
		TxHash:      common.Hash{},
		ReceiptHash: common.Hash{},
		Bloom:       types.Bloom{},
		Difficulty:  big.NewInt(1),
		Number:      big.NewInt(headerNumber),
		GasLimit:    0,
		GasUsed:     0,
		Time:        uint64(headerTime.Unix()),
		Extra:       extraDataBytes,
		MixDigest:   common.Hash{},
		Nonce:       types.BlockNonce{},
	}

	headerHash = ethash.SealHash(header)
	signature := herumi.Sign(privKey, headerHash.Bytes())
	compressedSignature := signature.Compress()
	header.MixDigest = common.BytesToHash(compressedSignature[:])
	mixDigest = header.MixDigest

	return
}
