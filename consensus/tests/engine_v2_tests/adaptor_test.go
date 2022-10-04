package engine_v2_tests

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/stretchr/testify/assert"
)

func TestAdaptorShouldGetAuthorForDifferentConsensusVersion(t *testing.T) {
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 900, params.TestXDPoSMockChainConfig, nil)
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
	// Insert block 901

	merkleRoot := "35999dded35e8db12de7e6c1471eb9670c162eec616ecebbaf4fddd4676fb930"
	header := &types.Header{
		Root:       common.HexToHash(merkleRoot),
		Number:     big.NewInt(int64(901)),
		ParentHash: currentBlock.Hash(),
		Coinbase:   signer,
	}

	header.Extra = generateV2Extra(1, currentBlock, signer, signFn, nil)

	block901, err := createBlockFromHeader(blockchain, header, nil, signer, signFn, blockchain.Config())
	if err != nil {
		t.Fatal(err)
	}
	err = blockchain.InsertBlock(block901)
	assert.Nil(t, err)

	addressFromAdaptor, errorAdaptor = adaptor.Author(block901.Header())
	if errorAdaptor != nil {
		t.Fatalf("Failed while trying to get Author from adaptor")
	}
	addressFromV2Engine, errV2 := adaptor.EngineV2.Author(block901.Header())
	if errV2 != nil {
		t.Fatalf("Failed while trying to get Author from engine v2")
	}
	// Make sure the value is exactly the same as from V2 engine
	assert.Equal(t, addressFromAdaptor, addressFromV2Engine)
}

