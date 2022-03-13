package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind/backends"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

func TestCountdownTimeoutToSendTimeoutMessage(t *testing.T) {
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 2251, params.TestXDPoSMockChainConfig, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	timeoutMsg := <-engineV2.BroadcastCh
	poolSize := engineV2.GetTimeoutPoolSizeFaker(timeoutMsg.(*utils.Timeout))
	assert.Equal(t, poolSize, 1)
	assert.NotNil(t, timeoutMsg)
	assert.Equal(t, uint64(1350), timeoutMsg.(*utils.Timeout).GapNumber)
	fmt.Println(timeoutMsg.(*utils.Timeout).GapNumber)
	assert.Equal(t, utils.Round(1), timeoutMsg.(*utils.Timeout).Round)
}

func TestCountdownTimeoutNotToSendTimeoutMessageIfNotInMasternodeList(t *testing.T) {
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 901, params.TestXDPoSMockChainConfig, 0)

	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2
	differentSigner, differentSignFn, err := backends.SimulateWalletAddressAndSignFn()
	assert.Nil(t, err)
	// Let's change the address
	engineV2.Authorize(differentSigner, differentSignFn)

	engineV2.SetNewRoundFaker(blockchain, 1, true)

	select {
	case <-engineV2.BroadcastCh:
		t.Fatalf("Not suppose to receive timeout msg")
	case <-time.After(15 * time.Second): //Countdown is only 1s wait, let's wait for 3s here
	}
}

func TestSyncInfoAfterReachTimeoutSnycThreadhold(t *testing.T) {
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 901, params.TestXDPoSMockChainConfig, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2
	engineV2.SetNewRoundFaker(blockchain, 1, true)

	// Because messages are sending async and on random order, so use this way to test
	var timeoutCounter, syncInfoCounter int
	for i := 0; i < 3; i++ {
		obj := <-engineV2.BroadcastCh
		switch v := obj.(type) {
		case *utils.Timeout:
			timeoutCounter++
		case *utils.SyncInfo:
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
		case *utils.Timeout:
			timeoutCounter++
		case *utils.SyncInfo:
			syncInfoCounter++
		default:
			log.Error("Unknown message type received", "value", v)
		}
	}
	assert.Equal(t, 4, timeoutCounter)
	assert.Equal(t, 2, syncInfoCounter)
}

// Timeout handler
func TestTimeoutMessageHandlerSuccessfullyGenerateTCandSyncInfo(t *testing.T) {
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 11, params.TestXDPoSMockChainConfig, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Set round to 1
	engineV2.SetNewRoundFaker(blockchain, utils.Round(1), false)
	// Create two timeout message which will not reach timeout pool threshold
	timeoutMsg := &utils.Timeout{
		Round:     utils.Round(1),
		Signature: []byte{1},
		GapNumber: 450,
	}

	err := engineV2.TimeoutHandler(blockchain, timeoutMsg)
	assert.Nil(t, err)
	currentRound, _, _, _, _, _ := engineV2.GetPropertiesFaker()
	assert.Equal(t, utils.Round(1), currentRound)
	timeoutMsg = &utils.Timeout{
		Round:     utils.Round(1),
		Signature: []byte{2},
		GapNumber: 450,
	}
	err = engineV2.TimeoutHandler(blockchain, timeoutMsg)
	assert.Nil(t, err)
	currentRound, _, _, _, _, _ = engineV2.GetPropertiesFaker()
	assert.Equal(t, utils.Round(1), currentRound)

	// Send a timeout with different gap number, it shall not trigger timeout pool hook
	timeoutMsg = &utils.Timeout{
		Round:     utils.Round(1),
		Signature: []byte{3},
		GapNumber: 1350,
	}
	err = engineV2.TimeoutHandler(blockchain, timeoutMsg)
	assert.Nil(t, err)
	currentRound, _, _, _, _, _ = engineV2.GetPropertiesFaker()
	assert.Equal(t, utils.Round(1), currentRound)

	// Create a timeout message that should trigger timeout pool hook
	timeoutMsg = &utils.Timeout{
		Round:     utils.Round(1),
		Signature: []byte{4},
		GapNumber: 450,
	}

	err = engineV2.TimeoutHandler(blockchain, timeoutMsg)
	assert.Nil(t, err)

	syncInfoMsg := <-engineV2.BroadcastCh

	currentRound, _, _, _, _, _ = engineV2.GetPropertiesFaker()

	assert.NotNil(t, syncInfoMsg)

	// Shouldn't have QC, however, we did not inilise it, hence will show default empty value
	qc := syncInfoMsg.(*utils.SyncInfo).HighestQuorumCert
	assert.Equal(t, utils.Round(0), qc.ProposedBlockInfo.Round)

	tc := syncInfoMsg.(*utils.SyncInfo).HighestTimeoutCert
	assert.NotNil(t, tc)
	assert.Equal(t, tc.Round, utils.Round(1))
	assert.Equal(t, uint64(450), tc.GapNumber)
	// The signatures shall not include the byte{3} from a different gap number
	sigatures := []utils.Signature{[]byte{1}, []byte{2}, []byte{4}}
	assert.ElementsMatch(t, tc.Signatures, sigatures)
	assert.Equal(t, utils.Round(2), currentRound)
}

