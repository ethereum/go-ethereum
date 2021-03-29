package ethash

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	common2 "github.com/silesiacoin/bls/common"
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
)

type fakeReader struct {
	chainConfig *params.ChainConfig
}

func (f fakeReader) Config() *params.ChainConfig {
	return f.chainConfig
}

func (f fakeReader) CurrentHeader() *types.Header {
	panic("implement me")
}

func (f fakeReader) GetHeader(hash common.Hash, number uint64) *types.Header {
	return &types.Header{Number: big.NewInt(1)}
}

func (f fakeReader) GetHeaderByNumber(number uint64) *types.Header {
	panic("implement me")
}

func (f fakeReader) GetHeaderByHash(hash common.Hash) *types.Header {
	panic("implement me")
}

var chainHeaderReader consensus.ChainHeaderReader = fakeReader{
	chainConfig: &params.ChainConfig{
		ChainID:             big.NewInt(256),
		HomesteadBlock:      big.NewInt(0),
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.Hash{},
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		SilesiaBlock:        big.NewInt(0),
	}}

// This file is used for exploration of possible ways to achieve pandora-vanguard block production
// Test RemoteSigner approach connected to each other
func TestCreateBlockByPandoraAndVanguard(t *testing.T) {
	lruCache := newlru("cache", 12, newCache)
	lruDataset := newlru("dataset", 12, newDataset)

	privKey, err := herumi.RandKey()
	assert.Nil(t, err)

	pubKey := privKey.PublicKey()

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
			signature := privKey.Sign(signatureBytes)

			isValidSignature := signature.Verify(pubKey, signatureBytes)

			if !isValidSignature {
				t.Errorf("Invalid signature received")
			}

			blsSignatureBytes := BlsSignatureBytes{}
			copy(blsSignatureBytes[:], signature.Marshal())

			//TODO: consider if it is needed
			header.MixDigest = common.BytesToHash(blsSignatureBytes[:])

			// Cast to []byte from [32]byte. This should prevent cropping
			workSubmittedLock.Done()

			// This in test is using work channel to push sealed block to pandora back
			submitWorkChannel <- &mineResult{
				nonce:     types.BlockNonce{},
				mixDigest: header.MixDigest,
				hash:      common.HexToHash(work[0]),
				blsSeal:   &blsSignatureBytes,
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
			//TODO: change this API, extend it or provide another one like SubmitSeal?
			submitted := ethashAPI.SubmitWorkBLS(
				submittedWork.nonce,
				submittedWork.hash,
				hexutil.Encode(submittedWork.blsSeal[:]),
			)

			//mixDigest := submittedWork.mixDigest
			blsSignatureBytes := submittedWork.blsSeal
			blsSignature, err := herumi.SignatureFromBytes(blsSignatureBytes[:])
			assert.Nil(t, err)
			assert.NotNil(t, blsSignature)

			hash := submittedWork.hash
			hashBytes := hash.Bytes()
			signatureValid := blsSignature.Verify(pubKey, hashBytes)
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

func TestEthash_Prepare_Pandora(t *testing.T) {
	lruCache := newlru("cache", 12, newCache)
	lruDataset := newlru("dataset", 12, newDataset)
	lruEpochSet := newlru("epochSet", 12, NewMinimalConsensusInfo)

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

	genesisEpoch := NewMinimalConsensusInfo(0).(*MinimalEpochConsensusInfo)
	genesisStart := time.Now()

	validatorPublicList := [32]common2.PublicKey{}
	validatorPrivateList := [32]common2.SecretKey{}

	for index, _ := range validatorPrivateList {
		privKey, err := herumi.RandKey()
		assert.Nil(t, err)
		pubKey := privKey.PublicKey()
		validatorPublicList[index] = pubKey
		validatorPrivateList[index] = privKey
	}

	genesisEpoch.AssignEpochStartFromGenesis(genesisStart)
	genesisEpoch.AssignValidators(validatorPublicList)
	lruEpochSet.cache.Add(0, genesisEpoch)

	header := &types.Header{
		ParentHash:  common.Hash{},
		UncleHash:   common.Hash{},
		Coinbase:    common.Address{},
		Root:        common.Hash{},
		TxHash:      common.Hash{},
		ReceiptHash: common.Hash{},
		Bloom:       types.Bloom{},
		Difficulty:  nil,
		Number:      big.NewInt(0),
		GasLimit:    0,
		GasUsed:     0,
		Time:        uint64(time.Now().Unix()),
		Extra:       nil,
		MixDigest:   common.Hash{},
		Nonce:       types.BlockNonce{},
	}

	err := ethash.Prepare(chainHeaderReader, header)
	assert.Nil(t, err)

	expectedData, err := NewPandoraExtraData(header, genesisEpoch)
	assert.Nil(t, err)

	expectedDataBytes, err := rlp.EncodeToBytes(expectedData)
	assert.Nil(t, err)

	assert.Equal(t, expectedDataBytes, header.Extra)
}

func TestMinimalEpochConsensusInfo_AssignEpochStartFromGenesis(t *testing.T) {
	maxIterations := 2 ^ 7
	genesisTime := time.Now()

	for i := 0; i < maxIterations; i++ {
		minimalEpochConsensusInfo := NewMinimalConsensusInfo(uint64(i))
		consensusInfo := minimalEpochConsensusInfo.(*MinimalEpochConsensusInfo)
		consensusInfo.AssignEpochStartFromGenesis(genesisTime)
		epochTimeStart := consensusInfo.EpochTimeStart
		seconds := time.Duration(slotTimeDuration) * time.Second * time.Duration(i)
		expectedEpochTime := genesisTime.Add(seconds)
		assert.Equal(t, expectedEpochTime.Unix(), epochTimeStart.Unix())
	}
}

func TestVerifySeal(t *testing.T) {
	lruCache := newlru("cache", 12, newCache)
	lruDataset := newlru("dataset", 12, newDataset)
	lruEpochSet := newlru("epochSet", 12, NewMinimalConsensusInfo)
	validatorPublicList := [32]common2.PublicKey{}
	validatorPrivateList := [32]common2.SecretKey{}

	for index, _ := range validatorPrivateList {
		privKey, err := herumi.RandKey()
		assert.Nil(t, err)
		pubKey := privKey.PublicKey()
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
			headerTime := genesisEpoch.EpochTimeStart.Add(slotTimeDuration * time.Second * time.Duration(index))
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
			headers = append(headers, header)

			err := ethash.verifySeal(nil, header, false)

			assert.Nil(
				t,
				err,
				fmt.Sprintf("failed on index: %d, with headerExtra: %v", index, header.Extra),
			)
		}

		t.Run("Should fail before overflowing the validator set length", func(t *testing.T) {
			// Check next epoch
			nextEpochHeaderNumber := len(validatorPrivateList) + 1
			headerTime := genesisEpoch.EpochTimeStart.Add(
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
		privateKey, err := herumi.RandKey()
		assert.Nil(t, err)

		for index, _ := range validatorPrivateList {
			headerTime := genesisEpoch.EpochTimeStart.Add(slotTimeDuration * time.Second * time.Duration(index))
			// Add additional second to not be on start of the slot
			randMin := 0
			randMax := 5
			randomInterval := mathRand.Intn(randMax-randMin) + randMin
			headerTime = headerTime.Add(time.Second * time.Duration(randomInterval))
			header, sealHash, blsSignature := generatePandoraSealedHeaderByKey(
				privateKey,
				int64(index),
				headerTime,
				&PandoraExtraData{},
			)
			headers = append(headers, header)

			expectedErr := fmt.Errorf(
				"invalid signature: %s in header hash: %s with sealHash: %s",
				blsSignature.Marshal(),
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
	privKey common2.SecretKey,
	headerNumber int64,
	headerTime time.Time,
	extraData *PandoraExtraData,
) (
	header *types.Header,
	headerHash common.Hash,
	blsSignature common2.Signature,
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
	signature := privKey.Sign(headerHash.Bytes())

	extraDataWithSignature := new(PandoraExtraDataSealed)
	extraDataWithSignature.FromExtraDataAndSignature(*extraData, signature)
	verified := signature.Verify(privKey.PublicKey(), headerHash.Bytes())

	if !verified {
		panic("Signature should be valid")
	}

	compressedSignature := signature.Marshal()[:herumi.CompressedSize]
	header.MixDigest = common.BytesToHash(compressedSignature[:])
	blsSignature = signature

	extraDataBytes, err := rlp.EncodeToBytes(extraDataWithSignature)

	if nil != err {
		panic(err.Error())
	}

	header.Extra = extraDataBytes

	return
}
