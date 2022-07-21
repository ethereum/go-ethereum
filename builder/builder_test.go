package builder

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/beacon"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/flashbots/go-boost-utils/bls"
	boostTypes "github.com/flashbots/go-boost-utils/types"
	"github.com/stretchr/testify/require"
)

/*
func NewBuilder(sk *bls.SecretKey, bc IBeaconClient, relay IRelay, builderSigningDomain boostTypes.Domain) *Builder {
func (b *Builder) onForkchoice(payloadAttributes *beacon.PayloadAttributesV1) {
func (b *Builder) newSealedBlock(data *beacon.ExecutableDataV1, block *types.Block, payloadAttributes *beacon.PayloadAttributesV1) {
*/

func TestOnNewSealedBlock(t *testing.T) {
	testBeacon := testBeaconClient{
		validator: NewRandomValidator(),
		slot:      56,
	}

	feeRecipient, _ := boostTypes.HexToAddress("0xabcf8e0d4e9587369b2301d0790347320302cc00")
	testRelay := testRelay{
		validator: ValidatorData{
			Pubkey:       PubkeyHex(testBeacon.validator.Pk),
			FeeRecipient: feeRecipient,
			GasLimit:     10,
			Timestamp:    15,
		},
	}

	sk, err := bls.SecretKeyFromBytes(hexutil.MustDecode("0x31ee185dad1220a8c88ca5275e64cf5a5cb09cb621cb30df52c9bee8fbaaf8d7"))
	require.NoError(t, err)

	bDomain := boostTypes.ComputeDomain(boostTypes.DomainTypeAppBuilder, [4]byte{0x02, 0x0, 0x0, 0x0}, boostTypes.Hash{})

	builder := NewBuilder(sk, &testBeacon, &testRelay, bDomain)

	testExecutableData := &beacon.ExecutableDataV1{
		ParentHash:   common.Hash{0x02, 0x03},
		FeeRecipient: common.Address{0x06, 0x15},
		StateRoot:    common.Hash{0x07, 0x16},
		ReceiptsRoot: common.Hash{0x08, 0x20},
		LogsBloom:    hexutil.MustDecode("0x000000000000000000000000000000"),
		Number:       uint64(10),
		GasLimit:     uint64(50),
		GasUsed:      uint64(100),
		Timestamp:    uint64(105),
		ExtraData:    hexutil.MustDecode("0x0042fafc"),

		BaseFeePerGas: big.NewInt(16),

		BlockHash:    common.Hash{0x09, 0xff},
		Transactions: [][]byte{},
	}

	testBlock := &types.Block{
		Profit: big.NewInt(10),
	}

	testPayloadAttributes := &beacon.PayloadAttributesV1{
		Timestamp:             uint64(104),
		Random:                common.Hash{0x05, 0x10},
		SuggestedFeeRecipient: common.Address{0x04, 0x10},
		GasLimit:              uint64(21),
		Slot:                  uint64(25),
	}

	builder.newSealedBlock(testExecutableData, testBlock, testPayloadAttributes)

	require.NotNil(t, testRelay.submittedMsg)

	expectedMessage := boostTypes.BidTraceMessage{
		Slot:                 uint64(25),
		ParentHash:           boostTypes.Hash{0x02, 0x03},
		BlockHash:            boostTypes.Hash{0x09, 0xff},
		BuilderPubkey:        builder.builderPublicKey,
		ProposerPubkey:       boostTypes.PublicKey{},
		ProposerFeeRecipient: boostTypes.Address{0x04, 0x10},
		Value:                boostTypes.U256Str{0x0a},
	}

	require.Equal(t, expectedMessage, *testRelay.submittedMsg.Message)

	expectedExecutionPayload := boostTypes.ExecutionPayload{
		ParentHash:    [32]byte(testExecutableData.ParentHash),
		FeeRecipient:  boostTypes.Address{0x6, 0x15},
		StateRoot:     [32]byte(testExecutableData.StateRoot),
		ReceiptsRoot:  [32]byte(testExecutableData.ReceiptsRoot),
		LogsBloom:     [256]byte{},
		Random:        [32]byte(testExecutableData.Random),
		BlockNumber:   testExecutableData.Number,
		GasLimit:      testExecutableData.GasLimit,
		GasUsed:       testExecutableData.GasUsed,
		Timestamp:     testExecutableData.Timestamp,
		ExtraData:     hexutil.MustDecode("0x0042fafc"),
		BaseFeePerGas: boostTypes.U256Str{0x10},
		BlockHash:     boostTypes.Hash{0x09, 0xff},
		Transactions:  []hexutil.Bytes{},
	}
	require.Equal(t, expectedExecutionPayload, *testRelay.submittedMsg.ExecutionPayload)

	expectedSignature, err := boostTypes.HexToSignature("0xb79f75f81c834d104afbf1fb45f2cc19d5b0b4367184a43b88e696c88e6ab1a150be1fde9de5d1ca28bd955063164ae001c99d516c6ccd278c6bfb2af9c08805e39698a4a4e0713681a012921c1e9d8d14be95b49f654aba1fb493892a00795d")

	require.NoError(t, err)
	require.Equal(t, expectedSignature, testRelay.submittedMsg.Signature)

	require.Equal(t, uint64(25), testRelay.requestedSlot)
}
