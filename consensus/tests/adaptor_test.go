package tests

import (
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

func TestAdaptorShouldGetAuthorForDifferentConsensusVersion(t *testing.T) {
	blockchain, backend, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 10, params.TestXDPoSMockChainConfigWithV2Engine, 0)
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
	header := &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(11)),
		ParentHash: currentBlock.Hash(),
		Coinbase:   common.HexToAddress(blockCoinBase),
	}
	err := generateSignature(backend, header)
	if err != nil {
		t.Fatal(err)
	}
	block11, err := insertBlock(blockchain, header)
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

func TestAdaptorGetMasternodesFromCheckpointHeader(t *testing.T) {
	blockchain, _, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 1, params.TestXDPoSMockChainConfigWithV2Engine, 0)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)
	headerV1 := currentBlock.Header()
	headerV1.Extra = common.Hex2Bytes("d7830100018358444388676f312e31352e38856c696e757800000000000000000278c350152e15fa6ffc712a5a73d704ce73e2e103d9e17ae3ff2c6712e44e25b09ac5ee91f6c9ff065551f0dcac6f00cae11192d462db709be3758ccef312ee5eea8d7bad5374c6a652150515d744508b61c1a4deb4e4e7bf057e4e3824c11fd2569bcb77a52905cda63b5a58507910bed335e4c9d87ae0ecdfafd400")
	masternodesV1 := adaptor.GetMasternodesFromCheckpointHeader(headerV1, 0, 0)
	headerV2 := currentBlock.Header()
	headerV2.Number.Add(blockchain.Config().XDPoS.XDPoSV2Block, big.NewInt(1))
	headerV2.Validators = common.Hex2Bytes("0278c350152e15fa6ffc712a5a73d704ce73e2e103d9e17ae3ff2c6712e44e25b09ac5ee91f6c9ff065551f0dcac6f00cae11192d462db709be3758c")
	masternodesV2 := adaptor.GetMasternodesFromCheckpointHeader(headerV2, 0, 0)
	assert.True(t, reflect.DeepEqual(masternodesV1, masternodesV2), "GetMasternodesFromCheckpointHeader in adaptor for v1 v2 not equal", "v1", masternodesV1, "v2", masternodesV2)
}
func TestAdaptorIsEpochSwitch(t *testing.T) {
	blockchain, _, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 1, params.TestXDPoSMockChainConfigWithV2Engine, 0)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)
	header := currentBlock.Header()
	// v1
	header.Number.SetUint64(0)
	assert.True(t, adaptor.IsEpochSwitch(header), "header should be epoch switch", header)
	header.Number.SetUint64(1)
	assert.False(t, adaptor.IsEpochSwitch(header), "header should not be epoch switch", header)
	// v2
	parentBlockInfo := &utils.BlockInfo{
		Hash:   header.ParentHash,
		Round:  utils.Round(0),
		Number: big.NewInt(0).Set(blockchain.Config().XDPoS.XDPoSV2Block),
	}
	quorumCert := &utils.QuorumCert{
		ProposedBlockInfo: parentBlockInfo,
		Signatures:        nil,
	}
	extra := utils.ExtraFields_v2{
		Round:      1,
		QuorumCert: quorumCert,
	}
	extraBytes, err := extra.EncodeToBytes()
	assert.Nil(t, err)
	header.Extra = extraBytes
	header.Number.Add(blockchain.Config().XDPoS.XDPoSV2Block, big.NewInt(1))
	assert.True(t, adaptor.IsEpochSwitch(header), "header should be epoch switch", header)
	parentBlockInfo = &utils.BlockInfo{
		Hash:   header.ParentHash,
		Round:  utils.Round(1),
		Number: big.NewInt(0).Add(blockchain.Config().XDPoS.XDPoSV2Block, big.NewInt(1)),
	}
	quorumCert = &utils.QuorumCert{
		ProposedBlockInfo: parentBlockInfo,
		Signatures:        nil,
	}
	extra = utils.ExtraFields_v2{
		Round:      2,
		QuorumCert: quorumCert,
	}
	extraBytes, err = extra.EncodeToBytes()
	assert.Nil(t, err)
	header.Extra = extraBytes
	header.Number.Add(blockchain.Config().XDPoS.XDPoSV2Block, big.NewInt(2))
	assert.False(t, adaptor.IsEpochSwitch(header), "header should not be epoch switch", header)
	parentBlockInfo = &utils.BlockInfo{
		Hash:   header.ParentHash,
		Round:  utils.Round(blockchain.Config().XDPoS.Epoch) - 1,
		Number: big.NewInt(0).Add(blockchain.Config().XDPoS.XDPoSV2Block, big.NewInt(100)),
	}
	quorumCert = &utils.QuorumCert{
		ProposedBlockInfo: parentBlockInfo,
		Signatures:        nil,
	}
	extra = utils.ExtraFields_v2{
		Round:      utils.Round(blockchain.Config().XDPoS.Epoch) + 1,
		QuorumCert: quorumCert,
	}
	extraBytes, err = extra.EncodeToBytes()
	assert.Nil(t, err)
	header.Extra = extraBytes
	header.Number.Add(blockchain.Config().XDPoS.XDPoSV2Block, big.NewInt(101))
	assert.True(t, adaptor.IsEpochSwitch(header), "header should be epoch switch", header)
	parentBlockInfo = &utils.BlockInfo{
		Hash:   header.ParentHash,
		Round:  utils.Round(blockchain.Config().XDPoS.Epoch) + 1,
		Number: big.NewInt(0).Add(blockchain.Config().XDPoS.XDPoSV2Block, big.NewInt(100)),
	}
	quorumCert = &utils.QuorumCert{
		ProposedBlockInfo: parentBlockInfo,
		Signatures:        nil,
	}
	extra = utils.ExtraFields_v2{
		Round:      utils.Round(blockchain.Config().XDPoS.Epoch) + 2,
		QuorumCert: quorumCert,
	}
	extraBytes, err = extra.EncodeToBytes()
	assert.Nil(t, err)
	header.Extra = extraBytes
	header.Number.Add(blockchain.Config().XDPoS.XDPoSV2Block, big.NewInt(101))
	assert.False(t, adaptor.IsEpochSwitch(header), "header should not be epoch switch", header)
}

