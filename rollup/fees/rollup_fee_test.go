package fees

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/params"
)

func TestL1DataFeeBeforeCurie(t *testing.T) {
	l1BaseFee := new(big.Int).SetUint64(15000000)
	overhead := new(big.Int).SetUint64(100)
	scalar := new(big.Int).SetUint64(10)

	data := []byte{0, 10, 1, 0}

	expected := new(big.Int).SetUint64(30) // 30.6
	actual := calculateEncodedL1DataFee(data, overhead, l1BaseFee, scalar)
	assert.Equal(t, expected, actual)
}

func TestL1DataFeeAfterCurie(t *testing.T) {
	l1BaseFee := new(big.Int).SetUint64(1500000000)
	l1BlobBaseFee := new(big.Int).SetUint64(150000000)
	commitScalar := new(big.Int).SetUint64(10)
	blobScalar := new(big.Int).SetUint64(10)

	data := []byte{0, 10, 1, 0}

	expected := new(big.Int).SetUint64(21)
	actual := calculateEncodedL1DataFeeCurie(data, l1BaseFee, l1BlobBaseFee, commitScalar, blobScalar)
	assert.Equal(t, expected, actual)
}

func TestL1DataFeeFeynman(t *testing.T) {
	l1BaseFee := new(big.Int).SetInt64(1_000_000_000)
	l1BlobBaseFee := new(big.Int).SetInt64(1_000_000_000)
	execScalar := new(big.Int).SetInt64(10)
	blobScalar := new(big.Int).SetInt64(20)
	penaltyThreshold := new(big.Int).SetInt64(6_000_000_000) // 6 * PRECISION
	penaltyFactor := new(big.Int).SetInt64(2_000_000_000)    // 2 * PRECISION (200% penalty)

	// Test case 1: No penalty (compression ratio >= threshold)
	t.Run("no penalty case", func(t *testing.T) {
		data := make([]byte, 100)                                // txSize = 100
		compressionRatio := new(big.Int).SetInt64(6_000_000_000) // exactly at threshold

		// Since compression ratio >= penaltyThreshold, penalty = 1 * PRECISION
		// feePerByte = execScalar * l1BaseFee + blobScalar * l1BlobBaseFee = 10 * 1_000_000_000 + 20 * 1_000_000_000 = 30_000_000_000
		// l1DataFee = feePerByte * txSize * penalty / PRECISION / PRECISION
		//           = 30_000_000_000 * 100 * 1_000_000_000 / 1_000_000_000 / 1_000_000_000 = 3000

		expected := new(big.Int).SetInt64(3000)

		actual := calculateEncodedL1DataFeeFeynman(
			data,
			l1BaseFee,
			l1BlobBaseFee,
			execScalar,
			blobScalar,
			penaltyThreshold,
			penaltyFactor,
			compressionRatio,
		)

		assert.Equal(t, expected, actual)
	})

	// Test case 2: With penalty (compression ratio < threshold)
	t.Run("with penalty case", func(t *testing.T) {
		data := make([]byte, 100)                                // txSize = 100
		compressionRatio := new(big.Int).SetInt64(5_000_000_000) // below threshold

		// Since compression ratio < penaltyThreshold, penalty = penaltyFactor
		// feePerByte = execScalar * l1BaseFee + blobScalar * l1BlobBaseFee = 30_000_000_000
		// l1DataFee = feePerByte * txSize * penaltyFactor / PRECISION / PRECISION
		//           = 30_000_000_000 * 100 * 2_000_000_000 / 1_000_000_000 / 1_000_000_000 = 6000

		expected := new(big.Int).SetInt64(6000)

		actual := calculateEncodedL1DataFeeFeynman(
			data,
			l1BaseFee,
			l1BlobBaseFee,
			execScalar,
			blobScalar,
			penaltyThreshold,
			penaltyFactor,
			compressionRatio,
		)

		assert.Equal(t, expected, actual)
	})
}

