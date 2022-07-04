package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type ExportBlockReplica struct {
	Type         string
	NetworkId    uint64
	Hash         common.Hash
	TotalDiff    *big.Int
	Header       *Header
	Transactions []*TransactionExportRLP
	Uncles       []*Header
	Receipts     []*ReceiptExportRLP
	Senders      []common.Address
	State        *StateSpecimen
}

type LogsExportRLP struct {
	Address     common.Address `json:"address"`
	Topics      []common.Hash  `json:"topics"`
	Data        []byte         `json:"data"`
	BlockNumber uint64         `json:"blockNumber"`
	TxHash      common.Hash    `json:"transactionHash"`
	TxIndex     uint           `json:"transactionIndex"`
	BlockHash   common.Hash    `json:"blockHash"`
	Index       uint           `json:"logIndex"`
	Removed     bool           `json:"removed"`
}

type ReceiptForExport Receipt

type ReceiptExportRLP struct {
	PostStateOrStatus []byte
	CumulativeGasUsed uint64
	TxHash            common.Hash
	ContractAddress   common.Address
	Logs              []*LogsExportRLP
	GasUsed           uint64
}

type TransactionForExport Transaction

type TransactionExportRLP struct {
	AccountNonce uint64          `json:"nonce"`
	Price        *big.Int        `json:"gasPrice"`
	GasLimit     uint64          `json:"gas"`
	Sender       common.Address  `json:"from"`
	Recipient    *common.Address `json:"to" rlp:"nil"` // nil means contract creation
	Amount       *big.Int        `json:"value"`
	Payload      []byte          `json:"input"`
}

func (r *ReceiptForExport) ExportReceipt() *ReceiptExportRLP {
	enc := &ReceiptExportRLP{
		PostStateOrStatus: (*Receipt)(r).statusEncoding(),
		GasUsed:           r.GasUsed,
		CumulativeGasUsed: r.CumulativeGasUsed,
		TxHash:            r.TxHash,
		ContractAddress:   r.ContractAddress,
		Logs:              make([]*LogsExportRLP, len(r.Logs)),
	}
	for i, log := range r.Logs {
		enc.Logs[i] = (*LogsExportRLP)(log)
	}
	return enc
}

func (tx *TransactionForExport) ExportTx() *TransactionExportRLP {
	var inner_tx *Transaction = (*Transaction)(tx)
	var signer Signer = FrontierSigner{}

	if inner_tx.Protected() {
		signer = NewEIP155Signer(inner_tx.ChainId())
	}
	from, _ := Sender(signer, inner_tx)

	txData := tx.inner

	return &TransactionExportRLP{
		AccountNonce: txData.nonce(),
		Price:        txData.gasPrice(),
		GasLimit:     txData.gas(),
		Sender:       from,
		Recipient:    txData.to(),
		Amount:       txData.value(),
		Payload:      txData.data(),
	}
}
