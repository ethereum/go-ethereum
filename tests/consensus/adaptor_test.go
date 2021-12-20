package consensus

import (
	"fmt"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

func TestAdaptorShouldGetAuthorForDifferentConsensusVersion(t *testing.T) {
	blockchain, _, currentBlock := PrepareXDCTestBlockChain(t, 10, params.TestXDPoSMockChainConfigWithV2Engine)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)

	addressFromAdaptor, errorAdaptor := adaptor.Author(currentBlock.Header())
	if errorAdaptor != nil {
		t.Fatalf("Failed while trying to get Author from adaptor")
	}
	addressFromV1Engine, errV1 := adaptor.EngineV1.Author(currentBlock.Header())
	if errV1 != nil {
		t.Fatalf("Failed while trying to get Author from engine v1")
	}
	// Make sure the value is exactly the same as from V1 engine
	assert.Equal(t, addressFromAdaptor, addressFromV1Engine)

	// Insert one more block to make it above 10, which means now we are on v2 of consensus engine
	// Insert block 11
	blockCoinBase := fmt.Sprintf("0x111000000000000000000000000000000%03d", 11)
	merkleRoot := "35999dded35e8db12de7e6c1471eb9670c162eec616ecebbaf4fddd4676fb930"
	block11, err := insertBlock(blockchain, 11, blockCoinBase, currentBlock, merkleRoot, 1)
	if err != nil {
		t.Fatal(err)
	}

	addressFromAdaptor, errorAdaptor = adaptor.Author(block11.Header())
	if errorAdaptor != nil {
		t.Fatalf("Failed while trying to get Author from adaptor")
	}
	addressFromV2Engine, errV2 := adaptor.EngineV2.Author(block11.Header())
	if errV2 != nil {
		t.Fatalf("Failed while trying to get Author from engine v2")
	}
	// Make sure the value is exactly the same as from V2 engine
	assert.Equal(t, addressFromAdaptor, addressFromV2Engine)
}