func TestEstimateTxCompressionRatio(t *testing.T) {
	// Mock config that would select a specific codec version
	// Note: You'll need to adjust this based on your actual params.ChainConfig structure

	t.Run("empty data", func(t *testing.T) {
		data := []byte{}
		// Should return 1.0 ratio (PRECISION)
		ratio, err := estimateTxCompressionRatio(data, 1000000, 1700000000, params.TestChainConfig)
		assert.Error(t, err, "raw data is empty")
		assert.Nil(t, ratio)
		// The exact value depends on rcfg.Precision, but should be the "1.0" equivalent
	})

	t.Run("non-empty data", func(t *testing.T) {
		// Create some compressible data
		data := make([]byte, 1000)
		for i := range data {
			data[i] = byte(i % 10) // Create patterns for better compression
		}

		ratio, err := estimateTxCompressionRatio(data, 1000000, 1700000000, params.TestChainConfig)
		assert.NoError(t, err)
		assert.NotNil(t, ratio)
		// Should return a ratio > 1.0 (since compressed size < original size)
	})

	t.Run("eth-transfer", func(t *testing.T) {
		// https://scrollscan.com/tx/0x8c7eba9a56e25c4402a1d9fdbe6fbe70e6f6f89484b2e4f5c329258a924193b4
		data := common.Hex2Bytes("02f86b83082750830461a40183830fb782523f94802b65b5d9016621e66003aed0b16615093f328b8080c001a0a1fa6bbede5ae355eaec83fdcda65eab240476895e649576552850de726596cca0424eb1f5221865817b270d85caf8611d35ea6d7c2e86c9c31af5c06df04a2587")
		ratio, err := estimateTxCompressionRatio(data, 1000000, 1700000000, params.TestChainConfig)
		assert.NoError(t, err)
		assert.NotNil(t, ratio)
		assert.Equal(t, big.NewInt(1_000_000_000), ratio) // 1x (not compressible)
	})

	t.Run("scr-transfer", func(t *testing.T) {
		// https://scrollscan.com/tx/0x7b681ce914c9774aff364d2b099b2ba41dea44bcd59dbebb9d4c4b6853893179
		data := common.Hex2Bytes("02f8b28308275001830f4240840279876683015c2894d29687c813d741e2f938f4ac377128810e217b1b80b844a9059cbb000000000000000000000000687b50a70d33d71f9a82dd330b8c091e4d77250800000000000000000000000000000000000000000000000ac96dda943e512bb9c080a0fdacacd07ed7c708e2193b803d731d3d288dcd39c317f321f243cd790406868ba0285444ab799632c88fd47c874c218bceb1589843949b5bc0f3ead1df069f3233")
		ratio, err := estimateTxCompressionRatio(data, 1000000, 1700000000, params.TestChainConfig)
		assert.NoError(t, err)
		assert.NotNil(t, ratio)
		assert.Equal(t, big.NewInt(1_117_283_950), ratio) // 1.1x
	})

	t.Run("syncswap-swap", func(t *testing.T) {
		// https://scrollscan.com/tx/0x59a7b72503400b6719f3cb670c7b1e7e45ce5076f30b98bdaad3b07a5d0fbc02
		data := common.Hex2Bytes("f902cf830887d783a57282830493e09480e38291e06339d10aab483c65695d004dbd5c6980b902642cc4081e00000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000005ec79b80000000000000000000000000000000000000000000000000003328b944c400000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000006000000000000000000000000053000000000000000000000000000000000000040000000000000000000000000000000000000000000000000091a94863ca800000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000020000000000000000000000000814a23b053fd0f102aeeda0459215c2444799c7000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000600000000000000000000000005300000000000000000000000000000000000004000000000000000000000000485ca81b70255da2fe3fd0814b57d1b08fce784e0000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000083104ec3a050db0fbfa3fd83aa9077abdd4edb3dc504661d6fb3b39f973fe994de8fb0ac41a044983fa3d16aa0e156a1b3382fa763f9831be5a5c158f849be524d41d100ab52")
		ratio, err := estimateTxCompressionRatio(data, 1000000, 1700000000, params.TestChainConfig)
		assert.NoError(t, err)
		assert.NotNil(t, ratio)
		assert.Equal(t, big.NewInt(3_059_322_033), ratio) // 3.1x
	})

	t.Run("uniswap-swap", func(t *testing.T) {
		// https://scrollscan.com/tx/0x65b268bd8ef416f44983ee277d748de044243272b0f106b71ff03cc8501a05da
		data := common.Hex2Bytes("f9014e830887e0836b92a7830493e094fc30937f5cde93df8d48acaf7e6f5d8d8a31f63680b8e45023b4df00000000000000000000000006efdbff2a14a7c8e15944d1f4a48f9f95f663a4000000000000000000000000530000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000001f4000000000000000000000000485ca81b70255da2fe3fd0814b57d1b08fce784e000000000000000000000000000000000000000000000000006a94d74f43000000000000000000000000000000000000000000000000000000000000045af675000000000000000000000000000000000000000000000000000000000000000083104ec4a0a527358d5bfb89dcc7939265c6add9faf4697415174723e509f795ad44021d98a0776f4a8a8a51da98b70d960a5bd1faf3c79b8dddc0bc2c642a4c2634c6990f02")
		ratio, err := estimateTxCompressionRatio(data, 1000000, 1700000000, params.TestChainConfig)
		assert.NoError(t, err)
		assert.NotNil(t, ratio)
		assert.Equal(t, big.NewInt(1_710_659_898), ratio) // 1.7x
	})

	t.Run("etherfi-deposit", func(t *testing.T) {
		// https://scrollscan.com/tx/0x41a77736afd54134b6c673e967c9801e326495074012b4033bd557920cbe5a71
		data := common.Hex2Bytes("02f901d58308275082f88a834c4b4084044c7166831066099462f623161fdb6564925c3f9b783cbdfef4ce8aec80b9016463baa26000000000000000000000000077a7e3215a621a9935d32a046212ebfcffa3bff900000000000000000000000006efdbff2a14a7c8e15944d1f4a48f9f95f663a400000000000000000000000008c6f91e2b681faf5e17227f2a44c307b3c1364c0000000000000000000000000000000000000000000000000000000002d4cae000000000000000000000000000000000000000000000000000000000028f7f83000000000000000000000000249e3fa81d73244f956ecd529715323b6d02f24b00000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000041a95314c3a11f86cc673f2afd60d27f559cb2edcc0da5af030adffc97f9a5edc3314efbadd32878e289017f644a4afa365da5367fefe583f7c4ff0c6047e2c1ff1b00000000000000000000000000000000000000000000000000000000000000c080a05b2d22b8aaf6d334471e74899cfc4c81186f8a94f278c97ee211727d8027ceafa0450352d9a1782180c27a03889d317d31e725dda38a5bd3c0531950b879ed50a1")
		ratio, err := estimateTxCompressionRatio(data, 1000000, 1700000000, params.TestChainConfig)
		assert.NoError(t, err)
		assert.NotNil(t, ratio)
		assert.Equal(t, big.NewInt(1_496_835_443), ratio) // 1.4x
	})

	t.Run("edgepushoracle-postupdate", func(t *testing.T) {
		// https://scrollscan.com/tx/0x8271c68146a3b07b1ebf52ce0b550751f49cbd72fa0596ef14ff56d1f23a0bec
		data := common.Hex2Bytes("f9046f83015e4c836e0b6b8303cbf8946a5b3ab3274b738eab25205af6e2d4dd7781292480b9040449a1a4fb000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000005f725f60000000000000000000000000000000000000000000000000000000003d0cac600000000000000000000000000000000000000000000000000000000685d50cd000000000000000000000000000000000000000000000000000000000000000500000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000001a0000000000000000000000000000000000000000000000000000000000000022000000000000000000000000000000000000000000000000000000000000002a0000000000000000000000000000000000000000000000000000000000000004155903b95865fc5a5dd7d4d876456140dd0b815695647fc41eb1924f4cfe267265130b5a5d77125c44cf6a5a81edba6d5850ba00f90ab83281c9b44e17528fd74010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000416f000e0498733998e6a1a6454e116c1b1f95f7e000400b6a54029406cf288bdc615b62de8e2db533d6010ca57001e0b8a4b3f05ed516a31830516c52b9df206e000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000410dabc77a807d729ff62c3be740d492d884f026ad2770fa7c4bdec569e201643656b07f2009d2129173738571417734a3df051cebc7b8233bec6d9471c21c098700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000041eb009614c939170e9ff3d3e06c3a2c45810fe46a364ce28ecec5e220f5fd86cd6e0f70ab9093dd6b22b69980246496b600c8fcb054047962d4128efa48b692f301000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000041a31b4dd4f0a482372d75c7a8c5f11aa8084a5f358579866f1d25a26a15beb2b5153400bfa7fa3d6fba138c02dd1eb8a5a97d62178d98c5632a153396a566e5ed000000000000000000000000000000000000000000000000000000000000000083104ec4a05cb4eee77676d432c672008594825b957d34ae5dd786ed294501849bb1ce285aa01325f37cdc945863ec0474932102bc944cb98a663db6d30ae23c2ebb5f9ce070")
		ratio, err := estimateTxCompressionRatio(data, 1000000, 1700000000, params.TestChainConfig)
		assert.NoError(t, err)
		assert.NotNil(t, ratio)
		assert.Equal(t, big.NewInt(2_139_097_744), ratio) // 2.1x
	})
}

func TestCalculatePenalty(t *testing.T) {
	precision := new(big.Int).SetInt64(1_000_000_000)        // PRECISION
	penaltyThreshold := new(big.Int).SetInt64(6_000_000_000) // 6 * PRECISION
	penaltyFactor := new(big.Int).SetInt64(2_000_000_000)    // 2 * PRECISION

	t.Run("no penalty when ratio >= threshold", func(t *testing.T) {
		compressionRatio := new(big.Int).SetInt64(6_000_000_000) // exactly at threshold
		penalty := calculatePenalty(compressionRatio, penaltyThreshold, penaltyFactor)
		assert.Equal(t, precision, penalty)
	})

	t.Run("penalty when ratio < threshold", func(t *testing.T) {
		compressionRatio := new(big.Int).SetInt64(5_000_000_000) // below threshold
		penalty := calculatePenalty(compressionRatio, penaltyThreshold, penaltyFactor)
		assert.Equal(t, penaltyFactor, penalty)
	})
}