func TestAdaptorGetMasternodesV2(t *testing.T) {
	// we skip test for v1 since it's hard to make a real genesis block
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 10, params.TestXDPoSMockChainConfigWithV2Engine, 0)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)
	blockNum := 11
	blockCoinBase := "0x111000000000000000000000000000000123"
	blockHeader := createBlock(params.TestXDPoSMockChainConfigWithV2Engine, currentBlock, blockNum, 1, blockCoinBase, signer, signFn)
	// it contains 3 master nodes
	blockHeader.Validators = common.Hex2Bytes("0278c350152e15fa6ffc712a5a73d704ce73e2e103d9e17ae3ff2c6712e44e25b09ac5ee91f6c9ff065551f0dcac6f00cae11192d462db709be3758c")
	// block 11 is the first v2 block, and is treated as epoch switch block
	currentBlock, err := insertBlock(blockchain, blockHeader)
	if err != nil {
		t.Fatal(err)
	}
	masternodes1 := adaptor.GetMasternodes(blockchain, currentBlock.Header())
	assert.Equal(t, 3, len(masternodes1))
	for blockNum = 12; blockNum < 15; blockNum++ {
		blockHeader = createBlock(params.TestXDPoSMockChainConfigWithV2Engine, currentBlock, blockNum, int64(blockNum-10), blockCoinBase, signer, signFn)
		currentBlock, err = insertBlock(blockchain, blockHeader)
		if err != nil {
			t.Fatal(err)
		}
		masternodes2 := adaptor.GetMasternodes(blockchain, currentBlock.Header())
		assert.True(t, reflect.DeepEqual(masternodes1, masternodes2), "at block number", blockNum)
	}
}
