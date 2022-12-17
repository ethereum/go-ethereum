package engine_v2_tests

import (
	"math/big"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind/backends"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

func TestCountdownTimeoutToSendTimeoutMessage(t *testing.T) {
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 901, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	timeoutMsg := <-engineV2.BroadcastCh
	poolSize := engineV2.GetTimeoutPoolSizeFaker(timeoutMsg.(*types.Timeout))
	assert.Equal(t, poolSize, 1)
	assert.NotNil(t, timeoutMsg)
	assert.Equal(t, uint64(450), timeoutMsg.(*types.Timeout).GapNumber)
	assert.Equal(t, types.Round(1), timeoutMsg.(*types.Timeout).Round)
}

func TestCountdownTimeoutNotToSendTimeoutMessageIfNotInMasternodeList(t *testing.T) {
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 901, params.TestXDPoSMockChainConfig, nil)

	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2
	differentSigner, differentSignFn, err := backends.SimulateWalletAddressAndSignFn()
	assert.Nil(t, err)
	// Let's change the address
	engineV2.Authorize(differentSigner, differentSignFn)

	engineV2.SetNewRoundFaker(blockchain, 1, true)

	select {
	case <-engineV2.BroadcastCh:
		t.Fatalf("Not suppose to receive timeout msg")
	case <-time.After(10 * time.Second): //Countdown is only 1s wait, let's wait for 3s here
	}
}

func TestSyncInfoAfterReachTimeoutSyncThreadhold(t *testing.T) {
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 901, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2
	engineV2.SetNewRoundFaker(blockchain, 1, true)

	// Because messages are sending async and on random order, so use this way to test
	var timeoutCounter, syncInfoCounter int
	for i := 0; i < 3; i++ {
		obj := <-engineV2.BroadcastCh
		switch v := obj.(type) {
		case *types.Timeout:
			timeoutCounter++
		case *types.SyncInfo:
			syncInfoCounter++
		default:
			log.Error("Unknown message type received", "value", v)
		}
	}
	assert.Equal(t, 2, timeoutCounter)
	assert.Equal(t, 1, syncInfoCounter)

	t.Log("waiting for another consecutive period")
	// another consecutive period
	for i := 0; i < 3; i++ {
		obj := <-engineV2.BroadcastCh
		switch v := obj.(type) {
		case *types.Timeout:
			timeoutCounter++
		case *types.SyncInfo:
			syncInfoCounter++
		default:
			log.Error("Unknown message type received", "value", v)
		}
	}
	assert.Equal(t, 4, timeoutCounter)
	assert.Equal(t, 2, syncInfoCounter)
}

func TestTimeoutPeriodAndThreadholdConfigChange(t *testing.T) {
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 1799, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2
	// engineV2.SetNewRoundFaker(blockchain, 1, true)

	// Because messages are sending async and on random order, so use this way to test
	var timeoutCounter, syncInfoCounter int
	for i := 0; i < 3; i++ {
		obj := <-engineV2.BroadcastCh
		switch v := obj.(type) {
		case *types.Timeout:
			timeoutCounter++
		case *types.SyncInfo:
			syncInfoCounter++
		default:
			log.Error("Unknown message type received", "value", v)
		}
	}

	assert.Equal(t, 2, timeoutCounter)
	assert.Equal(t, 1, syncInfoCounter)

	// Create another block to trigger update parameters
	blockNum := 1800
	blockCoinBase := "0x111000000000000000000000000000000123"
	currentBlock = CreateBlock(blockchain, params.TestXDPoSMockChainConfig, currentBlock, blockNum, 900, blockCoinBase, signer, signFn, nil, nil)
	currentBlockHeader := currentBlock.Header()
	currentBlockHeader.Time = big.NewInt(time.Now().Unix())
	err := blockchain.InsertBlock(currentBlock)
	assert.Nil(t, err)

	engineV2.UpdateParams(currentBlockHeader) // it will be triggered automatically on the real code by other process

	t.Log("waiting for another consecutive period")
	// another consecutive period
	t1 := time.Now()
	for i := 0; i < 5; i++ {
		obj := <-engineV2.BroadcastCh
		switch v := obj.(type) {
		case *types.Timeout:
			timeoutCounter++
		case *types.SyncInfo:
			syncInfoCounter++
		default:
			log.Error("Unknown message type received", "value", v)
		}
	}
	t2 := time.Now()
	timediff := t2.Sub(t1).Seconds()
	assert.Equal(t, 6, timeoutCounter)
	assert.Equal(t, 2, syncInfoCounter)
	assert.Less(t, timediff, float64(20))
}

