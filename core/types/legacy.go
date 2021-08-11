package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// v4StoredReceiptRLPWithLogs is the storage encoding of a receipt used in database version 4.
type v4StoredReceiptRLPWithLogs struct {
	PostStateOrStatus []byte
	CumulativeGasUsed uint64
	TxHash            common.Hash
	ContractAddress   common.Address
	Logs              []*legacyRlpStorageLog
	GasUsed           uint64
}

// v3StoredReceiptRLP is the original storage encoding of a receipt including some unnecessary fields.
type v3StoredReceiptRLPWithLogs struct {
	PostStateOrStatus []byte
	CumulativeGasUsed uint64
	Bloom             Bloom
	TxHash            common.Hash
	ContractAddress   common.Address
	Logs              []*legacyRlpStorageLog
	GasUsed           uint64
}

func encodeAsV4StoredReceiptRLPWithLogs(want *Receipt) ([]byte, error) {
	stored := &v4StoredReceiptRLPWithLogs{
		PostStateOrStatus: want.statusEncoding(),
		CumulativeGasUsed: want.CumulativeGasUsed,
		TxHash:            want.TxHash,
		ContractAddress:   want.ContractAddress,
		Logs:              make([]*legacyRlpStorageLog, len(want.Logs)),
		GasUsed:           want.GasUsed,
	}
	for i, log := range want.Logs {
		stored.Logs[i] = legacyFromLog(log)
	}
	return rlp.EncodeToBytes(stored)
}

func encodeAsV3StoredReceiptRLPWithLogs(want *Receipt) ([]byte, error) {
	stored := &v3StoredReceiptRLPWithLogs{
		PostStateOrStatus: want.statusEncoding(),
		CumulativeGasUsed: want.CumulativeGasUsed,
		Bloom:             want.Bloom,
		TxHash:            want.TxHash,
		ContractAddress:   want.ContractAddress,
		Logs:              make([]*legacyRlpStorageLog, len(want.Logs)),
		GasUsed:           want.GasUsed,
	}
	for i, log := range want.Logs {
		stored.Logs[i] = legacyFromLog(log)
	}
	return rlp.EncodeToBytes(stored)
}

func legacyFromLog(want *Log) *legacyRlpStorageLog {
	return &legacyRlpStorageLog{
		Address:     want.Address,
		Topics:      want.Topics,
		Data:        want.Data,
		BlockNumber: want.BlockNumber,
		TxHash:      want.TxHash,
		TxIndex:     want.TxIndex,
		BlockHash:   want.BlockHash,
		Index:       want.Index,
	}
}
