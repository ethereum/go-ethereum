package fees

import (
	"bytes"
	"fmt"
	"math"
	"math/big"

	"github.com/holiman/uint256"
	"github.com/scroll-tech/da-codec/encoding"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/rcfg"
)

var U256MAX *big.Int

func init() {
	U256MAX, _ = new(big.Int).SetString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)
}

var (
	// txExtraDataBytes is the number of bytes that we commit to L1 in addition
	// to the RLP-encoded signed transaction. Note that these are all assumed
	// to be non-zero.
	// - tx length prefix: 4 bytes
	txExtraDataBytes = uint64(4)

	// L1 data fee cap.
	l1DataFeeCap = new(big.Int).SetUint64(math.MaxUint64)
)

func MaxL1DataFee() *big.Int {
	return new(big.Int).Set(l1DataFeeCap)
}

// Message represents the interface of a message.
// It should be a subset of the methods found on
// types.Message
type Message interface {
	From() common.Address
	To() *common.Address
	GasPrice() *big.Int
	Gas() uint64
	GasFeeCap() *big.Int
	GasTipCap() *big.Int
	Value() *big.Int
	Nonce() uint64
	Data() []byte
	AccessList() types.AccessList
	IsL1MessageTx() bool
	SetCodeAuthorizations() []types.SetCodeAuthorization
}

// StateDB represents the StateDB interface
// required to compute the L1 fee
type StateDB interface {
	GetState(common.Address, common.Hash) common.Hash
}

type gpoState struct {
	l1BaseFee        *big.Int
	overhead         *big.Int
	scalar           *big.Int
	l1BlobBaseFee    *big.Int
	commitScalar     *big.Int
	blobScalar       *big.Int
	penaltyThreshold *big.Int
	penaltyFactor    *big.Int
}

func EstimateL1DataFeeForMessage(msg Message, baseFee *big.Int, config *params.ChainConfig, signer types.Signer, state StateDB, blockNumber *big.Int, blockTime uint64) (*big.Int, error) {
	if msg.IsL1MessageTx() {
		return big.NewInt(0), nil
	}

	unsigned := asUnsignedTx(msg, baseFee, config.ChainID)
	// with v=1
	tx, err := unsigned.WithSignature(signer, append(bytes.Repeat([]byte{0xff}, crypto.SignatureLength-1), 0x01))
	if err != nil {
		return nil, err
	}

	return CalculateL1DataFee(tx, state, config, blockNumber, blockTime)
}

// asUnsignedTx turns a Message into a types.Transaction
func asUnsignedTx(msg Message, baseFee, chainID *big.Int) *types.Transaction {
	if baseFee == nil {
		if msg.AccessList() == nil {
			return asUnsignedLegacyTx(msg)
		}

		return asUnsignedAccessListTx(msg, chainID)
	}

	if msg.SetCodeAuthorizations() == nil {
		return asUnsignedDynamicTx(msg, chainID)
	}

	return asUnsignedSetCodeTx(msg, chainID)
}

func asUnsignedLegacyTx(msg Message) *types.Transaction {
	return types.NewTx(&types.LegacyTx{
		Nonce:    msg.Nonce(),
		To:       msg.To(),
		Value:    msg.Value(),
		Gas:      msg.Gas(),
		GasPrice: msg.GasPrice(),
		Data:     msg.Data(),
	})
}

func asUnsignedAccessListTx(msg Message, chainID *big.Int) *types.Transaction {
	return types.NewTx(&types.AccessListTx{
		Nonce:      msg.Nonce(),
		To:         msg.To(),
		Value:      msg.Value(),
		Gas:        msg.Gas(),
		GasPrice:   msg.GasPrice(),
		Data:       msg.Data(),
		AccessList: msg.AccessList(),
		ChainID:    chainID,
	})
}

func asUnsignedDynamicTx(msg Message, chainID *big.Int) *types.Transaction {
	return types.NewTx(&types.DynamicFeeTx{
		Nonce:      msg.Nonce(),
		To:         msg.To(),
		Value:      msg.Value(),
		Gas:        msg.Gas(),
		GasFeeCap:  msg.GasFeeCap(),
		GasTipCap:  msg.GasTipCap(),
		Data:       msg.Data(),
		AccessList: msg.AccessList(),
		ChainID:    chainID,
	})
}

func asUnsignedSetCodeTx(msg Message, chainID *big.Int) *types.Transaction {
	tx := types.SetCodeTx{
		Nonce:      msg.Nonce(),
		Value:      uint256.MustFromBig(msg.Value()),
		Gas:        msg.Gas(),
		GasFeeCap:  uint256.MustFromBig(msg.GasFeeCap()),
		GasTipCap:  uint256.MustFromBig(msg.GasTipCap()),
		Data:       msg.Data(),
		AccessList: msg.AccessList(),
		AuthList:   msg.SetCodeAuthorizations(),
		ChainID:    uint256.MustFromBig(chainID),
	}
	if msg.To() != nil {
		tx.To = *msg.To()
	}
	return types.NewTx(&tx)
}