func TestAdaptorGetMasternodesFromCheckpointHeader(t *testing.T) {
	blockchain, _, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 1, params.TestXDPoSMockChainConfig, nil)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)
	headerV1 := currentBlock.Header()
	headerV1.Extra = common.Hex2Bytes("d7830100018358444388676f312e31352e38856c696e757800000000000000000278c350152e15fa6ffc712a5a73d704ce73e2e103d9e17ae3ff2c6712e44e25b09ac5ee91f6c9ff065551f0dcac6f00cae11192d462db709be3758ccef312ee5eea8d7bad5374c6a652150515d744508b61c1a4deb4e4e7bf057e4e3824c11fd2569bcb77a52905cda63b5a58507910bed335e4c9d87ae0ecdfafd400")
	masternodesV1 := adaptor.GetMasternodesFromCheckpointHeader(headerV1)
	headerV2 := currentBlock.Header()
	headerV2.Number.Add(blockchain.Config().XDPoS.V2.SwitchBlock, big.NewInt(1))
	headerV2.Validators = common.Hex2Bytes("0278c350152e15fa6ffc712a5a73d704ce73e2e103d9e17ae3ff2c6712e44e25b09ac5ee91f6c9ff065551f0dcac6f00cae11192d462db709be3758c")
	headerV2.Extra = []byte{2}
	masternodesV2 := adaptor.GetMasternodesFromCheckpointHeader(headerV2)
	assert.True(t, reflect.DeepEqual(masternodesV1, masternodesV2), "GetMasternodesFromCheckpointHeader in adaptor for v1 v2 not equal", "v1", masternodesV1, "v2", masternodesV2)
}
func TestAdaptorIsEpochSwitch(t *testing.T) {
	blockchain, _, currentBlock, _, _, _ := PrepareXDCTestBlockChainForV2Engine(t, 1, params.TestXDPoSMockChainConfig, nil)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)
	header := currentBlock.Header()
	// v1
	header.Number.SetUint64(0)

	isEpochSwitchBlock, epochNum, err := adaptor.IsEpochSwitch(header)
	assert.Nil(t, err)
	assert.True(t, isEpochSwitchBlock, "header should be epoch switch", header)
	assert.Equal(t, uint64(0), epochNum)
	header.Number.SetUint64(1)
	isEpochSwitchBlock, _, err = adaptor.IsEpochSwitch(header)
	assert.Nil(t, err)
	assert.False(t, isEpochSwitchBlock, "header should not be epoch switch", header)
	// v2
	parentBlockInfo := &types.BlockInfo{
		Hash:   header.ParentHash,
		Round:  types.Round(0),
		Number: big.NewInt(0).Set(blockchain.Config().XDPoS.V2.SwitchBlock),
	}
	quorumCert := &types.QuorumCert{
		ProposedBlockInfo: parentBlockInfo,
		Signatures:        nil,
		GapNumber:         blockchain.Config().XDPoS.V2.SwitchBlock.Uint64() - blockchain.Config().XDPoS.Gap,
	}
	extra := types.ExtraFields_v2{
		Round:      1,
		QuorumCert: quorumCert,
	}
	extraBytes, err := extra.EncodeToBytes()
	assert.Nil(t, err)
	header.Extra = extraBytes
	header.Number.Add(blockchain.Config().XDPoS.V2.SwitchBlock, big.NewInt(1))
	isEpochSwitchBlock, _, err = adaptor.IsEpochSwitch(header)
	assert.Nil(t, err)
	assert.True(t, isEpochSwitchBlock, "header should be epoch switch", header)
	parentBlockInfo = &types.BlockInfo{
		Hash:   header.ParentHash,
		Round:  types.Round(1),
		Number: big.NewInt(0).Add(blockchain.Config().XDPoS.V2.SwitchBlock, big.NewInt(1)),
	}
	quorumCert = &types.QuorumCert{
		ProposedBlockInfo: parentBlockInfo,
		Signatures:        nil,
		GapNumber:         blockchain.Config().XDPoS.V2.SwitchBlock.Uint64() - blockchain.Config().XDPoS.Gap,
	}
	extra = types.ExtraFields_v2{
		Round:      2,
		QuorumCert: quorumCert,
	}
	extraBytes, err = extra.EncodeToBytes()
	assert.Nil(t, err)
	header.Extra = extraBytes
	header.Number.Add(blockchain.Config().XDPoS.V2.SwitchBlock, big.NewInt(2))
	isEpochSwitchBlock, _, err = adaptor.IsEpochSwitch(header)
	assert.Nil(t, err)
	assert.False(t, isEpochSwitchBlock, "header should not be epoch switch", header)
	parentBlockInfo = &types.BlockInfo{
		Hash:   header.ParentHash,
		Round:  types.Round(blockchain.Config().XDPoS.Epoch) - 1,
		Number: big.NewInt(0).Add(blockchain.Config().XDPoS.V2.SwitchBlock, big.NewInt(100)),
	}
	quorumCert = &types.QuorumCert{
		ProposedBlockInfo: parentBlockInfo,
		Signatures:        nil,
		GapNumber:         blockchain.Config().XDPoS.V2.SwitchBlock.Uint64() - blockchain.Config().XDPoS.Gap,
	}
	extra = types.ExtraFields_v2{
		Round:      types.Round(blockchain.Config().XDPoS.Epoch) + 1,
		QuorumCert: quorumCert,
	}
	extraBytes, err = extra.EncodeToBytes()
	assert.Nil(t, err)
	header.Extra = extraBytes
	header.Number.Add(blockchain.Config().XDPoS.V2.SwitchBlock, big.NewInt(101))
	isEpochSwitchBlock, _, err = adaptor.IsEpochSwitch(header)
	assert.Nil(t, err)
	assert.True(t, isEpochSwitchBlock, "header should be epoch switch", header)
	parentBlockInfo = &types.BlockInfo{
		Hash:   header.ParentHash,
		Round:  types.Round(blockchain.Config().XDPoS.Epoch) + 1,
		Number: big.NewInt(0).Add(blockchain.Config().XDPoS.V2.SwitchBlock, big.NewInt(100)),
	}
	quorumCert = &types.QuorumCert{
		ProposedBlockInfo: parentBlockInfo,
		Signatures:        nil,
		GapNumber:         blockchain.Config().XDPoS.V2.SwitchBlock.Uint64() - blockchain.Config().XDPoS.Gap,
	}
	extra = types.ExtraFields_v2{
		Round:      types.Round(blockchain.Config().XDPoS.Epoch) + 2,
		QuorumCert: quorumCert,
	}
	extraBytes, err = extra.EncodeToBytes()
	assert.Nil(t, err)
	header.Extra = extraBytes
	header.Number.Add(blockchain.Config().XDPoS.V2.SwitchBlock, big.NewInt(101))
	isEpochSwitchBlock, _, err = adaptor.IsEpochSwitch(header)
	assert.Nil(t, err)
	assert.False(t, isEpochSwitchBlock, "header should not be epoch switch", header)
}