// Timeout handler
func TestTimeoutMessageHandlerSuccessfullyGenerateTCandSyncInfo(t *testing.T) {
	params.TestXDPoSMockChainConfig.XDPoS.V2.CurrentConfig = params.TestV2Configs[0]
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 11, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Set round to 1
	engineV2.SetNewRoundFaker(blockchain, types.Round(1), false)
	// Create two timeout message which will not reach timeout pool threshold
	timeoutMsg := &types.Timeout{
		Round:     types.Round(1),
		Signature: []byte{1},
		GapNumber: 450,
	}

	err := engineV2.TimeoutHandler(blockchain, timeoutMsg)
	assert.Nil(t, err)
	currentRound, _, _, _, _, _ := engineV2.GetPropertiesFaker()
	assert.Equal(t, types.Round(1), currentRound)
	timeoutMsg = &types.Timeout{
		Round:     types.Round(1),
		Signature: []byte{2},
		GapNumber: 450,
	}
	err = engineV2.TimeoutHandler(blockchain, timeoutMsg)
	assert.Nil(t, err)
	currentRound, _, _, _, _, _ = engineV2.GetPropertiesFaker()
	assert.Equal(t, types.Round(1), currentRound)

	// Send a timeout with different gap number, it shall not trigger timeout pool hook
	timeoutMsg = &types.Timeout{
		Round:     types.Round(1),
		Signature: []byte{3},
		GapNumber: 1350,
	}
	err = engineV2.TimeoutHandler(blockchain, timeoutMsg)
	assert.Nil(t, err)
	currentRound, _, _, _, _, _ = engineV2.GetPropertiesFaker()
	assert.Equal(t, types.Round(1), currentRound)

	// Create a timeout message that should trigger timeout pool hook
	timeoutMsg = &types.Timeout{
		Round:     types.Round(1),
		Signature: []byte{4},
		GapNumber: 450,
	}

	err = engineV2.TimeoutHandler(blockchain, timeoutMsg)
	assert.Nil(t, err)

	syncInfoMsg := <-engineV2.BroadcastCh

	currentRound, _, _, _, _, _ = engineV2.GetPropertiesFaker()

	assert.NotNil(t, syncInfoMsg)

	// Shouldn't have QC, however, we did not inilise it, hence will show default empty value
	qc := syncInfoMsg.(*types.SyncInfo).HighestQuorumCert
	assert.Equal(t, types.Round(0), qc.ProposedBlockInfo.Round)

	tc := syncInfoMsg.(*types.SyncInfo).HighestTimeoutCert
	assert.NotNil(t, tc)
	assert.Equal(t, tc.Round, types.Round(1))
	assert.Equal(t, uint64(450), tc.GapNumber)
	// The signatures shall not include the byte{3} from a different gap number
	sigatures := []types.Signature{[]byte{1}, []byte{2}, []byte{4}}
	assert.ElementsMatch(t, tc.Signatures, sigatures)
	assert.Equal(t, types.Round(2), currentRound)
}

func TestThrowErrorIfTimeoutMsgRoundNotEqualToCurrentRound(t *testing.T) {
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 11, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Set round to 3
	engineV2.SetNewRoundFaker(blockchain, types.Round(3), false)
	timeoutMsg := &types.Timeout{
		Round:     types.Round(2),
		Signature: []byte{1},
	}

	err := engineV2.TimeoutHandler(blockchain, timeoutMsg)
	assert.NotNil(t, err)
	// Timeout msg round > currentRound
	assert.Equal(t, "timeout message round number: 2 does not match currentRound: 3", err.Error())

	// Set round to 1
	engineV2.SetNewRoundFaker(blockchain, types.Round(1), false)
	err = engineV2.TimeoutHandler(blockchain, timeoutMsg)
	assert.NotNil(t, err)
	// Timeout msg round < currentRound
	assert.Equal(t, "timeout message round number: 2 does not match currentRound: 1", err.Error())
}