func readGPOStorageSlots(addr common.Address, state StateDB) gpoState {
	var gpoState gpoState
	gpoState.l1BaseFee = state.GetState(addr, rcfg.L1BaseFeeSlot).Big()
	gpoState.overhead = state.GetState(addr, rcfg.OverheadSlot).Big()
	gpoState.scalar = state.GetState(addr, rcfg.ScalarSlot).Big()
	gpoState.l1BlobBaseFee = state.GetState(addr, rcfg.L1BlobBaseFeeSlot).Big()
	gpoState.commitScalar = state.GetState(addr, rcfg.CommitScalarSlot).Big()
	gpoState.blobScalar = state.GetState(addr, rcfg.BlobScalarSlot).Big()
	gpoState.penaltyThreshold = state.GetState(addr, rcfg.PenaltyThresholdSlot).Big()
	gpoState.penaltyFactor = state.GetState(addr, rcfg.PenaltyFactorSlot).Big()
	return gpoState
}

// estimateTxCompressionRatio estimates the compression ratio for `data` using da-codec
// compression_ratio(tx) = size(tx) * PRECISION / size(zstd(tx))
func estimateTxCompressionRatio(data []byte, blockNumber uint64, blockTime uint64, config *params.ChainConfig) (*big.Int, error) {
	// By definition, the compression ratio of empty data is infinity
	if len(data) == 0 {
		return U256MAX, nil
	}

	// Compress data using da-codec
	compressed, err := encoding.CompressScrollBatchBytes(data, blockNumber, blockTime, config)
	if err != nil {
		log.Error("Batch compression failed, using 1.0 compression ratio", "error", err, "data size", len(data), "data", common.Bytes2Hex(data))
		return nil, fmt.Errorf("batch compression failed: %w", err)
	}

	if len(compressed) == 0 {
		log.Error("Compressed data is empty, using 1.0 compression ratio", "data size", len(data), "data", common.Bytes2Hex(data))
		return nil, fmt.Errorf("compressed data is empty")
	}

	// Make sure compression ratio >= 1 by checking if compressed data is bigger or equal to original data
	// This behavior is consistent with DA Batch compression in codecv7 and later versions
	if len(compressed) >= len(data) {
		return rcfg.Precision, nil
	}

	// compression_ratio = size(tx) * PRECISION / size(zstd(tx))
	originalSize := new(big.Int).SetUint64(uint64(len(data)))
	compressedSize := new(big.Int).SetUint64(uint64(len(compressed)))

	ratio := new(big.Int).Mul(originalSize, rcfg.Precision)
	ratio.Div(ratio, compressedSize)

	return ratio, nil
}

// calculateTxCompressedSize calculates the size of `data` after compression using da-codec.
// We constrain compressed_size so that it cannot exceed the original size:
//
//	compressed_size(tx) = min(size(zstd(rlp(tx))), size(rlp(tx)))
//
// This provides an upper bound on the rollup fee for a given transaction, regardless
// what compression algorithm the sequencer/prover uses.
func calculateTxCompressedSize(data []byte, blockNumber uint64, blockTime uint64, config *params.ChainConfig) (*big.Int, error) {
	// Compressed size of empty data is 0.
	// In practice, the rlp-encoded transaction is always non-empty.
	if len(data) == 0 {
		return common.Big0, nil
	}

	// Compress data using da-codec
	compressed, err := encoding.CompressScrollBatchBytes(data, blockNumber, blockTime, config)
	if err != nil {
		log.Error("Transaction compression failed", "error", err, "data size", len(data), "data", common.Bytes2Hex(data), "blockNumber", blockNumber, "blockTime", blockTime, "galileoTime", config.GalileoTime)
		return nil, fmt.Errorf("transaction compression failed: %w", err)
	}

	if len(compressed) < len(data) {
		return new(big.Int).SetUint64(uint64(len(compressed))), nil
	}
	return new(big.Int).SetUint64(uint64(len(data))), nil
}

// calculatePenalty computes the penalty multiplier based on compression ratio
// penalty(tx) = compression_ratio(tx) >= penalty_threshold ? 1 * PRECISION : penalty_factor
func calculatePenalty(compressionRatio, penaltyThreshold, penaltyFactor *big.Int) *big.Int {
	if compressionRatio.Cmp(penaltyThreshold) >= 0 {
		// No penalty
		return rcfg.Precision
	}
	// Apply penalty
	return penaltyFactor
}

