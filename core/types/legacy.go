package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// convertLegacyStoredReceipt takes a legacy RLP-encoded stored receipt
// and returns a fresh RLP-encoded stored receipt.
func convertLegacyStoredReceipt(raw []byte) ([]byte, error) {
	var receipt ReceiptForStorage
	if err := rlp.DecodeBytes(raw, &receipt); err != nil {
		return nil, err
	}
	return rlp.EncodeToBytes(&receipt)
}

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

func EncodeAsLegacyStoredReceiptsRLP(receipts []*Receipt) ([]byte, error) {
	stored := make([]v3StoredReceiptRLPWithLogs, len(receipts))
	for i, r := range receipts {
		stored[i] = *toV3StoredReceiptRLPWithLogs(r)
	}
	return rlp.EncodeToBytes(stored)
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

func toV3StoredReceiptRLPWithLogs(from *Receipt) *v3StoredReceiptRLPWithLogs {
	return &v3StoredReceiptRLPWithLogs{
		PostStateOrStatus: from.statusEncoding(),
		CumulativeGasUsed: from.CumulativeGasUsed,
		Bloom:             from.Bloom,
		TxHash:            from.TxHash,
		ContractAddress:   from.ContractAddress,
		Logs:              make([]*legacyRlpStorageLog, len(from.Logs)),
		GasUsed:           from.GasUsed,
	}
}

func encodeAsV3StoredReceiptRLPWithLogs(want *Receipt) ([]byte, error) {
	stored := toV3StoredReceiptRLPWithLogs(want)
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