func TestShouldVerifyTimeoutMessageForFirstV2Block(t *testing.T) {
	blockchain, _, _, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 901, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	signedHash, err := signFn(accounts.Account{Address: signer}, types.TimeoutSigHash(&types.TimeoutForSign{
		Round:     types.Round(1),
		GapNumber: 450,
	}).Bytes())
	assert.Nil(t, err)
	timeoutMsg := &types.Timeout{
		Round:     types.Round(1),
		GapNumber: 450,
		Signature: signedHash,
	}

	verified, err := engineV2.VerifyTimeoutMessage(blockchain, timeoutMsg)
	assert.Nil(t, err)
	assert.True(t, verified)

	signedHash, err = signFn(accounts.Account{Address: signer}, types.TimeoutSigHash(&types.TimeoutForSign{
		Round:     types.Round(2),
		GapNumber: 450,
	}).Bytes())
	assert.Nil(t, err)
	timeoutMsg = &types.Timeout{
		Round:     types.Round(2),
		GapNumber: 450,
		Signature: signedHash,
	}

	verified, err = engineV2.VerifyTimeoutMessage(blockchain, timeoutMsg)
	assert.Nil(t, err)
	assert.True(t, verified)
}

func TestShouldVerifyTimeoutMessage(t *testing.T) {
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 2251, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	signedHash := SignHashByPK(acc1Key, types.TimeoutSigHash(&types.TimeoutForSign{
		Round:     types.Round(5000),
		GapNumber: 2250,
	}).Bytes())
	timeoutMsg := &types.Timeout{
		Round:     types.Round(5000),
		GapNumber: 2250,
		Signature: signedHash,
	}

	verified, err := engineV2.VerifyTimeoutMessage(blockchain, timeoutMsg)
	assert.Nil(t, err)
	assert.True(t, verified)
}

func TestTimeoutPoolKeeyGoodHygiene(t *testing.T) {
	blockchain, _, _, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 905, params.TestXDPoSMockChainConfig, nil)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Set round to 5
	engineV2.SetNewRoundFaker(blockchain, types.Round(5), false)
	// Inject the first timeout with round 5

	signedHash, _ := signFn(accounts.Account{Address: signer}, types.TimeoutSigHash(&types.TimeoutForSign{
		Round:     types.Round(5),
		GapNumber: 450,
	}).Bytes())
	timeoutMsg := &types.Timeout{
		Round:     types.Round(5),
		GapNumber: 450,
		Signature: signedHash,
	}
	engineV2.TimeoutHandler(blockchain, timeoutMsg)

	// Inject a second timeout with round 16
	signedHash, _ = signFn(accounts.Account{Address: signer}, types.TimeoutSigHash(&types.TimeoutForSign{
		Round:     types.Round(16),
		GapNumber: 450,
	}).Bytes())
	timeoutMsg = &types.Timeout{
		Round:     types.Round(16),
		GapNumber: 450,
		Signature: signedHash,
	}
	// Set round to 16
	engineV2.SetNewRoundFaker(blockchain, types.Round(16), false)
	engineV2.TimeoutHandler(blockchain, timeoutMsg)

	// Inject a third timeout with round 17
	signedHash, _ = signFn(accounts.Account{Address: signer}, types.TimeoutSigHash(&types.TimeoutForSign{
		Round:     types.Round(17),
		GapNumber: 450,
	}).Bytes())
	timeoutMsg = &types.Timeout{
		Round:     types.Round(17),
		GapNumber: 450,
		Signature: signedHash,
	}
	// Set round to 16
	engineV2.SetNewRoundFaker(blockchain, types.Round(17), false)
	engineV2.TimeoutHandler(blockchain, timeoutMsg)

	// Let's keep good Hygiene
	engineV2.HygieneTimeoutPoolFaker()
	// Let's wait for 5 second for the goroutine
	<-time.After(5 * time.Second)
	keyList := engineV2.GetTimeoutPoolKeyListFaker()

	assert.Equal(t, 2, len(keyList))
	for _, k := range keyList {
		keyedRound, err := strconv.ParseInt(strings.Split(k, ":")[0], 10, 64)
		assert.Nil(t, err)
		if keyedRound < 25-10 {
			assert.Fail(t, "Did not clean up the timeout pool")
		}
	}
}