// calculateEncodedL1DataFee computes the L1 fee for an RLP-encoded tx
func calculateEncodedL1DataFee(data []byte, overhead, l1BaseFee *big.Int, scalar *big.Int) *big.Int {
	l1GasUsed := calculateL1GasUsed(data, overhead)
	l1DataFee := new(big.Int).Mul(l1GasUsed, l1BaseFee)
	return mulAndScale(l1DataFee, scalar, rcfg.Precision)
}

// calculateEncodedL1DataFeeCurie computes the L1 fee for an RLP-encoded tx, post Curie
func calculateEncodedL1DataFeeCurie(data []byte, l1BaseFee *big.Int, l1BlobBaseFee *big.Int, commitScalar *big.Int, blobScalar *big.Int) *big.Int {
	// calldata component of commit fees (calldata gas + execution)
	calldataGas := new(big.Int).Mul(commitScalar, l1BaseFee)

	// blob component of commit fees
	blobGas := big.NewInt(int64(len(data)))
	blobGas = new(big.Int).Mul(blobGas, l1BlobBaseFee)
	blobGas = new(big.Int).Mul(blobGas, blobScalar)

	// combined
	l1DataFee := new(big.Int).Add(calldataGas, blobGas)
	l1DataFee = new(big.Int).Quo(l1DataFee, rcfg.Precision)

	return l1DataFee
}

// calculateEncodedL1DataFeeFeynman computes the L1 fee for an RLP-encoded tx, post Feynman
//
// Post Feynman formula:
// rollup_fee(tx) = (execScalar * l1BaseFee + blobScalar * l1BlobBaseFee) * size(tx) * penalty(tx) / PRECISION / PRECISION
//
// Where:
// penalty(tx) = compression_ratio(tx) >= penalty_threshold ? 1 * PRECISION : penalty_factor
//
// compression_ratio(tx) = size(tx) * PRECISION / size(zstd(tx))
// exec_scalar = compression_scalar * (commit_scalar + verification_scalar)
// blob_scalar = compression_scalar * blob_scalar
func calculateEncodedL1DataFeeFeynman(
	data []byte,
	l1BaseFee *big.Int,
	l1BlobBaseFee *big.Int,
	execScalar *big.Int,
	blobScalar *big.Int,
	penaltyThreshold *big.Int,
	penaltyFactor *big.Int,
	compressionRatio *big.Int,
) *big.Int {
	// Calculate penalty multiplier
	penalty := calculatePenalty(compressionRatio, penaltyThreshold, penaltyFactor)

	// Transaction size (RLP-encoded)
	txSize := big.NewInt(int64(len(data)))

	// Compute gas components
	execGas := new(big.Int).Mul(execScalar, l1BaseFee)
	blobGas := new(big.Int).Mul(blobScalar, l1BlobBaseFee)

	// fee per byte = execGas + blobGas
	feePerByte := new(big.Int).Add(execGas, blobGas)

	// l1DataFee = feePerByte * txSize * penalty
	l1DataFee := new(big.Int).Mul(feePerByte, txSize)
	l1DataFee.Mul(l1DataFee, penalty)

	// Divide by rcfg.Precision (once for scalars, once for penalty)
	l1DataFee.Div(l1DataFee, rcfg.Precision) // account for scalars
	l1DataFee.Div(l1DataFee, rcfg.Precision) // accounts for penalty

	return l1DataFee
}

// calculateEncodedL1DataFeeGalileo computes the rollup fee for an RLP-encoded tx, post Galileo
//
// Post Galileo rollup fee formula:
// rollupFee(tx) = feePerByte * compressedSize(tx) * (1 + penalty(tx)) / PRECISION
//
// Where:
// feePerByte = (execScalar * l1BaseFee + blobScalar * l1BlobBaseFee)
// compressedSize(tx) = min(len(zstd(rlp(tx))), len(rlp(tx)))
// penalty(tx) = compressedSize(tx) / penaltyFactor
func calculateEncodedL1DataFeeGalileo(
	l1BaseFee *big.Int,
	l1BlobBaseFee *big.Int,
	execScalar *big.Int,
	blobScalar *big.Int,
	penaltyFactor *big.Int,
	compressedSize *big.Int,
) *big.Int {
	// Sanitize penalty factor.
	if penaltyFactor.Cmp(common.Big0) == 0 {
		penaltyFactor = common.Big1
	}

	// feePerByte = (execScalar * l1BaseFee) + (blobScalar * l1BlobBaseFee)
	execGas := new(big.Int).Mul(execScalar, l1BaseFee)
	blobGas := new(big.Int).Mul(blobScalar, l1BlobBaseFee)
	feePerByte := new(big.Int).Add(execGas, blobGas)

	// baseTerm = feePerByte * compressedSize
	baseTerm := new(big.Int).Mul(feePerByte, compressedSize)

	// penaltyTerm = (baseTerm * compressedSize) / penaltyFactor
	// Note: We divide by penaltyFactor after multiplication to preserve precision.
	penaltyTerm := new(big.Int).Mul(baseTerm, compressedSize)
	penaltyTerm.Div(penaltyTerm, penaltyFactor)

	// rollupFee = (baseTerm + penaltyTerm) / PRECISION
	rollupFee := new(big.Int).Add(baseTerm, penaltyTerm)
	rollupFee.Div(rollupFee, rcfg.Precision) // execScalar and blobScalar are scaled by PRECISION

	return rollupFee
}

