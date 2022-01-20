package tests

import (
	"math/big"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

func TestYourTurnInitialV2(t *testing.T) {
	config := params.TestXDPoSMockChainConfigWithV2EngineEpochSwitch
	blockchain, _, parentBlock, _ := PrepareXDCTestBlockChain(t, int(config.XDPoS.Epoch)-1, config)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)

	// Insert block 900
	t.Logf("Inserting block with propose at 900...")
	blockCoinbaseA := "0xaaa0000000000000000000000000000000000900"
	//Get from block validator error message
	merkleRoot := "35999dded35e8db12de7e6c1471eb9670c162eec616ecebbaf4fddd4676fb930"
	header := &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(900)),
		ParentHash: parentBlock.Hash(),
		Coinbase:   common.HexToAddress(blockCoinbaseA),
		Extra:      common.Hex2Bytes("d7830100018358444388676f312e31352e38856c696e757800000000000000000278c350152e15fa6ffc712a5a73d704ce73e2e103d9e17ae3ff2c6712e44e25b09ac5ee91f6c9ff065551f0dcac6f00cae11192d462db709be3758ccef312ee5eea8d7bad5374c6a652150515d744508b61c1a4deb4e4e7bf057e4e3824c11fd2569bcb77a52905cda63b5a58507910bed335e4c9d87ae0ecdfafd400"),
	}
	block900, err := insertBlock(blockchain, header)
	if err != nil {
		t.Fatal(err)
	}

	// YourTurn is called before mine first v2 block
	adaptor.YourTurn(blockchain, block900.Header(), common.HexToAddress("xdc0278C350152e15fa6FFC712a5A73D704Ce73E2E1"))
	assert.Equal(t, adaptor.EngineV2.GetCurrentRound(), utils.Round(1))

	snap, err := adaptor.EngineV2.GetSnapshot(blockchain, block900.Header())
	assert.Nil(t, err)
	assert.NotNil(t, snap)
	masterNodes := adaptor.EngineV1.GetMasternodesFromCheckpointHeader(block900.Header())
	for i := 0; i < len(masterNodes); i++ {
		assert.Equal(t, masterNodes[i].Hex(), snap.NextEpochMasterNodes[i].Hex())
	}
}
