package fees

import (
	"bytes"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rollup/rcfg"
)

var (
	// txExtraDataBytes is the number of bytes that we commit to L1 in addition
	// to the RLP-encoded signed transaction. Note that these are all assumed
	// to be non-zero.
	// - tx length prefix: 4 bytes
	txExtraDataBytes = uint64(4)
)

// Message represents the interface of a message.
// It should be a subset of the methods found on
// types.Message
type Message interface {
	GetFrom() common.Address
	GetTo() *common.Address
	GetGasPrice() *big.Int
	GetGasLimit() uint64
	GetGasFeeCap() *big.Int
	GetGasTipCap() *big.Int
	GetValue() *big.Int
	GetNonce() uint64
	GetData() []byte
	GetAccessList() types.AccessList
	GetIsL1MessageTx() bool
}

// StateDB represents the StateDB interface
// required to compute the L1 fee
type StateDB interface {
	GetState(common.Address, common.Hash) common.Hash
	GetBalance(addr common.Address) *big.Int
}

type gpoState struct {
	l1BaseFee     *big.Int
	overhead      *big.Int
	scalar        *big.Int
	l1BlobBaseFee *big.Int
	commitScalar  *big.Int
	blobScalar    *big.Int
}

func EstimateL1DataFeeForMessage(msg Message, baseFee *big.Int, config *params.ChainConfig, signer types.Signer, state StateDB, blockNumber *big.Int) (*big.Int, error) {
	if msg.GetIsL1MessageTx() {
		return big.NewInt(0), nil
	}

	unsigned := asUnsignedTx(msg, baseFee, config.ChainID)
	// with v=1
	tx, err := unsigned.WithSignature(signer, append(bytes.Repeat([]byte{0xff}, crypto.SignatureLength-1), 0x01))
	if err != nil {
		return nil, err
	}

	raw, err := tx.MarshalBinary()
	if err != nil {
		return nil, err
	}

	gpoState := readGPOStorageSlots(rcfg.L1GasPriceOracleAddress, state)

	var l1DataFee *big.Int

	if !config.IsCurie(blockNumber) {
		l1DataFee = calculateEncodedL1DataFee(raw, gpoState.overhead, gpoState.l1BaseFee, gpoState.scalar)
	} else {
		l1DataFee = calculateEncodedL1DataFeeCurie(raw, gpoState.l1BaseFee, gpoState.l1BlobBaseFee, gpoState.commitScalar, gpoState.blobScalar)
	}

	return l1DataFee, nil
}

// asUnsignedTx turns a Message into a types.Transaction
func asUnsignedTx(msg Message, baseFee, chainID *big.Int) *types.Transaction {
	if baseFee == nil {
		if msg.GetAccessList() == nil {
			return asUnsignedLegacyTx(msg)
		}

		return asUnsignedAccessListTx(msg, chainID)
	}

	return asUnsignedDynamicTx(msg, chainID)
}

func asUnsignedLegacyTx(msg Message) *types.Transaction {
	return types.NewTx(&types.LegacyTx{
		Nonce:    msg.GetNonce(),
		To:       msg.GetTo(),
		Value:    msg.GetValue(),
		Gas:      msg.GetGasLimit(),
		GasPrice: msg.GetGasPrice(),
		Data:     msg.GetData(),
	})
}

func asUnsignedAccessListTx(msg Message, chainID *big.Int) *types.Transaction {
	return types.NewTx(&types.AccessListTx{
		Nonce:      msg.GetNonce(),
		To:         msg.GetTo(),
		Value:      msg.GetValue(),
		Gas:        msg.GetGasLimit(),
		GasPrice:   msg.GetGasPrice(),
		Data:       msg.GetData(),
		AccessList: msg.GetAccessList(),
		ChainID:    chainID,
	})
}

func asUnsignedDynamicTx(msg Message, chainID *big.Int) *types.Transaction {
	return types.NewTx(&types.DynamicFeeTx{
		Nonce:      msg.GetNonce(),
		To:         msg.GetTo(),
		Value:      msg.GetValue(),
		Gas:        msg.GetGasLimit(),
		GasFeeCap:  msg.GetGasFeeCap(),
		GasTipCap:  msg.GetGasTipCap(),
		Data:       msg.GetData(),
		AccessList: msg.GetAccessList(),
		ChainID:    chainID,
	})
}

func readGPOStorageSlots(addr common.Address, state StateDB) gpoState {
	var gpoState gpoState
	gpoState.l1BaseFee = state.GetState(addr, rcfg.L1BaseFeeSlot).Big()
	gpoState.overhead = state.GetState(addr, rcfg.OverheadSlot).Big()
	gpoState.scalar = state.GetState(addr, rcfg.ScalarSlot).Big()
	gpoState.l1BlobBaseFee = state.GetState(addr, rcfg.L1BlobBaseFeeSlot).Big()
	gpoState.commitScalar = state.GetState(addr, rcfg.CommitScalarSlot).Big()
	gpoState.blobScalar = state.GetState(addr, rcfg.BlobScalarSlot).Big()
	return gpoState
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

func CalculateL1DataFee(tx *types.Transaction, state StateDB, config *params.ChainConfig, blockNumber *big.Int) (*big.Int, error) {
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
	} else {
		l1DataFee = calculateEncodedL1DataFeeCurie(raw, gpoState.l1BaseFee, gpoState.l1BlobBaseFee, gpoState.commitScalar, gpoState.blobScalar)
	}

	// ensure l1DataFee fits into uint64 for circuit compatibility
	// (note: in practice this value should never be this big)
	if !l1DataFee.IsUint64() {
		l1DataFee.SetUint64(math.MaxUint64)
	}

	return l1DataFee, nil
}

func GetL1BaseFee(state StateDB) *big.Int {
	return state.GetState(rcfg.L1GasPriceOracleAddress, rcfg.L1BaseFeeSlot).Big()
}