// calculateL1GasUsed computes the L1 gas used based on the calldata and
// constant sized overhead. The overhead can be decreased as the cost of the
// batch submission goes down via contract optimizations. This will not overflow
// under standard network conditions.
func calculateL1GasUsed(data []byte, overhead *big.Int) *big.Int {
	zeroes, ones := zeroesAndOnes(data)
	zeroesGas := zeroes * params.TxDataZeroGas
	onesGas := (ones + txExtraDataBytes) * params.TxDataNonZeroGasEIP2028
	l1Gas := new(big.Int).SetUint64(zeroesGas + onesGas)
	return new(big.Int).Add(l1Gas, overhead)
}

// zeroesAndOnes counts the number of 0 bytes and non 0 bytes in a byte slice
func zeroesAndOnes(data []byte) (uint64, uint64) {
	var zeroes uint64
	var ones uint64
	for _, byt := range data {
		if byt == 0 {
			zeroes++
		} else {
			ones++
		}
	}
	return zeroes, ones
}

// mulAndScale multiplies a big.Int by a big.Int and then scale it by precision,
// rounded towards zero
func mulAndScale(x *big.Int, y *big.Int, precision *big.Int) *big.Int {
	z := new(big.Int).Mul(x, y)
	return new(big.Int).Quo(z, precision)
}

func CalculateL1DataFee(tx *types.Transaction, state StateDB, config *params.ChainConfig, blockNumber *big.Int, blockTime uint64) (*big.Int, error) {
	if tx.IsL1MessageTx() {
		return big.NewInt(0), nil
	}

	raw, err := tx.MarshalBinary()
	if err != nil {
		return nil, err
	}

	gpoState := readGPOStorageSlots(rcfg.L1GasPriceOracleAddress, state)

	var l1DataFee *big.Int

	if !config.IsCurie(blockNumber) {
		l1DataFee = calculateEncodedL1DataFee(raw, gpoState.overhead, gpoState.l1BaseFee, gpoState.scalar)
	} else if !config.IsFeynman(blockTime) {
		l1DataFee = calculateEncodedL1DataFeeCurie(raw, gpoState.l1BaseFee, gpoState.l1BlobBaseFee, gpoState.commitScalar, gpoState.blobScalar)
	} else if !config.IsGalileo(blockTime) {
		// Calculate compression ratio for Feynman
		// Note: We compute the transaction ratio on tx.data, not on the full encoded transaction.
		compressionRatio, err := estimateTxCompressionRatio(tx.Data(), blockNumber.Uint64(), blockTime, config)
		if err != nil {
			return nil, fmt.Errorf("failed to estimate compression ratio: tx hash=%s: %w", tx.Hash().Hex(), err)
		}

		// The contract slot for commitScalar is changed to execScalar in Feynman
		l1DataFee = calculateEncodedL1DataFeeFeynman(
			raw,
			gpoState.l1BaseFee,
			gpoState.l1BlobBaseFee,
			gpoState.commitScalar, // now represents execScalar
			gpoState.blobScalar,
			gpoState.penaltyThreshold,
			gpoState.penaltyFactor,
			compressionRatio,
		)
	} else {
		// Note: In Galileo, we take the compressed size of the full RLP-encoded transaction.
		compressedSize, err := calculateTxCompressedSize(raw, blockNumber.Uint64(), blockTime, config)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate compressed size: tx hash=%s: %w", tx.Hash().Hex(), err)
		}

		l1DataFee = calculateEncodedL1DataFeeGalileo(
			gpoState.l1BaseFee,
			gpoState.l1BlobBaseFee,
			gpoState.commitScalar, // now represents execScalar
			gpoState.blobScalar,
			gpoState.penaltyFactor, // in Galileo, penaltyFactor is repurposed as a coefficient of the blob utilization penalty
			compressedSize,
		)
	}

	// ensure l1DataFee fits into uint64 for circuit compatibility
	// (note: in practice this value should never be this big)
	if l1DataFee.Cmp(l1DataFeeCap) > 0 {
		l1DataFee = new(big.Int).Set(l1DataFeeCap)
	}

	return l1DataFee, nil
}

func GetL1BaseFee(state StateDB) *big.Int {
	return state.GetState(rcfg.L1GasPriceOracleAddress, rcfg.L1BaseFeeSlot).Big()
}