func TestAdaptorGetMasternodesV2(t *testing.T) {
	// we skip test for v1 since it's hard to make a real genesis block
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 900, params.TestXDPoSMockChainConfig, nil)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)
	blockNum := 901
	blockCoinBase := "0x111000000000000000000000000000000123"
	currentBlock = CreateBlock(blockchain, params.TestXDPoSMockChainConfig, currentBlock, blockNum, 1, blockCoinBase, signer, signFn, nil, nil)

	// block 901 is the first v2 block, and is treated as epoch switch block
	err := blockchain.InsertBlock(currentBlock)
	assert.Nil(t, err)
	masternodes1 := adaptor.GetMasternodes(blockchain, currentBlock.Header())
	assert.Equal(t, 5, len(masternodes1))
	masternodes1ByNumber := adaptor.GetMasternodesByNumber(blockchain, currentBlock.NumberU64())
	assert.True(t, reflect.DeepEqual(masternodes1, masternodes1ByNumber), "at block number", blockNum)
	for blockNum = 902; blockNum < 915; blockNum++ {
		currentBlock = CreateBlock(blockchain, params.TestXDPoSMockChainConfig, currentBlock, blockNum, int64(blockNum-900), blockCoinBase, signer, signFn, nil, nil)
		err = blockchain.InsertBlock(currentBlock)
		assert.Nil(t, err)
		masternodes2 := adaptor.GetMasternodes(blockchain, currentBlock.Header())
		assert.True(t, reflect.DeepEqual(masternodes1, masternodes2), "at block number", blockNum)
		masternodes2ByNumber := adaptor.GetMasternodesByNumber(blockchain, currentBlock.NumberU64())
		assert.True(t, reflect.DeepEqual(masternodes2, masternodes2ByNumber), "at block number", blockNum)
	}
}

func TestGetCurrentEpochSwitchBlock(t *testing.T) {
	blockchain, _, currentBlock, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 900, params.TestXDPoSMockChainConfig, nil)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)

	// V1
	currentCheckpointNumber, epochNum, err := adaptor.GetCurrentEpochSwitchBlock(blockchain, big.NewInt(900))
	assert.Nil(t, err)
	assert.Equal(t, uint64(900), currentCheckpointNumber)
	assert.Equal(t, uint64(1), epochNum)

	// V2
	blockNum := 901
	blockCoinBase := "0x111000000000000000000000000000000123"
	currentBlock = CreateBlock(blockchain, params.TestXDPoSMockChainConfig, currentBlock, blockNum, 1, blockCoinBase, signer, signFn, nil, nil)
	err = blockchain.InsertBlock(currentBlock)
	assert.Nil(t, err)
	currentCheckpointNumber, epochNum, err = adaptor.GetCurrentEpochSwitchBlock(blockchain, currentBlock.Number())
	assert.Nil(t, err)
	assert.Equal(t, uint64(901), currentCheckpointNumber)
	assert.Equal(t, uint64(1), epochNum)

	for blockNum = 902; blockNum < 915; blockNum++ {
		currentBlock = CreateBlock(blockchain, params.TestXDPoSMockChainConfig, currentBlock, blockNum, int64(blockNum-900), blockCoinBase, signer, signFn, nil, nil)

		err = blockchain.InsertBlock(currentBlock)
		assert.Nil(t, err)
		currentCheckpointNumber, epochNum, err := adaptor.GetCurrentEpochSwitchBlock(blockchain, currentBlock.Number())
		assert.Nil(t, err)
		assert.Equal(t, uint64(901), currentCheckpointNumber)
		assert.Equal(t, uint64(1), epochNum)
	}
}

func TestGetParentBlock(t *testing.T) {
	blockchain, _, block900, signer, signFn, _ := PrepareXDCTestBlockChainForV2Engine(t, 900, params.TestXDPoSMockChainConfig, nil)
	adaptor := blockchain.Engine().(*XDPoS.XDPoS)

	// V1
	block := adaptor.FindParentBlockToAssign(blockchain, block900)
	assert.Equal(t, block, block900)

	// Initialise
	err := adaptor.EngineV2.Initial(blockchain, block.Header())
	assert.Nil(t, err)

	// V2
	blockNum := 901
	blockCoinBase := "0x111000000000000000000000000000000123"
	block901 := CreateBlock(blockchain, params.TestXDPoSMockChainConfig, block900, blockNum, 1, blockCoinBase, signer, signFn, nil, nil)
	err = blockchain.InsertBlock(block901)
	assert.Nil(t, err)

	// let's inject another one, but the highestedQC has not been updated, so it shall still point to 900
	blockNum = 902
	block902 := CreateBlock(blockchain, params.TestXDPoSMockChainConfig, block901, blockNum, 1, blockCoinBase, signer, signFn, nil, nil)
	err = blockchain.InsertBlock(block902)
	assert.Nil(t, err)
	block = adaptor.FindParentBlockToAssign(blockchain, block902)

	assert.Equal(t, block900.Hash(), block.Hash())
}
