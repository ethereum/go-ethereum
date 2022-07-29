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

func TestOnNewSealedBlock(t *testing.T) {
	vsk, err := bls.SecretKeyFromBytes(hexutil.MustDecode("0x370bb8c1a6e62b2882f6ec76762a67b39609002076b95aae5b023997cf9b2dc9"))
	require.NoError(t, err)
	validator := &ValidatorPrivateData{
		sk: vsk,
		Pk: hexutil.MustDecode("0xb67d2c11bcab8c4394fc2faa9601d0b99c7f4b37e14911101da7d97077917862eed4563203d34b91b5cf0aa44d6cfa05"),
	}

	testBeacon := testBeaconClient{
		validator: validator,
		slot:      56,
	}

	feeRecipient, _ := boostTypes.HexToAddress("0xabcf8e0d4e9587369b2301d0790347320302cc00")
	testRelay := testRelay{
		validator: ValidatorData{
			Pubkey:       PubkeyHex(testBeacon.validator.Pk.String()),
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

	expectedProposerPubkey, err := boostTypes.HexToPubkey(testBeacon.validator.Pk.String())
	require.NoError(t, err)

	expectedMessage := boostTypes.BidTrace{
		Slot:                 uint64(25),
		ParentHash:           boostTypes.Hash{0x02, 0x03},
		BlockHash:            boostTypes.Hash{0x09, 0xff},
		BuilderPubkey:        builder.builderPublicKey,
		ProposerPubkey:       expectedProposerPubkey,
		ProposerFeeRecipient: boostTypes.Address{0x04, 0x10},
		GasLimit:             uint64(50),
		GasUsed:              uint64(100),
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

	expectedSignature, err := boostTypes.HexToSignature("0xadebce714127deea6b04c8f63e650ad6b4c0d3df14ecd9759bef741cd6d72509090f5e172033ce40475c322c0c0e3fae0e78a880a66cb324913ea490472d93e187a9a91284b05137f1554688c5e9b1ee73539a2b005b103e8bd50e973e8e0f49")

	require.NoError(t, err)
	require.Equal(t, expectedSignature, testRelay.submittedMsg.Signature)

	require.Equal(t, uint64(25), testRelay.requestedSlot)
}
