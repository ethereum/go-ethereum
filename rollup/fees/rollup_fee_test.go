package fees

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

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