func TestThrowErrorIfTimeoutMsgRoundNotEqualToCurrentRound(t *testing.T) {
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 11, params.TestXDPoSMockChainConfig, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	// Set round to 3
	engineV2.SetNewRoundFaker(blockchain, utils.Round(3), false)
	timeoutMsg := &utils.Timeout{
		Round:     utils.Round(2),
		Signature: []byte{1},
	}

	err := engineV2.TimeoutHandler(blockchain, timeoutMsg)
	assert.NotNil(t, err)
	// Timeout msg round > currentRound
	assert.Equal(t, "timeout message round number: 2 does not match currentRound: 3", err.Error())

	// Set round to 1
	engineV2.SetNewRoundFaker(blockchain, utils.Round(1), false)
	err = engineV2.TimeoutHandler(blockchain, timeoutMsg)
	assert.NotNil(t, err)
	// Timeout msg round < currentRound
	assert.Equal(t, "timeout message round number: 2 does not match currentRound: 1", err.Error())
}

func TestShouldVerifyTimeoutMessageForFirstV2Block(t *testing.T) {
	blockchain, _, _, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 901, params.TestXDPoSMockChainConfig, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	signedHash, err := signFn(accounts.Account{Address: signer}, utils.TimeoutSigHash(&utils.TimeoutForSign{
		Round:     utils.Round(1),
		GapNumber: 450,
	}).Bytes())
	assert.Nil(t, err)
	timeoutMsg := &utils.Timeout{
		Round:     utils.Round(1),
		GapNumber: 450,
		Signature: signedHash,
	}

	verified, err := engineV2.VerifyTimeoutMessage(blockchain, timeoutMsg)
	assert.Nil(t, err)
	assert.True(t, verified)

	signedHash, err = signFn(accounts.Account{Address: signer}, utils.TimeoutSigHash(&utils.TimeoutForSign{
		Round:     utils.Round(2),
		GapNumber: 450,
	}).Bytes())
	assert.Nil(t, err)
	timeoutMsg = &utils.Timeout{
		Round:     utils.Round(2),
		GapNumber: 450,
		Signature: signedHash,
	}

	verified, err = engineV2.VerifyTimeoutMessage(blockchain, timeoutMsg)
	assert.Nil(t, err)
	assert.True(t, verified)
}

func TestShouldVerifyTimeoutMessage(t *testing.T) {
	blockchain, _, _, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 2251, params.TestXDPoSMockChainConfig, 0)
	engineV2 := blockchain.Engine().(*XDPoS.XDPoS).EngineV2

	signedHash := SignHashByPK(acc1Key, utils.TimeoutSigHash(&utils.TimeoutForSign{
		Round:     utils.Round(5000),
		GapNumber: 2250,
	}).Bytes())
	timeoutMsg := &utils.Timeout{
		Round:     utils.Round(5000),
		GapNumber: 2250,
		Signature: signedHash,
	}

	verified, err := engineV2.VerifyTimeoutMessage(blockchain, timeoutMsg)
	assert.Nil(t, err)
	assert.True(t, verified)
}
