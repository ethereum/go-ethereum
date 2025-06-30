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

	calcCompressionRatio := func(data []byte) (*big.Int, error) {
		return estimateTxCompressionRatio(data, 1000000, 1700000000, params.TestChainConfig)
	}

	t.Run("empty data", func(t *testing.T) {
		data := []byte{}
		ratio, err := calcCompressionRatio(data)
		assert.NoError(t, err)
		assert.NotNil(t, ratio)
		assert.Equal(t, U256MAX, ratio) // empty data has max compression ratio by definition
	})

	t.Run("non-empty data", func(t *testing.T) {
		// Create some compressible data
		data := make([]byte, 1000)
		for i := range data {
			data[i] = byte(i % 10) // Create patterns for better compression
		}

		ratio, err := calcCompressionRatio(data)
		assert.NoError(t, err)
		assert.NotNil(t, ratio)
		// Should return a ratio > 1.0 (since compressed size < original size)
	})

	t.Run("eth-transfer", func(t *testing.T) {
		data := common.Hex2Bytes("") // empty payload
		ratio, err := calcCompressionRatio(data)
		assert.NoError(t, err)
		assert.NotNil(t, ratio)
		assert.Equal(t, U256MAX, ratio) // empty data is infinitely compressible by definition
	})

	t.Run("scr-transfer", func(t *testing.T) {
		// https://scrollscan.com/tx/0x7b681ce914c9774aff364d2b099b2ba41dea44bcd59dbebb9d4c4b6853893179
		data := common.Hex2Bytes("a9059cbb000000000000000000000000687b50a70d33d71f9a82dd330b8c091e4d77250800000000000000000000000000000000000000000000000ac96dda943e512bb9")
		ratio, err := calcCompressionRatio(data)
		assert.NoError(t, err)
		assert.NotNil(t, ratio)
		assert.Equal(t, big.NewInt(1_387_755_102), ratio) // 1.4x
	})

	t.Run("syncswap-swap", func(t *testing.T) {
		// https://scrollscan.com/tx/0x59a7b72503400b6719f3cb670c7b1e7e45ce5076f30b98bdaad3b07a5d0fbc02
		data := common.Hex2Bytes("2cc4081e00000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000005ec79b80000000000000000000000000000000000000000000000000003328b944c400000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000006000000000000000000000000053000000000000000000000000000000000000040000000000000000000000000000000000000000000000000091a94863ca800000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000020000000000000000000000000814a23b053fd0f102aeeda0459215c2444799c7000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000000600000000000000000000000005300000000000000000000000000000000000004000000000000000000000000485ca81b70255da2fe3fd0814b57d1b08fce784e00000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000")
		ratio, err := calcCompressionRatio(data)
		assert.NoError(t, err)
		assert.NotNil(t, ratio)
		assert.Equal(t, big.NewInt(4_857_142_857), ratio) // 4.8x
	})

	t.Run("uniswap-swap", func(t *testing.T) {
		// https://scrollscan.com/tx/0x65b268bd8ef416f44983ee277d748de044243272b0f106b71ff03cc8501a05da
		data := common.Hex2Bytes("5023b4df00000000000000000000000006efdbff2a14a7c8e15944d1f4a48f9f95f663a4000000000000000000000000530000000000000000000000000000000000000400000000000000000000000000000000000000000000000000000000000001f4000000000000000000000000485ca81b70255da2fe3fd0814b57d1b08fce784e000000000000000000000000000000000000000000000000006a94d74f43000000000000000000000000000000000000000000000000000000000000045af6750000000000000000000000000000000000000000000000000000000000000000")
		ratio, err := calcCompressionRatio(data)
		assert.NoError(t, err)
		assert.NotNil(t, ratio)
		assert.Equal(t, big.NewInt(2_620_689_655), ratio) // 2.6x
	})

	t.Run("etherfi-deposit", func(t *testing.T) {
		// https://scrollscan.com/tx/0x41a77736afd54134b6c673e967c9801e326495074012b4033bd557920cbe5a71
		data := common.Hex2Bytes("63baa26000000000000000000000000077a7e3215a621a9935d32a046212ebfcffa3bff900000000000000000000000006efdbff2a14a7c8e15944d1f4a48f9f95f663a400000000000000000000000008c6f91e2b681faf5e17227f2a44c307b3c1364c0000000000000000000000000000000000000000000000000000000002d4cae000000000000000000000000000000000000000000000000000000000028f7f83000000000000000000000000249e3fa81d73244f956ecd529715323b6d02f24b00000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000041a95314c3a11f86cc673f2afd60d27f559cb2edcc0da5af030adffc97f9a5edc3314efbadd32878e289017f644a4afa365da5367fefe583f7c4ff0c6047e2c1ff1b00000000000000000000000000000000000000000000000000000000000000")
		ratio, err := calcCompressionRatio(data)
		assert.NoError(t, err)
		assert.NotNil(t, ratio)
		assert.Equal(t, big.NewInt(1_788_944_723), ratio) // 1.8x
	})

	t.Run("edgepushoracle-postupdate", func(t *testing.T) {
		// https://scrollscan.com/tx/0x8271c68146a3b07b1ebf52ce0b550751f49cbd72fa0596ef14ff56d1f23a0bec
		data := common.Hex2Bytes("49a1a4fb000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000005f725f60000000000000000000000000000000000000000000000000000000003d0cac600000000000000000000000000000000000000000000000000000000685d50cd000000000000000000000000000000000000000000000000000000000000000500000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000012000000000000000000000000000000000000000000000000000000000000001a0000000000000000000000000000000000000000000000000000000000000022000000000000000000000000000000000000000000000000000000000000002a0000000000000000000000000000000000000000000000000000000000000004155903b95865fc5a5dd7d4d876456140dd0b815695647fc41eb1924f4cfe267265130b5a5d77125c44cf6a5a81edba6d5850ba00f90ab83281c9b44e17528fd74010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000416f000e0498733998e6a1a6454e116c1b1f95f7e000400b6a54029406cf288bdc615b62de8e2db533d6010ca57001e0b8a4b3f05ed516a31830516c52b9df206e000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000410dabc77a807d729ff62c3be740d492d884f026ad2770fa7c4bdec569e201643656b07f2009d2129173738571417734a3df051cebc7b8233bec6d9471c21c098700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000041eb009614c939170e9ff3d3e06c3a2c45810fe46a364ce28ecec5e220f5fd86cd6e0f70ab9093dd6b22b69980246496b600c8fcb054047962d4128efa48b692f301000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000041a31b4dd4f0a482372d75c7a8c5f11aa8084a5f358579866f1d25a26a15beb2b5153400bfa7fa3d6fba138c02dd1eb8a5a97d62178d98c5632a153396a566e5ed0000000000000000000000000000000000000000000000000000000000000000")
		ratio, err := calcCompressionRatio(data)
		assert.NoError(t, err)
		assert.NotNil(t, ratio)
		assert.Equal(t, big.NewInt(2_441_805_225), ratio) // 2.4x
	})

	t.Run("intmax-post", func(t *testing.T) {
		// https://scrollscan.com/tx/0x7244e27223cdd79ba0f0e3990c746e5d524e35dbcc200f0a7e664ffdc6d08eef
		data := common.Hex2Bytes("9b6babf0f0372bb253e060ecbdd3dbef8b832b0e743148bd807bfcf665593a56a18bac69000000000000000000000000000000000000000000000000000000006861676d0000000000000000000000000000000000000000000000000000000000000015800000000000000000000000000000000000000000000000000000000000000029a690c4ef1e18884a11f73c8595fb721f964a3e2bee809800c474278f024bcd05a76119827e6c464cee8620f616a9a23d41305eb9f9682f9d2eaf964325fcd71147783453566f27ce103a2398d96719ee22ba51b89b92cdf952af817929329403b75ae310b23cf250041d53c82bef431fa2527e2dd68b49f45f06feb2bd09f011358fe2650b8987ea2bb39bb6e28ce770f4fc9c4f064d0ae7573a1450452b501a5b0d3454d254dbf9db7094f4ca1f5056143f5c70dee4126443a6150d9e51bd05dac7e9a2bd48a8797ac6e9379d400c5ce1815b10846eaf0d80dca3a727ffd0075387e0f1bc1b363c81ecf8d05a4b654ac6fbe1cdc7c741a5c0bbeabde4138906009129ca033af12094fd7306562d9735b2fe757f021b7eb3320f8a814a286a10130969de2783e49871b80e967cfba630e6bdef2fd1d2b1076c6c3f5fd9ae5800000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000000012a98f1556efe81340fad3e59044b8139ce62f1d5d50b44b680de9422b1ddbf1a")
		ratio, err := calcCompressionRatio(data)
		assert.NoError(t, err)
		assert.NotNil(t, ratio)
		assert.Equal(t, big.NewInt(1_298_578_199), ratio) // 1.3x
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
